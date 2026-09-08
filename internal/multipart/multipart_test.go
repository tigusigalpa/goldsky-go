package multipart

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"sync/atomic"
	"testing"
)

type trackingReader struct {
	*bytes.Reader
	closed atomic.Bool
}

func (r *trackingReader) Close() error {
	r.closed.Store(true)
	return nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type failingReadCloser struct {
	closed atomic.Bool
}

func (r *failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (r *failingReadCloser) Close() error {
	r.closed.Store(true)
	return nil
}

func TestNewBodyStreamsFieldsAndFile(t *testing.T) {
	fileReader := &trackingReader{Reader: bytes.NewReader([]byte("zip-data"))}
	body, err := NewBody([]Field{{Name: "description", Value: "hello"}}, File{
		FieldName:   "bundle",
		Filename:    `build "one".zip`,
		ContentType: "application/zip",
		Reader:      fileReader,
	})
	if err != nil {
		t.Fatalf("NewBody: %v", err)
	}
	defer func() { _ = body.Close() }()

	mediaType := body.ContentType()
	boundary := strings.TrimPrefix(mediaType, "multipart/form-data; boundary=")
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !fileReader.closed.Load() {
		t.Fatal("file reader was not closed")
	}
	mr := multipart.NewReader(bytes.NewReader(raw), boundary)
	form, err := mr.ReadForm(1024)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	defer func() { _ = form.RemoveAll() }()
	if got := form.Value["description"]; len(got) != 1 || got[0] != "hello" {
		t.Fatalf("description = %v", got)
	}
	files := form.File["bundle"]
	if len(files) != 1 || files[0].Filename != `build "one".zip` {
		t.Fatalf("files = %+v", files)
	}
	if got := escapeQuotes(`a\b"c`); got != `a\\b\"c` {
		t.Fatalf("escapeQuotes = %q", got)
	}
}

func TestNewBodyRejectsInvalidMetadata(t *testing.T) {
	tests := []File{
		{FieldName: "bundle\r\nx", Filename: "build.zip", Reader: strings.NewReader("x")},
		{FieldName: "bundle", Filename: "build.zip\r\nx", Reader: strings.NewReader("x")},
		{FieldName: "bundle", Filename: "build.zip", ContentType: "application/zip\r\nx", Reader: strings.NewReader("x")},
		{FieldName: "bundle", Filename: "build.zip"},
	}
	for i, file := range tests {
		if _, err := NewBody(nil, file); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
	if _, err := NewBody([]Field{{Name: "bad\nfield"}}, File{
		FieldName: "bundle", Filename: "build.zip", Reader: strings.NewReader("x"),
	}); err == nil {
		t.Fatal("expected invalid field-name error")
	}
}

func TestNewBodyPropagatesReaderFailure(t *testing.T) {
	body, err := NewBody(nil, File{
		FieldName: "bundle", Filename: "build.zip", Reader: failingReader{},
	})
	if err != nil {
		t.Fatalf("NewBody: %v", err)
	}
	defer func() { _ = body.Close() }()
	if _, err := io.ReadAll(body); err == nil || !strings.Contains(err.Error(), "read failed") {
		t.Fatalf("ReadAll error = %v", err)
	}
}

func TestNewBodyClosesFileReaderAfterCopyError(t *testing.T) {
	file := &failingReadCloser{}
	body, err := NewBody(nil, File{FieldName: "bundle", Filename: "build.zip", Reader: file})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = body.Close() }()

	if _, err := io.ReadAll(body); err == nil {
		t.Fatal("expected multipart copy error")
	}
	if !file.closed.Load() {
		t.Fatal("file reader was not closed")
	}
}

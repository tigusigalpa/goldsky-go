// Package multipart builds streaming multipart/form-data request bodies for
// subgraph deployment without buffering the entire bundle in memory.
package multipart

import (
	"io"
	"mime/multipart"
	"net/textproto"
)

// Field is a plain multipart form field sent as an ordinary value.
type Field struct {
	Name  string
	Value string
}

// File is a streaming file part. The Reader is consumed once and closed by the
// builder when it implements io.Closer.
type File struct {
	// FieldName is the multipart field name (e.g. "bundle").
	FieldName string
	// Filename is the file name reported in the part header.
	Filename string
	// ContentType is the MIME type reported in the part header; may be empty.
	ContentType string
	// Reader provides the file contents.
	Reader io.Reader
}

// Body is a streaming multipart/form-data body. Close releases the underlying
// pipe; the content type includes the generated boundary.
type Body struct {
	reader      io.ReadCloser
	contentType string
}

// Read reads from the streaming multipart body.
func (b *Body) Read(p []byte) (int, error) { return b.reader.Read(p) }

// Close releases the underlying pipe writer.
func (b *Body) Close() error { return b.reader.Close() }

// ContentType returns the multipart Content-Type header value, including the
// boundary string generated for this body.
func (b *Body) ContentType() string { return b.contentType }

// NewBody builds a streaming multipart/form-data body containing the supplied
// text fields followed by a single file part. The file's Reader is streamed and
// closed when it implements io.Closer. The returned Body must be closed when the
// request has been fully sent or cancelled.
func NewBody(fields []Field, file File) *Body {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	contentType := mw.FormDataContentType()

	go func() {
		var err error
		defer func() {
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()

		for _, f := range fields {
			if err = mw.WriteField(f.Name, f.Value); err != nil {
				return
			}
		}

		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition",
			`form-data; name="`+file.FieldName+`"; filename="`+file.Filename+`"`)
		if file.ContentType != "" {
			header.Set("Content-Type", file.ContentType)
		}
		var part io.Writer
		part, err = mw.CreatePart(header)
		if err != nil {
			return
		}
		if _, err = io.Copy(part, file.Reader); err != nil {
			return
		}
		if c, ok := file.Reader.(io.Closer); ok {
			_ = c.Close()
		}
		err = mw.Close()
	}()

	return &Body{reader: pr, contentType: contentType}
}

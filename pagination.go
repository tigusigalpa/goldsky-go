package goldsky

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// Pagination is the cursor metadata returned by paged list endpoints.
type Pagination struct {
	// NextPageToken is the cursor for the next page, or nil when the list is
	// complete. A page can hold fewer than page_size items and still have a
	// next page, so completion must be inferred from this field alone.
	NextPageToken *string `json:"next_page_token"`
	// PageSize is the page size the server applied.
	PageSize json.Number `json:"page_size"`
}

// Page is a single page of paged list results.
type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// HasMore reports whether another page is available.
func (p Page[T]) HasMore() bool {
	return p.Pagination.NextPageToken != nil && *p.Pagination.NextPageToken != ""
}

// LogRecord is a single pipeline or subgraph log line.
type LogRecord struct {
	Text      string      `json:"text"`
	Timestamp json.Number `json:"timestamp"`
	Level     string      `json:"level"`
}

// LogResults is the shape of the logs endpoint data envelope.
type LogResults struct {
	Results []LogRecord  `json:"results"`
	Cursor  *json.Number `json:"cursor,omitempty"`
}

// MetricPoint is a single time series sample in Edge endpoint metrics.
type MetricPoint struct {
	Time  string      `json:"time"`
	Value json.Number `json:"value"`
}

// EdgeMetricsData is the data envelope of the Edge metrics endpoint.
type EdgeMetricsData struct {
	Requests []MetricPoint `json:"requests"`
	Errors   []MetricPoint `json:"errors"`
}

// listPager is a generic pager over paged list endpoints. It is cancellation
// aware and never accumulates unbounded results; callers iterate page by page.
type listPager[T any] struct {
	client    *Client
	method    string
	segments  []string
	pageSize  int
	token     string
	first     bool
	queryHook func(url.Values)
}

// nextPage fetches the next page of results. It returns io.EOF-style
// completion via a nil token: when the server returns no next_page_token,
// the returned page's HasMore is false and subsequent calls return no data.
func (p *listPager[T]) nextPage(ctx context.Context) (Page[T], error) {
	q := make(url.Values)
	if p.pageSize > 0 {
		q.Set("page_size", strconv.Itoa(p.pageSize))
	}
	if p.token != "" {
		q.Set("page_token", p.token)
	}
	if p.queryHook != nil {
		p.queryHook(q)
	}
	resp, err := p.client.do(ctx, p.method, p.segments, requestOptions{query: q})
	if err != nil {
		return Page[T]{}, err
	}
	var page Page[T]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[T]{}, &TransportError{Op: p.method, Err: err}
	}
	if page.Pagination.NextPageToken != nil {
		p.token = *page.Pagination.NextPageToken
	} else {
		p.token = ""
	}
	return page, nil
}

package goldsky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// PipelineStatus is the lifecycle state of a pipeline. Known values are listed
// as constants; the wire value is preserved verbatim so future server enum
// additions do not cause decode failures.
type PipelineStatus string

const (
	PipelineStatusRunning    PipelineStatus = "RUNNING"
	PipelineStatusPaused     PipelineStatus = "PAUSED"
	PipelineStatusRestarting PipelineStatus = "RESTARTING"
	PipelineStatusDeploying  PipelineStatus = "DEPLOYING"
	PipelineStatusStopped    PipelineStatus = "STOPPED"
	PipelineStatusFailed     PipelineStatus = "FAILED"
	PipelineStatusSucceeded  PipelineStatus = "SUCCEEDED"
	PipelineStatusUnknown    PipelineStatus = "UNKNOWN"
)

// PipelineDefinition is the flexible pipeline authoring payload. Sources,
// transforms, and sinks are intentionally open object maps in the OpenAPI
// contract, so they are exposed as map[string]any.
type PipelineDefinition struct {
	Sources        map[string]any `json:"sources"`
	Transforms     map[string]any `json:"transforms"`
	Sinks          map[string]any `json:"sinks"`
	Description    string         `json:"description,omitempty"`
	ResourceSize   string         `json:"resource_size,omitempty"`
	UseDedicatedIP bool           `json:"use_dedicated_ip,omitempty"`
	Job            bool           `json:"job,omitempty"`
}

// Pipeline is a Turbo Pipeline resource.
type Pipeline struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"`
	Status       PipelineStatus     `json:"status"`
	Definition   PipelineDefinition `json:"definition"`
	ResourceSize string             `json:"resource_size,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	Version      json.Number        `json:"version,omitempty"`
	ProjectID    string             `json:"project_id,omitempty"`
}

// ListPipelinesOptions filters and pages the pipeline list.
type ListPipelinesOptions struct {
	// Type filters by pipeline type.
	Type string
	// PageSize is the page size, 1-200. Zero uses the server default.
	PageSize int
	// PageToken is the pagination cursor from a previous page.
	PageToken string
}

// CreatePipelineRequest creates a pipeline.
type CreatePipelineRequest struct {
	// Name matches ^[a-z0-9-]{1,50}$. May also be set inside Definition.
	Name           string             `json:"name,omitempty"`
	ResourceSize   string             `json:"resource_size,omitempty"`
	Description    string             `json:"description,omitempty"`
	UseDedicatedIP *bool              `json:"use_dedicated_ip,omitempty"`
	Definition     PipelineDefinition `json:"definition"`
}

// ValidatePipelineRequest validates a pipeline definition without creating it.
type ValidatePipelineRequest struct {
	Name           string             `json:"name,omitempty"`
	ResourceSize   string             `json:"resource_size,omitempty"`
	Description    string             `json:"description,omitempty"`
	UseDedicatedIP *bool              `json:"use_dedicated_ip,omitempty"`
	Definition     PipelineDefinition `json:"definition"`
}

// ValidationMessage names a single validation finding.
type ValidationMessage struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidatePipelineResponse is the result of pipeline validation.
type ValidatePipelineResponse struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationMessage `json:"errors"`
	Warnings []ValidationMessage `json:"warnings"`
}

// PreviewPipelineRequest previews a pipeline for a limited time.
type PreviewPipelineRequest struct {
	Definition PipelineDefinition `json:"definition"`
	// TTLSeconds is the preview lifetime, 1-600.
	TTLSeconds json.Number `json:"ttl_seconds,omitempty"`
}

// PreviewPipelineResponse is the result of a preview request.
type PreviewPipelineResponse struct {
	PipelineName string      `json:"pipeline_name"`
	TTLSeconds   json.Number `json:"ttl_seconds"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

// RestartPipelineRequest optionally clears pipeline state on restart.
type RestartPipelineRequest struct {
	ClearState bool `json:"clearState,omitempty"`
}

// PipelineLogsOptions filters pipeline logs.
type PipelineLogsOptions struct {
	// LogLevels is a comma-separated list of log levels.
	LogLevels string
	// Cursor is the log cursor from a previous response.
	Cursor *float64
	// After is a timestamp cursor.
	After *float64
	// Search filters log text.
	Search string
	// Direction is "asc" or "desc".
	Direction string
}

// PipelineLogsResponse is the pipeline logs endpoint envelope.
type PipelineLogsResponse struct {
	Data LogResults `json:"data"`
}

// PipelineErrorCountResponse is the pipeline error-count endpoint envelope.
type PipelineErrorCountResponse struct {
	Data struct {
		ErrorCount json.Number `json:"error_count"`
	} `json:"data"`
}

// PipelineStatusResponse is the pipeline status endpoint envelope.
type PipelineStatusResponse struct {
	Name   string         `json:"name"`
	Status PipelineStatus `json:"status"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// PipelineStateResponse wraps the raw pipeline state, whose schema is
// intentionally open in the OpenAPI contract.
type PipelineStateResponse struct {
	Data json.RawMessage `json:"data"`
}

var pipelineNameRe = regexp.MustCompile(`^[a-z0-9-]{1,50}$`)

// validatePipelineName returns an error if name does not match the documented
// pattern. It is a convenience only; server validation is authoritative.
func validatePipelineName(name string) error {
	if !pipelineNameRe.MatchString(name) {
		return fmt.Errorf("invalid pipeline name %q: must match ^[a-z0-9-]{1,50}$", name)
	}
	return nil
}

func validatePageSize(n int) error {
	if n != 0 && (n < 1 || n > 200) {
		return fmt.Errorf("page_size must be between 1 and 200, got %d", n)
	}
	return nil
}

// PipelineService manages Turbo Pipelines.
type PipelineService struct {
	client *Client
}

// List lists a single page of pipelines. Use NewPipelinePager for full
// iteration. See https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/listPipelines
func (s *PipelineService) List(ctx context.Context, opts ListPipelinesOptions) (Page[Pipeline], error) {
	if err := validatePageSize(opts.PageSize); err != nil {
		return Page[Pipeline]{}, err
	}
	q := make(url.Values)
	if opts.Type != "" {
		q.Set("type", opts.Type)
	}
	if opts.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(opts.PageSize))
	}
	if opts.PageToken != "" {
		q.Set("page_token", opts.PageToken)
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines"}, requestOptions{query: q})
	if err != nil {
		return Page[Pipeline]{}, err
	}
	var page Page[Pipeline]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[Pipeline]{}, &TransportError{Op: "listPipelines", Err: err}
	}
	return page, nil
}

// PipelinePager iterates pipeline pages, cancellation-aware.
type PipelinePager struct {
	p *listPager[Pipeline]
}

// NewPipelinePager returns a pager over pipelines starting at opts.PageToken.
func (s *PipelineService) NewPipelinePager(opts ListPipelinesOptions) *PipelinePager {
	p := &listPager[Pipeline]{
		client:   s.client,
		method:   "GET",
		segments: []string{"pipelines"},
		pageSize: opts.PageSize,
		token:    opts.PageToken,
		first:    true,
	}
	if opts.Type != "" {
		p.queryHook = func(q url.Values) { q.Set("type", opts.Type) }
	}
	return &PipelinePager{p: p}
}

// NextPage fetches the next page. When no more pages remain, the returned
// Page.HasMore is false and Data is empty on subsequent calls.
func (p *PipelinePager) NextPage(ctx context.Context) (Page[Pipeline], error) {
	return p.p.nextPage(ctx)
}

// Create creates a pipeline. See
// https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/createPipeline
func (s *PipelineService) Create(ctx context.Context, req CreatePipelineRequest) (Pipeline, error) {
	if req.Name != "" {
		if err := validatePipelineName(req.Name); err != nil {
			return Pipeline{}, err
		}
	}
	resp, err := s.client.do(ctx, "POST", []string{"pipelines"}, requestOptions{jsonBody: req})
	if err != nil {
		return Pipeline{}, err
	}
	var p Pipeline
	if err := decodeJSON(resp.body, &p); err != nil {
		return Pipeline{}, &TransportError{Op: "createPipeline", Err: err}
	}
	return p, nil
}

// Get fetches a pipeline by name. See
// https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/getPipeline
func (s *PipelineService) Get(ctx context.Context, name string) (Pipeline, error) {
	if err := validatePipelineName(name); err != nil {
		return Pipeline{}, err
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines", name}, requestOptions{})
	if err != nil {
		return Pipeline{}, err
	}
	var p Pipeline
	if err := decodeJSON(resp.body, &p); err != nil {
		return Pipeline{}, &TransportError{Op: "getPipeline", Err: err}
	}
	return p, nil
}

// Delete deletes a pipeline by name. See
// https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/deletePipeline
func (s *PipelineService) Delete(ctx context.Context, name string) error {
	if err := validatePipelineName(name); err != nil {
		return err
	}
	_, err := s.client.do(ctx, "DELETE", []string{"pipelines", name}, requestOptions{})
	return err
}

// Validate validates a pipeline definition without creating it. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Authoring/operation/validatePipeline
func (s *PipelineService) Validate(ctx context.Context, req ValidatePipelineRequest) (ValidatePipelineResponse, error) {
	if req.Name != "" {
		if err := validatePipelineName(req.Name); err != nil {
			return ValidatePipelineResponse{}, err
		}
	}
	resp, err := s.client.do(ctx, "POST", []string{"pipelines", "validate"}, requestOptions{jsonBody: req})
	if err != nil {
		return ValidatePipelineResponse{}, err
	}
	var v ValidatePipelineResponse
	if err := decodeJSON(resp.body, &v); err != nil {
		return ValidatePipelineResponse{}, &TransportError{Op: "validatePipeline", Err: err}
	}
	return v, nil
}

// Preview previews a pipeline for a limited time. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Authoring/operation/previewPipeline
func (s *PipelineService) Preview(ctx context.Context, req PreviewPipelineRequest) (PreviewPipelineResponse, error) {
	if req.TTLSeconds != "" {
		n, err := req.TTLSeconds.Int64()
		if err != nil {
			return PreviewPipelineResponse{}, fmt.Errorf("invalid ttl_seconds: %w", err)
		}
		if n < 1 || n > 600 {
			return PreviewPipelineResponse{}, fmt.Errorf("ttl_seconds must be between 1 and 600, got %d", n)
		}
	}
	resp, err := s.client.do(ctx, "POST", []string{"pipelines", "preview"}, requestOptions{jsonBody: req})
	if err != nil {
		return PreviewPipelineResponse{}, err
	}
	var p PreviewPipelineResponse
	if err := decodeJSON(resp.body, &p); err != nil {
		return PreviewPipelineResponse{}, &TransportError{Op: "previewPipeline", Err: err}
	}
	return p, nil
}

// Pause pauses a pipeline. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/pausePipeline
func (s *PipelineService) Pause(ctx context.Context, name string) error {
	if err := validatePipelineName(name); err != nil {
		return err
	}
	_, err := s.client.do(ctx, "PUT", []string{"pipelines", name, "pause"}, requestOptions{})
	return err
}

// Resume resumes a paused pipeline. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/resumePipeline
func (s *PipelineService) Resume(ctx context.Context, name string) error {
	if err := validatePipelineName(name); err != nil {
		return err
	}
	_, err := s.client.do(ctx, "PUT", []string{"pipelines", name, "resume"}, requestOptions{})
	return err
}

// Restart restarts a pipeline, optionally clearing state. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/restartPipeline
func (s *PipelineService) Restart(ctx context.Context, name string, req *RestartPipelineRequest) error {
	if err := validatePipelineName(name); err != nil {
		return err
	}
	opts := requestOptions{}
	if req != nil {
		opts.jsonBody = req
	}
	_, err := s.client.do(ctx, "PUT", []string{"pipelines", name, "restart"}, opts)
	return err
}

// Logs fetches a page of pipeline logs. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Logs/operation/getPipelineLogs
func (s *PipelineService) Logs(ctx context.Context, name string, opts PipelineLogsOptions) (PipelineLogsResponse, error) {
	if err := validatePipelineName(name); err != nil {
		return PipelineLogsResponse{}, err
	}
	q := make(url.Values)
	if opts.LogLevels != "" {
		q.Set("logLevels", opts.LogLevels)
	}
	if opts.Cursor != nil {
		q.Set("cursor", strconv.FormatFloat(*opts.Cursor, 'f', -1, 64))
	}
	if opts.After != nil {
		q.Set("after", strconv.FormatFloat(*opts.After, 'f', -1, 64))
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.Direction != "" {
		q.Set("direction", opts.Direction)
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines", name, "logs"}, requestOptions{query: q})
	if err != nil {
		return PipelineLogsResponse{}, err
	}
	var out PipelineLogsResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return PipelineLogsResponse{}, &TransportError{Op: "getPipelineLogs", Err: err}
	}
	return out, nil
}

// ErrorCount fetches the pipeline error count for the last hours. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Logs/operation/getPipelineErrorCount
func (s *PipelineService) ErrorCount(ctx context.Context, name string, sinceHours int) (PipelineErrorCountResponse, error) {
	if err := validatePipelineName(name); err != nil {
		return PipelineErrorCountResponse{}, err
	}
	if sinceHours != 0 && (sinceHours < 1 || sinceHours > 168) {
		return PipelineErrorCountResponse{}, fmt.Errorf("since_hours must be between 1 and 168, got %d", sinceHours)
	}
	q := make(url.Values)
	if sinceHours > 0 {
		q.Set("since_hours", strconv.Itoa(sinceHours))
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines", name, "logs", "error-count"}, requestOptions{query: q})
	if err != nil {
		return PipelineErrorCountResponse{}, err
	}
	var out PipelineErrorCountResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return PipelineErrorCountResponse{}, &TransportError{Op: "getPipelineErrorCount", Err: err}
	}
	return out, nil
}

// Status fetches the pipeline status. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Status/operation/getPipelineStatus
func (s *PipelineService) Status(ctx context.Context, name string) (PipelineStatusResponse, error) {
	if err := validatePipelineName(name); err != nil {
		return PipelineStatusResponse{}, err
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines", name, "status"}, requestOptions{})
	if err != nil {
		return PipelineStatusResponse{}, err
	}
	var out PipelineStatusResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return PipelineStatusResponse{}, &TransportError{Op: "getPipelineStatus", Err: err}
	}
	return out, nil
}

// State fetches the pipeline state. The OpenAPI contract leaves the state
// schema open, so the raw JSON is returned. See
// https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Status/operation/getPipelineState
func (s *PipelineService) State(ctx context.Context, name string) (PipelineStateResponse, error) {
	if err := validatePipelineName(name); err != nil {
		return PipelineStateResponse{}, err
	}
	resp, err := s.client.do(ctx, "GET", []string{"pipelines", name, "state"}, requestOptions{})
	if err != nil {
		return PipelineStateResponse{}, err
	}
	var out PipelineStateResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		// The state body may not be wrapped in {data:...}; fall back to raw.
		if len(resp.body) > 0 {
			out.Data = resp.body
		}
	}
	return out, nil
}

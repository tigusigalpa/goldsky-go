package goldsky

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/tigusigalpa/goldsky-go/internal/multipart"
)

// SubgraphStatus is the lifecycle state of a deployed subgraph version.
type SubgraphStatus string

// Known subgraph lifecycle states.
const (
	SubgraphStatusActive SubgraphStatus = "ACTIVE"
	SubgraphStatusPaused SubgraphStatus = "PAUSED"
)

// SubgraphHealth is the indexing health of a subgraph deployment.
type SubgraphHealth string

// Known subgraph indexing health values.
const (
	SubgraphHealthHealthy   SubgraphHealth = "HEALTHY"
	SubgraphHealthUnhealthy SubgraphHealth = "UNHEALTHY"
	SubgraphHealthFailed    SubgraphHealth = "FAILED"
	SubgraphHealthUnknown   SubgraphHealth = "UNKNOWN"
)

// SubgraphTag points a tag at a target version.
type SubgraphTag struct {
	TargetVersion string `json:"target_version"`
}

// IndexingProgress is the per-network sync progress of a deployment.
type IndexingProgress struct {
	Network              string      `json:"network"`
	ProgressPercent      json.Number `json:"progress_percent"`
	ChainHeadBlock       json.Number `json:"chain_head_block"`
	DeploymentHeadBlock  json.Number `json:"deployment_head_block"`
	DeploymentStartBlock json.Number `json:"deployment_start_block"`
	Synced               bool        `json:"synced"`
}

// SubgraphDeployment is a single deployment of a subgraph version.
type SubgraphDeployment struct {
	DeploymentID     string            `json:"deployment_id"`
	CreatedAt        time.Time         `json:"created_at"`
	Health           SubgraphHealth    `json:"health"`
	Synced           bool              `json:"synced"`
	FatalError       *string           `json:"fatal_error"`
	NonFatalErrors   []string          `json:"non_fatal_errors"`
	IndexingProgress *IndexingProgress `json:"indexing_progress,omitempty"`
}

// Subgraph is a subgraph version with its deployments and optional tag.
type Subgraph struct {
	Name                   string               `json:"name"`
	Version                string               `json:"version"`
	Tag                    *SubgraphTag         `json:"tag,omitempty"`
	Status                 SubgraphStatus       `json:"status"`
	Network                string               `json:"network"`
	Health                 SubgraphHealth       `json:"health"`
	Synced                 bool                 `json:"synced"`
	GraphQLEndpoint        string               `json:"graphql_endpoint"`
	PrivateGraphQLEndpoint string               `json:"private_graphql_endpoint"`
	PublicEndpointEnabled  bool                 `json:"public_endpoint_enabled"`
	PrivateEndpointEnabled bool                 `json:"private_endpoint_enabled"`
	Description            *string              `json:"description"`
	Deployments            []SubgraphDeployment `json:"deployments"`
}

// ListSubgraphsOptions pages the subgraph list.
type ListSubgraphsOptions struct {
	PageSize  int
	PageToken string
}

// SubgraphChainsResponse lists supported deployment chains.
type SubgraphChainsResponse struct {
	Data struct {
		SupportedChains []string `json:"supported_chains"`
	} `json:"data"`
}

// UpdateSubgraphVersionRequest updates endpoint settings on a version/tag.
type UpdateSubgraphVersionRequest struct {
	PublicEndpointEnabled  *bool   `json:"public_endpoint_enabled,omitempty"`
	PrivateEndpointEnabled *bool   `json:"private_endpoint_enabled,omitempty"`
	Description            *string `json:"description,omitempty"`
}

// SetSubgraphTagRequest points a tag at a target version.
type SetSubgraphTagRequest struct {
	TargetVersion string `json:"target_version"`
}

// SubgraphLogsOptions filters subgraph indexing logs.
type SubgraphLogsOptions struct {
	Cursor    *float64
	After     *float64
	Direction string
	Search    string
	LogLevel  string
	LogLevels string
}

// SubgraphLogsResponse is the subgraph logs endpoint envelope.
type SubgraphLogsResponse struct {
	Data LogResults `json:"data"`
}

// DeploySubgraphOptions deploys a compiled subgraph bundle.
//
// Bundle is streamed as a multipart file part and is never fully buffered in
// memory. The server accepts a maximum 50 MB compressed bundle and 100 MB
// extracted bundle. Overwrite is deprecated and "1" is rejected by the server;
// to replace a version, delete it and deploy again, or move a tag to it.
type DeploySubgraphOptions struct {
	// Bundle is the zip of the compiled subgraph build directory. Required.
	Bundle io.Reader
	// BundleFilename is the file name reported in the multipart part. Required.
	BundleFilename string
	// Overwrite is deprecated; "1" is rejected by the server. Omit or set "0".
	Overwrite string
	// RemoveGraft set to "1" strips the graft from the manifest before deploy.
	RemoveGraft string
	// SkipGraftValidation set to "1" skips validation of the graft base.
	SkipGraftValidation string
	// StartBlock is a block number string to start indexing from.
	StartBlock string
	// GraftFrom is "name/version" of an existing subgraph to graft from.
	GraftFrom string
	// Description is a human-readable description (max 500 characters).
	Description string
	// GraphNodeShard pins the deployment to a specific indexing shard. Advanced.
	GraphNodeShard string
}

// SubgraphService manages Subgraphs and their versions, tags, and deployments.
type SubgraphService struct {
	client *Client
}

// List lists a single page of subgraphs. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/listSubgraphs
func (s *SubgraphService) List(ctx context.Context, opts ListSubgraphsOptions) (Page[Subgraph], error) {
	if err := validatePageSize(opts.PageSize); err != nil {
		return Page[Subgraph]{}, err
	}
	q := make(url.Values)
	if opts.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(opts.PageSize))
	}
	if opts.PageToken != "" {
		q.Set("page_token", opts.PageToken)
	}
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs"}, requestOptions{query: q})
	if err != nil {
		return Page[Subgraph]{}, err
	}
	var page Page[Subgraph]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[Subgraph]{}, &TransportError{Op: "listSubgraphs", Err: err}
	}
	return page, nil
}

// SubgraphPager iterates subgraph pages, cancellation-aware.
type SubgraphPager struct {
	p *listPager[Subgraph]
}

// NewSubgraphPager returns a pager over subgraphs starting at opts.PageToken.
func (s *SubgraphService) NewSubgraphPager(opts ListSubgraphsOptions) *SubgraphPager {
	return &SubgraphPager{p: &listPager[Subgraph]{
		client:   s.client,
		method:   "GET",
		segments: []string{"subgraphs"},
		pageSize: opts.PageSize,
		token:    opts.PageToken,
	}}
}

// NextPage fetches the next page.
func (p *SubgraphPager) NextPage(ctx context.Context) (Page[Subgraph], error) {
	return p.p.nextPage(ctx)
}

// Get fetches a subgraph with its versions and tags. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/getSubgraph
func (s *SubgraphService) Get(ctx context.Context, name string) (Page[Subgraph], error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", name}, requestOptions{})
	if err != nil {
		return Page[Subgraph]{}, err
	}
	// getSubgraph returns {data: [...], pagination:{...}}.
	var page Page[Subgraph]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[Subgraph]{}, &TransportError{Op: "getSubgraph", Err: err}
	}
	return page, nil
}

// SupportedChains lists supported deployment chains. See
// https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listSubgraphChains
func (s *SubgraphService) SupportedChains(ctx context.Context) (SubgraphChainsResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", "supported-chains"}, requestOptions{})
	if err != nil {
		return SubgraphChainsResponse{}, err
	}
	var out SubgraphChainsResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return SubgraphChainsResponse{}, &TransportError{Op: "listSubgraphChains", Err: err}
	}
	return out, nil
}

// GetVersion fetches a subgraph tag or deployed version. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/getSubgraphVersion
func (s *SubgraphService) GetVersion(ctx context.Context, name, version string) (Page[Subgraph], error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", name, version}, requestOptions{})
	if err != nil {
		return Page[Subgraph]{}, err
	}
	var page Page[Subgraph]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[Subgraph]{}, &TransportError{Op: "getSubgraphVersion", Err: err}
	}
	return page, nil
}

// UpdateVersion updates endpoint settings on a version or tag. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/updateSubgraphVersion
func (s *SubgraphService) UpdateVersion(ctx context.Context, name, version string, req UpdateSubgraphVersionRequest) (Subgraph, error) {
	resp, err := s.client.do(ctx, "PATCH", []string{"subgraphs", name, version}, requestOptions{jsonBody: req})
	if err != nil {
		return Subgraph{}, err
	}
	var out struct {
		Data Subgraph `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return Subgraph{}, &TransportError{Op: "updateSubgraphVersion", Err: err}
	}
	return out.Data, nil
}

// Logs fetches a page of subgraph indexing logs. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Logs/operation/getSubgraphLogs
func (s *SubgraphService) Logs(ctx context.Context, name, version string, opts SubgraphLogsOptions) (SubgraphLogsResponse, error) {
	q := make(url.Values)
	if opts.Cursor != nil {
		q.Set("cursor", strconv.FormatFloat(*opts.Cursor, 'f', -1, 64))
	}
	if opts.After != nil {
		q.Set("after", strconv.FormatFloat(*opts.After, 'f', -1, 64))
	}
	if opts.Direction != "" {
		q.Set("direction", opts.Direction)
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.LogLevel != "" {
		q.Set("log_level", opts.LogLevel)
	}
	if opts.LogLevels != "" {
		q.Set("log_levels", opts.LogLevels)
	}
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", name, version, "logs"}, requestOptions{query: q})
	if err != nil {
		return SubgraphLogsResponse{}, err
	}
	var out SubgraphLogsResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return SubgraphLogsResponse{}, &TransportError{Op: "getSubgraphLogs", Err: err}
	}
	return out, nil
}

// Pause pauses a deployed subgraph version. Pause/resume targets a deployed
// version, not a moving tag. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/pauseSubgraph
func (s *SubgraphService) Pause(ctx context.Context, name, version string) error {
	_, err := s.client.do(ctx, "PUT", []string{"subgraphs", name, version, "pause"}, requestOptions{})
	return err
}

// Resume resumes a paused deployed subgraph version. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/resumeSubgraph
func (s *SubgraphService) Resume(ctx context.Context, name, version string) error {
	_, err := s.client.do(ctx, "PUT", []string{"subgraphs", name, version, "resume"}, requestOptions{})
	return err
}

// SetTag creates or moves a tag to a target version. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Tags/operation/setSubgraphTag
func (s *SubgraphService) SetTag(ctx context.Context, name, version string, req SetSubgraphTagRequest) (Subgraph, error) {
	resp, err := s.client.do(ctx, "PUT", []string{"subgraphs", name, "tags", version}, requestOptions{jsonBody: req})
	if err != nil {
		return Subgraph{}, err
	}
	var out struct {
		Data Subgraph `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return Subgraph{}, &TransportError{Op: "setSubgraphTag", Err: err}
	}
	return out.Data, nil
}

// DeleteTag deletes a tag. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Tags/operation/deleteSubgraphTag
func (s *SubgraphService) DeleteTag(ctx context.Context, name, version string) error {
	_, err := s.client.do(ctx, "DELETE", []string{"subgraphs", name, "tags", version}, requestOptions{})
	return err
}

// DeleteDeployment deletes a deployment. This fails with HTTP 422 while the
// deployment is referenced by a tag, pipeline, or webhook. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Deployments/operation/deleteSubgraphDeployment
func (s *SubgraphService) DeleteDeployment(ctx context.Context, name, version string) error {
	_, err := s.client.do(ctx, "DELETE", []string{"subgraphs", name, "deployments", version}, requestOptions{})
	return err
}

// Deploy deploys a compiled subgraph bundle as streaming multipart/form-data.
// See https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Deployments/operation/deploySubgraph
func (s *SubgraphService) Deploy(ctx context.Context, name, version string, opts DeploySubgraphOptions) (Subgraph, error) {
	if opts.Bundle == nil {
		return Subgraph{}, fmt.Errorf("goldsky: Deploy requires a Bundle reader")
	}
	if opts.BundleFilename == "" {
		return Subgraph{}, fmt.Errorf("goldsky: Deploy requires a BundleFilename")
	}
	if opts.Overwrite == "1" {
		return Subgraph{}, fmt.Errorf("goldsky: overwrite=1 is rejected by the server; delete the version and redeploy, or move a tag")
	}

	var fields []multipart.Field
	if opts.Overwrite != "" {
		fields = append(fields, multipart.Field{Name: "overwrite", Value: opts.Overwrite})
	}
	if opts.RemoveGraft != "" {
		fields = append(fields, multipart.Field{Name: "remove_graft", Value: opts.RemoveGraft})
	}
	if opts.SkipGraftValidation != "" {
		fields = append(fields, multipart.Field{Name: "skip_graft_validation", Value: opts.SkipGraftValidation})
	}
	if opts.StartBlock != "" {
		fields = append(fields, multipart.Field{Name: "start_block", Value: opts.StartBlock})
	}
	if opts.GraftFrom != "" {
		fields = append(fields, multipart.Field{Name: "graft_from", Value: opts.GraftFrom})
	}
	if opts.Description != "" {
		fields = append(fields, multipart.Field{Name: "description", Value: opts.Description})
	}
	if opts.GraphNodeShard != "" {
		fields = append(fields, multipart.Field{Name: "graph_node_shard", Value: opts.GraphNodeShard})
	}

	body, err := multipart.NewBody(fields, multipart.File{
		FieldName:   "bundle",
		Filename:    opts.BundleFilename,
		ContentType: "application/zip",
		Reader:      opts.Bundle,
	})
	if err != nil {
		return Subgraph{}, fmt.Errorf("goldsky: build deployment body: %w", err)
	}
	defer body.Close()

	resp, err := s.client.do(ctx, "PUT", []string{"subgraphs", name, "deployments", version}, requestOptions{multipart: body})
	if err != nil {
		return Subgraph{}, err
	}
	var out Subgraph
	if err := decodeJSON(resp.body, &out); err != nil {
		return Subgraph{}, &TransportError{Op: "deploySubgraph", Err: err}
	}
	return out, nil
}

// WebhookEntityColumn describes a column of a webhook-able entity.
type WebhookEntityColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// WebhookEntity is a subgraph entity (table) available for webhooks.
type WebhookEntity struct {
	Name    string                `json:"name"`
	Rows    string                `json:"rows"`
	Columns []WebhookEntityColumn `json:"columns"`
}

// WebhookEntitiesResponse lists webhook-able entities for a subgraph version.
type WebhookEntitiesResponse struct {
	Data struct {
		Entities []WebhookEntity `json:"entities"`
	} `json:"data"`
}

// WebhookEntities lists webhook-able entities for a subgraph version. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/listWebhookEntities
func (s *SubgraphService) WebhookEntities(ctx context.Context, name, version string) (WebhookEntitiesResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", name, version, "entities"}, requestOptions{})
	if err != nil {
		return WebhookEntitiesResponse{}, err
	}
	var out WebhookEntitiesResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return WebhookEntitiesResponse{}, &TransportError{Op: "listWebhookEntities", Err: err}
	}
	return out, nil
}

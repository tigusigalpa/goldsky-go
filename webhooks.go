package goldsky

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Webhook is a Subgraph entity webhook. The delivery secret is returned only at
// create time and is not present in list responses.
type Webhook struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	WebhookURL      string    `json:"webhook_url"`
	Entity          string    `json:"entity"`
	SubgraphName    string    `json:"subgraph_name"`
	SubgraphVersion string    `json:"subgraph_version"`
	CreatedAt       time.Time `json:"created_at"`
}

// WebhookListResponse lists project webhooks.
type WebhookListResponse struct {
	Data []Webhook `json:"data"`
}

// CreateWebhookRequest creates an entity webhook.
//
// Secret is sent as the goldsky-webhook-secret header on every delivery. If
// omitted, the server generates one and returns it once in the create response.
// num_retries is 0-10; retry_interval_seconds and retry_timeout_seconds are >=1.
type CreateWebhookRequest struct {
	Name                 string  `json:"name"`
	SubgraphName         string  `json:"subgraph_name"`
	SubgraphVersion      string  `json:"subgraph_version"`
	Entity               string  `json:"entity"`
	WebhookURL           string  `json:"webhook_url"`
	Secret               *string `json:"secret,omitempty"`
	NumRetries           *int    `json:"num_retries,omitempty"`
	RetryIntervalSeconds *int    `json:"retry_interval_seconds,omitempty"`
	RetryTimeoutSeconds  *int    `json:"retry_timeout_seconds,omitempty"`
}

// CreateWebhookResponse is the create response. WebhookSecret is a one-time
// secret: it is returned here and via no other endpoint, and must be stored by
// the caller immediately. It is never logged by the SDK.
type CreateWebhookResponse struct {
	Data struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		WebhookSecret string `json:"webhook_secret"`
	} `json:"data"`
}

// WebhookService manages Subgraph entity webhooks.
type WebhookService struct {
	client *Client
}

// List lists project webhooks. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/listWebhooks
func (s *WebhookService) List(ctx context.Context) (WebhookListResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"subgraphs", "webhooks"}, requestOptions{})
	if err != nil {
		return WebhookListResponse{}, err
	}
	var out WebhookListResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return WebhookListResponse{}, &TransportError{Op: "listWebhooks", Err: err}
	}
	return out, nil
}

// Create creates an entity webhook and returns the one-time delivery secret.
// See https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/createWebhook
func (s *WebhookService) Create(ctx context.Context, req CreateWebhookRequest) (CreateWebhookResponse, error) {
	if err := validateResourceName("webhook", req.Name); err != nil {
		return CreateWebhookResponse{}, err
	}
	if len(req.Name) > 42 {
		return CreateWebhookResponse{}, fmt.Errorf("goldsky: webhook name must be at most 42 characters")
	}
	if err := validateSubgraphTarget(req.SubgraphName, req.SubgraphVersion); err != nil {
		return CreateWebhookResponse{}, err
	}
	if strings.TrimSpace(req.Entity) == "" {
		return CreateWebhookResponse{}, fmt.Errorf("goldsky: webhook entity is required")
	}
	webhookURL, err := url.ParseRequestURI(req.WebhookURL)
	if err != nil || (webhookURL.Scheme != "http" && webhookURL.Scheme != "https") || webhookURL.Host == "" {
		return CreateWebhookResponse{}, fmt.Errorf("goldsky: webhook URL must be an absolute HTTP(S) URL")
	}
	if req.NumRetries != nil && (*req.NumRetries < 0 || *req.NumRetries > 10) {
		return CreateWebhookResponse{}, fmt.Errorf("num_retries must be between 0 and 10, got %d", *req.NumRetries)
	}
	if req.RetryIntervalSeconds != nil && *req.RetryIntervalSeconds < 1 {
		return CreateWebhookResponse{}, fmt.Errorf("retry_interval_seconds must be >= 1")
	}
	if req.RetryTimeoutSeconds != nil && *req.RetryTimeoutSeconds < 1 {
		return CreateWebhookResponse{}, fmt.Errorf("retry_timeout_seconds must be >= 1")
	}
	resp, err := s.client.do(ctx, "POST", []string{"subgraphs", "webhooks"}, requestOptions{jsonBody: req})
	if err != nil {
		return CreateWebhookResponse{}, err
	}
	var out CreateWebhookResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return CreateWebhookResponse{}, &TransportError{Op: "createWebhook", Err: err}
	}
	return out, nil
}

// Delete deletes a webhook by name. See
// https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/deleteWebhook
func (s *WebhookService) Delete(ctx context.Context, name string) error {
	if err := validateResourceName("webhook", name); err != nil {
		return err
	}
	_, err := s.client.do(ctx, "DELETE", []string{"subgraphs", "webhooks", name}, requestOptions{})
	return err
}

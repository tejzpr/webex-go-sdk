/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// Webhook represents a Webex webhook
type Webhook struct {
	ID        string                  `json:"id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	TargetURL string                  `json:"targetUrl,omitempty"`
	Resource  string                  `json:"resource,omitempty"`
	Event     string                  `json:"event,omitempty"`
	Filter    string                  `json:"filter,omitempty"`
	Secret    string                  `json:"secret,omitempty"`
	Status    string                  `json:"status,omitempty"`
	Created   *time.Time              `json:"created,omitempty"`
	Errors    webexsdk.ResourceErrors `json:"errors,omitempty"`
}

// ListOptions contains the options for listing webhooks
type ListOptions struct {
	Max int `url:"max,omitempty"`
}

// WebhooksPage represents a paginated list of webhooks
type WebhooksPage struct {
	Items []Webhook `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Webhooks plugin
type Config struct {
	// Any configuration settings for the webhooks plugin can go here
}

// DefaultConfig returns the default configuration for the Webhooks plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the webhooks API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Webhooks plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of webhooks
func (c *Client) List(options *ListOptions) (*WebhooksPage, error) {
	params := url.Values{}
	if options != nil && options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "webhooks", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "webhooks")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Webhooks
	webhooksPage := &WebhooksPage{
		Page:  page,
		Items: make([]Webhook, len(page.Items)),
	}

	for i, item := range page.Items {
		var webhook Webhook
		if err := json.Unmarshal(item, &webhook); err != nil {
			return nil, err
		}
		webhooksPage.Items[i] = webhook
	}

	return webhooksPage, nil
}

// Create creates a new webhook
func (c *Client) Create(webhook *Webhook) (*Webhook, error) {
	if webhook.Name == "" {
		return nil, fmt.Errorf("webhook name is required")
	}
	if webhook.TargetURL == "" {
		return nil, fmt.Errorf("webhook targetUrl is required")
	}
	if webhook.Resource == "" {
		return nil, fmt.Errorf("webhook resource is required")
	}
	if webhook.Event == "" {
		return nil, fmt.Errorf("webhook event is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "webhooks", nil, webhook)
	if err != nil {
		return nil, err
	}

	var result Webhook
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns details for a webhook
func (c *Client) Get(webhookID string) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}

	path := fmt.Sprintf("webhooks/%s", webhookID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := webexsdk.ParseResponse(resp, &webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// Update updates an existing webhook
func (c *Client) Update(webhookID string, webhook *Webhook) (*Webhook, error) {
	if webhookID == "" {
		return nil, fmt.Errorf("webhookID is required")
	}
	if webhook.Name == "" {
		return nil, fmt.Errorf("webhook name is required")
	}
	if webhook.TargetURL == "" {
		return nil, fmt.Errorf("webhook targetUrl is required")
	}
	// Status is optional, but if provided, validate it
	if webhook.Status != "" && webhook.Status != "active" && webhook.Status != "inactive" {
		return nil, fmt.Errorf("webhook status must be either 'active' or 'inactive'")
	}

	// Create a new webhook object with only the allowed fields for updates
	updateData := &struct {
		Name      string `json:"name"`
		TargetURL string `json:"targetUrl"`
		Secret    string `json:"secret,omitempty"`
		Status    string `json:"status,omitempty"`
	}{
		Name:      webhook.Name,
		TargetURL: webhook.TargetURL,
	}

	// Only include optional fields if they're set
	if webhook.Secret != "" {
		updateData.Secret = webhook.Secret
	}
	if webhook.Status != "" {
		updateData.Status = webhook.Status
	}

	path := fmt.Sprintf("webhooks/%s", webhookID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, updateData)
	if err != nil {
		return nil, err
	}

	var result Webhook
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a webhook
func (c *Client) Delete(webhookID string) error {
	if webhookID == "" {
		return fmt.Errorf("webhookID is required")
	}

	path := fmt.Sprintf("webhooks/%s", webhookID)
	resp, err := c.webexClient.Request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// For DELETE operations, we just check the status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// NewUpdateWebhook creates a webhook struct with only the fields that can be updated
func NewUpdateWebhook(name, targetURL, secret string, status string) *Webhook {
	return &Webhook{
		Name:      name,
		TargetURL: targetURL,
		Secret:    secret,
		Status:    status,
	}
}

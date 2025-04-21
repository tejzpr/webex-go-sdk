/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package attachmentactions

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

// AttachmentAction represents a Webex attachment action
type AttachmentAction struct {
	ID        string                 `json:"id,omitempty"`
	Type      string                 `json:"type,omitempty"`
	MessageID string                 `json:"messageId,omitempty"`
	Inputs    map[string]interface{} `json:"inputs,omitempty"`
	PersonID  string                 `json:"personId,omitempty"`
	RoomID    string                 `json:"roomId,omitempty"`
	Created   *time.Time             `json:"created,omitempty"`
}

// Config holds the configuration for the AttachmentActions plugin
type Config struct {
	// Any configuration settings for the attachment actions plugin can go here
}

// DefaultConfig returns the default configuration for the AttachmentActions plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the attachment actions API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new AttachmentActions plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// Create submits an attachment action for a message with an adaptive card
func (c *Client) Create(action *AttachmentAction) (*AttachmentAction, error) {
	if action.MessageID == "" {
		return nil, fmt.Errorf("messageId is required")
	}

	if action.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "attachment/actions", nil, action)
	if err != nil {
		return nil, err
	}

	var result AttachmentAction
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns a single attachment action by ID
func (c *Client) Get(actionID string) (*AttachmentAction, error) {
	if actionID == "" {
		return nil, fmt.Errorf("actionID is required")
	}

	path := fmt.Sprintf("attachment/actions/%s", actionID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var action AttachmentAction
	if err := webexsdk.ParseResponse(resp, &action); err != nil {
		return nil, err
	}

	return &action, nil
}

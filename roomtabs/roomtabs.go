/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package roomtabs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// RoomTab represents a Webex room tab
type RoomTab struct {
	ID          string                  `json:"id,omitempty"`
	RoomID      string                  `json:"roomId,omitempty"`
	RoomType    string                  `json:"roomType,omitempty"`
	DisplayName string                  `json:"displayName,omitempty"`
	ContentURL  string                  `json:"contentUrl,omitempty"`
	CreatorID   string                  `json:"creatorId,omitempty"`
	Created     *time.Time              `json:"created,omitempty"`
	Errors      webexsdk.ResourceErrors `json:"errors,omitempty"`
}

// ListOptions contains the options for listing room tabs
type ListOptions struct {
	RoomID string `url:"roomId,omitempty"`
}

// RoomTabsPage represents a paginated list of room tabs
type RoomTabsPage struct {
	Items []RoomTab `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the RoomTabs plugin
type Config struct {
	// Any configuration settings for the roomtabs plugin can go here
}

// DefaultConfig returns the default configuration for the RoomTabs plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the room tabs API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new RoomTabs plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of room tabs for a specified room
func (c *Client) List(options *ListOptions) (*RoomTabsPage, error) {
	if options == nil || options.RoomID == "" {
		return nil, fmt.Errorf("roomId is required")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("roomId", options.RoomID)

	resp, err := c.webexClient.Request(http.MethodGet, "room/tabs", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, webexsdk.ResourceRoomTabs)
	if err != nil {
		return nil, err
	}

	// Unmarshal items into RoomTabs
	roomTabsPage := &RoomTabsPage{
		Page:  page,
		Items: make([]RoomTab, len(page.Items)),
	}

	for i, item := range page.Items {
		var roomTab RoomTab
		if err := json.Unmarshal(item, &roomTab); err != nil {
			return nil, err
		}
		roomTabsPage.Items[i] = roomTab
	}

	return roomTabsPage, nil
}

// Create creates a new room tab
func (c *Client) Create(tab *RoomTab) (*RoomTab, error) {
	if tab.RoomID == "" {
		return nil, fmt.Errorf("roomId is required")
	}
	if tab.ContentURL == "" {
		return nil, fmt.Errorf("contentUrl is required")
	}
	if tab.DisplayName == "" {
		return nil, fmt.Errorf("displayName is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "room/tabs", nil, tab)
	if err != nil {
		return nil, err
	}

	var result RoomTab
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns details for a room tab
func (c *Client) Get(tabID string) (*RoomTab, error) {
	if tabID == "" {
		return nil, fmt.Errorf("tabID is required")
	}

	path := fmt.Sprintf("room/tabs/%s", tabID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var roomTab RoomTab
	if err := webexsdk.ParseResponse(resp, &roomTab); err != nil {
		return nil, err
	}

	return &roomTab, nil
}

// Update updates an existing room tab
func (c *Client) Update(tabID string, tab *RoomTab) (*RoomTab, error) {
	if tabID == "" {
		return nil, fmt.Errorf("tabID is required")
	}
	if tab.RoomID == "" {
		return nil, fmt.Errorf("roomId is required")
	}
	if tab.ContentURL == "" {
		return nil, fmt.Errorf("contentUrl is required")
	}
	if tab.DisplayName == "" {
		return nil, fmt.Errorf("displayName is required")
	}

	path := fmt.Sprintf("room/tabs/%s", tabID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, tab)
	if err != nil {
		return nil, err
	}

	var result RoomTab
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a room tab
func (c *Client) Delete(tabID string) error {
	if tabID == "" {
		return fmt.Errorf("tabID is required")
	}

	path := fmt.Sprintf("room/tabs/%s", tabID)
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

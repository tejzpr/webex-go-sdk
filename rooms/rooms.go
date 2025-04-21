/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package rooms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Room represents a Webex room
type Room struct {
	ID           string     `json:"id,omitempty"`
	Title        string     `json:"title,omitempty"`
	TeamID       string     `json:"teamId,omitempty"`
	IsLocked     bool       `json:"isLocked,omitempty"`
	Type         string     `json:"type,omitempty"`
	CreatorID    string     `json:"creatorId,omitempty"`
	Created      *time.Time `json:"created,omitempty"`
	LastActivity *time.Time `json:"lastActivity,omitempty"`
}

// RoomWithReadStatus represents a room with read status information
type RoomWithReadStatus struct {
	ID               string     `json:"id,omitempty"`
	Title            string     `json:"title,omitempty"`
	Type             string     `json:"type,omitempty"`
	IsLocked         bool       `json:"isLocked,omitempty"`
	TeamID           string     `json:"teamId,omitempty"`
	LastActivityDate *time.Time `json:"lastActivityDate,omitempty"`
	LastSeenDate     *time.Time `json:"lastSeenDate,omitempty"`
}

// ListOptions contains the options for listing rooms
type ListOptions struct {
	TeamID string `url:"teamId,omitempty"`
	Type   string `url:"type,omitempty"`
	SortBy string `url:"sortBy,omitempty"`
	Max    int    `url:"max,omitempty"`
}

// RoomsPage represents a paginated list of rooms
type RoomsPage struct {
	Items []Room `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Rooms plugin
type Config struct {
	// Any configuration settings for the rooms plugin can go here
}

// DefaultConfig returns the default configuration for the Rooms plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the rooms API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Rooms plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// Create creates a new room
func (c *Client) Create(room *Room) (*Room, error) {
	if room.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "rooms", nil, room)
	if err != nil {
		return nil, err
	}

	var result Room
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns a single room by ID
func (c *Client) Get(roomID string) (*Room, error) {
	if roomID == "" {
		return nil, fmt.Errorf("roomID is required")
	}

	path := fmt.Sprintf("rooms/%s", roomID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var room Room
	if err := webexsdk.ParseResponse(resp, &room); err != nil {
		return nil, err
	}

	return &room, nil
}

// List returns a list of rooms
func (c *Client) List(options *ListOptions) (*RoomsPage, error) {
	if options == nil {
		options = &ListOptions{}
	}

	// Build query parameters
	params := url.Values{}

	if options.TeamID != "" {
		params.Set("teamId", options.TeamID)
	}

	if options.Type != "" {
		params.Set("type", options.Type)
	}

	if options.SortBy != "" {
		params.Set("sortBy", options.SortBy)
	}

	if options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "rooms", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "rooms")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Rooms
	roomsPage := &RoomsPage{
		Page:  page,
		Items: make([]Room, len(page.Items)),
	}

	for i, item := range page.Items {
		var room Room
		if err := json.Unmarshal(item, &room); err != nil {
			return nil, err
		}
		roomsPage.Items[i] = room
	}

	return roomsPage, nil
}

// Update updates an existing room
func (c *Client) Update(roomID string, room *Room) (*Room, error) {
	if roomID == "" {
		return nil, fmt.Errorf("roomID is required")
	}

	path := fmt.Sprintf("rooms/%s", roomID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, room)
	if err != nil {
		return nil, err
	}

	var result Room
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete removes a room
func (c *Client) Delete(roomID string) error {
	if roomID == "" {
		return fmt.Errorf("roomID is required")
	}

	path := fmt.Sprintf("rooms/%s", roomID)
	resp, err := c.webexClient.Request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	// For DELETE operations, we just check the status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package memberships

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// Membership represents a Webex membership
type Membership struct {
	ID                string                  `json:"id,omitempty"`
	RoomID            string                  `json:"roomId,omitempty"`
	PersonID          string                  `json:"personId,omitempty"`
	PersonEmail       string                  `json:"personEmail,omitempty"`
	PersonDisplayName string                  `json:"personDisplayName,omitempty"`
	PersonOrgID       string                  `json:"personOrgId,omitempty"`
	IsModerator       bool                    `json:"isModerator,omitempty"`
	IsMonitor         bool                    `json:"isMonitor,omitempty"`
	IsRoomHidden      bool                    `json:"isRoomHidden,omitempty"`
	Created           *time.Time              `json:"created,omitempty"`
	RoomType          string                  `json:"roomType,omitempty"`
	LastSeenID        string                  `json:"lastSeenId,omitempty"`
	LastSeenDate      *time.Time              `json:"lastSeenDate,omitempty"`
	Errors            webexsdk.ResourceErrors `json:"errors,omitempty"`
}

// ListOptions contains the options for listing memberships
type ListOptions struct {
	RoomID      string `url:"roomId,omitempty"`
	PersonID    string `url:"personId,omitempty"`
	PersonEmail string `url:"personEmail,omitempty"`
	Max         int    `url:"max,omitempty"`
}

// MembershipsPage represents a paginated list of memberships
type MembershipsPage struct {
	Items []Membership `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Memberships plugin
type Config struct {
	// Any configuration settings for the memberships plugin can go here
}

// DefaultConfig returns the default configuration for the Memberships plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the memberships API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Memberships plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// Create adds a person to a room
func (c *Client) Create(membership *Membership) (*Membership, error) {
	if membership.RoomID == "" {
		return nil, fmt.Errorf("roomId is required")
	}

	if membership.PersonID == "" && membership.PersonEmail == "" {
		return nil, fmt.Errorf("either personId or personEmail is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "memberships", nil, membership)
	if err != nil {
		return nil, err
	}

	var result Membership
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns a single membership by ID
func (c *Client) Get(membershipID string) (*Membership, error) {
	if membershipID == "" {
		return nil, fmt.Errorf("membershipID is required")
	}

	path := fmt.Sprintf("memberships/%s", membershipID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var membership Membership
	if err := webexsdk.ParseResponse(resp, &membership); err != nil {
		return nil, err
	}

	return &membership, nil
}

// List returns a list of memberships
func (c *Client) List(options *ListOptions) (*MembershipsPage, error) {
	if options == nil {
		options = &ListOptions{}
	}

	// Build query parameters
	params := url.Values{}

	if options.RoomID != "" {
		params.Set("roomId", options.RoomID)
	}

	if options.PersonID != "" {
		params.Set("personId", options.PersonID)
	}

	if options.PersonEmail != "" {
		params.Set("personEmail", options.PersonEmail)
	}

	if options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "memberships", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "memberships")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Memberships
	membershipsPage := &MembershipsPage{
		Page:  page,
		Items: make([]Membership, len(page.Items)),
	}

	for i, item := range page.Items {
		var membership Membership
		if err := json.Unmarshal(item, &membership); err != nil {
			return nil, err
		}
		membershipsPage.Items[i] = membership
	}

	return membershipsPage, nil
}

// Update updates an existing membership
func (c *Client) Update(membershipID string, membership *Membership) (*Membership, error) {
	if membershipID == "" {
		return nil, fmt.Errorf("membershipID is required")
	}

	path := fmt.Sprintf("memberships/%s", membershipID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, membership)
	if err != nil {
		return nil, err
	}

	var result Membership
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete removes a person from a room
func (c *Client) Delete(membershipID string) error {
	if membershipID == "" {
		return fmt.Errorf("membershipID is required")
	}

	path := fmt.Sprintf("memberships/%s", membershipID)
	resp, err := c.webexClient.Request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// For DELETE operations, we just check the status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

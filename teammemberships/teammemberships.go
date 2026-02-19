/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package teammemberships

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v1/webexsdk"
)

// TeamMembership represents a Webex team membership
type TeamMembership struct {
	ID                string     `json:"id,omitempty"`
	TeamID            string     `json:"teamId,omitempty"`
	PersonID          string     `json:"personId,omitempty"`
	PersonEmail       string     `json:"personEmail,omitempty"`
	PersonDisplayName string     `json:"personDisplayName,omitempty"`
	IsModerator       bool       `json:"isModerator,omitempty"`
	Created           *time.Time `json:"created,omitempty"`
}

// ListOptions contains the options for listing team memberships
type ListOptions struct {
	TeamID string `url:"teamId,omitempty"`
	Max    int    `url:"max,omitempty"`
}

// TeamMembershipsPage represents a paginated list of team memberships
type TeamMembershipsPage struct {
	Items []TeamMembership `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the TeamMemberships plugin
type Config struct {
	// Any configuration settings for the teammemberships plugin can go here
}

// DefaultConfig returns the default configuration for the TeamMemberships plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the team memberships API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new TeamMemberships plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of team memberships for a specified team
func (c *Client) List(options *ListOptions) (*TeamMembershipsPage, error) {
	if options == nil || options.TeamID == "" {
		return nil, fmt.Errorf("teamId is required")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("teamId", options.TeamID)
	if options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "team/memberships", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "team/memberships")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into TeamMemberships
	teamMembershipsPage := &TeamMembershipsPage{
		Page:  page,
		Items: make([]TeamMembership, len(page.Items)),
	}

	for i, item := range page.Items {
		var teamMembership TeamMembership
		if err := json.Unmarshal(item, &teamMembership); err != nil {
			return nil, err
		}
		teamMembershipsPage.Items[i] = teamMembership
	}

	return teamMembershipsPage, nil
}

// Create creates a new team membership
func (c *Client) Create(membership *TeamMembership) (*TeamMembership, error) {
	if membership.TeamID == "" {
		return nil, fmt.Errorf("teamId is required")
	}
	if membership.PersonID == "" && membership.PersonEmail == "" {
		return nil, fmt.Errorf("either personId or personEmail is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "team/memberships", nil, membership)
	if err != nil {
		return nil, err
	}

	var result TeamMembership
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns details for a team membership
func (c *Client) Get(membershipID string) (*TeamMembership, error) {
	if membershipID == "" {
		return nil, fmt.Errorf("membershipID is required")
	}

	path := fmt.Sprintf("team/memberships/%s", membershipID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var teamMembership TeamMembership
	if err := webexsdk.ParseResponse(resp, &teamMembership); err != nil {
		return nil, err
	}

	return &teamMembership, nil
}

// Update updates an existing team membership
func (c *Client) Update(membershipID string, isModerator bool) (*TeamMembership, error) {
	if membershipID == "" {
		return nil, fmt.Errorf("membershipID is required")
	}

	// Only isModerator can be updated
	updatedMembership := &TeamMembership{
		IsModerator: isModerator,
	}

	path := fmt.Sprintf("team/memberships/%s", membershipID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, updatedMembership)
	if err != nil {
		return nil, err
	}

	var result TeamMembership
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a team membership
func (c *Client) Delete(membershipID string) error {
	if membershipID == "" {
		return fmt.Errorf("membershipID is required")
	}

	path := fmt.Sprintf("team/memberships/%s", membershipID)
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

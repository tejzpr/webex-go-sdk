/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package teams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

// Team represents a Webex team
type Team struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	CreatorID   string     `json:"creatorId,omitempty"`
	Created     *time.Time `json:"created,omitempty"`
}

// ListOptions contains the options for listing teams
type ListOptions struct {
	Max int `url:"max,omitempty"`
}

// TeamsPage represents a paginated list of teams
type TeamsPage struct {
	Items []Team `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Teams plugin
type Config struct {
	// Any configuration settings for the teams plugin can go here
}

// DefaultConfig returns the default configuration for the Teams plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the teams API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Teams plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of teams
func (c *Client) List(options *ListOptions) (*TeamsPage, error) {
	params := url.Values{}
	if options != nil && options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "teams", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "teams")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Teams
	teamsPage := &TeamsPage{
		Page:  page,
		Items: make([]Team, len(page.Items)),
	}

	for i, item := range page.Items {
		var team Team
		if err := json.Unmarshal(item, &team); err != nil {
			return nil, err
		}
		teamsPage.Items[i] = team
	}

	return teamsPage, nil
}

// Create creates a new team
func (c *Client) Create(team *Team) (*Team, error) {
	if team.Name == "" {
		return nil, fmt.Errorf("team name is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "teams", nil, team)
	if err != nil {
		return nil, err
	}

	var result Team
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns details for a team
func (c *Client) Get(teamID string) (*Team, error) {
	if teamID == "" {
		return nil, fmt.Errorf("teamID is required")
	}

	path := fmt.Sprintf("teams/%s", teamID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var team Team
	if err := webexsdk.ParseResponse(resp, &team); err != nil {
		return nil, err
	}

	return &team, nil
}

// Update updates an existing team
func (c *Client) Update(teamID string, team *Team) (*Team, error) {
	if teamID == "" {
		return nil, fmt.Errorf("teamID is required")
	}
	if team.Name == "" {
		return nil, fmt.Errorf("team name is required")
	}

	path := fmt.Sprintf("teams/%s", teamID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, team)
	if err != nil {
		return nil, err
	}

	var result Team
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a team
func (c *Client) Delete(teamID string) error {
	if teamID == "" {
		return fmt.Errorf("teamID is required")
	}

	path := fmt.Sprintf("teams/%s", teamID)
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

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webexsdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Plugin represents a Webex API plugin
type Plugin interface {
	// Name returns the name of the plugin
	Name() string
}

// Client is the main Webex client struct
type Client struct {
	// HTTP client used to communicate with the API
	HttpClient *http.Client

	// Base URL for API requests
	BaseURL *url.URL

	// Access token for API authentication
	AccessToken string

	// Plugins registered with the client
	plugins map[string]Plugin

	// Configuration for the client
	Config *Config
}

// Config holds the configuration for the Webex client
type Config struct {
	// BaseURL is the base URL of the Webex API
	BaseURL string

	// Timeout for API requests
	Timeout time.Duration

	// Default headers to include in API requests
	DefaultHeaders map[string]string

	// Custom HTTP client to use instead of the default one
	// If nil, a default client will be created with the specified Timeout
	HttpClient *http.Client
}

// DefaultConfig returns a default configuration for the Webex client
func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://webexapis.com/v1",
		Timeout:        30 * time.Second,
		DefaultHeaders: make(map[string]string),
		HttpClient:     nil,
	}
}

// NewClient creates a new Webex client with the given access token and optional configuration
func NewClient(accessToken string, config *Config) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	if config == nil {
		config = DefaultConfig()
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}

	// Create HTTP client - either use the provided custom client or create a default one
	httpClient := config.HttpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	client := &Client{
		HttpClient:  httpClient,
		BaseURL:     baseURL,
		AccessToken: accessToken,
		plugins:     make(map[string]Plugin),
		Config:      config,
	}

	return client, nil
}

// RegisterPlugin registers a plugin with the client
func (c *Client) RegisterPlugin(plugin Plugin) {
	c.plugins[plugin.Name()] = plugin
}

// GetPlugin returns a plugin by name
func (c *Client) GetPlugin(name string) (Plugin, bool) {
	plugin, ok := c.plugins[name]
	return plugin, ok
}

// Request performs an HTTP request to the Webex API
func (c *Client) Request(method, path string, params url.Values, body interface{}) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL.String() + "/" + path)
	if err != nil {
		return nil, err
	}

	if params != nil {
		u.RawQuery = params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Add default headers
	for k, v := range c.Config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	return c.HttpClient.Do(req)
}

// Page represents a paginated response from the Webex API
type Page struct {
	Items    []json.RawMessage `json:"items"`
	NextPage string            `json:"nextPage,omitempty"`
	PrevPage string            `json:"prevPage,omitempty"`
	HasNext  bool              `json:"-"`
	HasPrev  bool              `json:"-"`
	Client   *Client           `json:"-"`
	Resource string            `json:"-"`
}

// ParseResponse parses an HTTP response into the given interface
func ParseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, v)
}

// NewPage creates a new Page from an HTTP response
func NewPage(resp *http.Response, client *Client, resource string) (*Page, error) {
	page := &Page{
		Client:   client,
		Resource: resource,
	}

	if err := ParseResponse(resp, page); err != nil {
		return nil, err
	}

	page.HasNext = page.NextPage != ""
	page.HasPrev = page.PrevPage != ""

	return page, nil
}

// Next retrieves the next page of results
func (p *Page) Next() (*Page, error) {
	if !p.HasNext {
		return nil, fmt.Errorf("no next page")
	}

	resp, err := p.Client.Request(http.MethodGet, p.NextPage, nil, nil)
	if err != nil {
		return nil, err
	}

	return NewPage(resp, p.Client, p.Resource)
}

// Prev retrieves the previous page of results
func (p *Page) Prev() (*Page, error) {
	if !p.HasPrev {
		return nil, fmt.Errorf("no previous page")
	}

	resp, err := p.Client.Request(http.MethodGet, p.PrevPage, nil, nil)
	if err != nil {
		return nil, err
	}

	return NewPage(resp, p.Client, p.Resource)
}

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webexsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Logger is the interface for SDK logging. Any logger that implements Printf
// (such as the standard library's *log.Logger) can be used.
type Logger interface {
	Printf(format string, v ...any)
}

// Plugin represents a Webex API plugin
type Plugin interface {
	// Name returns the name of the plugin
	Name() string
}

// Client is the main Webex client struct
type Client struct {
	// HTTP client used to communicate with the API
	httpClient *http.Client

	// Base URL for API requests
	BaseURL *url.URL

	// Access token for API authentication
	accessToken string

	// Plugins registered with the client
	plugins map[string]Plugin

	// Configuration for the client
	Config *Config

	// Logger for SDK operations
	logger Logger
}

// GetAccessToken returns the access token used for API authentication
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// GetHTTPClient returns the HTTP client used for API requests
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// GetLogger returns the logger used by the SDK.
func (c *Client) GetLogger() Logger {
	return c.logger
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

	// MaxRetries is the maximum number of retries for transient errors (429, 502, 503, 504).
	// Set to 0 to disable retries. Default: 3.
	MaxRetries int

	// RetryBaseDelay is the initial delay between retries. Default: 1s.
	// Subsequent retries use exponential backoff (delay * 2^attempt).
	RetryBaseDelay time.Duration

	// Logger is the logger for SDK operations. If nil, the standard library's
	// default logger (log.Default()) is used.
	Logger Logger
}

// DefaultConfig returns a default configuration for the Webex client
func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://webexapis.com/v1",
		Timeout:        30 * time.Second,
		DefaultHeaders: make(map[string]string),
		HttpClient:     nil,
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Second,
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

	// Set up logger - use provided logger or default
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}

	client := &Client{
		httpClient:  httpClient,
		BaseURL:     baseURL,
		accessToken: accessToken,
		plugins:     make(map[string]Plugin),
		logger:      logger,
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

// Request performs an HTTP request to the Webex API.
// The caller is responsible for closing the response body when done.
func (c *Client) Request(method, path string, params url.Values, body interface{}) (*http.Response, error) {
	return c.RequestWithContext(context.Background(), method, path, params, body)
}

// RequestWithContext performs an HTTP request to the Webex API with the given context.
// The context can be used for per-request timeouts and cancellation.
// The caller is responsible for closing the response body when done.
func (c *Client) RequestWithContext(ctx context.Context, method, path string, params url.Values, body interface{}) (*http.Response, error) {
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

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Add default headers
	for k, v := range c.Config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// RequestWithRetry performs an HTTP request with automatic retry for transient errors.
// It retries on HTTP 429 (Too Many Requests, respecting Retry-After header) and
// transient server errors (502, 503, 504) using exponential backoff.
// The caller is responsible for closing the response body when done.
func (c *Client) RequestWithRetry(ctx context.Context, method, path string, params url.Values, body interface{}) (*http.Response, error) {
	maxRetries := c.Config.MaxRetries
	baseDelay := c.Config.RetryBaseDelay
	if baseDelay == 0 {
		baseDelay = 1 * time.Second
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.RequestWithContext(ctx, method, path, params, body)
		if err != nil {
			return nil, err
		}

		// Check if we should retry
		if !isRetryableStatus(resp.StatusCode) || attempt == maxRetries {
			return resp, nil
		}

		// Determine delay
		delay := retryDelay(resp, baseDelay, attempt)

		// Close the response body before retrying
		resp.Body.Close()

		// Wait with context cancellation support
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return resp, err
}

// isRetryableStatus returns true for HTTP status codes that should be retried.
func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// retryDelay calculates the delay before the next retry attempt.
// For 429 responses, it respects the Retry-After header if present.
// Otherwise, it uses exponential backoff: baseDelay * 2^attempt.
func retryDelay(resp *http.Response, baseDelay time.Duration, attempt int) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	// Exponential backoff
	return baseDelay * (1 << uint(attempt))
}

// MultipartField represents a text field in a multipart request.
type MultipartField struct {
	Name  string
	Value string
}

// MultipartFile represents a file to upload in a multipart request.
type MultipartFile struct {
	FieldName string // Form field name (e.g., "files")
	FileName  string // Original filename (e.g., "report.pdf")
	Content   []byte // Raw file bytes
}

// RequestMultipart performs a multipart/form-data POST request to the Webex API.
// This is required for local file uploads (e.g., sending messages with attachments).
// The caller is responsible for closing the response body when done.
func (c *Client) RequestMultipart(path string, fields []MultipartField, files []MultipartFile) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL.String() + "/" + path)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Write text fields
	for _, f := range fields {
		if err := writer.WriteField(f.Name, f.Value); err != nil {
			return nil, fmt.Errorf("error writing field %s: %w", f.Name, err)
		}
	}

	// Write file parts
	for _, f := range files {
		part, err := writer.CreateFormFile(f.FieldName, f.FileName)
		if err != nil {
			return nil, fmt.Errorf("error creating form file %s: %w", f.FileName, err)
		}
		if _, err := part.Write(f.Content); err != nil {
			return nil, fmt.Errorf("error writing file %s: %w", f.FileName, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error closing multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add default headers
	for k, v := range c.Config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
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

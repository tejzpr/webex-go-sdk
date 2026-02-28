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

// Request performs an HTTP request to the Webex API with automatic retry
// for transient errors (429, 502, 503, 504).
// The caller is responsible for closing the response body when done.
func (c *Client) Request(method, path string, params url.Values, body interface{}) (*http.Response, error) {
	return c.RequestWithRetry(context.Background(), method, path, params, body)
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
// Includes 423 Locked (file being scanned for malware) per Webex API spec.
func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusLocked || // 423 — anti-malware scanning in progress
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// retryDelay calculates the delay before the next retry attempt.
// For 429 responses, it respects the Retry-After header if present.
// Otherwise, it uses exponential backoff: baseDelay * 2^attempt.
func retryDelay(resp *http.Response, baseDelay time.Duration, attempt int) time.Duration {
	// Respect Retry-After header for 429 (rate limit) and 423 (malware scanning)
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusLocked {
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

// RequestMultipart performs a multipart/form-data POST request to the Webex API
// with automatic retry for transient errors (429, 502, 503, 504).
// This is required for local file uploads (e.g., sending messages with attachments).
// The caller is responsible for closing the response body when done.
func (c *Client) RequestMultipart(path string, fields []MultipartField, files []MultipartFile) (*http.Response, error) {
	return c.RequestMultipartWithRetry(context.Background(), path, fields, files)
}

// RequestMultipartWithRetry performs a multipart/form-data POST with retry support.
// The multipart body is rebuilt on each retry attempt.
func (c *Client) RequestMultipartWithRetry(ctx context.Context, path string, fields []MultipartField, files []MultipartFile) (*http.Response, error) {
	maxRetries := c.Config.MaxRetries
	baseDelay := c.Config.RetryBaseDelay
	if baseDelay == 0 {
		baseDelay = 1 * time.Second
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.doMultipartRequest(ctx, path, fields, files)
		if err != nil {
			return nil, err
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == maxRetries {
			return resp, nil
		}

		delay := retryDelay(resp, baseDelay, attempt)
		resp.Body.Close()

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

// doMultipartRequest performs a single multipart/form-data POST request.
func (c *Client) doMultipartRequest(ctx context.Context, path string, fields []MultipartField, files []MultipartFile) (*http.Response, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &body)
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

// Page represents a paginated response from the Webex API.
// Pagination follows RFC 5988 (Web Linking) — next/prev URLs are parsed
// from the response's Link header.
type Page struct {
	Items    []json.RawMessage `json:"items"`
	NextPage string            `json:"-"`
	PrevPage string            `json:"-"`
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
		return NewAPIError(resp, body)
	}

	return json.Unmarshal(body, v)
}

// RequestURL performs an HTTP GET request to a full URL (not relative to BaseURL).
// This is used for pagination where Link headers contain absolute URLs.
// The request includes the same authentication and default headers as regular requests.
// The caller is responsible for closing the response body when done.
func (c *Client) RequestURL(method, fullURL string, body interface{}) (*http.Response, error) {
	return c.RequestURLWithRetry(context.Background(), method, fullURL, body)
}

// RequestURLWithRetry performs an HTTP request to a full URL with retry logic.
func (c *Client) RequestURLWithRetry(ctx context.Context, method, fullURL string, body interface{}) (*http.Response, error) {
	maxRetries := c.Config.MaxRetries
	baseDelay := c.Config.RetryBaseDelay
	if baseDelay == 0 {
		baseDelay = 1 * time.Second
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.doRequestURL(ctx, method, fullURL, body)
		if err != nil {
			return nil, err
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == maxRetries {
			return resp, nil
		}

		delay := retryDelay(resp, baseDelay, attempt)
		resp.Body.Close()

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

// doRequestURL performs a single HTTP request to a full URL.
func (c *Client) doRequestURL(ctx context.Context, method, fullURL string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range c.Config.DefaultHeaders {
		req.Header.Set(k, v)
	}

	return c.httpClient.Do(req)
}

// parseLinkHeader parses an RFC 5988 Link header value and returns a map
// of rel type to URL. For example:
//
//	<https://example.com/items?page=2>; rel="next"
//
// returns {"next": "https://example.com/items?page=2"}.
func parseLinkHeader(header string) map[string]string {
	links := make(map[string]string)
	if header == "" {
		return links
	}

	// Split by comma for multiple links
	parts := splitLinks(header)
	for _, part := range parts {
		// Extract URL between < and >
		urlStart := indexOf(part, '<')
		urlEnd := indexOf(part, '>')
		if urlStart < 0 || urlEnd < 0 || urlEnd <= urlStart+1 {
			continue
		}
		linkURL := part[urlStart+1 : urlEnd]

		// Extract rel value
		relStart := indexOfSubstring(part, `rel="`)
		if relStart < 0 {
			continue
		}
		relStart += 5 // len(`rel="`)
		relEnd := indexOf(part[relStart:], '"')
		if relEnd < 0 {
			continue
		}
		rel := part[relStart : relStart+relEnd]

		links[rel] = linkURL
	}

	return links
}

// splitLinks splits a Link header value by commas, respecting angle brackets.
func splitLinks(header string) []string {
	var parts []string
	inBrackets := false
	start := 0
	for i := 0; i < len(header); i++ {
		switch header[i] {
		case '<':
			inBrackets = true
		case '>':
			inBrackets = false
		case ',':
			if !inBrackets {
				parts = append(parts, trimSpace(header[start:i]))
				start = i + 1
			}
		}
	}
	if start < len(header) {
		parts = append(parts, trimSpace(header[start:]))
	}
	return parts
}

// indexOf returns the index of the first occurrence of c in s, or -1 if not found.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// indexOfSubstring returns the index of the first occurrence of sub in s, or -1.
func indexOfSubstring(s, sub string) int {
	if len(sub) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// trimSpace trims leading and trailing whitespace from s.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// NewPage creates a new Page from an HTTP response.
// It parses the Link header (RFC 5988) for pagination URLs and
// the JSON body for the items array.
func NewPage(resp *http.Response, client *Client, resource string) (*Page, error) {
	// Parse Link header before reading/closing the body
	linkHeader := resp.Header.Get("Link")
	links := parseLinkHeader(linkHeader)

	page := &Page{
		Client:   client,
		Resource: resource,
	}

	if err := ParseResponse(resp, page); err != nil {
		return nil, err
	}

	// Set pagination URLs from Link header
	page.NextPage = links["next"]
	page.PrevPage = links["prev"]
	page.HasNext = page.NextPage != ""
	page.HasPrev = page.PrevPage != ""

	return page, nil
}

// Next retrieves the next page of results using the URL from the Link header.
func (p *Page) Next() (*Page, error) {
	if !p.HasNext {
		return nil, fmt.Errorf("no next page")
	}

	// Link header URLs are absolute — use RequestURL
	resp, err := p.Client.RequestURL(http.MethodGet, p.NextPage, nil)
	if err != nil {
		return nil, err
	}

	return NewPage(resp, p.Client, p.Resource)
}

// Prev retrieves the previous page of results using the URL from the Link header.
func (p *Page) Prev() (*Page, error) {
	if !p.HasPrev {
		return nil, fmt.Errorf("no previous page")
	}

	resp, err := p.Client.RequestURL(http.MethodGet, p.PrevPage, nil)
	if err != nil {
		return nil, err
	}

	return NewPage(resp, p.Client, p.Resource)
}

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webexsdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// MockPlugin implements the Plugin interface for testing
type MockPlugin struct {
	name string
}

func (m *MockPlugin) Name() string {
	return m.name
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		config      *Config
		expectError bool
	}{
		{
			name:        "Valid with default config",
			accessToken: "valid-token",
			config:      nil,
			expectError: false,
		},
		{
			name:        "Valid with custom config",
			accessToken: "valid-token",
			config: &Config{
				BaseURL: "https://api.example.com",
				Timeout: 60 * time.Second,
				DefaultHeaders: map[string]string{
					"X-Custom-Header": "value",
				},
			},
			expectError: false,
		},
		{
			name:        "Empty access token",
			accessToken: "",
			config:      nil,
			expectError: true,
		},
		{
			name:        "Invalid base URL",
			accessToken: "valid-token",
			config: &Config{
				BaseURL: ":", // Invalid URL
				Timeout: 30 * time.Second,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.accessToken, tc.config)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client")
				return
			}

			if client.GetAccessToken() != tc.accessToken {
				t.Errorf("Expected AccessToken %q, got %q", tc.accessToken, client.GetAccessToken())
			}

			if tc.config != nil {
				if client.BaseURL.String() != tc.config.BaseURL {
					t.Errorf("Expected BaseURL %q, got %q", tc.config.BaseURL, client.BaseURL.String())
				}

				if client.GetHTTPClient().Timeout != tc.config.Timeout {
					t.Errorf("Expected Timeout %v, got %v", tc.config.Timeout, client.GetHTTPClient().Timeout)
				}

				// Check custom headers were set in config
				for k, v := range tc.config.DefaultHeaders {
					if client.Config.DefaultHeaders[k] != v {
						t.Errorf("Expected header %q: %q, got %q", k, v, client.Config.DefaultHeaders[k])
					}
				}
			} else {
				// Check default config values
				defaultConfig := DefaultConfig()
				if client.BaseURL.String() != defaultConfig.BaseURL {
					t.Errorf("Expected default BaseURL %q, got %q", defaultConfig.BaseURL, client.BaseURL.String())
				}
				if client.GetHTTPClient().Timeout != defaultConfig.Timeout {
					t.Errorf("Expected default Timeout %v, got %v", defaultConfig.Timeout, client.GetHTTPClient().Timeout)
				}
			}
		})
	}
}

func TestRegisterAndGetPlugin(t *testing.T) {
	client, _ := NewClient("test-token", nil)

	// Register a mock plugin
	mockPlugin := &MockPlugin{name: "mock-plugin"}
	client.RegisterPlugin(mockPlugin)

	// Test getting the registered plugin
	plugin, ok := client.GetPlugin("mock-plugin")
	if !ok {
		t.Errorf("Expected to find plugin 'mock-plugin', but not found")
	}

	if plugin != mockPlugin {
		t.Errorf("Expected to get the same plugin instance that was registered")
	}

	// Test getting a non-existent plugin
	_, ok = client.GetPlugin("non-existent")
	if ok {
		t.Errorf("Expected not to find plugin 'non-existent', but found")
	}
}

func TestRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %q", authHeader)
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got %q", contentType)
		}

		// Check custom header
		customHeader := r.Header.Get("X-Custom-Header")
		if customHeader != "custom-value" {
			t.Errorf("Expected X-Custom-Header 'custom-value', got %q", customHeader)
		}

		// Check method
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check path
		if r.URL.Path != "/test" {
			t.Errorf("Expected path '/test', got %q", r.URL.Path)
		}

		// Check query parameters
		if r.URL.Query().Get("param1") != "value1" {
			t.Errorf("Expected query param 'param1=value1', got %q", r.URL.Query().Get("param1"))
		}
		if r.URL.Query().Get("param2") != "value2" {
			t.Errorf("Expected query param 'param2=value2', got %q", r.URL.Query().Get("param2"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status": "success"}`)
	}))
	defer server.Close()

	// Create a client with a custom config
	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		DefaultHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	// Create query parameters
	params := url.Values{}
	params.Set("param1", "value1")
	params.Set("param2", "value2")

	// Make the request
	resp, err := client.Request(http.MethodGet, "test", params, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Parse response
	var responseData struct {
		Status string `json:"status"`
	}

	err = ParseResponse(resp, &responseData)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response data
	if responseData.Status != "success" {
		t.Errorf("Expected status 'success', got %q", responseData.Status)
	}
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "Valid response",
			statusCode:   http.StatusOK,
			responseBody: `{"key": "value"}`,
			expectError:  false,
		},
		{
			name:         "Error response",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": "Bad request"}`,
			expectError:  true,
		},
		{
			name:         "Invalid JSON",
			statusCode:   http.StatusOK,
			responseBody: `{"key": "value"`, // Incomplete JSON
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test response
			resp := &http.Response{
				StatusCode: tc.statusCode,
				Body:       newMockReadCloser(tc.responseBody),
			}

			var data map[string]string
			err := ParseResponse(resp, &data)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check parsed data
			if len(data) == 0 {
				t.Errorf("Expected non-empty data")
			}
		})
	}
}

func TestPageNavigation(t *testing.T) {
	// Use var to avoid circular reference in closure
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/items":
			// First page — Link header with next only
			w.Header().Set("Link", `<`+serverURL+`/next-page>; rel="next"`)
			fmt.Fprintln(w, `{"items": [{"id": "item1"}, {"id": "item2"}]}`)
		case "/next-page":
			// Second page — Link header with prev only
			w.Header().Set("Link", `<`+serverURL+`/prev-page>; rel="prev"`)
			fmt.Fprintln(w, `{"items": [{"id": "item3"}, {"id": "item4"}]}`)
		case "/prev-page":
			// Back to first page — Link header with next only
			w.Header().Set("Link", `<`+serverURL+`/next-page>; rel="next"`)
			fmt.Fprintln(w, `{"items": [{"id": "item1"}, {"id": "item2"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()
	serverURL = server.URL

	// Create client
	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	// Get first page
	resp, err := client.Request(http.MethodGet, "items", nil, nil)
	if err != nil {
		t.Fatalf("Failed to get first page: %v", err)
	}

	page, err := NewPage(resp, client, "items")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Check first page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items on first page, got %d", len(page.Items))
	}
	if !page.HasNext {
		t.Errorf("Expected first page to have next page")
	}
	if page.HasPrev {
		t.Errorf("Expected first page not to have previous page")
	}

	// Navigate to next page
	nextPage, err := page.Next()
	if err != nil {
		t.Fatalf("Failed to navigate to next page: %v", err)
	}

	// Check second page
	if len(nextPage.Items) != 2 {
		t.Errorf("Expected 2 items on second page, got %d", len(nextPage.Items))
	}
	if nextPage.HasNext {
		t.Errorf("Expected second page not to have next page")
	}
	if !nextPage.HasPrev {
		t.Errorf("Expected second page to have previous page")
	}

	// Navigate back to first page
	prevPage, err := nextPage.Prev()
	if err != nil {
		t.Fatalf("Failed to navigate to previous page: %v", err)
	}

	// Check first page again
	if len(prevPage.Items) != 2 {
		t.Errorf("Expected 2 items on first page, got %d", len(prevPage.Items))
	}

	// Test error cases
	// Try to go to prev page when there isn't one
	_, err = page.Prev()
	if err == nil {
		t.Errorf("Expected error when navigating to non-existent previous page")
	}

	// Try to go to next page when there isn't one
	_, err = nextPage.Next()
	if err == nil {
		t.Errorf("Expected error when navigating to non-existent next page")
	}
}

func TestRequestMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Check auth header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer test-token, got %q", r.Header.Get("Authorization"))
		}

		// Check content type is multipart
		ct := r.Header.Get("Content-Type")
		if ct == "" || len(ct) < 19 {
			t.Fatalf("Expected multipart content type, got %q", ct)
		}
		if ct[:19] != "multipart/form-data" {
			t.Fatalf("Expected multipart/form-data, got %q", ct)
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart: %v", err)
		}

		// Check text fields
		if r.FormValue("roomId") != "room-abc" {
			t.Errorf("Expected roomId 'room-abc', got '%s'", r.FormValue("roomId"))
		}
		if r.FormValue("text") != "hello" {
			t.Errorf("Expected text 'hello', got '%s'", r.FormValue("text"))
		}

		// Check file
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("Failed to get form file: %v", err)
		}
		defer file.Close()

		if header.Filename != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got '%s'", header.Filename)
		}

		content, _ := io.ReadAll(file)
		if string(content) != "file content here" {
			t.Errorf("Expected 'file content here', got '%s'", content)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id": "msg-123"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	fields := []MultipartField{
		{Name: "roomId", Value: "room-abc"},
		{Name: "text", Value: "hello"},
	}
	files := []MultipartFile{
		{FieldName: "files", FileName: "test.txt", Content: []byte("file content here")},
	}

	resp, err := client.RequestMultipart("messages", fields, files)
	if err != nil {
		t.Fatalf("RequestMultipart failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := ParseResponse(resp, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got '%s'", result.ID)
	}
}

// Mock ReadCloser for testing ParseResponse
type mockReadCloser struct {
	data  string
	index int
}

func newMockReadCloser(data string) *mockReadCloser {
	return &mockReadCloser{data: data}
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	if m.index >= len(m.data) {
		return 0, io.EOF
	}

	n = copy(p, m.data[m.index:])
	m.index += n
	return n, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}

// --- Retry tests ---

func TestRequest_Retries429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintln(w, `{"message":"rate limited"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRequest_Retries502(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintln(w, `{"message":"bad gateway"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRequest_Retries423Locked(t *testing.T) {
	// 423 Locked (anti-malware scanning) should be retried with Retry-After
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusLocked)
			fmt.Fprintln(w, `{"message":"file is being scanned"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestRequest_NoRetryOn400(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, `{"message":"bad request"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
	}
}

func TestRequest_NoRetryOn401(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"message":"unauthorized"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry for 401), got %d", attempts)
	}
}

func TestRequest_ExhaustsRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, `{"message":"unavailable"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     2,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "300")
		fmt.Fprintln(w, `{"message":"rate limited"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     5,
		RetryBaseDelay: 10 * time.Second,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.RequestWithRetry(ctx, http.MethodGet, "test", nil, nil)
	if err == nil {
		t.Fatal("Expected error from context cancellation")
	}
}

func TestRequestMultipart_Retries429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintln(w, `{"message":"rate limited"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"id":"msg-123"}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		MaxRetries:     3,
		RetryBaseDelay: 1 * time.Millisecond,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	fields := []MultipartField{{Name: "roomId", Value: "room-1"}}
	files := []MultipartFile{{FieldName: "files", FileName: "test.txt", Content: []byte("data")}}

	resp, err := client.RequestMultipart("messages", fields, files)
	if err != nil {
		t.Fatalf("RequestMultipart failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

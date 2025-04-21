/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package webexsdk

import (
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

			if client.AccessToken != tc.accessToken {
				t.Errorf("Expected AccessToken %q, got %q", tc.accessToken, client.AccessToken)
			}

			if tc.config != nil {
				if client.BaseURL.String() != tc.config.BaseURL {
					t.Errorf("Expected BaseURL %q, got %q", tc.config.BaseURL, client.BaseURL.String())
				}

				if client.HttpClient.Timeout != tc.config.Timeout {
					t.Errorf("Expected Timeout %v, got %v", tc.config.Timeout, client.HttpClient.Timeout)
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
				if client.HttpClient.Timeout != defaultConfig.Timeout {
					t.Errorf("Expected default Timeout %v, got %v", defaultConfig.Timeout, client.HttpClient.Timeout)
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
	// Create test server
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Simulate different pages based on path
		switch r.URL.Path {
		case "/items":
			// First page
			fmt.Fprintln(w, `{
				"items": [{"id": "item1"}, {"id": "item2"}],
				"nextPage": "next-page",
				"prevPage": ""
			}`)
		case "/next-page":
			// Second page
			fmt.Fprintln(w, `{
				"items": [{"id": "item3"}, {"id": "item4"}],
				"nextPage": "",
				"prevPage": "prev-page"
			}`)
		case "/prev-page":
			// Back to first page
			fmt.Fprintln(w, `{
				"items": [{"id": "item1"}, {"id": "item2"}],
				"nextPage": "next-page",
				"prevPage": ""
			}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		pageCount++
	}))
	defer server.Close()

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

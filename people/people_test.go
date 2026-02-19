/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package people

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/people/me" {
			t.Errorf("Expected path '/people/me', got '%s'", r.URL.Path)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample person
		person := Person{
			ID:          "test-id",
			Emails:      []string{"test@example.com"},
			DisplayName: "Test User",
			Created:     time.Now(),
		}

		// Write response
		_ = json.NewEncoder(w).Encode(person)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(), // Use the test server's client
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL in case NewClient didn't use our URL
	client.BaseURL = baseURL

	// Create people plugin
	peoplePlugin := New(client, nil)

	// Get person
	person, err := peoplePlugin.Get("me")
	if err != nil {
		t.Fatalf("Error getting person: %v", err)
	}

	// Check person
	if person.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", person.ID)
	}
	if len(person.Emails) != 1 || person.Emails[0] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%v'", person.Emails)
	}
	if person.DisplayName != "Test User" {
		t.Errorf("Expected display name 'Test User', got '%s'", person.DisplayName)
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/people" {
			t.Errorf("Expected path '/people', got '%s'", r.URL.Path)
		}

		email := r.URL.Query().Get("email")
		if email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", email)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		person := Person{
			ID:          "test-id",
			Emails:      []string{"test@example.com"},
			DisplayName: "Test User",
			Created:     time.Now(),
		}

		response := struct {
			Items []Person `json:"items"`
		}{
			Items: []Person{person},
		}

		// Write response
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(), // Use the test server's client
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL in case NewClient didn't use our URL
	client.BaseURL = baseURL

	// Create people plugin
	peoplePlugin := New(client, nil)

	// List people
	options := &ListOptions{
		Email: "test@example.com",
	}

	page, err := peoplePlugin.List(options)
	if err != nil {
		t.Fatalf("Error listing people: %v", err)
	}

	// Check people
	if len(page.Items) != 1 {
		t.Fatalf("Expected 1 person, got %d", len(page.Items))
	}

	person := page.Items[0]
	if person.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", person.ID)
	}
	if len(person.Emails) != 1 || person.Emails[0] != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%v'", person.Emails)
	}
	if person.DisplayName != "Test User" {
		t.Errorf("Expected display name 'Test User', got '%s'", person.DisplayName)
	}
}

func TestInferPersonIDFromUUID(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Regular UUID",
			input:    "12345678-1234-1234-1234-123456789012",
			expected: EncodeBase64("ciscospark://us/PEOPLE/12345678-1234-1234-1234-123456789012"),
		},
		{
			name:     "Already encoded Hydra ID",
			input:    EncodeBase64("ciscospark://us/PEOPLE/12345678-1234-1234-1234-123456789012"),
			expected: EncodeBase64("ciscospark://us/PEOPLE/12345678-1234-1234-1234-123456789012"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := InferPersonIDFromUUID(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetMe(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/people/me" {
			t.Errorf("Expected path '/people/me', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got '%s'", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample person
		person := Person{
			ID:          "me-user-id",
			Emails:      []string{"me@example.com"},
			DisplayName: "Current User",
			FirstName:   "Current",
			LastName:    "User",
			Created:     time.Now(),
		}

		// Write response
		_ = json.NewEncoder(w).Encode(person)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create people plugin
	peoplePlugin := New(client, nil)

	// Get current user
	me, err := peoplePlugin.GetMe()
	if err != nil {
		t.Fatalf("Error getting me: %v", err)
	}

	// Check person
	if me.ID != "me-user-id" {
		t.Errorf("Expected ID 'me-user-id', got '%s'", me.ID)
	}
	if len(me.Emails) != 1 || me.Emails[0] != "me@example.com" {
		t.Errorf("Expected email 'me@example.com', got '%v'", me.Emails)
	}
	if me.DisplayName != "Current User" {
		t.Errorf("Expected display name 'Current User', got '%s'", me.DisplayName)
	}
	if me.FirstName != "Current" {
		t.Errorf("Expected first name 'Current', got '%s'", me.FirstName)
	}
	if me.LastName != "User" {
		t.Errorf("Expected last name 'User', got '%s'", me.LastName)
	}
}

func newTestPeopleServer(t *testing.T, people []Person) (*httptest.Server, *webexsdk.Client) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/people" {
			// Filter by requested IDs
			requestedIDs := r.URL.Query()["id"]
			var result []Person
			for _, p := range people {
				for _, id := range requestedIDs {
					if p.ID == id {
						result = append(result, p)
					}
				}
			}
			if result == nil {
				result = []Person{}
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(struct {
				Items []Person `json:"items"`
			}{Items: result})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = baseURL
	return server, client
}

func TestBatchRequest_Empty(t *testing.T) {
	server, client := newTestPeopleServer(t, nil)
	defer server.Close()

	batcher := NewBatcher(client, DefaultConfig())
	result, err := batcher.BatchRequest([]string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}
}

func TestBatchRequest_SingleID(t *testing.T) {
	// BatchRequest converts IDs via InferPersonIDFromUUID, so store with Hydra IDs
	people := []Person{
		{ID: InferPersonIDFromUUID("id-1"), DisplayName: "Alice", Emails: []string{"alice@example.com"}},
		{ID: InferPersonIDFromUUID("id-2"), DisplayName: "Bob", Emails: []string{"bob@example.com"}},
	}
	server, client := newTestPeopleServer(t, people)
	defer server.Close()

	batcher := NewBatcher(client, DefaultConfig())
	result, err := batcher.BatchRequest([]string{"id-1"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if result[0].DisplayName != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", result[0].DisplayName)
	}
}

func TestBatchRequest_MultipleIDs(t *testing.T) {
	people := []Person{
		{ID: InferPersonIDFromUUID("id-1"), DisplayName: "Alice"},
		{ID: InferPersonIDFromUUID("id-2"), DisplayName: "Bob"},
		{ID: InferPersonIDFromUUID("id-3"), DisplayName: "Charlie"},
	}
	server, client := newTestPeopleServer(t, people)
	defer server.Close()

	batcher := NewBatcher(client, DefaultConfig())
	result, err := batcher.BatchRequest([]string{"id-1", "id-3"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}

	names := map[string]bool{}
	for _, p := range result {
		names[p.DisplayName] = true
	}
	if !names["Alice"] || !names["Charlie"] {
		t.Errorf("Expected Alice and Charlie, got %v", names)
	}
}

func TestBatchRequest_NotFound(t *testing.T) {
	people := []Person{
		{ID: "id-1", DisplayName: "Alice"},
	}
	server, client := newTestPeopleServer(t, people)
	defer server.Close()

	batcher := NewBatcher(client, DefaultConfig())
	result, err := batcher.BatchRequest([]string{"id-nonexistent"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 results for nonexistent ID, got %d", len(result))
	}
}

func TestBatchRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = baseURL

	batcher := NewBatcher(client, DefaultConfig())
	_, err = batcher.BatchRequest([]string{"id-1"})
	if err == nil {
		t.Fatal("Expected error for server error response, got nil")
	}
}

func TestBatcher_AsyncRequest(t *testing.T) {
	people := []Person{
		{ID: InferPersonIDFromUUID("uuid-1"), DisplayName: "Alice"},
	}
	server, client := newTestPeopleServer(t, people)
	defer server.Close()

	cfg := DefaultConfig()
	cfg.BatcherWait = 50 * time.Millisecond
	batcher := NewBatcher(client, cfg)

	person, err := batcher.Request("uuid-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if person.DisplayName != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", person.DisplayName)
	}
}

func TestBatcher_AsyncRequest_NotFound(t *testing.T) {
	server, client := newTestPeopleServer(t, nil)
	defer server.Close()

	cfg := DefaultConfig()
	cfg.BatcherWait = 50 * time.Millisecond
	batcher := NewBatcher(client, cfg)

	_, err := batcher.Request("nonexistent-uuid")
	if err == nil {
		t.Fatal("Expected error for nonexistent person, got nil")
	}
}

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

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
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
		json.NewEncoder(w).Encode(person)
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
		json.NewEncoder(w).Encode(response)
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

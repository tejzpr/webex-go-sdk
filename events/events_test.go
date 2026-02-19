/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package events

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v1/webexsdk"
)

func TestList(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check path
		if r.URL.Path != "/events" {
			t.Errorf("Expected path /events, got %s", r.URL.Path)
		}

		// Check query parameters
		query := r.URL.Query()
		if resource := query.Get("resource"); resource != "messages" {
			t.Errorf("Expected resource parameter 'messages', got '%s'", resource)
		}
		if eventType := query.Get("type"); eventType != "created" {
			t.Errorf("Expected type parameter 'created', got '%s'", eventType)
		}
		if actorId := query.Get("actorId"); actorId != "actor123" {
			t.Errorf("Expected actorId parameter 'actor123', got '%s'", actorId)
		}
		if from := query.Get("from"); from != "2023-01-01T00:00:00Z" {
			t.Errorf("Expected from parameter '2023-01-01T00:00:00Z', got '%s'", from)
		}
		if to := query.Get("to"); to != "2023-01-02T00:00:00Z" {
			t.Errorf("Expected to parameter '2023-01-02T00:00:00Z', got '%s'", to)
		}
		if max := query.Get("max"); max != "10" {
			t.Errorf("Expected max parameter '10', got '%s'", max)
		}

		// Return sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{
					"id": "event123",
					"resource": "messages",
					"type": "created",
					"actorId": "actor123",
					"orgId": "org123",
					"created": "2023-01-01T12:00:00Z",
					"data": {
						"id": "msg123",
						"roomId": "room123",
						"roomType": "group",
						"personId": "person123",
						"personEmail": "test@example.com"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	// Setup client
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		HttpClient: http.DefaultClient,
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	eventsClient := New(client, nil)

	// Call List with options
	fromTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	toTime, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	options := &ListOptions{
		Resource: "messages",
		Type:     "created",
		ActorID:  "actor123",
		From:     fromTime.Format(time.RFC3339),
		To:       toTime.Format(time.RFC3339),
		Max:      10,
	}
	eventsPage, err := eventsClient.List(options)

	// Check results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if eventsPage == nil {
		t.Fatal("Expected events page, got nil")
	}
	if len(eventsPage.Items) != 1 {
		t.Errorf("Expected 1 event, got %d", len(eventsPage.Items))
	}

	// Check event details
	event := eventsPage.Items[0]
	if event.ID != "event123" {
		t.Errorf("Expected event ID 'event123', got '%s'", event.ID)
	}
	if event.Resource != "messages" {
		t.Errorf("Expected resource 'messages', got '%s'", event.Resource)
	}
	if event.Type != "created" {
		t.Errorf("Expected type 'created', got '%s'", event.Type)
	}
	if event.ActorID != "actor123" {
		t.Errorf("Expected actor ID 'actor123', got '%s'", event.ActorID)
	}
	if event.Data.RoomID != "room123" {
		t.Errorf("Expected room ID 'room123', got '%s'", event.Data.RoomID)
	}
}

func TestGet(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check path
		if r.URL.Path != "/events/event123" {
			t.Errorf("Expected path /events/event123, got %s", r.URL.Path)
		}

		// Return sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "event123",
			"resource": "messages",
			"type": "created",
			"actorId": "actor123",
			"orgId": "org123",
			"created": "2023-01-01T12:00:00Z",
			"data": {
				"id": "msg123",
				"roomId": "room123",
				"roomType": "group",
				"personId": "person123",
				"personEmail": "test@example.com"
			}
		}`))
	}))
	defer server.Close()

	// Setup client
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		HttpClient: http.DefaultClient,
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	eventsClient := New(client, nil)

	// Call Get with event ID
	event, err := eventsClient.Get("event123")

	// Check results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if event == nil {
		t.Fatal("Expected event, got nil")
	}

	// Check event details
	if event.ID != "event123" {
		t.Errorf("Expected event ID 'event123', got '%s'", event.ID)
	}
	if event.Resource != "messages" {
		t.Errorf("Expected resource 'messages', got '%s'", event.Resource)
	}
	if event.Type != "created" {
		t.Errorf("Expected type 'created', got '%s'", event.Type)
	}
	if event.ActorID != "actor123" {
		t.Errorf("Expected actor ID 'actor123', got '%s'", event.ActorID)
	}
	if event.Data.RoomID != "room123" {
		t.Errorf("Expected room ID 'room123', got '%s'", event.Data.RoomID)
	}
}

func TestGetError(t *testing.T) {
	// Setup client with nil event ID
	config := &webexsdk.Config{
		BaseURL:    "https://example.com",
		HttpClient: http.DefaultClient,
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	eventsClient := New(client, nil)

	// Call Get with empty event ID
	_, err = eventsClient.Get("")

	// Check error
	if err == nil {
		t.Error("Expected error for empty event ID, got nil")
	}
}

func TestListError(t *testing.T) {
	// Setup test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Invalid request"}`))
	}))
	defer server.Close()

	// Setup client
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		HttpClient: http.DefaultClient,
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	eventsClient := New(client, nil)

	// Call List with options
	options := &ListOptions{
		Resource: "messages",
		Type:     "created",
	}
	_, err = eventsClient.List(options)

	// Check error
	if err == nil {
		t.Error("Expected error for bad request, got nil")
	}
}

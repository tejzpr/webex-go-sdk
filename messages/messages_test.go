/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package messages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path '/messages', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var message Message
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify message
		if message.RoomID != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", message.RoomID)
		}
		if message.Text != "Hello, World!" {
			t.Errorf("Expected text 'Hello, World!', got '%s'", message.Text)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMsg := Message{
			ID:          "test-message-id",
			RoomID:      message.RoomID,
			Text:        message.Text,
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseMsg)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Create message
	message := &Message{
		RoomID: "test-room-id",
		Text:   "Hello, World!",
	}

	result, err := messagesPlugin.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Check message
	if result.ID != "test-message-id" {
		t.Errorf("Expected ID 'test-message-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", result.Text)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", result.PersonEmail)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages/test-message-id" {
			t.Errorf("Expected path '/messages/test-message-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		message := Message{
			ID:          "test-message-id",
			RoomID:      "test-room-id",
			Text:        "Hello, World!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(message)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Get message
	message, err := messagesPlugin.Get("test-message-id")
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	// Check message
	if message.ID != "test-message-id" {
		t.Errorf("Expected ID 'test-message-id', got '%s'", message.ID)
	}
	if message.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", message.RoomID)
	}
	if message.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", message.Text)
	}
	if message.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", message.PersonID)
	}
	if message.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", message.PersonEmail)
	}
	if message.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path '/messages', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("roomId") != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", r.URL.Query().Get("roomId"))
		}
		if r.URL.Query().Get("max") != "50" {
			t.Errorf("Expected max '50', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		message1 := Message{
			ID:          "test-message-id-1",
			RoomID:      "test-room-id",
			Text:        "Hello, World!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}
		message2 := Message{
			ID:          "test-message-id-2",
			RoomID:      "test-room-id",
			Text:        "Hello again!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Create response with items array
		response := struct {
			Items []Message `json:"items"`
		}{
			Items: []Message{message1, message2},
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
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// List messages
	options := &ListOptions{
		RoomID: "test-room-id",
		Max:    50,
	}

	page, err := messagesPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Check messages
	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(page.Items))
	}

	message1 := page.Items[0]
	if message1.ID != "test-message-id-1" {
		t.Errorf("Expected ID 'test-message-id-1', got '%s'", message1.ID)
	}
	if message1.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", message1.Text)
	}

	message2 := page.Items[1]
	if message2.ID != "test-message-id-2" {
		t.Errorf("Expected ID 'test-message-id-2', got '%s'", message2.ID)
	}
	if message2.Text != "Hello again!" {
		t.Errorf("Expected text 'Hello again!', got '%s'", message2.Text)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages/test-message-id" {
			t.Errorf("Expected path '/messages/test-message-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}

		// Write response with no content
		w.WriteHeader(http.StatusNoContent)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Delete message
	err = messagesPlugin.Delete("test-message-id")
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}
}

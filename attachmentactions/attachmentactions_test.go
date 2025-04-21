/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package attachmentactions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/attachment/actions" {
			t.Errorf("Expected path '/attachment/actions', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var action AttachmentAction
		if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify action
		if action.MessageID != "test-message-id" {
			t.Errorf("Expected messageId 'test-message-id', got '%s'", action.MessageID)
		}
		if action.Type != "submit" {
			t.Errorf("Expected type 'submit', got '%s'", action.Type)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseAction := AttachmentAction{
			ID:        "test-action-id",
			Type:      action.Type,
			MessageID: action.MessageID,
			Inputs:    action.Inputs,
			PersonID:  "test-person-id",
			RoomID:    "test-room-id",
			Created:   &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseAction)
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

	// Create attachment actions plugin
	actionsPlugin := New(client, nil)

	// Create inputs for the action
	inputs := map[string]interface{}{
		"textInput": "Hello, World!",
		"checkBox":  true,
	}

	// Create action
	action := &AttachmentAction{
		Type:      "submit",
		MessageID: "test-message-id",
		Inputs:    inputs,
	}

	result, err := actionsPlugin.Create(action)
	if err != nil {
		t.Fatalf("Failed to create attachment action: %v", err)
	}

	// Check action
	if result.ID != "test-action-id" {
		t.Errorf("Expected ID 'test-action-id', got '%s'", result.ID)
	}
	if result.Type != "submit" {
		t.Errorf("Expected type 'submit', got '%s'", result.Type)
	}
	if result.MessageID != "test-message-id" {
		t.Errorf("Expected messageId 'test-message-id', got '%s'", result.MessageID)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}

	// Check inputs
	if input, ok := result.Inputs["textInput"]; !ok || input != "Hello, World!" {
		t.Errorf("Expected textInput 'Hello, World!', got '%v'", input)
	}
	if input, ok := result.Inputs["checkBox"]; !ok || input != true {
		t.Errorf("Expected checkBox true, got '%v'", input)
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/attachment/actions/test-action-id" {
			t.Errorf("Expected path '/attachment/actions/test-action-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		inputs := map[string]interface{}{
			"textInput": "Hello, World!",
			"checkBox":  true,
		}
		action := AttachmentAction{
			ID:        "test-action-id",
			Type:      "submit",
			MessageID: "test-message-id",
			Inputs:    inputs,
			PersonID:  "test-person-id",
			RoomID:    "test-room-id",
			Created:   &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(action)
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

	// Create attachment actions plugin
	actionsPlugin := New(client, nil)

	// Get action
	action, err := actionsPlugin.Get("test-action-id")
	if err != nil {
		t.Fatalf("Failed to get attachment action: %v", err)
	}

	// Check action
	if action.ID != "test-action-id" {
		t.Errorf("Expected ID 'test-action-id', got '%s'", action.ID)
	}
	if action.Type != "submit" {
		t.Errorf("Expected type 'submit', got '%s'", action.Type)
	}
	if action.MessageID != "test-message-id" {
		t.Errorf("Expected messageId 'test-message-id', got '%s'", action.MessageID)
	}
	if action.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", action.PersonID)
	}
	if action.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", action.RoomID)
	}
	if action.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}

	// Check inputs
	if input, ok := action.Inputs["textInput"]; !ok || input != "Hello, World!" {
		t.Errorf("Expected textInput 'Hello, World!', got '%v'", input)
	}
	if input, ok := action.Inputs["checkBox"]; !ok || input != true {
		t.Errorf("Expected checkBox true, got '%v'", input)
	}
}

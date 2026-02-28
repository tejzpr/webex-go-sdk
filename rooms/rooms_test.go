/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package rooms

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/rooms" {
			t.Errorf("Expected path '/rooms', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var room Room
		if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify room
		if room.Title != "Test Room" {
			t.Errorf("Expected title 'Test Room', got '%s'", room.Title)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		lastActivity := time.Now()
		responseRoom := Room{
			ID:           "test-room-id",
			Title:        room.Title,
			TeamID:       room.TeamID,
			Type:         "group",
			IsLocked:     false,
			CreatorID:    "test-creator-id",
			Created:      &createdAt,
			LastActivity: &lastActivity,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseRoom)
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

	// Create rooms plugin
	roomsPlugin := New(client, nil)

	// Create room
	room := &Room{
		Title:  "Test Room",
		TeamID: "test-team-id",
	}

	result, err := roomsPlugin.Create(room)
	if err != nil {
		t.Fatalf("Failed to create room: %v", err)
	}

	// Check room
	if result.ID != "test-room-id" {
		t.Errorf("Expected ID 'test-room-id', got '%s'", result.ID)
	}
	if result.Title != "Test Room" {
		t.Errorf("Expected title 'Test Room', got '%s'", result.Title)
	}
	if result.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", result.TeamID)
	}
	if result.Type != "group" {
		t.Errorf("Expected type 'group', got '%s'", result.Type)
	}
	if result.IsLocked {
		t.Errorf("Expected isLocked false, got true")
	}
	if result.CreatorID != "test-creator-id" {
		t.Errorf("Expected creatorId 'test-creator-id', got '%s'", result.CreatorID)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
	if result.LastActivity == nil {
		t.Error("Expected lastActivity timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/rooms/test-room-id" {
			t.Errorf("Expected path '/rooms/test-room-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		lastActivity := time.Now()
		room := Room{
			ID:           "test-room-id",
			Title:        "Test Room",
			TeamID:       "test-team-id",
			Type:         "group",
			IsLocked:     false,
			CreatorID:    "test-creator-id",
			Created:      &createdAt,
			LastActivity: &lastActivity,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(room)
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

	// Create rooms plugin
	roomsPlugin := New(client, nil)

	// Get room
	room, err := roomsPlugin.Get("test-room-id")
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	// Check room
	if room.ID != "test-room-id" {
		t.Errorf("Expected ID 'test-room-id', got '%s'", room.ID)
	}
	if room.Title != "Test Room" {
		t.Errorf("Expected title 'Test Room', got '%s'", room.Title)
	}
	if room.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", room.TeamID)
	}
	if room.Type != "group" {
		t.Errorf("Expected type 'group', got '%s'", room.Type)
	}
	if room.IsLocked {
		t.Errorf("Expected isLocked false, got true")
	}
	if room.CreatorID != "test-creator-id" {
		t.Errorf("Expected creatorId 'test-creator-id', got '%s'", room.CreatorID)
	}
	if room.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
	if room.LastActivity == nil {
		t.Error("Expected lastActivity timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/rooms" {
			t.Errorf("Expected path '/rooms', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("teamId") != "test-team-id" {
			t.Errorf("Expected teamId 'test-team-id', got '%s'", r.URL.Query().Get("teamId"))
		}
		if r.URL.Query().Get("max") != "10" {
			t.Errorf("Expected max '10', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		lastActivity := time.Now()
		room1 := Room{
			ID:           "test-room-id-1",
			Title:        "Test Room 1",
			TeamID:       "test-team-id",
			Type:         "group",
			IsLocked:     false,
			CreatorID:    "test-creator-id",
			Created:      &createdAt,
			LastActivity: &lastActivity,
		}
		room2 := Room{
			ID:           "test-room-id-2",
			Title:        "Test Room 2",
			TeamID:       "test-team-id",
			Type:         "group",
			IsLocked:     true,
			CreatorID:    "test-creator-id",
			Created:      &createdAt,
			LastActivity: &lastActivity,
		}

		// Create response with items array
		response := struct {
			Items []Room `json:"items"`
		}{
			Items: []Room{room1, room2},
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

	// Create rooms plugin
	roomsPlugin := New(client, nil)

	// List rooms
	options := &ListOptions{
		TeamID: "test-team-id",
		Max:    10,
	}

	page, err := roomsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list rooms: %v", err)
	}

	// Check rooms
	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 rooms, got %d", len(page.Items))
	}

	room1 := page.Items[0]
	if room1.ID != "test-room-id-1" {
		t.Errorf("Expected ID 'test-room-id-1', got '%s'", room1.ID)
	}
	if room1.Title != "Test Room 1" {
		t.Errorf("Expected title 'Test Room 1', got '%s'", room1.Title)
	}
	if room1.IsLocked {
		t.Errorf("Expected isLocked false, got true")
	}

	room2 := page.Items[1]
	if room2.ID != "test-room-id-2" {
		t.Errorf("Expected ID 'test-room-id-2', got '%s'", room2.ID)
	}
	if room2.Title != "Test Room 2" {
		t.Errorf("Expected title 'Test Room 2', got '%s'", room2.Title)
	}
	if !room2.IsLocked {
		t.Errorf("Expected isLocked true, got false")
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/rooms/test-room-id" {
			t.Errorf("Expected path '/rooms/test-room-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var room Room
		if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify room
		if room.Title != "Updated Room Title" {
			t.Errorf("Expected title 'Updated Room Title', got '%s'", room.Title)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		lastActivity := time.Now()
		responseRoom := Room{
			ID:           "test-room-id",
			Title:        room.Title,
			TeamID:       "test-team-id",
			Type:         "group",
			IsLocked:     false,
			CreatorID:    "test-creator-id",
			Created:      &createdAt,
			LastActivity: &lastActivity,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseRoom)
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

	// Create rooms plugin
	roomsPlugin := New(client, nil)

	// Update room
	room := &Room{
		Title: "Updated Room Title",
	}

	result, err := roomsPlugin.Update("test-room-id", room)
	if err != nil {
		t.Fatalf("Failed to update room: %v", err)
	}

	// Check room
	if result.ID != "test-room-id" {
		t.Errorf("Expected ID 'test-room-id', got '%s'", result.ID)
	}
	if result.Title != "Updated Room Title" {
		t.Errorf("Expected title 'Updated Room Title', got '%s'", result.Title)
	}
	if result.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", result.TeamID)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/rooms/test-room-id" {
			t.Errorf("Expected path '/rooms/test-room-id', got '%s'", r.URL.Path)
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

	// Create rooms plugin
	roomsPlugin := New(client, nil)

	// Delete room
	err = roomsPlugin.Delete("test-room-id")
	if err != nil {
		t.Fatalf("Failed to delete room: %v", err)
	}
}

func TestList_PartialFailures(t *testing.T) {
	// Per eval.md: list responses may contain items with field-level errors
	// (e.g., encrypted titles that KMS failed to decrypt)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{
			"items": [
				{
					"id": "room-1",
					"title": "Normal Room",
					"type": "group",
					"isLocked": false
				},
				{
					"id": "room-2",
					"title": "eyJhbGciOiIiLCJraWQiOiIiLCJlbmMiOiIifQ....",
					"errors": {
						"title": {
							"code": "kms_failure",
							"reason": "Key management server failed to respond appropriately."
						}
					}
				}
			]
		}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, _ := webexsdk.NewClient("test-token", config)
	client.BaseURL = baseURL

	roomsPlugin := New(client, nil)
	page, err := roomsPlugin.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(page.Items))
	}

	// First room should have no errors
	if page.Items[0].Errors.HasErrors() {
		t.Error("Expected no errors on first room")
	}
	if page.Items[0].Title != "Normal Room" {
		t.Errorf("Expected 'Normal Room', got %q", page.Items[0].Title)
	}

	// Second room should have a title error
	if !page.Items[1].Errors.HasErrors() {
		t.Fatal("Expected errors on second room")
	}
	if !page.Items[1].Errors.HasFieldError("title") {
		t.Error("Expected title field error")
	}
	titleErr := page.Items[1].Errors["title"]
	if titleErr.Code != "kms_failure" {
		t.Errorf("Expected code 'kms_failure', got %q", titleErr.Code)
	}
	if page.Items[1].ID != "room-2" {
		t.Errorf("Expected id 'room-2', got %q", page.Items[1].ID)
	}
}

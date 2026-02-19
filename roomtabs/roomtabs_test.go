/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package roomtabs

import (
	"encoding/json"
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
		if r.URL.Path != "/room/tabs" {
			t.Errorf("Expected path '/room/tabs', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var tab RoomTab
		if err := json.NewDecoder(r.Body).Decode(&tab); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify tab
		if tab.RoomID != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", tab.RoomID)
		}
		if tab.DisplayName != "Test Tab" {
			t.Errorf("Expected displayName 'Test Tab', got '%s'", tab.DisplayName)
		}
		if tab.ContentURL != "https://example.com" {
			t.Errorf("Expected contentUrl 'https://example.com', got '%s'", tab.ContentURL)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseTab := RoomTab{
			ID:          "test-tab-id",
			RoomID:      tab.RoomID,
			RoomType:    "group",
			DisplayName: tab.DisplayName,
			ContentURL:  tab.ContentURL,
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseTab)
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

	// Create roomtabs plugin
	roomTabsPlugin := New(client, nil)

	// Create room tab
	tab := &RoomTab{
		RoomID:      "test-room-id",
		DisplayName: "Test Tab",
		ContentURL:  "https://example.com",
	}

	result, err := roomTabsPlugin.Create(tab)
	if err != nil {
		t.Fatalf("Failed to create room tab: %v", err)
	}

	// Check room tab
	if result.ID != "test-tab-id" {
		t.Errorf("Expected ID 'test-tab-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.RoomType != "group" {
		t.Errorf("Expected roomType 'group', got '%s'", result.RoomType)
	}
	if result.DisplayName != "Test Tab" {
		t.Errorf("Expected displayName 'Test Tab', got '%s'", result.DisplayName)
	}
	if result.ContentURL != "https://example.com" {
		t.Errorf("Expected contentUrl 'https://example.com', got '%s'", result.ContentURL)
	}
	if result.CreatorID != "test-creator-id" {
		t.Errorf("Expected creatorId 'test-creator-id', got '%s'", result.CreatorID)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/room/tabs/test-tab-id" {
			t.Errorf("Expected path '/room/tabs/test-tab-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		tab := RoomTab{
			ID:          "test-tab-id",
			RoomID:      "test-room-id",
			RoomType:    "group",
			DisplayName: "Test Tab",
			ContentURL:  "https://example.com",
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(tab)
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

	// Create roomtabs plugin
	roomTabsPlugin := New(client, nil)

	// Get room tab
	tab, err := roomTabsPlugin.Get("test-tab-id")
	if err != nil {
		t.Fatalf("Failed to get room tab: %v", err)
	}

	// Check room tab
	if tab.ID != "test-tab-id" {
		t.Errorf("Expected ID 'test-tab-id', got '%s'", tab.ID)
	}
	if tab.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", tab.RoomID)
	}
	if tab.RoomType != "group" {
		t.Errorf("Expected roomType 'group', got '%s'", tab.RoomType)
	}
	if tab.DisplayName != "Test Tab" {
		t.Errorf("Expected displayName 'Test Tab', got '%s'", tab.DisplayName)
	}
	if tab.ContentURL != "https://example.com" {
		t.Errorf("Expected contentUrl 'https://example.com', got '%s'", tab.ContentURL)
	}
	if tab.CreatorID != "test-creator-id" {
		t.Errorf("Expected creatorId 'test-creator-id', got '%s'", tab.CreatorID)
	}
	if tab.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/room/tabs" {
			t.Errorf("Expected path '/room/tabs', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("roomId") != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", r.URL.Query().Get("roomId"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		tabs := []RoomTab{
			{
				ID:          "test-tab-id-1",
				RoomID:      "test-room-id",
				RoomType:    "group",
				DisplayName: "Test Tab 1",
				ContentURL:  "https://example.com/1",
				CreatorID:   "test-creator-id",
				Created:     &createdAt,
			},
			{
				ID:          "test-tab-id-2",
				RoomID:      "test-room-id",
				RoomType:    "group",
				DisplayName: "Test Tab 2",
				ContentURL:  "https://example.com/2",
				CreatorID:   "test-creator-id",
				Created:     &createdAt,
			},
		}

		// Prepare response with items
		response := struct {
			Items []RoomTab `json:"items"`
		}{
			Items: tabs,
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

	// Create roomtabs plugin
	roomTabsPlugin := New(client, nil)

	// List room tabs
	options := &ListOptions{
		RoomID: "test-room-id",
	}
	page, err := roomTabsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list room tabs: %v", err)
	}

	// Check page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	// Check first tab
	tab := page.Items[0]
	if tab.ID != "test-tab-id-1" {
		t.Errorf("Expected ID 'test-tab-id-1', got '%s'", tab.ID)
	}
	if tab.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", tab.RoomID)
	}
	if tab.DisplayName != "Test Tab 1" {
		t.Errorf("Expected displayName 'Test Tab 1', got '%s'", tab.DisplayName)
	}
	if tab.ContentURL != "https://example.com/1" {
		t.Errorf("Expected contentUrl 'https://example.com/1', got '%s'", tab.ContentURL)
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/room/tabs/test-tab-id" {
			t.Errorf("Expected path '/room/tabs/test-tab-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var tab RoomTab
		if err := json.NewDecoder(r.Body).Decode(&tab); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify tab
		if tab.RoomID != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", tab.RoomID)
		}
		if tab.DisplayName != "Updated Tab" {
			t.Errorf("Expected displayName 'Updated Tab', got '%s'", tab.DisplayName)
		}
		if tab.ContentURL != "https://example.com/updated" {
			t.Errorf("Expected contentUrl 'https://example.com/updated', got '%s'", tab.ContentURL)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseTab := RoomTab{
			ID:          "test-tab-id",
			RoomID:      tab.RoomID,
			RoomType:    "group",
			DisplayName: tab.DisplayName,
			ContentURL:  tab.ContentURL,
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseTab)
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

	// Create roomtabs plugin
	roomTabsPlugin := New(client, nil)

	// Update room tab
	tab := &RoomTab{
		RoomID:      "test-room-id",
		DisplayName: "Updated Tab",
		ContentURL:  "https://example.com/updated",
	}

	result, err := roomTabsPlugin.Update("test-tab-id", tab)
	if err != nil {
		t.Fatalf("Failed to update room tab: %v", err)
	}

	// Check room tab
	if result.ID != "test-tab-id" {
		t.Errorf("Expected ID 'test-tab-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.DisplayName != "Updated Tab" {
		t.Errorf("Expected displayName 'Updated Tab', got '%s'", result.DisplayName)
	}
	if result.ContentURL != "https://example.com/updated" {
		t.Errorf("Expected contentUrl 'https://example.com/updated', got '%s'", result.ContentURL)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/room/tabs/test-tab-id" {
			t.Errorf("Expected path '/room/tabs/test-tab-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}

		// Write response
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

	// Create roomtabs plugin
	roomTabsPlugin := New(client, nil)

	// Delete room tab
	err = roomTabsPlugin.Delete("test-tab-id")
	if err != nil {
		t.Fatalf("Failed to delete room tab: %v", err)
	}
}

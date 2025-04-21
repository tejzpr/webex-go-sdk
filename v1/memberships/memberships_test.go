/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package memberships

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
		if r.URL.Path != "/memberships" {
			t.Errorf("Expected path '/memberships', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var membership Membership
		if err := json.NewDecoder(r.Body).Decode(&membership); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify membership
		if membership.RoomID != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", membership.RoomID)
		}
		if membership.PersonEmail != "test@example.com" {
			t.Errorf("Expected personEmail 'test@example.com', got '%s'", membership.PersonEmail)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMembership := Membership{
			ID:          "test-membership-id",
			RoomID:      membership.RoomID,
			PersonID:    "test-person-id",
			PersonEmail: membership.PersonEmail,
			IsModerator: membership.IsModerator,
			IsMonitor:   false,
			Created:     &createdAt,
			RoomType:    "group",
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseMembership)
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

	// Create memberships plugin
	membershipsPlugin := New(client, nil)

	// Create membership
	membership := &Membership{
		RoomID:      "test-room-id",
		PersonEmail: "test@example.com",
		IsModerator: true,
	}

	result, err := membershipsPlugin.Create(membership)
	if err != nil {
		t.Fatalf("Failed to create membership: %v", err)
	}

	// Check membership
	if result.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", result.PersonEmail)
	}
	if !result.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
	if result.IsMonitor {
		t.Errorf("Expected isMonitor false, got true")
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
	if result.RoomType != "group" {
		t.Errorf("Expected roomType 'group', got '%s'", result.RoomType)
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/memberships/test-membership-id" {
			t.Errorf("Expected path '/memberships/test-membership-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		membership := Membership{
			ID:          "test-membership-id",
			RoomID:      "test-room-id",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			IsModerator: true,
			IsMonitor:   false,
			Created:     &createdAt,
			RoomType:    "group",
		}

		// Write response
		_ = json.NewEncoder(w).Encode(membership)
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

	// Create memberships plugin
	membershipsPlugin := New(client, nil)

	// Get membership
	membership, err := membershipsPlugin.Get("test-membership-id")
	if err != nil {
		t.Fatalf("Failed to get membership: %v", err)
	}

	// Check membership
	if membership.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", membership.ID)
	}
	if membership.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", membership.RoomID)
	}
	if membership.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", membership.PersonID)
	}
	if membership.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", membership.PersonEmail)
	}
	if !membership.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
	if membership.IsMonitor {
		t.Errorf("Expected isMonitor false, got true")
	}
	if membership.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
	if membership.RoomType != "group" {
		t.Errorf("Expected roomType 'group', got '%s'", membership.RoomType)
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/memberships" {
			t.Errorf("Expected path '/memberships', got '%s'", r.URL.Path)
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
		membership1 := Membership{
			ID:          "test-membership-id-1",
			RoomID:      "test-room-id",
			PersonID:    "test-person-id-1",
			PersonEmail: "test1@example.com",
			IsModerator: true,
			IsMonitor:   false,
			Created:     &createdAt,
			RoomType:    "group",
		}
		membership2 := Membership{
			ID:          "test-membership-id-2",
			RoomID:      "test-room-id",
			PersonID:    "test-person-id-2",
			PersonEmail: "test2@example.com",
			IsModerator: false,
			IsMonitor:   false,
			Created:     &createdAt,
			RoomType:    "group",
		}

		// Create response with items array
		response := struct {
			Items []Membership `json:"items"`
		}{
			Items: []Membership{membership1, membership2},
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

	// Create memberships plugin
	membershipsPlugin := New(client, nil)

	// List memberships
	options := &ListOptions{
		RoomID: "test-room-id",
		Max:    50,
	}

	page, err := membershipsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list memberships: %v", err)
	}

	// Check memberships
	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 memberships, got %d", len(page.Items))
	}

	membership1 := page.Items[0]
	if membership1.ID != "test-membership-id-1" {
		t.Errorf("Expected ID 'test-membership-id-1', got '%s'", membership1.ID)
	}
	if membership1.PersonEmail != "test1@example.com" {
		t.Errorf("Expected personEmail 'test1@example.com', got '%s'", membership1.PersonEmail)
	}
	if !membership1.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}

	membership2 := page.Items[1]
	if membership2.ID != "test-membership-id-2" {
		t.Errorf("Expected ID 'test-membership-id-2', got '%s'", membership2.ID)
	}
	if membership2.PersonEmail != "test2@example.com" {
		t.Errorf("Expected personEmail 'test2@example.com', got '%s'", membership2.PersonEmail)
	}
	if membership2.IsModerator {
		t.Errorf("Expected isModerator false, got true")
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/memberships/test-membership-id" {
			t.Errorf("Expected path '/memberships/test-membership-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var membership Membership
		if err := json.NewDecoder(r.Body).Decode(&membership); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify membership
		if !membership.IsModerator {
			t.Errorf("Expected isModerator true, got false")
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMembership := Membership{
			ID:          "test-membership-id",
			RoomID:      "test-room-id",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			IsModerator: true,
			IsMonitor:   false,
			Created:     &createdAt,
			RoomType:    "group",
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseMembership)
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

	// Create memberships plugin
	membershipsPlugin := New(client, nil)

	// Update membership
	membership := &Membership{
		IsModerator: true,
	}

	result, err := membershipsPlugin.Update("test-membership-id", membership)
	if err != nil {
		t.Fatalf("Failed to update membership: %v", err)
	}

	// Check membership
	if result.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", result.PersonEmail)
	}
	if !result.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/memberships/test-membership-id" {
			t.Errorf("Expected path '/memberships/test-membership-id', got '%s'", r.URL.Path)
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

	// Create memberships plugin
	membershipsPlugin := New(client, nil)

	// Delete membership
	err = membershipsPlugin.Delete("test-membership-id")
	if err != nil {
		t.Fatalf("Failed to delete membership: %v", err)
	}
}

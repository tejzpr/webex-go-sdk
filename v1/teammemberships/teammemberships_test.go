/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package teammemberships

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
		if r.URL.Path != "/team/memberships" {
			t.Errorf("Expected path '/team/memberships', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var membership TeamMembership
		if err := json.NewDecoder(r.Body).Decode(&membership); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify membership
		if membership.TeamID != "test-team-id" {
			t.Errorf("Expected teamId 'test-team-id', got '%s'", membership.TeamID)
		}
		if membership.PersonEmail != "test@example.com" {
			t.Errorf("Expected personEmail 'test@example.com', got '%s'", membership.PersonEmail)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMembership := TeamMembership{
			ID:                "test-membership-id",
			TeamID:            membership.TeamID,
			PersonID:          "test-person-id",
			PersonEmail:       membership.PersonEmail,
			PersonDisplayName: "Test User",
			IsModerator:       membership.IsModerator,
			Created:           &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(responseMembership)
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

	// Create teammemberships plugin
	teamMembershipsPlugin := New(client, nil)

	// Create team membership
	membership := &TeamMembership{
		TeamID:      "test-team-id",
		PersonEmail: "test@example.com",
		IsModerator: true,
	}

	result, err := teamMembershipsPlugin.Create(membership)
	if err != nil {
		t.Fatalf("Failed to create team membership: %v", err)
	}

	// Check team membership
	if result.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", result.ID)
	}
	if result.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", result.TeamID)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", result.PersonEmail)
	}
	if result.PersonDisplayName != "Test User" {
		t.Errorf("Expected personDisplayName 'Test User', got '%s'", result.PersonDisplayName)
	}
	if !result.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/team/memberships/test-membership-id" {
			t.Errorf("Expected path '/team/memberships/test-membership-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		membership := TeamMembership{
			ID:                "test-membership-id",
			TeamID:            "test-team-id",
			PersonID:          "test-person-id",
			PersonEmail:       "test@example.com",
			PersonDisplayName: "Test User",
			IsModerator:       true,
			Created:           &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(membership)
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

	// Create teammemberships plugin
	teamMembershipsPlugin := New(client, nil)

	// Get team membership
	membership, err := teamMembershipsPlugin.Get("test-membership-id")
	if err != nil {
		t.Fatalf("Failed to get team membership: %v", err)
	}

	// Check team membership
	if membership.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", membership.ID)
	}
	if membership.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", membership.TeamID)
	}
	if membership.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", membership.PersonID)
	}
	if membership.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", membership.PersonEmail)
	}
	if membership.PersonDisplayName != "Test User" {
		t.Errorf("Expected personDisplayName 'Test User', got '%s'", membership.PersonDisplayName)
	}
	if !membership.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
	if membership.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/team/memberships" {
			t.Errorf("Expected path '/team/memberships', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("teamId") != "test-team-id" {
			t.Errorf("Expected teamId 'test-team-id', got '%s'", r.URL.Query().Get("teamId"))
		}
		if r.URL.Query().Get("max") != "50" {
			t.Errorf("Expected max '50', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		memberships := []TeamMembership{
			{
				ID:                "test-membership-id-1",
				TeamID:            "test-team-id",
				PersonID:          "test-person-id-1",
				PersonEmail:       "test1@example.com",
				PersonDisplayName: "Test User 1",
				IsModerator:       true,
				Created:           &createdAt,
			},
			{
				ID:                "test-membership-id-2",
				TeamID:            "test-team-id",
				PersonID:          "test-person-id-2",
				PersonEmail:       "test2@example.com",
				PersonDisplayName: "Test User 2",
				IsModerator:       false,
				Created:           &createdAt,
			},
		}

		// Prepare response with items
		response := struct {
			Items []TeamMembership `json:"items"`
		}{
			Items: memberships,
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
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create teammemberships plugin
	teamMembershipsPlugin := New(client, nil)

	// List team memberships
	options := &ListOptions{
		TeamID: "test-team-id",
		Max:    50,
	}
	page, err := teamMembershipsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list team memberships: %v", err)
	}

	// Check page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	// Check first membership
	membership := page.Items[0]
	if membership.ID != "test-membership-id-1" {
		t.Errorf("Expected ID 'test-membership-id-1', got '%s'", membership.ID)
	}
	if membership.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", membership.TeamID)
	}
	if membership.PersonID != "test-person-id-1" {
		t.Errorf("Expected personId 'test-person-id-1', got '%s'", membership.PersonID)
	}
	if membership.PersonEmail != "test1@example.com" {
		t.Errorf("Expected personEmail 'test1@example.com', got '%s'", membership.PersonEmail)
	}
	if !membership.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/team/memberships/test-membership-id" {
			t.Errorf("Expected path '/team/memberships/test-membership-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var membership TeamMembership
		if err := json.NewDecoder(r.Body).Decode(&membership); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify isModerator
		if !membership.IsModerator {
			t.Errorf("Expected isModerator true, got false")
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMembership := TeamMembership{
			ID:                "test-membership-id",
			TeamID:            "test-team-id",
			PersonID:          "test-person-id",
			PersonEmail:       "test@example.com",
			PersonDisplayName: "Test User",
			IsModerator:       true,
			Created:           &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(responseMembership)
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

	// Create teammemberships plugin
	teamMembershipsPlugin := New(client, nil)

	// Update team membership
	result, err := teamMembershipsPlugin.Update("test-membership-id", true)
	if err != nil {
		t.Fatalf("Failed to update team membership: %v", err)
	}

	// Check team membership
	if result.ID != "test-membership-id" {
		t.Errorf("Expected ID 'test-membership-id', got '%s'", result.ID)
	}
	if result.TeamID != "test-team-id" {
		t.Errorf("Expected teamId 'test-team-id', got '%s'", result.TeamID)
	}
	if !result.IsModerator {
		t.Errorf("Expected isModerator true, got false")
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/team/memberships/test-membership-id" {
			t.Errorf("Expected path '/team/memberships/test-membership-id', got '%s'", r.URL.Path)
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

	// Create teammemberships plugin
	teamMembershipsPlugin := New(client, nil)

	// Delete team membership
	err = teamMembershipsPlugin.Delete("test-membership-id")
	if err != nil {
		t.Fatalf("Failed to delete team membership: %v", err)
	}
}

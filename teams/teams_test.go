/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package teams

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
		if r.URL.Path != "/teams" {
			t.Errorf("Expected path '/teams', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var team Team
		if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify team
		if team.Name != "Test Team" {
			t.Errorf("Expected name 'Test Team', got '%s'", team.Name)
		}
		if team.Description != "Team for testing" {
			t.Errorf("Expected description 'Team for testing', got '%s'", team.Description)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseTeam := Team{
			ID:          "test-team-id",
			Name:        team.Name,
			Description: team.Description,
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseTeam)
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

	// Create teams plugin
	teamsPlugin := New(client, nil)

	// Create team
	team := &Team{
		Name:        "Test Team",
		Description: "Team for testing",
	}

	result, err := teamsPlugin.Create(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Check team
	if result.ID != "test-team-id" {
		t.Errorf("Expected ID 'test-team-id', got '%s'", result.ID)
	}
	if result.Name != "Test Team" {
		t.Errorf("Expected name 'Test Team', got '%s'", result.Name)
	}
	if result.Description != "Team for testing" {
		t.Errorf("Expected description 'Team for testing', got '%s'", result.Description)
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
		if r.URL.Path != "/teams/test-team-id" {
			t.Errorf("Expected path '/teams/test-team-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		team := Team{
			ID:          "test-team-id",
			Name:        "Test Team",
			Description: "Team for testing",
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(team)
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

	// Create teams plugin
	teamsPlugin := New(client, nil)

	// Get team
	team, err := teamsPlugin.Get("test-team-id")
	if err != nil {
		t.Fatalf("Failed to get team: %v", err)
	}

	// Check team
	if team.ID != "test-team-id" {
		t.Errorf("Expected ID 'test-team-id', got '%s'", team.ID)
	}
	if team.Name != "Test Team" {
		t.Errorf("Expected name 'Test Team', got '%s'", team.Name)
	}
	if team.Description != "Team for testing" {
		t.Errorf("Expected description 'Team for testing', got '%s'", team.Description)
	}
	if team.CreatorID != "test-creator-id" {
		t.Errorf("Expected creatorId 'test-creator-id', got '%s'", team.CreatorID)
	}
	if team.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/teams" {
			t.Errorf("Expected path '/teams', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("max") != "50" {
			t.Errorf("Expected max '50', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		teams := []Team{
			{
				ID:          "test-team-id-1",
				Name:        "Test Team 1",
				Description: "Team for testing 1",
				CreatorID:   "test-creator-id",
				Created:     &createdAt,
			},
			{
				ID:          "test-team-id-2",
				Name:        "Test Team 2",
				Description: "Team for testing 2",
				CreatorID:   "test-creator-id",
				Created:     &createdAt,
			},
		}

		// Prepare response with items
		response := struct {
			Items []Team `json:"items"`
		}{
			Items: teams,
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

	// Create teams plugin
	teamsPlugin := New(client, nil)

	// List teams
	options := &ListOptions{
		Max: 50,
	}
	page, err := teamsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list teams: %v", err)
	}

	// Check page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	// Check first team
	team := page.Items[0]
	if team.ID != "test-team-id-1" {
		t.Errorf("Expected ID 'test-team-id-1', got '%s'", team.ID)
	}
	if team.Name != "Test Team 1" {
		t.Errorf("Expected name 'Test Team 1', got '%s'", team.Name)
	}
	if team.Description != "Team for testing 1" {
		t.Errorf("Expected description 'Team for testing 1', got '%s'", team.Description)
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/teams/test-team-id" {
			t.Errorf("Expected path '/teams/test-team-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var team Team
		if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify team
		if team.Name != "Updated Team" {
			t.Errorf("Expected name 'Updated Team', got '%s'", team.Name)
		}
		if team.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", team.Description)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseTeam := Team{
			ID:          "test-team-id",
			Name:        team.Name,
			Description: team.Description,
			CreatorID:   "test-creator-id",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseTeam)
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

	// Create teams plugin
	teamsPlugin := New(client, nil)

	// Update team
	team := &Team{
		Name:        "Updated Team",
		Description: "Updated description",
	}

	result, err := teamsPlugin.Update("test-team-id", team)
	if err != nil {
		t.Fatalf("Failed to update team: %v", err)
	}

	// Check team
	if result.ID != "test-team-id" {
		t.Errorf("Expected ID 'test-team-id', got '%s'", result.ID)
	}
	if result.Name != "Updated Team" {
		t.Errorf("Expected name 'Updated Team', got '%s'", result.Name)
	}
	if result.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", result.Description)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/teams/test-team-id" {
			t.Errorf("Expected path '/teams/test-team-id', got '%s'", r.URL.Path)
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

	// Create teams plugin
	teamsPlugin := New(client, nil)

	// Delete team
	err = teamsPlugin.Delete("test-team-id")
	if err != nil {
		t.Fatalf("Failed to delete team: %v", err)
	}
}

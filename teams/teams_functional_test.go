//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package teams

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func functionalClient(t *testing.T) *webexsdk.Client {
	t.Helper()
	token := os.Getenv("WEBEX_ACCESS_TOKEN")
	if token == "" {
		t.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
	}
	client, err := webexsdk.NewClient(token, &webexsdk.Config{
		BaseURL: "https://webexapis.com/v1",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Webex client: %v", err)
	}
	return client
}

// TestFunctionalTeamsCRUD tests the full Create → Get → Update → Delete lifecycle
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamsCRUD -v ./teams/
func TestFunctionalTeamsCRUD(t *testing.T) {
	client := functionalClient(t)
	teamsClient := New(client, nil)

	// Create
	team, err := teamsClient.Create(&Team{Name: "SDK Func Test Team"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if err := teamsClient.Delete(team.ID); err != nil {
			t.Logf("Warning: cleanup delete failed: %v", err)
		}
	}()

	if team.ID == "" {
		t.Fatal("Created team has empty ID")
	}
	t.Logf("Created team: ID=%s Name=%q", team.ID, team.Name)

	// Get
	got, err := teamsClient.Get(team.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != team.ID {
		t.Errorf("Get ID mismatch: got %s, want %s", got.ID, team.ID)
	}
	if got.Name != "SDK Func Test Team" {
		t.Errorf("Get name mismatch: got %q", got.Name)
	}
	t.Logf("Get confirmed: Name=%q CreatorID=%s", got.Name, got.CreatorID)

	// Update
	updated, err := teamsClient.Update(team.ID, &Team{Name: "SDK Func Test Team Updated"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "SDK Func Test Team Updated" {
		t.Errorf("Update name mismatch: got %q", updated.Name)
	}
	t.Logf("Updated team name to: %q", updated.Name)
}

// TestFunctionalTeamsList tests listing teams
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamsList -v ./teams/
func TestFunctionalTeamsList(t *testing.T) {
	client := functionalClient(t)
	teamsClient := New(client, nil)

	page, err := teamsClient.List(&ListOptions{Max: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d teams", len(page.Items))
	for i, team := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Name=%q CreatorID=%s\n",
			i+1, team.ID, team.Name, team.CreatorID)
	}
}

// TestFunctionalTeamsListPagination tests pagination through teams
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamsListPagination -v ./teams/
func TestFunctionalTeamsListPagination(t *testing.T) {
	client := functionalClient(t)
	teamsClient := New(client, nil)

	// Create 4 teams for pagination
	createdIDs := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		team, err := teamsClient.Create(&Team{
			Name: fmt.Sprintf("SDK Pagination Team %d", i+1),
		})
		if err != nil {
			t.Fatalf("Failed to create team %d: %v", i+1, err)
		}
		createdIDs = append(createdIDs, team.ID)
	}
	defer func() {
		for _, id := range createdIDs {
			if err := teamsClient.Delete(id); err != nil {
				t.Logf("Warning: cleanup delete team %s failed: %v", id, err)
			}
		}
	}()

	t.Logf("Created %d teams for pagination test", len(createdIDs))

	page, err := teamsClient.List(&ListOptions{Max: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	totalItems := len(page.Items)
	pageCount := 1
	t.Logf("Page %d: %d items, hasNext=%v", pageCount, len(page.Items), page.HasNext)

	for page.HasNext && pageCount < 10 {
		nextPage, err := page.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		page.Page = nextPage
		pageCount++
		totalItems += len(nextPage.Items)
		t.Logf("Page %d: %d raw items, hasNext=%v", pageCount, len(nextPage.Items), nextPage.HasNext)
	}

	t.Logf("Pagination complete: %d total items across %d pages", totalItems, pageCount)
}

// TestFunctionalTeamsNotFound tests structured error on invalid team ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamsNotFound -v ./teams/
func TestFunctionalTeamsNotFound(t *testing.T) {
	client := functionalClient(t)
	teamsClient := New(client, nil)

	_, err := teamsClient.Get("invalid-team-id")
	if err == nil {
		t.Fatal("Expected error for invalid team ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

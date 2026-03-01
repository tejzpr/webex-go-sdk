//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package teammemberships

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/teams"
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

// TestFunctionalTeamMembershipsLifecycle tests listing and getting team memberships
// The creator is automatically a member of the team, so we can list/get without adding anyone
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamMembershipsLifecycle -v ./teammemberships/
func TestFunctionalTeamMembershipsLifecycle(t *testing.T) {
	client := functionalClient(t)
	tmClient := New(client, nil)
	teamsClient := teams.New(client, nil)

	// Create a team
	team, err := teamsClient.Create(&teams.Team{Name: "SDK TeamMemberships Lifecycle Test"})
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}
	defer teamsClient.Delete(team.ID)

	// List memberships for the team
	page, err := tmClient.List(&ListOptions{TeamID: team.ID})
	if err != nil {
		t.Fatalf("List team memberships failed: %v", err)
	}

	t.Logf("Found %d team memberships", len(page.Items))
	if len(page.Items) == 0 {
		t.Fatal("Expected at least 1 membership (the creator)")
	}

	// The first membership should be the creator
	membership := page.Items[0]
	t.Logf("First membership: ID=%s PersonID=%s DisplayName=%s IsModerator=%v",
		membership.ID, membership.PersonID, membership.PersonDisplayName, membership.IsModerator)

	if membership.TeamID != team.ID {
		t.Errorf("Membership TeamID = %s, want %s", membership.TeamID, team.ID)
	}

	// Get by ID
	got, err := tmClient.Get(membership.ID)
	if err != nil {
		t.Fatalf("Get team membership failed: %v", err)
	}
	if got.ID != membership.ID {
		t.Errorf("Get returned ID=%s, want %s", got.ID, membership.ID)
	}
	t.Logf("Get membership: ID=%s PersonDisplayName=%s", got.ID, got.PersonDisplayName)

	// Update moderator status — toggle to false (creator default is moderator)
	updated, err := tmClient.Update(membership.ID, false)
	if err != nil {
		// Some accounts may not allow removing last moderator
		t.Logf("Update moderator to false returned error (may be expected): %v", err)
	} else {
		t.Logf("Updated membership: IsModerator=%v", updated.IsModerator)
		// Restore moderator
		_, _ = tmClient.Update(membership.ID, true)
	}
}

// TestFunctionalTeamMembershipsListByTeam tests listing with Max parameter
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamMembershipsListByTeam -v ./teammemberships/
func TestFunctionalTeamMembershipsListByTeam(t *testing.T) {
	client := functionalClient(t)
	tmClient := New(client, nil)
	teamsClient := teams.New(client, nil)

	team, err := teamsClient.Create(&teams.Team{Name: "SDK TeamMemberships List Test"})
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}
	defer teamsClient.Delete(team.ID)

	page, err := tmClient.List(&ListOptions{TeamID: team.ID, Max: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Listed %d memberships (max 10)", len(page.Items))
	for _, m := range page.Items {
		t.Logf("  Membership: ID=%s Person=%s Moderator=%v", m.ID, m.PersonDisplayName, m.IsModerator)
	}
}

// TestFunctionalTeamMembershipsNotFound tests structured error for invalid membership ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamMembershipsNotFound -v ./teammemberships/
func TestFunctionalTeamMembershipsNotFound(t *testing.T) {
	client := functionalClient(t)
	tmClient := New(client, nil)

	_, err := tmClient.Get("invalid-team-membership-id")
	if err == nil {
		t.Fatal("Expected error for invalid membership ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) {
		t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
	} else {
		t.Logf("Error is not an APIError: %v", err)
	}

	if webexsdk.IsNotFound(err) {
		t.Log("Correctly identified as NotFound error")
	}
}

// TestFunctionalTeamMembershipsCursorNavigation tests PageFromCursor with team memberships
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTeamMembershipsCursorNavigation -v ./teammemberships/
func TestFunctionalTeamMembershipsCursorNavigation(t *testing.T) {
	client := functionalClient(t)
	tmClient := New(client, nil)
	teamsClient := teams.New(client, nil)

	// Create a team to have at least one team membership
	team, err := teamsClient.Create(&teams.Team{Name: "SDK TeamMemberships Cursor Test"})
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}
	defer func() {
		if err := teamsClient.Delete(team.ID); err != nil {
			t.Logf("Warning: cleanup delete team failed: %v", err)
		}
	}()

	page, err := tmClient.List(&ListOptions{TeamID: team.ID, Max: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if !page.HasNext {
		t.Log("Only one page of results — skipping cursor navigation test")
		return
	}

	cursor := page.NextPage
	t.Logf("Saved cursor: %s", cursor)

	directPage, err := client.PageFromCursor(cursor)
	if err != nil {
		t.Fatalf("PageFromCursor failed: %v", err)
	}

	t.Logf("Direct cursor navigation: got %d items, hasNext=%v", len(directPage.Items), directPage.HasNext)
	if len(directPage.Items) == 0 {
		t.Error("Expected items from cursor navigation")
	}
}

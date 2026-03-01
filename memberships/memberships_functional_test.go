//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package memberships

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/rooms"
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

// TestFunctionalMembershipsLifecycle tests listing/getting memberships in a room
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMembershipsLifecycle -v ./memberships/
func TestFunctionalMembershipsLifecycle(t *testing.T) {
	client := functionalClient(t)
	membershipsClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	// Create a test room (the authenticated user is automatically a member)
	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Memberships Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// List memberships for this room
	page, err := membershipsClient.List(&ListOptions{RoomID: room.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d memberships in test room", len(page.Items))
	if len(page.Items) == 0 {
		t.Fatal("Expected at least 1 membership (creator)")
	}

	for i, m := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s PersonEmail=%s DisplayName=%q Moderator=%v\n",
			i+1, m.ID, m.PersonEmail, m.PersonDisplayName, m.IsModerator)
	}

	// Get a specific membership
	membership := page.Items[0]
	got, err := membershipsClient.Get(membership.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != membership.ID {
		t.Errorf("Get ID mismatch: got %s, want %s", got.ID, membership.ID)
	}
	t.Logf("Get confirmed: PersonEmail=%s RoomID=%s", got.PersonEmail, got.RoomID)
}

// TestFunctionalMembershipsListByPerson tests listing memberships by person
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMembershipsListByPerson -v ./memberships/
func TestFunctionalMembershipsListByPerson(t *testing.T) {
	client := functionalClient(t)
	membershipsClient := New(client, nil)

	// Get the current user's email by calling people/me
	resp, err := client.Request("GET", "people/me", nil, nil)
	if err != nil {
		t.Fatalf("Failed to get current user: %v", err)
	}

	var me struct {
		Emails []string `json:"emails"`
	}
	if err := webexsdk.ParseResponse(resp, &me); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(me.Emails) == 0 {
		t.Fatal("Current user has no emails")
	}

	// List memberships for the current user
	page, err := membershipsClient.List(&ListOptions{
		PersonEmail: me.Emails[0],
		Max:         10,
	})
	if err != nil {
		t.Fatalf("List by person email failed: %v", err)
	}

	t.Logf("Found %d memberships for %s", len(page.Items), me.Emails[0])
	for i, m := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] RoomID=%s Type=%s Moderator=%v\n",
			i+1, m.RoomID, m.RoomType, m.IsModerator)
	}
}

// TestFunctionalMembershipsNotFound tests structured error on invalid membership ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMembershipsNotFound -v ./memberships/
func TestFunctionalMembershipsNotFound(t *testing.T) {
	client := functionalClient(t)
	membershipsClient := New(client, nil)

	_, err := membershipsClient.Get("invalid-membership-id")
	if err == nil {
		t.Fatal("Expected error for invalid membership ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}
// TestFunctionalMembershipsCursorNavigation tests PageFromCursor with memberships
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMembershipsCursorNavigation -v ./memberships/
func TestFunctionalMembershipsCursorNavigation(t *testing.T) {
	client := functionalClient(t)
	membershipsClient := New(client, nil)

	page, err := membershipsClient.List(&ListOptions{Max: 1})
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
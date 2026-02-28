//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package roomtabs

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/rooms"
	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// skipOnAPIError skips the test if the error is an API error (bots may lack room tab permissions).
func skipOnAPIError(t *testing.T, err error, msg string) {
	t.Helper()
	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) && (apiErr.StatusCode == 400 || apiErr.StatusCode == 403) {
		t.Skipf("Skipping: %s: %v", msg, err)
	}
}

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

// TestFunctionalRoomTabsCRUD tests the full lifecycle: Create → Get → Update → Delete
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomTabsCRUD -v ./roomtabs/
func TestFunctionalRoomTabsCRUD(t *testing.T) {
	client := functionalClient(t)
	tabsClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	// Create a group room for room tabs
	room, err := roomsClient.Create(&rooms.Room{Title: "SDK RoomTabs CRUD Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Create a tab
	tab, err := tabsClient.Create(&RoomTab{
		RoomID:      room.ID,
		DisplayName: "SDK Test Tab",
		ContentURL:  "https://www.example.com",
	})
	if err != nil {
		skipOnAPIError(t, err, "bots may not support room tabs")
		t.Fatalf("Create tab failed: %v", err)
	}
	defer tabsClient.Delete(tab.ID)

	t.Logf("Created tab: ID=%s DisplayName=%s ContentURL=%s", tab.ID, tab.DisplayName, tab.ContentURL)

	if tab.ID == "" {
		t.Error("Tab ID is empty")
	}
	if tab.RoomID != room.ID {
		t.Errorf("Tab RoomID = %s, want %s", tab.RoomID, room.ID)
	}
	if tab.DisplayName != "SDK Test Tab" {
		t.Errorf("Tab DisplayName = %s, want %s", tab.DisplayName, "SDK Test Tab")
	}

	// Get
	got, err := tabsClient.Get(tab.ID)
	if err != nil {
		t.Fatalf("Get tab failed: %v", err)
	}
	if got.ID != tab.ID {
		t.Errorf("Get returned ID=%s, want %s", got.ID, tab.ID)
	}
	t.Logf("Get tab: ID=%s DisplayName=%s", got.ID, got.DisplayName)

	// Update
	updated, err := tabsClient.Update(tab.ID, &RoomTab{
		RoomID:      room.ID,
		DisplayName: "Updated SDK Tab",
		ContentURL:  "https://www.example.com/updated",
	})
	if err != nil {
		t.Fatalf("Update tab failed: %v", err)
	}
	if updated.DisplayName != "Updated SDK Tab" {
		t.Errorf("Updated DisplayName = %s, want %s", updated.DisplayName, "Updated SDK Tab")
	}
	t.Logf("Updated tab: DisplayName=%s ContentURL=%s", updated.DisplayName, updated.ContentURL)

	// Delete
	err = tabsClient.Delete(tab.ID)
	if err != nil {
		t.Fatalf("Delete tab failed: %v", err)
	}
	t.Log("Deleted tab successfully")

	// Verify deletion
	_, err = tabsClient.Get(tab.ID)
	if err == nil {
		t.Error("Expected error after deleting tab, got nil")
	}
}

// TestFunctionalRoomTabsList tests listing tabs in a room
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomTabsList -v ./roomtabs/
func TestFunctionalRoomTabsList(t *testing.T) {
	client := functionalClient(t)
	tabsClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK RoomTabs List Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Create a couple tabs
	tab1, err := tabsClient.Create(&RoomTab{
		RoomID:      room.ID,
		DisplayName: "Tab One",
		ContentURL:  "https://www.example.com/one",
	})
	if err != nil {
		skipOnAPIError(t, err, "bots may not support room tabs")
		t.Fatalf("Create tab1 failed: %v", err)
	}
	defer tabsClient.Delete(tab1.ID)

	tab2, err := tabsClient.Create(&RoomTab{
		RoomID:      room.ID,
		DisplayName: "Tab Two",
		ContentURL:  "https://www.example.com/two",
	})
	if err != nil {
		t.Fatalf("Create tab2 failed: %v", err)
	}
	defer tabsClient.Delete(tab2.ID)

	// List
	page, err := tabsClient.List(&ListOptions{RoomID: room.ID})
	if err != nil {
		t.Fatalf("List tabs failed: %v", err)
	}

	t.Logf("Listed %d tabs in room", len(page.Items))
	if len(page.Items) < 2 {
		t.Errorf("Expected at least 2 tabs, got %d", len(page.Items))
	}

	for _, tab := range page.Items {
		t.Logf("  Tab: ID=%s DisplayName=%s", tab.ID, tab.DisplayName)
	}
}

// TestFunctionalRoomTabsNotFound tests structured error handling for invalid tab ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomTabsNotFound -v ./roomtabs/
func TestFunctionalRoomTabsNotFound(t *testing.T) {
	client := functionalClient(t)
	tabsClient := New(client, nil)

	_, err := tabsClient.Get("invalid-tab-id-does-not-exist")
	if err == nil {
		t.Fatal("Expected error for invalid tab ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) {
		t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
		if !webexsdk.IsNotFound(err) {
			t.Logf("Error is not NotFound (status=%d), which may be expected for malformed IDs", apiErr.StatusCode)
		}
	} else {
		t.Logf("Error is not an APIError: %v", err)
	}
}

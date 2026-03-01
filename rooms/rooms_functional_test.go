//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package rooms

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// helper to create a Webex client from env token
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

// TestFunctionalRoomsCRUD tests the full Create → Get → Update → Delete lifecycle
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsCRUD -v ./rooms/
func TestFunctionalRoomsCRUD(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	// Create
	room, err := roomsClient.Create(&Room{Title: "SDK Func Test Room"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if err := roomsClient.Delete(room.ID); err != nil {
			t.Logf("Warning: cleanup delete failed: %v", err)
		}
	}()

	if room.ID == "" {
		t.Fatal("Created room has empty ID")
	}
	t.Logf("Created room: ID=%s Title=%q Type=%s", room.ID, room.Title, room.Type)

	// Get
	got, err := roomsClient.Get(room.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != room.ID {
		t.Errorf("Get returned wrong ID: got %s, want %s", got.ID, room.ID)
	}
	if got.Title != "SDK Func Test Room" {
		t.Errorf("Get title mismatch: got %q", got.Title)
	}
	t.Logf("Get confirmed: Title=%q CreatorID=%s", got.Title, got.CreatorID)

	// Update
	updated, err := roomsClient.Update(room.ID, &Room{Title: "SDK Func Test Room Updated"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Title != "SDK Func Test Room Updated" {
		t.Errorf("Update title mismatch: got %q", updated.Title)
	}
	t.Logf("Updated room title to: %q", updated.Title)

	// Delete is handled by defer
}

// TestFunctionalRoomsList tests listing rooms
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsList -v ./rooms/
func TestFunctionalRoomsList(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	page, err := roomsClient.List(&ListOptions{Max: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d rooms", len(page.Items))
	for i, r := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Type=%s Locked=%v\n",
			i+1, r.ID, r.Title, r.Type, r.IsLocked)
	}
}

// TestFunctionalRoomsListByType tests listing rooms filtered by type
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsListByType -v ./rooms/
func TestFunctionalRoomsListByType(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	for _, roomType := range []string{"direct", "group"} {
		t.Run(roomType, func(t *testing.T) {
			page, err := roomsClient.List(&ListOptions{Type: roomType, Max: 5})
			if err != nil {
				t.Fatalf("List type=%s failed: %v", roomType, err)
			}
			t.Logf("Type %s: found %d rooms", roomType, len(page.Items))
			for i, r := range page.Items {
				_, _ = fmt.Fprintf(os.Stdout, "  [%d] %q (type=%s)\n", i+1, r.Title, r.Type)
			}
		})
	}
}

// TestFunctionalRoomsListPagination tests Link-header based pagination
// Creates 6 rooms, lists with Max=2, traverses all pages
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsListPagination -v ./rooms/
func TestFunctionalRoomsListPagination(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	// Create 6 test rooms for pagination
	const numRooms = 6
	createdIDs := make([]string, 0, numRooms)
	for i := 0; i < numRooms; i++ {
		room, err := roomsClient.Create(&Room{
			Title: fmt.Sprintf("SDK Pagination Test %d", i+1),
		})
		if err != nil {
			t.Fatalf("Failed to create room %d: %v", i+1, err)
		}
		createdIDs = append(createdIDs, room.ID)
	}
	defer func() {
		for _, id := range createdIDs {
			if err := roomsClient.Delete(id); err != nil {
				t.Logf("Warning: cleanup delete room %s failed: %v", id, err)
			}
		}
	}()

	t.Logf("Created %d rooms for pagination test", len(createdIDs))

	// List with small page size
	page, err := roomsClient.List(&ListOptions{Max: 2, Type: "group"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	totalItems := len(page.Items)
	pageCount := 1
	t.Logf("Page %d: %d items, hasNext=%v", pageCount, len(page.Items), page.HasNext)

	// Save cursor for direct navigation test
	var page2Cursor string
	if page.HasNext {
		page2Cursor = page.NextPage
		t.Logf("Saved page 2 cursor: %s", page2Cursor)
	}

	// Traverse pages (limit to 10 to avoid runaway)
	for page.HasNext && pageCount < 10 {
		nextPage, err := page.Next()
		if err != nil {
			t.Fatalf("Next() failed on page %d: %v", pageCount, err)
		}

		page.Page = nextPage
		pageCount++
		totalItems += len(nextPage.Items)
		t.Logf("Page %d: %d raw items, hasNext=%v", pageCount, len(nextPage.Items), nextPage.HasNext)
	}

	t.Logf("Pagination complete: %d total items across %d pages", totalItems, pageCount)
	if pageCount < 2 {
		t.Logf("Note: only 1 page returned; the account may have fewer group rooms than Max")
	}

	// Test direct cursor navigation (PageFromCursor)
	if page2Cursor != "" {
		t.Log("Testing direct cursor navigation to page 2...")
		directPage, err := client.PageFromCursor(page2Cursor)
		if err != nil {
			t.Fatalf("PageFromCursor failed: %v", err)
		}
		t.Logf("Direct cursor navigation: got %d items, hasNext=%v", len(directPage.Items), directPage.HasNext)
		if len(directPage.Items) == 0 {
			t.Error("Expected items from direct cursor navigation")
		}
	} else {
		t.Log("Skipping cursor navigation test — only one page of results")
	}
}

// TestFunctionalRoomsNotFound tests structured error on invalid room ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsNotFound -v ./rooms/
func TestFunctionalRoomsNotFound(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	_, err := roomsClient.Get("invalid-room-id-does-not-exist")
	if err == nil {
		t.Fatal("Expected error for invalid room ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)

	if webexsdk.IsNotFound(err) {
		t.Logf("IsNotFound correctly returned true")
	} else {
		t.Logf("IsNotFound returned false (status was %d)", apiErr.StatusCode)
	}
}

// TestFunctionalRoomsPartialFailures tests observational partial failures (ResourceErrors)
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRoomsPartialFailures -v ./rooms/
func TestFunctionalRoomsPartialFailures(t *testing.T) {
	client := functionalClient(t)
	roomsClient := New(client, nil)

	page, err := roomsClient.List(&ListOptions{Max: 50})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	errCount := 0
	for _, r := range page.Items {
		if r.Errors.HasErrors() {
			errCount++
			t.Logf("Room %s has partial errors:", r.ID)
			for field, fieldErr := range r.Errors {
				t.Logf("  field=%s code=%s reason=%s", field, fieldErr.Code, fieldErr.Reason)
			}
		}
	}

	if errCount == 0 {
		t.Logf("No partial failures found in %d rooms (this is normal)", len(page.Items))
	} else {
		t.Logf("Found %d rooms with partial failures out of %d", errCount, len(page.Items))
	}
}

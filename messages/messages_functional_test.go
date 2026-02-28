//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package messages

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/rooms"
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

// TestFunctionalMessagesCRUD tests Create → Get → Update → Delete for messages
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesCRUD -v ./messages/
func TestFunctionalMessagesCRUD(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	// Create a test room
	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Messages CRUD Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer func() {
		if err := roomsClient.Delete(room.ID); err != nil {
			t.Logf("Warning: cleanup room delete failed: %v", err)
		}
	}()

	// Create message
	msg, err := messagesClient.Create(&Message{
		RoomID: room.ID,
		Text:   "Hello from SDK functional test!",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if err := messagesClient.Delete(msg.ID); err != nil {
			t.Logf("Warning: cleanup message delete failed: %v", err)
		}
	}()

	if msg.ID == "" {
		t.Fatal("Created message has empty ID")
	}
	t.Logf("Created message: ID=%s Text=%q", msg.ID, msg.Text)

	// Get message
	got, err := messagesClient.Get(msg.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != msg.ID {
		t.Errorf("Get returned wrong ID: got %s, want %s", got.ID, msg.ID)
	}
	if got.Text != "Hello from SDK functional test!" {
		t.Errorf("Get text mismatch: got %q", got.Text)
	}
	t.Logf("Get confirmed: Text=%q PersonEmail=%s", got.Text, got.PersonEmail)

	// Update message
	updated, err := messagesClient.Update(msg.ID, &Message{
		RoomID: room.ID,
		Text:   "Updated message from SDK test",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Text != "Updated message from SDK test" {
		t.Errorf("Update text mismatch: got %q", updated.Text)
	}
	t.Logf("Updated message text to: %q", updated.Text)
}

// TestFunctionalMessagesMarkdown tests creating a message with markdown
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesMarkdown -v ./messages/
func TestFunctionalMessagesMarkdown(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Markdown Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	msg, err := messagesClient.Create(&Message{
		RoomID:   room.ID,
		Markdown: "**Bold** _italic_ `code` [link](https://example.com)",
	})
	if err != nil {
		t.Fatalf("Create markdown message failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	t.Logf("Created markdown message: ID=%s", msg.ID)
	if msg.HTML != "" {
		t.Logf("HTML rendered: %s", msg.HTML)
	}
}

// TestFunctionalMessagesRemoteFileAttachment tests creating a message with a remote file URL
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesRemoteFileAttachment -v ./messages/
func TestFunctionalMessagesRemoteFileAttachment(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Remote File Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	msg, err := messagesClient.Create(&Message{
		RoomID: room.ID,
		Text:   "Here is a test image",
		Files:  []string{"https://www.webex.com/content/dam/wbx/global/images/webex-logo.png"},
	})
	if err != nil {
		t.Fatalf("Create message with remote file failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	t.Logf("Created message with remote file: ID=%s Files=%v", msg.ID, msg.Files)
	if len(msg.Files) == 0 {
		t.Error("Expected at least one file URL in response")
	}
}

// TestFunctionalMessagesLocalFileAttachment tests creating a message with an inline file
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesLocalFileAttachment -v ./messages/
func TestFunctionalMessagesLocalFileAttachment(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK File Upload Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Upload a small text file using FileBytes
	fileContent := []byte("This is a test file from the SDK functional test.\nLine 2.\n")
	msg, err := messagesClient.CreateWithAttachment(
		&Message{RoomID: room.ID, Text: "Test file upload"},
		&FileUpload{FileName: "test-upload.txt", FileBytes: fileContent},
	)
	if err != nil {
		t.Fatalf("CreateWithAttachment failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	t.Logf("Created message with attachment: ID=%s Files=%v", msg.ID, msg.Files)
	if len(msg.Files) == 0 {
		t.Error("Expected file URL in response")
	}
}

// TestFunctionalMessagesAdaptiveCard tests creating a message with an adaptive card
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesAdaptiveCard -v ./messages/
func TestFunctionalMessagesAdaptiveCard(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Adaptive Card Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	cardBody := map[string]interface{}{
		"type":    "AdaptiveCard",
		"version": "1.3",
		"body": []map[string]interface{}{
			{
				"type": "TextBlock",
				"text": "SDK Functional Test Card",
				"size": "Medium",
			},
			{
				"type": "TextBlock",
				"text": "This card was created by the Go SDK functional test suite.",
				"wrap": true,
			},
		},
	}

	card := NewAdaptiveCard(cardBody)
	msg, err := messagesClient.CreateWithAdaptiveCard(
		&Message{RoomID: room.ID},
		card,
		"Adaptive Card (fallback text)",
	)
	if err != nil {
		t.Fatalf("CreateWithAdaptiveCard failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	t.Logf("Created adaptive card message: ID=%s", msg.ID)
	if len(msg.Attachments) > 0 {
		t.Logf("Attachments: %d", len(msg.Attachments))
	}
}

// TestFunctionalMessagesListPagination tests listing messages with pagination
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesListPagination -v ./messages/
func TestFunctionalMessagesListPagination(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Messages Pagination Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Create 6 messages
	for i := 0; i < 6; i++ {
		_, err := messagesClient.Create(&Message{
			RoomID: room.ID,
			Text:   fmt.Sprintf("Pagination test message %d", i+1),
		})
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i+1, err)
		}
	}

	// List with small page: note that the room creation message (+bot join) may
	// add extra items, so there could be 7+ messages
	page, err := messagesClient.List(&ListOptions{RoomID: room.ID, Max: 2})
	if err != nil {
		// Bots may get 403 when listing messages; skip if so
		var apiErr *webexsdk.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
			t.Skipf("Skipping: bot got 403 listing messages (expected for some token types): %v", err)
		}
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

// TestFunctionalMessagesNotFound tests structured error on invalid message ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMessagesNotFound -v ./messages/
func TestFunctionalMessagesNotFound(t *testing.T) {
	client := functionalClient(t)
	messagesClient := New(client, nil)

	_, err := messagesClient.Get("invalid-message-id")
	if err == nil {
		t.Fatal("Expected error for invalid message ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

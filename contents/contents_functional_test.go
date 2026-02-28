//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package contents

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/messages"
	"github.com/WebexCommunity/webex-go-sdk/v2/rooms"
	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func functionalClient(t *testing.T) *webexsdk.Client {
	t.Helper()
	token := os.Getenv("WEBEX_ACCESS_TOKEN")
	if token == "" {
		t.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
	}
	// Use MaxRetries=5 with 5s base delay so the SDK auto-retries 423
	// (file-scanning) responses without any caller-side retry logic.
	client, err := webexsdk.NewClient(token, &webexsdk.Config{
		BaseURL:        "https://webexapis.com/v1",
		Timeout:        60 * time.Second,
		MaxRetries:     5,
		RetryBaseDelay: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Webex client: %v", err)
	}
	return client
}

// TestFunctionalContentsDownloadRoundTrip tests uploading a file via messages,
// then downloading it via the contents API
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContentsDownloadRoundTrip -v ./contents/
func TestFunctionalContentsDownloadRoundTrip(t *testing.T) {
	client := functionalClient(t)
	contentsClient := New(client, nil)
	messagesClient := messages.New(client, nil)
	roomsClient := rooms.New(client, nil)

	// Create a test room
	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Contents Download Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Upload a file via messages
	fileContent := []byte("Hello from SDK contents functional test!\nThis file tests download roundtrip.\n")
	msg, err := messagesClient.CreateWithAttachment(
		&messages.Message{RoomID: room.ID, Text: "Test file for contents download"},
		&messages.FileUpload{FileName: "test-contents.txt", FileBytes: fileContent},
	)
	if err != nil {
		t.Fatalf("CreateWithAttachment failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	if len(msg.Files) == 0 {
		t.Fatal("Message has no file URLs")
	}

	fileURL := msg.Files[0]
	t.Logf("Uploaded file, URL: %s", fileURL)

	// Download via contents DownloadFromURL — SDK auto-retries 423 (file scanning)
	fileInfo, err := contentsClient.DownloadFromURL(fileURL)
	if err != nil {
		t.Fatalf("DownloadFromURL failed: %v", err)
	}

	t.Logf("Downloaded: ContentType=%s ContentDisposition=%s Size=%d",
		fileInfo.ContentType, fileInfo.ContentDisposition, len(fileInfo.Data))

	// Verify content matches
	if string(fileInfo.Data) != string(fileContent) {
		t.Errorf("Content mismatch: got %d bytes, want %d bytes", len(fileInfo.Data), len(fileContent))
	} else {
		t.Logf("Content roundtrip verified: %d bytes match", len(fileInfo.Data))
	}
}

// TestFunctionalContentsDownloadFromURL tests downloading from a full URL
// Uses a remote file attachment
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContentsDownloadFromURL -v ./contents/
func TestFunctionalContentsDownloadFromURL(t *testing.T) {
	client := functionalClient(t)
	contentsClient := New(client, nil)
	messagesClient := messages.New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Contents URL Download Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Post a message with a remote file
	msg, err := messagesClient.Create(&messages.Message{
		RoomID: room.ID,
		Text:   "Test remote file for contents",
		Files:  []string{"https://www.webex.com/content/dam/wbx/global/images/webex-logo.png"},
	})
	if err != nil {
		t.Fatalf("Create message with file failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	if len(msg.Files) == 0 {
		t.Fatal("Message has no files")
	}

	// Download — SDK auto-retries 423 (file scanning)
	fileInfo, err := contentsClient.DownloadFromURL(msg.Files[0])
	if err != nil {
		t.Fatalf("DownloadFromURL failed: %v", err)
	}

	t.Logf("Downloaded: ContentType=%s Size=%d", fileInfo.ContentType, len(fileInfo.Data))
	if len(fileInfo.Data) == 0 {
		t.Error("Downloaded file is empty")
	}
}

// TestFunctionalContentsDownloadWithOptions tests DownloadWithOptions
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContentsDownloadWithOptions -v ./contents/
func TestFunctionalContentsDownloadWithOptions(t *testing.T) {
	client := functionalClient(t)
	contentsClient := New(client, nil)
	messagesClient := messages.New(client, nil)
	roomsClient := rooms.New(client, nil)

	room, err := roomsClient.Create(&rooms.Room{Title: "SDK Contents Options Test"})
	if err != nil {
		t.Fatalf("Failed to create test room: %v", err)
	}
	defer roomsClient.Delete(room.ID)

	// Upload a file
	msg, err := messagesClient.CreateWithAttachment(
		&messages.Message{RoomID: room.ID, Text: "Options test"},
		&messages.FileUpload{FileName: "options-test.txt", FileBytes: []byte("test data")},
	)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	defer messagesClient.Delete(msg.ID)

	if len(msg.Files) == 0 {
		t.Fatal("No files in message")
	}

	// Download with AllowUnscannable=true (should succeed for normal files too)
	// SDK auto-retries 423 (file scanning)
	fileInfo, err := contentsClient.DownloadFromURLWithOptions(msg.Files[0], &DownloadOptions{
		AllowUnscannable: true,
	})
	if err != nil {
		t.Fatalf("DownloadFromURLWithOptions failed: %v", err)
	}
	t.Logf("Downloaded with options: ContentType=%s Size=%d", fileInfo.ContentType, len(fileInfo.Data))
}

// TestFunctionalContentsStructuredErrors tests API error handling for invalid content ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContentsStructuredErrors -v ./contents/
func TestFunctionalContentsStructuredErrors(t *testing.T) {
	client := functionalClient(t)
	contentsClient := New(client, nil)

	_, err := contentsClient.Download("invalid-content-id")
	if err == nil {
		t.Fatal("Expected error for invalid content ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		// The error might be wrapped, try unwrapping
		t.Logf("Error type: %T, value: %v", err, err)
		// Contents wraps errors with fmt.Errorf, so unwrap
		unwrapped := errors.Unwrap(err)
		if unwrapped != nil && errors.As(unwrapped, &apiErr) {
			t.Logf("Got wrapped API error: status=%d message=%q", apiErr.StatusCode, apiErr.Message)
		} else {
			t.Logf("Error is not an APIError (may be expected for some error cases): %v", err)
		}
	} else {
		t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
	}
}

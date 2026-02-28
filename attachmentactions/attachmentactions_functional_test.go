//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package attachmentactions

import (
	"errors"
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

// TestFunctionalAttachmentActionsGet tests retrieving an attachment action by ID
// Note: Creating an attachment action programmatically requires a user to submit
// an adaptive card, which cannot be automated via API. This test verifies the
// Get endpoint returns a structured error for an invalid ID.
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalAttachmentActionsGet -v ./attachmentactions/
func TestFunctionalAttachmentActionsGet(t *testing.T) {
	client := functionalClient(t)
	aaClient := New(client, nil)

	_, err := aaClient.Get("invalid-action-id")
	if err == nil {
		t.Fatal("Expected error for invalid action ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) {
		t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
	} else {
		t.Logf("Error: %v", err)
	}
}

// TestFunctionalAttachmentActionsCreateValidation tests client-side validation
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalAttachmentActionsCreateValidation -v ./attachmentactions/
func TestFunctionalAttachmentActionsCreateValidation(t *testing.T) {
	client := functionalClient(t)
	aaClient := New(client, nil)

	// Missing messageId should fail with client-side validation
	_, err := aaClient.Create(&AttachmentAction{
		Type:   "submit",
		Inputs: map[string]interface{}{"key": "value"},
	})
	if err == nil {
		t.Fatal("Expected error for missing messageId, got nil")
	}
	t.Logf("Correct validation error for missing messageId: %v", err)

	// Missing type should fail
	_, err = aaClient.Create(&AttachmentAction{
		MessageID: "some-message-id",
		Inputs:    map[string]interface{}{"key": "value"},
	})
	if err == nil {
		t.Fatal("Expected error for missing type, got nil")
	}
	t.Logf("Correct validation error for missing type: %v", err)
}

// TestFunctionalAttachmentActionsCreateWithInvalidMessage tests server-side error
// for a create action with a non-existent message ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalAttachmentActionsCreateWithInvalidMessage -v ./attachmentactions/
func TestFunctionalAttachmentActionsCreateWithInvalidMessage(t *testing.T) {
	client := functionalClient(t)
	aaClient := New(client, nil)

	_, err := aaClient.Create(&AttachmentAction{
		Type:      "submit",
		MessageID: "invalid-message-id-does-not-exist",
		Inputs:    map[string]interface{}{"key": "value"},
	})
	if err == nil {
		t.Fatal("Expected error for invalid message ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) {
		t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
			apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
	} else {
		t.Logf("Error: %v", err)
	}
}

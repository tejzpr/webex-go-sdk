//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package events

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// skipOn403 skips the test if the error is an API 403 (missing scopes/compliance role).
func skipOn403(t *testing.T, err error) {
	t.Helper()
	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		t.Skipf("Skipping: token lacks required scopes: %v", err)
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

// TestFunctionalEventsList tests listing events from the last 7 days
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalEventsList -v ./events/
func TestFunctionalEventsList(t *testing.T) {
	client := functionalClient(t)
	eventsClient := New(client, nil)

	page, err := eventsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  20,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d events from the last 7 days", len(page.Items))
	for i, e := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Resource=%s Type=%s ActorID=%s Created=%s\n",
			i+1, e.ID, e.Resource, e.Type, e.ActorID, e.Created.Format(time.RFC3339))
		if e.Data.RoomID != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    RoomID: %s\n", e.Data.RoomID)
		}
		if e.Data.PersonEmail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    PersonEmail: %s\n", e.Data.PersonEmail)
		}
	}
}

// TestFunctionalEventsListByResource tests listing events filtered by resource
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalEventsListByResource -v ./events/
func TestFunctionalEventsListByResource(t *testing.T) {
	client := functionalClient(t)
	eventsClient := New(client, nil)

	resources := []string{"messages", "memberships", "rooms"}
	for _, resource := range resources {
		t.Run(resource, func(t *testing.T) {
			page, err := eventsClient.List(&ListOptions{
				Resource: resource,
				From:     time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
				To:       time.Now().Format(time.RFC3339),
				Max:      10,
			})
			if err != nil {
				skipOn403(t, err)
				t.Fatalf("List resource=%s failed: %v", resource, err)
			}
			t.Logf("Resource %s: found %d events", resource, len(page.Items))
			for i, e := range page.Items {
				_, _ = fmt.Fprintf(os.Stdout, "  [%d] Type=%s ActorID=%s\n",
					i+1, e.Type, e.ActorID)
			}
		})
	}
}

// TestFunctionalEventsGet tests getting a specific event
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalEventsGet -v ./events/
func TestFunctionalEventsGet(t *testing.T) {
	client := functionalClient(t)
	eventsClient := New(client, nil)

	// First list events to find one
	page, err := eventsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  5,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("List failed: %v", err)
	}
	if len(page.Items) == 0 {
		t.Skip("No events found in the last 7 days")
	}

	event := page.Items[0]
	got, err := eventsClient.Get(event.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != event.ID {
		t.Errorf("Get ID mismatch: got %s, want %s", got.ID, event.ID)
	}
	t.Logf("Get confirmed: ID=%s Resource=%s Type=%s", got.ID, got.Resource, got.Type)
}

// TestFunctionalEventsNotFound tests structured error on invalid event ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalEventsNotFound -v ./events/
func TestFunctionalEventsNotFound(t *testing.T) {
	client := functionalClient(t)
	eventsClient := New(client, nil)

	_, err := eventsClient.Get("invalid-event-id")
	if err == nil {
		t.Fatal("Expected error for invalid event ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

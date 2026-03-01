//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package people

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

// TestFunctionalPeopleGetMe tests getting the current authenticated user
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalPeopleGetMe -v ./people/
func TestFunctionalPeopleGetMe(t *testing.T) {
	client := functionalClient(t)
	peopleClient := New(client, nil)

	me, err := peopleClient.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	if me.ID == "" {
		t.Fatal("GetMe returned empty ID")
	}
	t.Logf("Current user: ID=%s DisplayName=%q", me.ID, me.DisplayName)
	if len(me.Emails) > 0 {
		t.Logf("Email: %s", me.Emails[0])
	}
	if me.OrgID != "" {
		t.Logf("OrgID: %s", me.OrgID)
	}
	if me.Status != "" {
		t.Logf("Status: %s", me.Status)
	}
	if me.Avatar != "" {
		t.Logf("Avatar: %s", me.Avatar)
	}
}

// TestFunctionalPeopleGet tests getting a specific person by ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalPeopleGet -v ./people/
func TestFunctionalPeopleGet(t *testing.T) {
	client := functionalClient(t)
	peopleClient := New(client, nil)

	// First get own ID via GetMe
	me, err := peopleClient.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	// Now get by ID
	person, err := peopleClient.Get(me.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if person.ID != me.ID {
		t.Errorf("Get returned wrong ID: got %s, want %s", person.ID, me.ID)
	}
	if person.DisplayName != me.DisplayName {
		t.Errorf("DisplayName mismatch: got %q, want %q", person.DisplayName, me.DisplayName)
	}
	t.Logf("Get confirmed: ID=%s DisplayName=%q", person.ID, person.DisplayName)
}

// TestFunctionalPeopleListByEmail tests listing people by email
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalPeopleListByEmail -v ./people/
func TestFunctionalPeopleListByEmail(t *testing.T) {
	client := functionalClient(t)
	peopleClient := New(client, nil)

	// Get own email
	me, err := peopleClient.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}
	if len(me.Emails) == 0 {
		t.Fatal("Current user has no emails")
	}

	// List by email
	page, err := peopleClient.List(&ListOptions{
		Email: me.Emails[0],
		Max:   5,
	})
	if err != nil {
		t.Fatalf("List by email failed: %v", err)
	}

	t.Logf("Found %d people matching email %s", len(page.Items), me.Emails[0])
	for i, p := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s DisplayName=%q Status=%s\n",
			i+1, p.ID, p.DisplayName, p.Status)
	}
}

// TestFunctionalPeopleBatcher tests the batch request mechanism
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalPeopleBatcher -v ./people/
func TestFunctionalPeopleBatcher(t *testing.T) {
	client := functionalClient(t)
	peopleClient := New(client, nil)

	// Get own ID
	me, err := peopleClient.GetMe()
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	// Use the Batcher to request our own person record
	person, err := peopleClient.Get(me.ID)
	if err != nil {
		t.Fatalf("Batcher Request failed: %v", err)
	}
	if person.DisplayName != me.DisplayName {
		t.Errorf("Batcher result mismatch: got %q, want %q", person.DisplayName, me.DisplayName)
	}
	t.Logf("Batcher result: ID=%s DisplayName=%q", person.ID, person.DisplayName)

	// Test BatchRequest with a list of IDs
	persons, err := peopleClient.List(&ListOptions{IDs: []string{me.ID}})
	if err != nil {
		// Bots may get 400 when using batch people request; skip if so
		var apiErr *webexsdk.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 400 || apiErr.StatusCode == 403) {
			t.Skipf("Skipping batch request: %v", err)
		}
		t.Fatalf("BatchRequest failed: %v", err)
	}
	t.Logf("BatchRequest returned %d persons", len(persons.Items))
	for i, p := range persons.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s DisplayName=%q\n", i+1, p.ID, p.DisplayName)
	}
}

// TestFunctionalPeopleNotFound tests structured error on invalid person ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalPeopleNotFound -v ./people/
func TestFunctionalPeopleNotFound(t *testing.T) {
	client := functionalClient(t)
	peopleClient := New(client, nil)

	_, err := peopleClient.Get("invalid-person-id")
	if err == nil {
		t.Fatal("Expected error for invalid person ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

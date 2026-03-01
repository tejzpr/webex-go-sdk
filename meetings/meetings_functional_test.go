//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package meetings

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// skipOn403 skips the test if the error is an API 403 (missing scopes/licenses).
func skipOn403(t *testing.T, err error) {
	t.Helper()
	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		t.Skipf("Skipping: token lacks required meeting scopes: %v", err)
	}
}

// TestFunctionalListMeetings lists all meetings between Feb 25 2026 and Feb 26 2026
// using the real Webex API.
//
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListMeetings -v ./meetings/
func TestFunctionalListMeetings(t *testing.T) {
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

	meetingsClient := New(client, nil)

	options := &ListOptions{
		MeetingType: "scheduledMeeting",
		From:        "2026-02-25T00:00:00-08:00",
		To:          "2026-02-26T23:59:59-08:00",
		Max:         100,
	}

	page, err := meetingsClient.List(options)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list meetings: %v", err)
	}

	t.Logf("Found %d scheduled meetings between Feb 25-26, 2026", len(page.Items))
	for i, m := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s Host=%s WebLink=%s\n",
			i+1, m.ID, m.Title, m.Start, m.End, m.MeetingType, m.State, m.HostEmail, m.WebLink)

		// Log additional fields if available
		if m.EnabledAutoRecordMeeting {
			_, _ = fmt.Fprintf(os.Stdout, "    Auto Recording: Enabled\n")
		}
		if m.EnabledJoinBeforeHost {
			_, _ = fmt.Fprintf(os.Stdout, "    Join Before Host: Enabled (%d minutes)\n", m.JoinBeforeHostMinutes)
		}
		if len(m.IntegrationTags) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", m.IntegrationTags)
		}
		if m.Telephony != nil && m.Telephony.AccessCode != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Telephony Access Code: %s\n", m.Telephony.AccessCode)
		}
	}

	// Also check ended meeting instances in the same window
	endedOptions := &ListOptions{
		MeetingType: "meeting",
		State:       "ended",
		From:        "2026-02-25T00:00:00-08:00",
		To:          "2026-02-26T23:59:59-08:00",
		Max:         100,
	}

	endedPage, err := meetingsClient.List(endedOptions)
	if err != nil {
		t.Fatalf("Failed to list ended meetings: %v", err)
	}

	t.Logf("Found %d ended meeting instances between Feb 25-26, 2026", len(endedPage.Items))
	for i, m := range endedPage.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s Host=%s HasRecording=%v HasTranscription=%v\n",
			i+1, m.ID, m.Title, m.Start, m.End, m.MeetingType, m.State, m.HostEmail, m.HasRecording, m.HasTranscription)

		// Log additional fields for ended meetings
		if m.MeetingNumber != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Meeting Number: %s\n", m.MeetingNumber)
		}
		if m.SiteURL != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Site URL: %s\n", m.SiteURL)
		}
		if m.HasSummary {
			_, _ = fmt.Fprintf(os.Stdout, "    Summary: Available\n")
		}
		if m.HasClosedCaption {
			_, _ = fmt.Fprintf(os.Stdout, "    Closed Caption: Available\n")
		}
		if len(m.TrackingCodes) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Tracking Codes: %d entries\n", len(m.TrackingCodes))
			for _, tc := range m.TrackingCodes {
				_, _ = fmt.Fprintf(os.Stdout, "      - %s: %s\n", tc.Name, tc.Value)
			}
		}
	}
}

// TestFunctionalCreateGetUpdateDeleteMeeting tests the full CRUD lifecycle
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCreateGetUpdateDeleteMeeting -v ./meetings/
func TestFunctionalCreateGetUpdateDeleteMeeting(t *testing.T) {
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

	meetingsClient := New(client, nil)

	// Create a test meeting
	startTime := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endTime := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)

	meeting := &Meeting{
		Title:                    "Functional Test Meeting",
		Agenda:                   "Testing the Webex Go SDK",
		Start:                    startTime,
		End:                      endTime,
		EnabledAutoRecordMeeting: true,
		EnabledJoinBeforeHost:    true,
		JoinBeforeHostMinutes:    5,
		PublicMeeting:            false,
		IntegrationTags:          []string{"test-sdk", "functional-test"},
	}

	t.Logf("Creating meeting: %s from %s to %s", meeting.Title, meeting.Start, meeting.End)
	created, err := meetingsClient.Create(meeting)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to create meeting: %v", err)
	}

	t.Logf("Created meeting: ID=%s WebLink=%s", created.ID, created.WebLink)
	t.Logf("Meeting details: Host=%s, Number=%s, Site=%s",
		created.HostEmail, created.MeetingNumber, created.SiteURL)
	t.Logf("Features: AutoRecord=%v, JoinBeforeHost=%v, Public=%v",
		created.EnabledAutoRecordMeeting, created.EnabledJoinBeforeHost, created.PublicMeeting)
	defer func() {
		// Clean up
		if err := meetingsClient.Delete(created.ID); err != nil {
			t.Logf("Warning: Failed to delete test meeting %s: %v", created.ID, err)
		}
	}()

	// Get the meeting
	retrieved, err := meetingsClient.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get meeting: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.Title != created.Title {
		t.Errorf("Expected title %s, got %s", created.Title, retrieved.Title)
	}

	t.Logf("Retrieved meeting: %s", retrieved.Title)

	// Update the meeting
	updatedMeeting := &Meeting{
		Title:                    "Updated Functional Test Meeting",
		Agenda:                   "Updated agenda for testing",
		Start:                    startTime,
		End:                      endTime,
		EnabledAutoRecordMeeting: false,
		AllowAnyUserToBeCoHost:   true,
		IntegrationTags:          []string{"test-sdk", "updated"},
	}

	updated, err := meetingsClient.Update(created.ID, updatedMeeting)
	if err != nil {
		t.Fatalf("Failed to update meeting: %v", err)
	}

	if updated.Title != updatedMeeting.Title {
		t.Errorf("Expected updated title %s, got %s", updatedMeeting.Title, updated.Title)
	}

	t.Logf("Updated meeting title to: %s", updated.Title)
	t.Logf("Updated features: AutoRecord=%v, CoHost=%v",
		updated.EnabledAutoRecordMeeting, updated.AllowAnyUserToBeCoHost)

	// Patch the meeting
	patchData := map[string]interface{}{
		"agenda": "Patched agenda via PATCH",
	}

	patched, err := meetingsClient.Patch(created.ID, patchData)
	if err != nil {
		t.Fatalf("Failed to patch meeting: %v", err)
	}

	if patched.Agenda != "Patched agenda via PATCH" {
		t.Errorf("Expected patched agenda, got %s", patched.Agenda)
	}

	t.Logf("Patched meeting agenda to: %s", patched.Agenda)
}

// TestFunctionalListParticipants tests listing participants for a meeting
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListParticipants -v ./meetings/
func TestFunctionalListParticipants(t *testing.T) {
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

	meetingsClient := New(client, nil)

	// First, get a list of ended meetings to find one with participants
	options := &ListOptions{
		MeetingType: "meeting",
		State:       "ended",
		From:        "2026-02-24T00:00:00-08:00",
		To:          "2026-02-26T23:59:59-08:00",
		Max:         10,
	}

	page, err := meetingsClient.List(options)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list meetings: %v", err)
	}

	if len(page.Items) == 0 {
		t.Skip("No ended meetings found in the specified date range")
	}

	// Try each meeting until we find one with participants
	for _, meeting := range page.Items {
		t.Logf("Checking participants for meeting: %s (%s)", meeting.Title, meeting.ID)

		participants, err := meetingsClient.ListParticipants(&ParticipantListOptions{
			MeetingID: meeting.ID,
			Max:       50,
		})

		if err != nil {
			t.Logf("Failed to list participants for meeting %s: %v", meeting.ID, err)
			continue
		}

		t.Logf("Found %d participants for meeting %s", len(participants.Items), meeting.ID)

		for i, p := range participants.Items {
			_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Email=%s DisplayName=%q Host=%v CoHost=%v State=%s\n",
				i+1, p.ID, p.Email, p.DisplayName, p.Host, p.CoHost, p.State)

			// Show additional participant details
			if p.JoinedTime != "" {
				_, _ = fmt.Fprintf(os.Stdout, "    Joined: %s\n", p.JoinedTime)
			}
			if p.LeftTime != "" {
				_, _ = fmt.Fprintf(os.Stdout, "    Left: %s\n", p.LeftTime)
			}
			if len(p.Devices) > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "    Devices: %d\n", len(p.Devices))
				for _, d := range p.Devices {
					_, _ = fmt.Fprintf(os.Stdout, "      - %s (Audio: %s)\n", d.DeviceType, d.AudioType)
				}
			}
		}

		if len(participants.Items) > 0 {
			// Test getting a specific participant
			participant := participants.Items[0]
			retrieved, err := meetingsClient.GetParticipant(participant.ID, meeting.ID)
			if err != nil {
				t.Logf("Failed to get participant %s: %v", participant.ID, err)
			} else {
				t.Logf("Retrieved participant: %s (%s)", retrieved.DisplayName, retrieved.Email)
			}
			break // Found a meeting with participants
		}
	}
}

// TestFunctionalListAllMeetingTypes tests listing different types of meetings
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListAllMeetingTypes -v ./meetings/
func TestFunctionalListAllMeetingTypes(t *testing.T) {
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

	meetingsClient := New(client, nil)

	// Test different meeting types
	testCases := []struct {
		name        string
		options     *ListOptions
		description string
	}{
		{
			name: "Meeting Series",
			options: &ListOptions{
				MeetingType: "meetingSeries",
				Max:         10,
			},
			description: "Recurring meeting definitions",
		},
		{
			name: "Scheduled Meetings",
			options: &ListOptions{
				MeetingType: "scheduledMeeting",
				Max:         10,
			},
			description: "Upcoming scheduled occurrences",
		},
		{
			name: "Active Meetings",
			options: &ListOptions{
				MeetingType: "meetingSeries",
				Max:         10,
			},
			description: "Meeting series (active state)",
		},
		{
			name: "Ended Meetings (Last 7 Days)",
			options: &ListOptions{
				MeetingType: "meeting",
				State:       "ended",
				From:        time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
				To:          time.Now().Format(time.RFC3339),
				Max:         20,
			},
			description: "Past meeting instances",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			page, err := meetingsClient.List(tc.options)
			if err != nil {
				skipOn403(t, err)
				t.Fatalf("Failed to list %s: %v", tc.name, err)
			}

			t.Logf("%s: Found %d meetings (%s)", tc.name, len(page.Items), tc.description)

			for i, m := range page.Items {
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s\n",
					i+1, m.ID, m.Title, m.Start, m.End, m.MeetingType, m.State)
			}
		})
	}
}

// TestFunctionalListTomorrowMeetings lists meetings scheduled for tomorrow
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListTomorrowMeetings -v ./meetings/
func TestFunctionalListTomorrowMeetings(t *testing.T) {
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

	meetingsClient := New(client, nil)

	// Calculate tomorrow's date range in UTC
	tomorrow := time.Now().AddDate(0, 0, 1)
	tomorrowStart := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowEnd := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, time.UTC)

	t.Logf("Looking for meetings scheduled for %s", tomorrow.Format("2006-01-02"))

	// List scheduled meetings for tomorrow
	options := &ListOptions{
		MeetingType: "scheduledMeeting",
		From:        tomorrowStart.Format(time.RFC3339),
		To:          tomorrowEnd.Format(time.RFC3339),
		Max:         50,
	}

	page, err := meetingsClient.List(options)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list meetings: %v", err)
	}

	t.Logf("Found %d meetings scheduled for tomorrow", len(page.Items))
	for i, meeting := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q\n",
			i+1, meeting.ID, meeting.Title)

		// Log additional meeting details
		if meeting.Start != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Start: %s\n", meeting.Start)
		}
		if meeting.End != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    End: %s\n", meeting.End)
		}
		if meeting.State != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    State: %s\n", meeting.State)
		}
		if meeting.MeetingType != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Type: %s\n", meeting.MeetingType)
		}
		if meeting.HostEmail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Host: %s\n", meeting.HostEmail)
		}
		if meeting.EnabledAutoRecordMeeting {
			_, _ = fmt.Fprintf(os.Stdout, "    Auto Record: Enabled\n")
		}
		if meeting.JoinBeforeHostMinutes > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Join Before Host: %d minutes\n", meeting.JoinBeforeHostMinutes)
		}
		if len(meeting.IntegrationTags) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", meeting.IntegrationTags)
		}
		if meeting.MeetingNumber != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Meeting Number: %s\n", meeting.MeetingNumber)
		}
		if meeting.SiteURL != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Site URL: %s\n", meeting.SiteURL)
		}
		if len(meeting.Invitees) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Invitees (%d):\n", len(meeting.Invitees))
			for j, invitee := range meeting.Invitees {
				if j < 5 { // Limit to first 5 invitees
					_, _ = fmt.Fprintf(os.Stdout, "      - %s (%s)\n", invitee.DisplayName, invitee.Email)
				}
			}
			if len(meeting.Invitees) > 5 {
				_, _ = fmt.Fprintf(os.Stdout, "      ... and %d more invitees\n", len(meeting.Invitees)-5)
			}
		}
		_, _ = fmt.Fprintf(os.Stdout, "\n")
	}
}

// TestFunctionalMeetingsNotFound tests structured error on invalid meeting ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMeetingsNotFound -v ./meetings/
func TestFunctionalMeetingsNotFound(t *testing.T) {
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

	meetingsClient := New(client, nil)

	_, err = meetingsClient.Get("invalid-meeting-id-does-not-exist")
	if err == nil {
		t.Fatal("Expected error for invalid meeting ID, got nil")
	}

	// Verify structured error
	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)

	// Check IsNotFound convenience function
	if webexsdk.IsNotFound(err) {
		t.Logf("IsNotFound correctly returned true")
	} else {
		t.Logf("IsNotFound returned false (status was %d)", apiErr.StatusCode)
	}
}

// TestFunctionalMeetingsListPagination tests pagination through meetings
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalMeetingsListPagination -v ./meetings/
func TestFunctionalMeetingsListPagination(t *testing.T) {
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

	meetingsClient := New(client, nil)

	// Request small pages to test pagination with ended meetings from last 30 days
	options := &ListOptions{
		MeetingType: "meeting",
		State:       "ended",
		From:        time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:          time.Now().Format(time.RFC3339),
		Max:         2,
	}

	page, err := meetingsClient.List(options)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list meetings: %v", err)
	}

	totalItems := len(page.Items)
	pageCount := 1
	t.Logf("Page %d: %d items", pageCount, len(page.Items))

	// Traverse pages (limit to 5 pages to avoid runaway)
	for page.HasNext && pageCount < 5 {
		nextPage, err := page.Page.Next()
		if err != nil {
			t.Fatalf("Failed to get next page: %v", err)
		}

		page.Page = nextPage
		pageCount++
		totalItems += len(nextPage.Items)
		t.Logf("Page %d: %d items", pageCount, len(nextPage.Items))
	}

	t.Logf("Total: %d items across %d pages", totalItems, pageCount)
}

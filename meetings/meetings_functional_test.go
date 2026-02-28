//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package meetings

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

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
		t.Fatalf("Failed to list meetings: %v", err)
	}

	t.Logf("Found %d scheduled meetings between Feb 25-26, 2026", len(page.Items))
	for i, m := range page.Items {
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s Host=%s WebLink=%s\n",
			i+1, m.ID, m.Title, m.Start, m.End, m.MeetingType, m.State, m.HostEmail, m.WebLink)

		// Log additional fields if available
		if m.EnabledAutoRecordMeeting {
			fmt.Fprintf(os.Stdout, "    Auto Recording: Enabled\n")
		}
		if m.EnabledJoinBeforeHost {
			fmt.Fprintf(os.Stdout, "    Join Before Host: Enabled (%d minutes)\n", m.JoinBeforeHostMinutes)
		}
		if len(m.IntegrationTags) > 0 {
			fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", m.IntegrationTags)
		}
		if m.Telephony != nil && m.Telephony.AccessCode != "" {
			fmt.Fprintf(os.Stdout, "    Telephony Access Code: %s\n", m.Telephony.AccessCode)
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
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s Host=%s HasRecording=%v HasTranscription=%v\n",
			i+1, m.ID, m.Title, m.Start, m.End, m.MeetingType, m.State, m.HostEmail, m.HasRecording, m.HasTranscription)

		// Log additional fields for ended meetings
		if m.MeetingNumber != "" {
			fmt.Fprintf(os.Stdout, "    Meeting Number: %s\n", m.MeetingNumber)
		}
		if m.SiteURL != "" {
			fmt.Fprintf(os.Stdout, "    Site URL: %s\n", m.SiteURL)
		}
		if m.HasSummary {
			fmt.Fprintf(os.Stdout, "    Summary: Available\n")
		}
		if m.HasClosedCaption {
			fmt.Fprintf(os.Stdout, "    Closed Caption: Available\n")
		}
		if len(m.TrackingCodes) > 0 {
			fmt.Fprintf(os.Stdout, "    Tracking Codes: %d entries\n", len(m.TrackingCodes))
			for _, tc := range m.TrackingCodes {
				fmt.Fprintf(os.Stdout, "      - %s: %s\n", tc.Name, tc.Value)
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
			fmt.Fprintf(os.Stdout, "[%d] ID=%s Email=%s DisplayName=%q Host=%v CoHost=%v State=%s\n",
				i+1, p.ID, p.Email, p.DisplayName, p.Host, p.CoHost, p.State)

			// Show additional participant details
			if p.JoinedTime != "" {
				fmt.Fprintf(os.Stdout, "    Joined: %s\n", p.JoinedTime)
			}
			if p.LeftTime != "" {
				fmt.Fprintf(os.Stdout, "    Left: %s\n", p.LeftTime)
			}
			if len(p.Devices) > 0 {
				fmt.Fprintf(os.Stdout, "    Devices: %d\n", len(p.Devices))
				for _, d := range p.Devices {
					fmt.Fprintf(os.Stdout, "      - %s (Audio: %s)\n", d.DeviceType, d.AudioType)
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
				t.Fatalf("Failed to list %s: %v", tc.name, err)
			}

			t.Logf("%s: Found %d meetings (%s)", tc.name, len(page.Items), tc.description)

			for i, m := range page.Items {
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q Start=%s End=%s Type=%s State=%s\n",
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
		t.Fatalf("Failed to list meetings: %v", err)
	}

	t.Logf("Found %d meetings scheduled for tomorrow", len(page.Items))
	for i, meeting := range page.Items {
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Title=%q\n",
			i+1, meeting.ID, meeting.Title)

		// Log additional meeting details
		if meeting.Start != "" {
			fmt.Fprintf(os.Stdout, "    Start: %s\n", meeting.Start)
		}
		if meeting.End != "" {
			fmt.Fprintf(os.Stdout, "    End: %s\n", meeting.End)
		}
		if meeting.State != "" {
			fmt.Fprintf(os.Stdout, "    State: %s\n", meeting.State)
		}
		if meeting.MeetingType != "" {
			fmt.Fprintf(os.Stdout, "    Type: %s\n", meeting.MeetingType)
		}
		if meeting.HostEmail != "" {
			fmt.Fprintf(os.Stdout, "    Host: %s\n", meeting.HostEmail)
		}
		if meeting.EnabledAutoRecordMeeting {
			fmt.Fprintf(os.Stdout, "    Auto Record: Enabled\n")
		}
		if meeting.JoinBeforeHostMinutes > 0 {
			fmt.Fprintf(os.Stdout, "    Join Before Host: %d minutes\n", meeting.JoinBeforeHostMinutes)
		}
		if len(meeting.IntegrationTags) > 0 {
			fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", meeting.IntegrationTags)
		}
		if meeting.MeetingNumber != "" {
			fmt.Fprintf(os.Stdout, "    Meeting Number: %s\n", meeting.MeetingNumber)
		}
		if meeting.SiteURL != "" {
			fmt.Fprintf(os.Stdout, "    Site URL: %s\n", meeting.SiteURL)
		}
		if len(meeting.Invitees) > 0 {
			fmt.Fprintf(os.Stdout, "    Invitees (%d):\n", len(meeting.Invitees))
			for j, invitee := range meeting.Invitees {
				if j < 5 { // Limit to first 5 invitees
					fmt.Fprintf(os.Stdout, "      - %s (%s)\n", invitee.DisplayName, invitee.Email)
				}
			}
			if len(meeting.Invitees) > 5 {
				fmt.Fprintf(os.Stdout, "      ... and %d more invitees\n", len(meeting.Invitees)-5)
			}
		}
		fmt.Fprintf(os.Stdout, "\n")
	}
}

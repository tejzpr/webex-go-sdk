//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package recordings

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// TestFunctionalListRecordings lists recordings from the last 7 days
// using the real Webex API.
//
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListRecordings -v ./recordings/
func TestFunctionalListRecordings(t *testing.T) {
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

	recordingsClient := New(client, nil)

	// List recordings from the last 7 days
	options := &ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  50,
	}

	page, err := recordingsClient.List(options)
	if err != nil {
		t.Fatalf("Failed to list recordings: %v", err)
	}

	t.Logf("Found %d recordings from the last 7 days", len(page.Items))
	for i, r := range page.Items {
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s Duration=%ds Size=%dMB Status=%s\n",
			i+1, r.ID, r.Topic, r.Format, r.DurationSeconds, r.SizeBytes/(1024*1024), r.Status)

		// Log additional recording details
		if r.HostEmail != "" {
			fmt.Fprintf(os.Stdout, "    Host: %s\n", r.HostEmail)
		}
		if r.SiteURL != "" {
			fmt.Fprintf(os.Stdout, "    Site: %s\n", r.SiteURL)
		}
		if r.ServiceType != "" {
			fmt.Fprintf(os.Stdout, "    Service: %s\n", r.ServiceType)
		}
		if r.PlaybackURL != "" {
			fmt.Fprintf(os.Stdout, "    Playback: %s\n", r.PlaybackURL)
		}
		if r.ShareToMe {
			fmt.Fprintf(os.Stdout, "    Shared to me: Yes\n")
		}
		if len(r.IntegrationTags) > 0 {
			fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", r.IntegrationTags)
		}
		if r.TemporaryDirectDownloadLinks != nil {
			fmt.Fprintf(os.Stdout, "    Direct Downloads: Available\n")
			if r.TemporaryDirectDownloadLinks.RecordingDownloadLink != "" {
				fmt.Fprintf(os.Stdout, "      - Recording: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.AudioDownloadLink != "" {
				fmt.Fprintf(os.Stdout, "      - Audio: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.TranscriptDownloadLink != "" {
				fmt.Fprintf(os.Stdout, "      - Transcript: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.Expiration != "" {
				fmt.Fprintf(os.Stdout, "      - Expires: %s\n", r.TemporaryDirectDownloadLinks.Expiration)
			}
		}
	}
}

// TestFunctionalListRecordingsByMeeting lists recordings for a specific meeting
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListRecordingsByMeeting -v ./recordings/
func TestFunctionalListRecordingsByMeeting(t *testing.T) {
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

	recordingsClient := New(client, nil)

	// First, get a list of meetings to find one with recordings
	resp, err := client.Request("GET", "meetings", nil, nil)
	if err != nil {
		t.Fatalf("Failed to list meetings: %v", err)
	}
	defer resp.Body.Close()

	page, err := webexsdk.NewPage(resp, client, "meetings")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	if len(page.Items) == 0 {
		t.Skip("No meetings found in the last 7 days")
	}

	// Try to find recordings for each meeting
	for _, item := range page.Items {
		var meeting struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(item, &meeting); err != nil {
			continue
		}

		t.Logf("Checking recordings for meeting: %s (%s)", meeting.Title, meeting.ID)

		options := &ListOptions{
			MeetingID: meeting.ID,
			Max:       10,
		}

		recordingsPage, err := recordingsClient.List(options)
		if err != nil {
			t.Logf("Failed to list recordings for meeting %s: %v", meeting.ID, err)
			continue
		}

		if len(recordingsPage.Items) > 0 {
			t.Logf("Found %d recordings for meeting %s", len(recordingsPage.Items), meeting.ID)
			for i, r := range recordingsPage.Items {
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s Duration=%ds\n",
					i+1, r.ID, r.Topic, r.Format, r.DurationSeconds)

				if r.TimeRecorded != "" {
					fmt.Fprintf(os.Stdout, "    Recorded: %s\n", r.TimeRecorded)
				}
				if r.CreateTime != "" {
					fmt.Fprintf(os.Stdout, "    Created: %s\n", r.CreateTime)
				}
				if r.DownloadURL != "" {
					fmt.Fprintf(os.Stdout, "    Download: %s\n", r.DownloadURL)
				}
			}
			break // Found a meeting with recordings
		}
	}
}

// TestFunctionalListRecordingsByFormat lists recordings by different formats
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListRecordingsByFormat -v ./recordings/
func TestFunctionalListRecordingsByFormat(t *testing.T) {
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

	recordingsClient := New(client, nil)

	// Test different formats
	formats := []string{"MP4", "ARF"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			options := &ListOptions{
				Format: format,
				From:   time.Now().AddDate(0, 0, -30).Format(time.RFC3339), // Last 30 days
				To:     time.Now().Format(time.RFC3339),
				Max:    20,
			}

			page, err := recordingsClient.List(options)
			if err != nil {
				t.Fatalf("Failed to list %s recordings: %v", format, err)
			}

			t.Logf("%s format: Found %d recordings", format, len(page.Items))

			for i, r := range page.Items {
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds Size=%dMB\n",
					i+1, r.ID, r.Topic, r.DurationSeconds, r.SizeBytes/(1024*1024))

				if r.ServiceType != "" {
					fmt.Fprintf(os.Stdout, "    Service: %s\n", r.ServiceType)
				}
				if r.Status != "" {
					fmt.Fprintf(os.Stdout, "    Status: %s\n", r.Status)
				}
			}
		})
	}
}

// TestFunctionalListRecordingsByStatus lists recordings by different statuses
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListRecordingsByStatus -v ./recordings/
func TestFunctionalListRecordingsByStatus(t *testing.T) {
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

	recordingsClient := New(client, nil)

	// Test different statuses
	statuses := []string{"available", "deleted"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			options := &ListOptions{
				Status: status,
				Max:    20,
			}

			page, err := recordingsClient.List(options)
			if err != nil {
				t.Fatalf("Failed to list %s recordings: %v", status, err)
			}

			t.Logf("%s status: Found %d recordings", status, len(page.Items))

			for i, r := range page.Items {
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s\n",
					i+1, r.ID, r.Topic, r.Format)

				if r.MeetingID != "" {
					fmt.Fprintf(os.Stdout, "    Meeting ID: %s\n", r.MeetingID)
				}
				if r.HostEmail != "" {
					fmt.Fprintf(os.Stdout, "    Host: %s\n", r.HostEmail)
				}
				if r.TimeRecorded != "" {
					fmt.Fprintf(os.Stdout, "    Recorded: %s\n", r.TimeRecorded)
				}
			}
		})
	}
}

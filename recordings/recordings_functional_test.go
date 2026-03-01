//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package recordings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// skipOn403 skips the test if the error is an API 403 (missing scopes).
func skipOn403(t *testing.T, err error) {
	t.Helper()
	var apiErr *webexsdk.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		t.Skipf("Skipping: token lacks required scopes: %v", err)
	}
}

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
		skipOn403(t, err)
		t.Fatalf("Failed to list recordings: %v", err)
	}

	t.Logf("Found %d recordings from the last 7 days", len(page.Items))
	for i, r := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s Duration=%ds Size=%dMB Status=%s\n",
			i+1, r.ID, r.Topic, r.Format, r.DurationSeconds, r.SizeBytes/(1024*1024), r.Status)

		// Log additional recording details
		if r.HostEmail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Host: %s\n", r.HostEmail)
		}
		if r.SiteURL != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Site: %s\n", r.SiteURL)
		}
		if r.ServiceType != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Service: %s\n", r.ServiceType)
		}
		if r.PlaybackURL != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Playback: %s\n", r.PlaybackURL)
		}
		if r.ShareToMe {
			_, _ = fmt.Fprintf(os.Stdout, "    Shared to me: Yes\n")
		}
		if len(r.IntegrationTags) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Integration Tags: %v\n", r.IntegrationTags)
		}
		if r.TemporaryDirectDownloadLinks != nil {
			_, _ = fmt.Fprintf(os.Stdout, "    Direct Downloads: Available\n")
			if r.TemporaryDirectDownloadLinks.RecordingDownloadLink != "" {
				_, _ = fmt.Fprintf(os.Stdout, "      - Recording: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.AudioDownloadLink != "" {
				_, _ = fmt.Fprintf(os.Stdout, "      - Audio: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.TranscriptDownloadLink != "" {
				_, _ = fmt.Fprintf(os.Stdout, "      - Transcript: Available\n")
			}
			if r.TemporaryDirectDownloadLinks.Expiration != "" {
				_, _ = fmt.Fprintf(os.Stdout, "      - Expires: %s\n", r.TemporaryDirectDownloadLinks.Expiration)
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
		skipOn403(t, err)
		t.Fatalf("Failed to list meetings: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	page, err := webexsdk.NewPage(resp, client, webexsdk.ResourceMeetings)
	if err != nil {
		skipOn403(t, err)
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
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s Duration=%ds\n",
					i+1, r.ID, r.Topic, r.Format, r.DurationSeconds)

				if r.TimeRecorded != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Recorded: %s\n", r.TimeRecorded)
				}
				if r.CreateTime != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Created: %s\n", r.CreateTime)
				}
				if r.DownloadURL != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Download: %s\n", r.DownloadURL)
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
				skipOn403(t, err)
				t.Fatalf("Failed to list %s recordings: %v", format, err)
			}

			t.Logf("%s format: Found %d recordings", format, len(page.Items))

			for i, r := range page.Items {
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds Size=%dMB\n",
					i+1, r.ID, r.Topic, r.DurationSeconds, r.SizeBytes/(1024*1024))

				if r.ServiceType != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Service: %s\n", r.ServiceType)
				}
				if r.Status != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Status: %s\n", r.Status)
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
				skipOn403(t, err)
				t.Fatalf("Failed to list %s recordings: %v", status, err)
			}

			t.Logf("%s status: Found %d recordings", status, len(page.Items))

			for i, r := range page.Items {
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Format=%s\n",
					i+1, r.ID, r.Topic, r.Format)

				if r.MeetingID != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Meeting ID: %s\n", r.MeetingID)
				}
				if r.HostEmail != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Host: %s\n", r.HostEmail)
				}
				if r.TimeRecorded != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Recorded: %s\n", r.TimeRecorded)
				}
			}
		})
	}
}

// TestFunctionalRecordingsGetWithDownloadLinks tests getting a recording with temporary download links
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRecordingsGetWithDownloadLinks -v ./recordings/
func TestFunctionalRecordingsGetWithDownloadLinks(t *testing.T) {
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

	page, err := recordingsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  5,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list recordings: %v", err)
	}
	if len(page.Items) == 0 {
		t.Skip("No recordings found in the last 30 days")
	}

	recording, err := recordingsClient.Get(page.Items[0].ID)
	if err != nil {
		t.Fatalf("Failed to get recording: %v", err)
	}

	t.Logf("Recording: ID=%s Topic=%q Format=%s Duration=%ds", recording.ID, recording.Topic, recording.Format, recording.DurationSeconds)

	if recording.TemporaryDirectDownloadLinks != nil {
		links := recording.TemporaryDirectDownloadLinks
		t.Logf("Download links available (expires: %s)", links.Expiration)
		if links.RecordingDownloadLink != "" {
			t.Logf("  Recording: available")
		}
		if links.AudioDownloadLink != "" {
			t.Logf("  Audio: available")
		}
		if links.TranscriptDownloadLink != "" {
			t.Logf("  Transcript: available")
		}
	} else {
		t.Logf("No temporary download links (may require specific permissions)")
	}
}

// TestFunctionalRecordingsDownloadAudio tests downloading audio from a recording
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRecordingsDownloadAudio -v ./recordings/
func TestFunctionalRecordingsDownloadAudio(t *testing.T) {
	token := os.Getenv("WEBEX_ACCESS_TOKEN")
	if token == "" {
		t.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
	}

	client, err := webexsdk.NewClient(token, &webexsdk.Config{
		BaseURL: "https://webexapis.com/v1",
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Webex client: %v", err)
	}

	recordingsClient := New(client, nil)

	page, err := recordingsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  10,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list recordings: %v", err)
	}
	if len(page.Items) == 0 {
		t.Skip("No recordings found")
	}

	for _, rec := range page.Items {
		audioURL, _, err := recordingsClient.GetAudioDownloadLink(rec.ID)
		if err != nil {
			t.Logf("No audio link for recording %s: %v", rec.ID, err)
			continue
		}

		t.Logf("Downloading audio from recording %s", rec.ID)
		content, err := recordingsClient.DownloadAudio(rec.ID)
		if err != nil {
			t.Fatalf("Failed to download audio: %v", err)
		}

		t.Logf("Audio downloaded: ContentType=%s Size=%d bytes", content.ContentType, len(content.Data))
		_ = audioURL
		return
	}
	t.Skip("No recordings with audio download links found")
}

// TestFunctionalRecordingsNotFound tests structured error on invalid recording ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRecordingsNotFound -v ./recordings/
func TestFunctionalRecordingsNotFound(t *testing.T) {
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

	_, err = recordingsClient.Get("invalid-recording-id")
	if err == nil {
		t.Fatal("Expected error for invalid recording ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

// TestFunctionalRecordingsCursorNavigation tests PageFromCursor with recordings
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalRecordingsCursorNavigation -v ./recordings/
func TestFunctionalRecordingsCursorNavigation(t *testing.T) {
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

	page, err := recordingsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  1,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("List failed: %v", err)
	}

	if !page.HasNext {
		t.Log("Only one page of results — skipping cursor navigation test")
		return
	}

	cursor := page.NextPage
	t.Logf("Saved cursor: %s", cursor)

	directPage, err := client.PageFromCursor(cursor)
	if err != nil {
		t.Fatalf("PageFromCursor failed: %v", err)
	}

	t.Logf("Direct cursor navigation: got %d items, hasNext=%v", len(directPage.Items), directPage.HasNext)
	if len(directPage.Items) == 0 {
		t.Error("Expected items from cursor navigation")
	}
}
//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package transcripts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

// TestFunctionalListTranscripts lists transcripts from the last 7 days
// using the real Webex API.
//
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListTranscripts -v ./transcripts/
func TestFunctionalListTranscripts(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// List transcripts from the last 7 days
	options := &ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  50,
	}

	page, err := transcriptsClient.List(options)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	t.Logf("Found %d transcripts from the last 7 days", len(page.Items))
	for i, tr := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds Status=%s\n",
			i+1, tr.ID, tr.MeetingTopic, tr.Duration, tr.Status)

		// Log additional transcript details
		if tr.HostEmail != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Host: %s\n", tr.HostEmail)
		}
		if tr.SiteURL != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Site: %s\n", tr.SiteURL)
		}
		if tr.StartTime != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Start: %s\n", tr.StartTime)
		}
		if tr.EndTime != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    End: %s\n", tr.EndTime)
		}
		if tr.VttDownloadLink != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    VTT Download: %s\n", tr.VttDownloadLink)
		}
		if tr.TxtDownloadLink != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Text Download: %s\n", tr.TxtDownloadLink)
		}
		if tr.MeetingID != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Meeting ID: %s\n", tr.MeetingID)
		}
	}
}

// TestFunctionalListTranscriptsByMeeting lists transcripts for a specific meeting
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListTranscriptsByMeeting -v ./transcripts/
func TestFunctionalListTranscriptsByMeeting(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// Get meeting instances (not series) from the last 30 days
	params := url.Values{}
	params.Set("meetingType", "meeting")
	params.Set("state", "ended")
	params.Set("from", time.Now().AddDate(0, 0, -30).Format(time.RFC3339))
	params.Set("to", time.Now().Format(time.RFC3339))
	params.Set("max", "10")

	resp, err := client.Request("GET", "meetings", params, nil)
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list meeting instances: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	page, err := webexsdk.NewPage(resp, client, "meetings")
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to create page: %v", err)
	}

	if len(page.Items) == 0 {
		t.Skip("No meeting instances found in the last 30 days")
	}

	// Try to find transcripts for each meeting
	for _, item := range page.Items {
		var meeting struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(item, &meeting); err != nil {
			continue
		}

		t.Logf("Checking transcripts for meeting: %s (%s)", meeting.Title, meeting.ID)

		options := &ListOptions{
			MeetingID: meeting.ID,
			Max:       10,
		}

		transcriptsPage, err := transcriptsClient.List(options)
		if err != nil {
			t.Logf("Failed to list transcripts for meeting %s: %v", meeting.ID, err)
			continue
		}

		if len(transcriptsPage.Items) > 0 {
			t.Logf("Found %d transcripts for meeting %s", len(transcriptsPage.Items), meeting.ID)
			for i, tr := range transcriptsPage.Items {
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds\n",
					i+1, tr.ID, tr.MeetingTopic, tr.Duration)

				if tr.StartTime != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Start: %s\n", tr.StartTime)
				}
				if tr.EndTime != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    End: %s\n", tr.EndTime)
				}
				if tr.Status != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Status: %s\n", tr.Status)
				}
			}
			break // Found a meeting with transcripts
		}
	}
}

// TestFunctionalListTranscriptSnippets lists snippets from transcripts
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListTranscriptSnippets -v ./transcripts/
func TestFunctionalListTranscriptSnippets(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// First, get a list of transcripts to find one with snippets
	transcriptsPage, err := transcriptsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  10,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	if len(transcriptsPage.Items) == 0 {
		t.Skip("No transcripts found in the last 7 days")
	}

	// Try to get snippets for each transcript
	for _, transcript := range transcriptsPage.Items {
		t.Logf("Checking snippets for transcript: %s", transcript.ID)

		snippetsPage, err := transcriptsClient.ListSnippets(transcript.ID, &SnippetListOptions{
			Max: 20,
		})

		if err != nil {
			t.Logf("Failed to list snippets for transcript %s: %v", transcript.ID, err)
			continue
		}

		if len(snippetsPage.Items) > 0 {
			t.Logf("Found %d snippets for transcript %s", len(snippetsPage.Items), transcript.ID)

			for i, s := range snippetsPage.Items {
				_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Speaker=%q Text=%q\n",
					i+1, s.ID, s.PersonName, s.Text)

				if s.PersonEmail != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Email: %s\n", s.PersonEmail)
				}
				if s.DurationMillisecond > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "    Duration: %dms\n", s.DurationMillisecond)
				}
				if s.OffsetMillisecond > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "    Offset: %dms\n", s.OffsetMillisecond)
				}
				if s.Language != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Language: %s\n", s.Language)
				}
				if s.Confidence > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "    Confidence: %.2f\n", s.Confidence)
				}
				if s.StartTime != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    Start: %s\n", s.StartTime)
				}
				if s.EndTime != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    End: %s\n", s.EndTime)
				}
			}
			break // Found a transcript with snippets
		}
	}
}

// TestFunctionalListTranscriptSnippetsByPerson lists snippets by a specific person
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalListTranscriptSnippetsByPerson -v ./transcripts/
func TestFunctionalListTranscriptSnippetsByPerson(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// First, get a list of transcripts to find one with snippets
	transcriptsPage, err := transcriptsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  5,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	if len(transcriptsPage.Items) == 0 {
		t.Skip("No transcripts found in the last 7 days")
	}

	// Get snippets from the first transcript to find a person
	transcript := transcriptsPage.Items[0]
	snippetsPage, err := transcriptsClient.ListSnippets(transcript.ID, &SnippetListOptions{
		Max: 50,
	})

	if err != nil {
		t.Fatalf("Failed to list snippets for transcript %s: %v", transcript.ID, err)
	}

	if len(snippetsPage.Items) == 0 {
		t.Skipf("No snippets found in transcript %s", transcript.ID)
	}

	// Find the first person with multiple snippets
	personEmail := snippetsPage.Items[0].PersonEmail
	if personEmail == "" {
		t.Skip("No person email found in snippets")
	}

	t.Logf("Filtering snippets for person: %s", personEmail)

	// Now get snippets filtered by this person
	filteredSnippetsPage, err := transcriptsClient.ListSnippets(transcript.ID, &SnippetListOptions{
		Max:         20,
		PersonEmail: personEmail,
	})

	if err != nil {
		t.Fatalf("Failed to list filtered snippets: %v", err)
	}

	t.Logf("Found %d snippets for person %s", len(filteredSnippetsPage.Items), personEmail)

	for i, s := range filteredSnippetsPage.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Text=%q\n",
			i+1, s.ID, s.Text)

		if s.DurationMillisecond > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Duration: %dms\n", s.DurationMillisecond)
		}
		if s.OffsetMillisecond > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Offset: %dms (%.1fs into meeting)\n",
				s.OffsetMillisecond, float64(s.OffsetMillisecond)/1000)
		}
		if s.StartTime != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Start: %s\n", s.StartTime)
		}
	}
}

// TestFunctionalTranscriptsDownload tests downloading transcript content
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTranscriptsDownload -v ./transcripts/
func TestFunctionalTranscriptsDownload(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// Find a transcript to download
	page, err := transcriptsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  5,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list transcripts: %v", err)
	}
	if len(page.Items) == 0 {
		t.Skip("No transcripts found in the last 30 days")
	}

	transcript := page.Items[0]
	t.Logf("Downloading transcript: %s (topic=%q)", transcript.ID, transcript.MeetingTopic)

	// Test TXT format download
	txtContent, err := transcriptsClient.Download(transcript.ID, "txt")
	if err != nil {
		t.Fatalf("Failed to download TXT transcript: %v", err)
	}
	t.Logf("TXT content length: %d bytes", len(txtContent))
	if len(txtContent) > 200 {
		_, _ = fmt.Fprintf(os.Stdout, "TXT preview: %s...\n", txtContent[:200])
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "TXT content: %s\n", txtContent)
	}

	// Test VTT format download
	vttContent, err := transcriptsClient.Download(transcript.ID, "vtt")
	if err != nil {
		t.Fatalf("Failed to download VTT transcript: %v", err)
	}
	t.Logf("VTT content length: %d bytes", len(vttContent))
	if len(vttContent) > 200 {
		_, _ = fmt.Fprintf(os.Stdout, "VTT preview: %s...\n", vttContent[:200])
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "VTT content: %s\n", vttContent)
	}
}

// TestFunctionalTranscriptsGetSnippet tests getting a single snippet
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTranscriptsGetSnippet -v ./transcripts/
func TestFunctionalTranscriptsGetSnippet(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// Find a transcript with snippets
	tPage, err := transcriptsClient.List(&ListOptions{
		From: time.Now().AddDate(0, 0, -30).Format(time.RFC3339),
		To:   time.Now().Format(time.RFC3339),
		Max:  5,
	})
	if err != nil {
		skipOn403(t, err)
		t.Fatalf("Failed to list transcripts: %v", err)
	}
	if len(tPage.Items) == 0 {
		t.Skip("No transcripts found")
	}

	var foundSnippet *Snippet
	var transcriptID string
	for _, tr := range tPage.Items {
		sPage, err := transcriptsClient.ListSnippets(tr.ID, &SnippetListOptions{Max: 5})
		if err != nil {
			continue
		}
		if len(sPage.Items) > 0 {
			foundSnippet = &sPage.Items[0]
			transcriptID = tr.ID
			break
		}
	}
	if foundSnippet == nil {
		t.Skip("No snippets found in any transcript")
	}

	// Get the specific snippet
	snippet, err := transcriptsClient.GetSnippet(transcriptID, foundSnippet.ID)
	if err != nil {
		t.Fatalf("Failed to get snippet: %v", err)
	}

	if snippet.ID != foundSnippet.ID {
		t.Errorf("Expected snippet ID %s, got %s", foundSnippet.ID, snippet.ID)
	}
	t.Logf("Got snippet: ID=%s Speaker=%q Text=%q", snippet.ID, snippet.PersonName, snippet.Text)
}

// TestFunctionalTranscriptsNotFound tests structured error on invalid transcript ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalTranscriptsNotFound -v ./transcripts/
func TestFunctionalTranscriptsNotFound(t *testing.T) {
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

	transcriptsClient := New(client, nil)

	// Try to download a transcript with an invalid ID
	_, err = transcriptsClient.Download("invalid-transcript-id", "txt")
	if err == nil {
		t.Fatal("Expected error for invalid transcript ID, got nil")
	}

	// Verify structured error type
	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

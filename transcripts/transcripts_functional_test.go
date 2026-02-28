//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package transcripts

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

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
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	t.Logf("Found %d transcripts from the last 7 days", len(page.Items))
	for i, tr := range page.Items {
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds Status=%s\n",
			i+1, tr.ID, tr.MeetingTopic, tr.Duration, tr.Status)

		// Log additional transcript details
		if tr.HostEmail != "" {
			fmt.Fprintf(os.Stdout, "    Host: %s\n", tr.HostEmail)
		}
		if tr.SiteURL != "" {
			fmt.Fprintf(os.Stdout, "    Site: %s\n", tr.SiteURL)
		}
		if tr.StartTime != "" {
			fmt.Fprintf(os.Stdout, "    Start: %s\n", tr.StartTime)
		}
		if tr.EndTime != "" {
			fmt.Fprintf(os.Stdout, "    End: %s\n", tr.EndTime)
		}
		if tr.VttDownloadLink != "" {
			fmt.Fprintf(os.Stdout, "    VTT Download: %s\n", tr.VttDownloadLink)
		}
		if tr.TxtDownloadLink != "" {
			fmt.Fprintf(os.Stdout, "    Text Download: %s\n", tr.TxtDownloadLink)
		}
		if tr.MeetingID != "" {
			fmt.Fprintf(os.Stdout, "    Meeting ID: %s\n", tr.MeetingID)
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

	// First, get a list of meetings to find one with transcripts
	resp, err := client.Request("GET", "meetings", nil, nil)
	if err != nil {
		t.Fatalf("Failed to list meetings: %v", err)
	}
	defer resp.Body.Close()

	// Add query parameters to get meeting instances (not series)
	params := url.Values{}
	params.Set("meetingType", "meeting")
	params.Set("state", "ended")
	params.Set("from", time.Now().AddDate(0, 0, -30).Format(time.RFC3339)) // Last 30 days
	params.Set("to", time.Now().Format(time.RFC3339))
	params.Set("max", "10")

	resp, err = client.Request("GET", "meetings?"+params.Encode(), nil, nil)
	if err != nil {
		t.Fatalf("Failed to list meeting instances: %v", err)
	}
	defer resp.Body.Close()

	page, err := webexsdk.NewPage(resp, client, "meetings")
	if err != nil {
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
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Topic=%q Duration=%ds\n",
					i+1, tr.ID, tr.MeetingTopic, tr.Duration)

				if tr.StartTime != "" {
					fmt.Fprintf(os.Stdout, "    Start: %s\n", tr.StartTime)
				}
				if tr.EndTime != "" {
					fmt.Fprintf(os.Stdout, "    End: %s\n", tr.EndTime)
				}
				if tr.Status != "" {
					fmt.Fprintf(os.Stdout, "    Status: %s\n", tr.Status)
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
				fmt.Fprintf(os.Stdout, "[%d] ID=%s Speaker=%q Text=%q\n",
					i+1, s.ID, s.PersonName, s.Text)

				if s.PersonEmail != "" {
					fmt.Fprintf(os.Stdout, "    Email: %s\n", s.PersonEmail)
				}
				if s.DurationMillisecond > 0 {
					fmt.Fprintf(os.Stdout, "    Duration: %dms\n", s.DurationMillisecond)
				}
				if s.OffsetMillisecond > 0 {
					fmt.Fprintf(os.Stdout, "    Offset: %dms\n", s.OffsetMillisecond)
				}
				if s.Language != "" {
					fmt.Fprintf(os.Stdout, "    Language: %s\n", s.Language)
				}
				if s.Confidence > 0 {
					fmt.Fprintf(os.Stdout, "    Confidence: %.2f\n", s.Confidence)
				}
				if s.StartTime != "" {
					fmt.Fprintf(os.Stdout, "    Start: %s\n", s.StartTime)
				}
				if s.EndTime != "" {
					fmt.Fprintf(os.Stdout, "    End: %s\n", s.EndTime)
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
		fmt.Fprintf(os.Stdout, "[%d] ID=%s Text=%q\n",
			i+1, s.ID, s.Text)

		if s.DurationMillisecond > 0 {
			fmt.Fprintf(os.Stdout, "    Duration: %dms\n", s.DurationMillisecond)
		}
		if s.OffsetMillisecond > 0 {
			fmt.Fprintf(os.Stdout, "    Offset: %dms (%.1fs into meeting)\n",
				s.OffsetMillisecond, float64(s.OffsetMillisecond)/1000)
		}
		if s.StartTime != "" {
			fmt.Fprintf(os.Stdout, "    Start: %s\n", s.StartTime)
		}
	}
}

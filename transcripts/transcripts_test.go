/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package transcripts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = baseURL

	return New(client, nil), server
}

func TestList(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts" {
			t.Errorf("Expected path '/meetingTranscripts', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		if r.URL.Query().Get("meetingId") != "test-meeting-id" {
			t.Errorf("Expected meetingId 'test-meeting-id', got '%s'", r.URL.Query().Get("meetingId"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		transcripts := []Transcript{
			{
				ID:                 "transcript-1",
				MeetingID:          "test-meeting-id",
				MeetingTopic:       "Meeting Transcript 1",
				SiteURL:            "example.webex.com",
				ScheduledMeetingID: "scheduled-1",
				MeetingSeriesID:    "series-1",
				HostUserID:         "host-user-1",
				StartTime:          "2026-01-15T10:00:00Z",
				Status:             "available",
				VttDownloadLink:    "https://example.com/transcript-1.vtt",
				TxtDownloadLink:    "https://example.com/transcript-1.txt",
			},
			{
				ID:                 "transcript-2",
				MeetingID:          "test-meeting-id",
				MeetingTopic:       "Meeting Transcript 2",
				SiteURL:            "example.webex.com",
				ScheduledMeetingID: "scheduled-2",
				MeetingSeriesID:    "series-1",
				HostUserID:         "host-user-1",
				StartTime:          "2026-01-16T14:00:00Z",
				Status:             "available",
				VttDownloadLink:    "https://example.com/transcript-2.vtt",
				TxtDownloadLink:    "https://example.com/transcript-2.txt",
			},
		}

		response := struct {
			Items []Transcript `json:"items"`
		}{
			Items: transcripts,
		}

		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	options := &ListOptions{
		MeetingID: "test-meeting-id",
	}
	page, err := transcriptsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	if page.Items[0].ID != "transcript-1" {
		t.Errorf("Expected ID 'transcript-1', got '%s'", page.Items[0].ID)
	}
	if page.Items[0].MeetingTopic != "Meeting Transcript 1" {
		t.Errorf("Expected meetingTopic 'Meeting Transcript 1', got '%s'", page.Items[0].MeetingTopic)
	}
	if page.Items[0].SiteURL != "example.webex.com" {
		t.Errorf("Expected siteUrl 'example.webex.com', got '%s'", page.Items[0].SiteURL)
	}
	if page.Items[0].HostUserID != "host-user-1" {
		t.Errorf("Expected hostUserId 'host-user-1', got '%s'", page.Items[0].HostUserID)
	}
	if page.Items[0].Status != "available" {
		t.Errorf("Expected status 'available', got '%s'", page.Items[0].Status)
	}
	if page.Items[1].ID != "transcript-2" {
		t.Errorf("Expected ID 'transcript-2', got '%s'", page.Items[1].ID)
	}
}

func TestListWithFilters(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("hostEmail") != "host@example.com" {
			t.Errorf("Expected hostEmail 'host@example.com', got '%s'", r.URL.Query().Get("hostEmail"))
		}
		if r.URL.Query().Get("from") != "2026-01-01T00:00:00Z" {
			t.Errorf("Expected from '2026-01-01T00:00:00Z', got '%s'", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("max") != "10" {
			t.Errorf("Expected max '10', got '%s'", r.URL.Query().Get("max"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Transcript `json:"items"`
		}{
			Items: []Transcript{},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	options := &ListOptions{
		HostEmail: "host@example.com",
		From:      "2026-01-01T00:00:00Z",
		Max:       10,
	}
	_, err := transcriptsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list transcripts with filters: %v", err)
	}
}

func TestListDefaultDateRange(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// When no meetingId and no date range is specified, the SDK should
		// default to the last 30 days
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		if from == "" {
			t.Error("Expected 'from' query parameter to be set by default")
		}
		if to == "" {
			t.Error("Expected 'to' query parameter to be set by default")
		}

		// Verify the range is approximately 30 days
		if from != "" && to != "" {
			fromTime, err1 := time.Parse(time.RFC3339, from)
			toTime, err2 := time.Parse(time.RFC3339, to)
			if err1 == nil && err2 == nil {
				diff := toTime.Sub(fromTime)
				if diff < 29*24*time.Hour || diff > 31*24*time.Hour {
					t.Errorf("Expected ~30 day range, got %v", diff)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Transcript `json:"items"`
		}{
			Items: []Transcript{},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	// Call List with nil options - should auto-set date range
	_, err := transcriptsPlugin.List(nil)
	if err != nil {
		t.Fatalf("Failed to list transcripts with default date range: %v", err)
	}
}

func TestListByMeetingIDNoDefaultDateRange(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// When meetingId is specified, no default date range should be added
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		meetingID := r.URL.Query().Get("meetingId")

		if meetingID != "test-meeting-id" {
			t.Errorf("Expected meetingId 'test-meeting-id', got '%s'", meetingID)
		}
		if from != "" {
			t.Errorf("Expected no 'from' param when meetingId is set, got '%s'", from)
		}
		if to != "" {
			t.Errorf("Expected no 'to' param when meetingId is set, got '%s'", to)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Transcript `json:"items"`
		}{
			Items: []Transcript{},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	_, err := transcriptsPlugin.List(&ListOptions{MeetingID: "test-meeting-id"})
	if err != nil {
		t.Fatalf("Failed to list transcripts by meetingId: %v", err)
	}
}

func TestDownload(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts/transcript-1/download" {
			t.Errorf("Expected path '/meetingTranscripts/transcript-1/download', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		format := r.URL.Query().Get("format")
		if format != "txt" {
			t.Errorf("Expected format 'txt', got '%s'", format)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Speaker 1: Hello everyone.\nSpeaker 2: Hi, how are you?\nSpeaker 1: Let's get started."))
	})
	defer server.Close()

	content, err := transcriptsPlugin.Download("transcript-1", "txt")
	if err != nil {
		t.Fatalf("Failed to download transcript: %v", err)
	}

	expected := "Speaker 1: Hello everyone.\nSpeaker 2: Hi, how are you?\nSpeaker 1: Let's get started."
	if content != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, content)
	}
}

func TestDownloadVtt(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")
		if format != "vtt" {
			t.Errorf("Expected format 'vtt', got '%s'", format)
		}

		w.Header().Set("Content-Type", "text/vtt")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("WEBVTT\n\n00:00:01.000 --> 00:00:03.000\nHello everyone."))
	})
	defer server.Close()

	content, err := transcriptsPlugin.Download("transcript-1", "vtt")
	if err != nil {
		t.Fatalf("Failed to download transcript: %v", err)
	}

	if content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestDownloadDefaultFormat(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")
		if format != "txt" {
			t.Errorf("Expected default format 'txt', got '%s'", format)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("transcript content"))
	})
	defer server.Close()

	content, err := transcriptsPlugin.Download("transcript-1", "")
	if err != nil {
		t.Fatalf("Failed to download transcript: %v", err)
	}

	if content != "transcript content" {
		t.Errorf("Expected 'transcript content', got '%s'", content)
	}
}

func TestDownloadWithMeetingID(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts/transcript-1/download" {
			t.Errorf("Expected path '/meetingTranscripts/transcript-1/download', got '%s'", r.URL.Path)
		}

		meetingID := r.URL.Query().Get("meetingId")
		if meetingID != "meeting-123" {
			t.Errorf("Expected meetingId 'meeting-123', got '%s'", meetingID)
		}

		format := r.URL.Query().Get("format")
		if format != "txt" {
			t.Errorf("Expected format 'txt', got '%s'", format)
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("transcript content with meeting id"))
	})
	defer server.Close()

	content, err := transcriptsPlugin.Download("transcript-1", "txt", &DownloadOptions{
		MeetingID: "meeting-123",
	})
	if err != nil {
		t.Fatalf("Failed to download transcript with meeting ID: %v", err)
	}

	if content != "transcript content with meeting id" {
		t.Errorf("Expected 'transcript content with meeting id', got '%s'", content)
	}
}

func TestDownloadValidation(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := transcriptsPlugin.Download("", "txt")
	if err == nil {
		t.Error("Expected error for empty transcriptID")
	}

	_, err = transcriptsPlugin.Download("transcript-1", "pdf")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestListSnippets(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts/transcript-1/snippets" {
			t.Errorf("Expected path '/meetingTranscripts/transcript-1/snippets', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		snippets := []Snippet{
			{
				ID:           "snippet-1",
				TranscriptID: "transcript-1",
				Text:         "Hello everyone, welcome to the meeting.",
				PersonName:   "John Doe",
				PersonEmail:  "john@example.com",
				StartTime:    "2026-01-15T10:00:01Z",
				EndTime:      "2026-01-15T10:00:05Z",
				Duration:     4.0,
				Language:     "en",
			},
			{
				ID:           "snippet-2",
				TranscriptID: "transcript-1",
				Text:         "Thanks John. Let's review the agenda.",
				PersonName:   "Jane Smith",
				PersonEmail:  "jane@example.com",
				StartTime:    "2026-01-15T10:00:06Z",
				EndTime:      "2026-01-15T10:00:10Z",
				Duration:     4.0,
				Language:     "en",
			},
		}

		response := struct {
			Items []Snippet `json:"items"`
		}{
			Items: snippets,
		}

		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	page, err := transcriptsPlugin.ListSnippets("transcript-1", nil)
	if err != nil {
		t.Fatalf("Failed to list snippets: %v", err)
	}

	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	if page.Items[0].ID != "snippet-1" {
		t.Errorf("Expected ID 'snippet-1', got '%s'", page.Items[0].ID)
	}
	if page.Items[0].Text != "Hello everyone, welcome to the meeting." {
		t.Errorf("Expected text 'Hello everyone, welcome to the meeting.', got '%s'", page.Items[0].Text)
	}
	if page.Items[0].PersonName != "John Doe" {
		t.Errorf("Expected personName 'John Doe', got '%s'", page.Items[0].PersonName)
	}
}

func TestListSnippetsValidation(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := transcriptsPlugin.ListSnippets("", nil)
	if err == nil {
		t.Error("Expected error for empty transcriptID")
	}
}

func TestGetSnippet(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts/transcript-1/snippets/snippet-1" {
			t.Errorf("Expected path '/meetingTranscripts/transcript-1/snippets/snippet-1', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		snippet := Snippet{
			ID:           "snippet-1",
			TranscriptID: "transcript-1",
			Text:         "Hello everyone, welcome to the meeting.",
			PersonName:   "John Doe",
			PersonEmail:  "john@example.com",
			StartTime:    "2026-01-15T10:00:01Z",
			EndTime:      "2026-01-15T10:00:05Z",
			Duration:     4.0,
			Language:     "en",
		}

		_ = json.NewEncoder(w).Encode(snippet)
	})
	defer server.Close()

	snippet, err := transcriptsPlugin.GetSnippet("transcript-1", "snippet-1")
	if err != nil {
		t.Fatalf("Failed to get snippet: %v", err)
	}

	if snippet.ID != "snippet-1" {
		t.Errorf("Expected ID 'snippet-1', got '%s'", snippet.ID)
	}
	if snippet.Text != "Hello everyone, welcome to the meeting." {
		t.Errorf("Expected text 'Hello everyone, welcome to the meeting.', got '%s'", snippet.Text)
	}
	if snippet.PersonName != "John Doe" {
		t.Errorf("Expected personName 'John Doe', got '%s'", snippet.PersonName)
	}
}

func TestGetSnippetValidation(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := transcriptsPlugin.GetSnippet("", "snippet-1")
	if err == nil {
		t.Error("Expected error for empty transcriptID")
	}

	_, err = transcriptsPlugin.GetSnippet("transcript-1", "")
	if err == nil {
		t.Error("Expected error for empty snippetID")
	}
}

func TestUpdateSnippet(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingTranscripts/transcript-1/snippets/snippet-1" {
			t.Errorf("Expected path '/meetingTranscripts/transcript-1/snippets/snippet-1', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if body.Text != "Updated text" {
			t.Errorf("Expected text 'Updated text', got '%s'", body.Text)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		snippet := Snippet{
			ID:           "snippet-1",
			TranscriptID: "transcript-1",
			Text:         body.Text,
			PersonName:   "John Doe",
			PersonEmail:  "john@example.com",
		}

		_ = json.NewEncoder(w).Encode(snippet)
	})
	defer server.Close()

	snippet := &Snippet{Text: "Updated text"}
	result, err := transcriptsPlugin.UpdateSnippet("transcript-1", "snippet-1", snippet)
	if err != nil {
		t.Fatalf("Failed to update snippet: %v", err)
	}

	if result.Text != "Updated text" {
		t.Errorf("Expected text 'Updated text', got '%s'", result.Text)
	}
}

func TestUpdateSnippetValidation(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := transcriptsPlugin.UpdateSnippet("", "snippet-1", &Snippet{Text: "test"})
	if err == nil {
		t.Error("Expected error for empty transcriptID")
	}

	_, err = transcriptsPlugin.UpdateSnippet("transcript-1", "", &Snippet{Text: "test"})
	if err == nil {
		t.Error("Expected error for empty snippetID")
	}

	_, err = transcriptsPlugin.UpdateSnippet("transcript-1", "snippet-1", &Snippet{})
	if err == nil {
		t.Error("Expected error for empty text")
	}
}

func TestTranscriptNewFields(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Transcript `json:"items"`
		}{
			Items: []Transcript{
				{
					ID:              "transcript-1",
					MeetingID:       "meeting-1",
					MeetingTopic:    "Test Meeting",
					StartTime:       "2026-01-15T10:00:00Z",
					EndTime:         "2026-01-15T11:00:00Z",
					Duration:        3600,
					Status:          "available",
					Created:         "2026-01-15T11:05:00Z",
					Updated:         "2026-01-15T11:10:00Z",
					VttDownloadLink: "https://example.com/download.vtt",
					TxtDownloadLink: "https://example.com/download.txt",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	page, err := transcriptsPlugin.List(&ListOptions{MeetingID: "meeting-1"})
	if err != nil {
		t.Fatalf("Failed to list transcripts: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(page.Items))
	}

	tr := page.Items[0]
	if tr.EndTime != "2026-01-15T11:00:00Z" {
		t.Errorf("Expected endTime '2026-01-15T11:00:00Z', got '%s'", tr.EndTime)
	}
	if tr.Duration != 3600 {
		t.Errorf("Expected duration 3600, got %d", tr.Duration)
	}
	if tr.Created != "2026-01-15T11:05:00Z" {
		t.Errorf("Expected created '2026-01-15T11:05:00Z', got '%s'", tr.Created)
	}
	if tr.Updated != "2026-01-15T11:10:00Z" {
		t.Errorf("Expected updated '2026-01-15T11:10:00Z', got '%s'", tr.Updated)
	}
}

func TestSnippetConfidenceField(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Snippet `json:"items"`
		}{
			Items: []Snippet{
				{
					ID:           "snippet-1",
					TranscriptID: "transcript-1",
					Text:         "Hello everyone.",
					PersonName:   "John Doe",
					Confidence:   0.95,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	page, err := transcriptsPlugin.ListSnippets("transcript-1", nil)
	if err != nil {
		t.Fatalf("Failed to list snippets: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(page.Items))
	}

	if page.Items[0].Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", page.Items[0].Confidence)
	}
}

func TestListSnippetsWithFilters(t *testing.T) {
	transcriptsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("personEmail") != "speaker@example.com" {
			t.Errorf("Expected personEmail 'speaker@example.com', got '%s'", r.URL.Query().Get("personEmail"))
		}
		if r.URL.Query().Get("peopleId") != "people-123" {
			t.Errorf("Expected peopleId 'people-123', got '%s'", r.URL.Query().Get("peopleId"))
		}
		if r.URL.Query().Get("from") != "2026-01-15T10:00:00Z" {
			t.Errorf("Expected from '2026-01-15T10:00:00Z', got '%s'", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "2026-01-15T11:00:00Z" {
			t.Errorf("Expected to '2026-01-15T11:00:00Z', got '%s'", r.URL.Query().Get("to"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Items []Snippet `json:"items"`
		}{Items: []Snippet{}})
	})
	defer server.Close()

	_, err := transcriptsPlugin.ListSnippets("transcript-1", &SnippetListOptions{
		PersonEmail: "speaker@example.com",
		PeopleID:    "people-123",
		From:        "2026-01-15T10:00:00Z",
		To:          "2026-01-15T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("Failed to list snippets with filters: %v", err)
	}
}

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package recordings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = baseURL

	return New(client, nil), server
}

func TestList(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recordings" {
			t.Errorf("Expected path '/recordings', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("max") != "5" {
			t.Errorf("Expected max '5', got '%s'", r.URL.Query().Get("max"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			Items []Recording `json:"items"`
		}{
			Items: []Recording{
				{
					ID:              "rec-1",
					MeetingID:       "meeting-1",
					Topic:           "Team Standup",
					Format:          "MP4",
					DurationSeconds: 1800,
					SizeBytes:       50000000,
					Status:          "available",
					HostEmail:       "host@example.com",
					DownloadURL:     "https://example.com/download/rec-1",
					PlaybackURL:     "https://example.com/play/rec-1",
				},
				{
					ID:              "rec-2",
					MeetingID:       "meeting-2",
					Topic:           "Sprint Review",
					Format:          "MP4",
					DurationSeconds: 3600,
					SizeBytes:       100000000,
					Status:          "available",
					HostEmail:       "host@example.com",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	page, err := rc.List(&ListOptions{Max: 5})
	if err != nil {
		t.Fatalf("Failed to list recordings: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(page.Items))
	}
	if page.Items[0].ID != "rec-1" {
		t.Errorf("Expected ID 'rec-1', got '%s'", page.Items[0].ID)
	}
	if page.Items[0].Topic != "Team Standup" {
		t.Errorf("Expected topic 'Team Standup', got '%s'", page.Items[0].Topic)
	}
	if page.Items[0].DurationSeconds != 1800 {
		t.Errorf("Expected durationSeconds 1800, got %d", page.Items[0].DurationSeconds)
	}
	if page.Items[0].Format != "MP4" {
		t.Errorf("Expected format 'MP4', got '%s'", page.Items[0].Format)
	}
	if page.Items[1].ID != "rec-2" {
		t.Errorf("Expected ID 'rec-2', got '%s'", page.Items[1].ID)
	}
}

func TestListWithFilters(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("meetingId") != "meeting-123" {
			t.Errorf("Expected meetingId 'meeting-123', got '%s'", q.Get("meetingId"))
		}
		if q.Get("hostEmail") != "host@example.com" {
			t.Errorf("Expected hostEmail, got '%s'", q.Get("hostEmail"))
		}
		if q.Get("siteUrl") != "cisco.webex.com" {
			t.Errorf("Expected siteUrl 'cisco.webex.com', got '%s'", q.Get("siteUrl"))
		}
		if q.Get("from") != "2026-01-01T00:00:00Z" {
			t.Errorf("Expected from date, got '%s'", q.Get("from"))
		}
		if q.Get("to") != "2026-02-01T00:00:00Z" {
			t.Errorf("Expected to date, got '%s'", q.Get("to"))
		}
		if q.Get("status") != "available" {
			t.Errorf("Expected status 'available', got '%s'", q.Get("status"))
		}
		if q.Get("format") != "MP4" {
			t.Errorf("Expected format 'MP4', got '%s'", q.Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Items []Recording `json:"items"`
		}{Items: []Recording{}})
	})
	defer server.Close()

	_, err := rc.List(&ListOptions{
		MeetingID: "meeting-123",
		HostEmail: "host@example.com",
		SiteURL:   "cisco.webex.com",
		From:      "2026-01-01T00:00:00Z",
		To:        "2026-02-01T00:00:00Z",
		Status:    "available",
		Format:    "MP4",
	})
	if err != nil {
		t.Fatalf("Failed to list with filters: %v", err)
	}
}

func TestGet(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recordings/rec-123" {
			t.Errorf("Expected path '/recordings/rec-123', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Recording{
			ID:              "rec-123",
			Topic:           "Test Meeting",
			Format:          "MP4",
			DurationSeconds: 2646,
			SizeBytes:       95769395,
			Status:          "available",
			DownloadURL:     "https://example.com/download",
			PlaybackURL:     "https://example.com/play",
			TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
				RecordingDownloadLink:  "https://example.com/direct/video.mp4",
				AudioDownloadLink:      "https://example.com/direct/audio.mp3",
				TranscriptDownloadLink: "https://example.com/direct/transcript.txt",
				Expiration:             "2026-02-07T02:19:56Z",
			},
		})
	})
	defer server.Close()

	recording, err := rc.Get("rec-123")
	if err != nil {
		t.Fatalf("Failed to get recording: %v", err)
	}

	if recording.ID != "rec-123" {
		t.Errorf("Expected ID 'rec-123', got '%s'", recording.ID)
	}
	if recording.Topic != "Test Meeting" {
		t.Errorf("Expected topic 'Test Meeting', got '%s'", recording.Topic)
	}
	if recording.DurationSeconds != 2646 {
		t.Errorf("Expected durationSeconds 2646, got %d", recording.DurationSeconds)
	}
	if recording.TemporaryDirectDownloadLinks == nil {
		t.Fatal("Expected temporaryDirectDownloadLinks to be non-nil")
	}
	if recording.TemporaryDirectDownloadLinks.AudioDownloadLink != "https://example.com/direct/audio.mp3" {
		t.Errorf("Expected audio link, got '%s'", recording.TemporaryDirectDownloadLinks.AudioDownloadLink)
	}
	if recording.TemporaryDirectDownloadLinks.RecordingDownloadLink != "https://example.com/direct/video.mp4" {
		t.Errorf("Expected recording link, got '%s'", recording.TemporaryDirectDownloadLinks.RecordingDownloadLink)
	}
	if recording.TemporaryDirectDownloadLinks.TranscriptDownloadLink != "https://example.com/direct/transcript.txt" {
		t.Errorf("Expected transcript link, got '%s'", recording.TemporaryDirectDownloadLinks.TranscriptDownloadLink)
	}
	if recording.TemporaryDirectDownloadLinks.Expiration != "2026-02-07T02:19:56Z" {
		t.Errorf("Expected expiration, got '%s'", recording.TemporaryDirectDownloadLinks.Expiration)
	}
}

func TestGetValidation(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := rc.Get("")
	if err == nil {
		t.Error("Expected error for empty recordingID")
	}
}

func TestDelete(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recordings/rec-123" {
			t.Errorf("Expected path '/recordings/rec-123', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := rc.Delete("rec-123")
	if err != nil {
		t.Fatalf("Failed to delete recording: %v", err)
	}
}

func TestDeleteValidation(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	err := rc.Delete("")
	if err == nil {
		t.Error("Expected error for empty recordingID")
	}
}

func TestGetAudioDownloadLink(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{
			ID: "rec-123",
			TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
				AudioDownloadLink: "https://example.com/audio.mp3",
				Expiration:        "2026-02-07T02:19:56Z",
			},
		})
	})
	defer server.Close()

	audioURL, recording, err := rc.GetAudioDownloadLink("rec-123")
	if err != nil {
		t.Fatalf("Failed to get audio link: %v", err)
	}
	if audioURL != "https://example.com/audio.mp3" {
		t.Errorf("Expected audio URL, got '%s'", audioURL)
	}
	if recording.ID != "rec-123" {
		t.Errorf("Expected recording ID 'rec-123', got '%s'", recording.ID)
	}
}

func TestGetAudioDownloadLink_NoLinks(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{
			ID: "rec-no-links",
		})
	})
	defer server.Close()

	_, _, err := rc.GetAudioDownloadLink("rec-no-links")
	if err == nil {
		t.Error("Expected error when no download links available")
	}
}

func TestGetAudioDownloadLink_NoAudioLink(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{
			ID: "rec-no-audio",
			TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
				RecordingDownloadLink: "https://example.com/video.mp4",
			},
		})
	})
	defer server.Close()

	_, _, err := rc.GetAudioDownloadLink("rec-no-audio")
	if err == nil {
		t.Error("Expected error when no audio link available")
	}
}

func TestDownloadAudio(t *testing.T) {
	audioContent := []byte("fake mp3 audio content")

	mux := http.NewServeMux()
	mux.HandleFunc("/recordings/rec-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{
			ID: "rec-123",
			TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
				AudioDownloadLink: "", // will be set dynamically
			},
		})
	})

	// Use a single server for both API and download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/recordings/rec-123":
			w.Header().Set("Content-Type", "application/json")
			recording := Recording{
				ID: "rec-123",
				TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
					AudioDownloadLink: "http://" + r.Host + "/download/audio.mp3",
				},
			}
			_ = json.NewEncoder(w).Encode(recording)
		case "/download/audio.mp3":
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("Expected auth header on download request")
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Content-Disposition", `attachment; filename="audio.mp3"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(audioContent)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, _ := webexsdk.NewClient("test-token", config)
	client.BaseURL = baseURL
	rc := New(client, nil)

	content, err := rc.DownloadAudio("rec-123")
	if err != nil {
		t.Fatalf("DownloadAudio failed: %v", err)
	}

	if content.ContentType != "audio/mpeg" {
		t.Errorf("Expected content type 'audio/mpeg', got '%s'", content.ContentType)
	}
	if string(content.Data) != string(audioContent) {
		t.Errorf("Audio data mismatch")
	}
}

func TestDownloadRecording(t *testing.T) {
	videoContent := []byte("fake mp4 video content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/recordings/rec-123":
			w.Header().Set("Content-Type", "application/json")
			recording := Recording{
				ID: "rec-123",
				TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
					RecordingDownloadLink: "http://" + r.Host + "/download/video.mp4",
				},
			}
			_ = json.NewEncoder(w).Encode(recording)
		case "/download/video.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(videoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, _ := webexsdk.NewClient("test-token", config)
	client.BaseURL = baseURL
	rc := New(client, nil)

	content, err := rc.DownloadRecording("rec-123")
	if err != nil {
		t.Fatalf("DownloadRecording failed: %v", err)
	}

	if content.ContentType != "video/mp4" {
		t.Errorf("Expected content type 'video/mp4', got '%s'", content.ContentType)
	}
	if string(content.Data) != string(videoContent) {
		t.Errorf("Video data mismatch")
	}
}

func TestDownloadTranscript(t *testing.T) {
	transcriptContent := []byte("Speaker 1: Hello\nSpeaker 2: Hi there")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/recordings/rec-123":
			w.Header().Set("Content-Type", "application/json")
			recording := Recording{
				ID: "rec-123",
				TemporaryDirectDownloadLinks: &TemporaryDownloadLinks{
					TranscriptDownloadLink: "http://" + r.Host + "/download/transcript.txt",
				},
			}
			_ = json.NewEncoder(w).Encode(recording)
		case "/download/transcript.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(transcriptContent); err != nil {
				t.Logf("Failed to write response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	client, _ := webexsdk.NewClient("test-token", config)
	client.BaseURL = baseURL
	rc := New(client, nil)

	content, err := rc.DownloadTranscript("rec-123")
	if err != nil {
		t.Fatalf("DownloadTranscript failed: %v", err)
	}

	if content.ContentType != "text/plain" {
		t.Errorf("Expected content type 'text/plain', got '%s'", content.ContentType)
	}
	if string(content.Data) != string(transcriptContent) {
		t.Errorf("Transcript data mismatch")
	}
}

func TestDownloadRecording_NoLinks(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{ID: "rec-no-links"})
	})
	defer server.Close()

	_, err := rc.DownloadRecording("rec-no-links")
	if err == nil {
		t.Error("Expected error when no download links available")
	}
}

func TestDownloadTranscript_NoLinks(t *testing.T) {
	rc, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Recording{ID: "rec-no-links"})
	})
	defer server.Close()

	_, err := rc.DownloadTranscript("rec-no-links")
	if err == nil {
		t.Error("Expected error when no download links available")
	}
}

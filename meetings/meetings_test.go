/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package meetings

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

func TestCreate(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings" {
			t.Errorf("Expected path '/meetings', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		var meeting Meeting
		if err := json.NewDecoder(r.Body).Decode(&meeting); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if meeting.Title != "Test Meeting" {
			t.Errorf("Expected title 'Test Meeting', got '%s'", meeting.Title)
		}
		if meeting.Start != "2026-02-01T10:00:00Z" {
			t.Errorf("Expected start '2026-02-01T10:00:00Z', got '%s'", meeting.Start)
		}
		if meeting.End != "2026-02-01T11:00:00Z" {
			t.Errorf("Expected end '2026-02-01T11:00:00Z', got '%s'", meeting.End)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		createdAt := time.Now()
		responseMeeting := Meeting{
			ID:            "test-meeting-id",
			Title:         meeting.Title,
			Start:         meeting.Start,
			End:           meeting.End,
			Timezone:      "UTC",
			MeetingType:   "meetingSeries",
			State:         "active",
			HostEmail:     "host@example.com",
			WebLink:       "https://meet.webex.com/test",
			MeetingNumber: "1234567890",
			Created:       &createdAt,
		}

		_ = json.NewEncoder(w).Encode(responseMeeting)
	})
	defer server.Close()

	meeting := &Meeting{
		Title: "Test Meeting",
		Start: "2026-02-01T10:00:00Z",
		End:   "2026-02-01T11:00:00Z",
	}

	result, err := meetingsPlugin.Create(meeting)
	if err != nil {
		t.Fatalf("Failed to create meeting: %v", err)
	}

	if result.ID != "test-meeting-id" {
		t.Errorf("Expected ID 'test-meeting-id', got '%s'", result.ID)
	}
	if result.Title != "Test Meeting" {
		t.Errorf("Expected title 'Test Meeting', got '%s'", result.Title)
	}
	if result.MeetingNumber != "1234567890" {
		t.Errorf("Expected meetingNumber '1234567890', got '%s'", result.MeetingNumber)
	}
	if result.WebLink != "https://meet.webex.com/test" {
		t.Errorf("Expected webLink 'https://meet.webex.com/test', got '%s'", result.WebLink)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestCreateValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	// Missing title
	_, err := meetingsPlugin.Create(&Meeting{Start: "2026-02-01T10:00:00Z", End: "2026-02-01T11:00:00Z"})
	if err == nil {
		t.Error("Expected error for missing title")
	}

	// Missing start
	_, err = meetingsPlugin.Create(&Meeting{Title: "Test", End: "2026-02-01T11:00:00Z"})
	if err == nil {
		t.Error("Expected error for missing start")
	}

	// Missing end
	_, err = meetingsPlugin.Create(&Meeting{Title: "Test", Start: "2026-02-01T10:00:00Z"})
	if err == nil {
		t.Error("Expected error for missing end")
	}
}

func TestGet(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings/test-meeting-id" {
			t.Errorf("Expected path '/meetings/test-meeting-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		createdAt := time.Now()
		meeting := Meeting{
			ID:            "test-meeting-id",
			Title:         "Test Meeting",
			Start:         "2026-02-01T10:00:00Z",
			End:           "2026-02-01T11:00:00Z",
			MeetingType:   "meetingSeries",
			State:         "active",
			HostEmail:     "host@example.com",
			MeetingNumber: "1234567890",
			Created:       &createdAt,
		}

		_ = json.NewEncoder(w).Encode(meeting)
	})
	defer server.Close()

	meeting, err := meetingsPlugin.Get("test-meeting-id")
	if err != nil {
		t.Fatalf("Failed to get meeting: %v", err)
	}

	if meeting.ID != "test-meeting-id" {
		t.Errorf("Expected ID 'test-meeting-id', got '%s'", meeting.ID)
	}
	if meeting.Title != "Test Meeting" {
		t.Errorf("Expected title 'Test Meeting', got '%s'", meeting.Title)
	}
	if meeting.HostEmail != "host@example.com" {
		t.Errorf("Expected hostEmail 'host@example.com', got '%s'", meeting.HostEmail)
	}
}

func TestGetValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := meetingsPlugin.Get("")
	if err == nil {
		t.Error("Expected error for empty meetingID")
	}
}

func TestList(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings" {
			t.Errorf("Expected path '/meetings', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		if r.URL.Query().Get("max") != "10" {
			t.Errorf("Expected max '10', got '%s'", r.URL.Query().Get("max"))
		}
		if r.URL.Query().Get("meetingType") != "meetingSeries" {
			t.Errorf("Expected meetingType 'meetingSeries', got '%s'", r.URL.Query().Get("meetingType"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		createdAt := time.Now()
		meetings := []Meeting{
			{
				ID:            "meeting-1",
				Title:         "Meeting 1",
				Start:         "2026-02-01T10:00:00Z",
				End:           "2026-02-01T11:00:00Z",
				MeetingType:   "meetingSeries",
				State:         "active",
				MeetingNumber: "1111111111",
				Created:       &createdAt,
			},
			{
				ID:            "meeting-2",
				Title:         "Meeting 2",
				Start:         "2026-02-02T14:00:00Z",
				End:           "2026-02-02T15:00:00Z",
				MeetingType:   "meetingSeries",
				State:         "active",
				MeetingNumber: "2222222222",
				Created:       &createdAt,
			},
		}

		response := struct {
			Items []Meeting `json:"items"`
		}{
			Items: meetings,
		}

		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	options := &ListOptions{
		MeetingType: "meetingSeries",
		Max:         10,
	}
	page, err := meetingsPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list meetings: %v", err)
	}

	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	if page.Items[0].ID != "meeting-1" {
		t.Errorf("Expected ID 'meeting-1', got '%s'", page.Items[0].ID)
	}
	if page.Items[0].Title != "Meeting 1" {
		t.Errorf("Expected title 'Meeting 1', got '%s'", page.Items[0].Title)
	}
	if page.Items[1].ID != "meeting-2" {
		t.Errorf("Expected ID 'meeting-2', got '%s'", page.Items[1].ID)
	}
}

func TestUpdate(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings/test-meeting-id" {
			t.Errorf("Expected path '/meetings/test-meeting-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		var meeting Meeting
		if err := json.NewDecoder(r.Body).Decode(&meeting); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if meeting.Title != "Updated Meeting" {
			t.Errorf("Expected title 'Updated Meeting', got '%s'", meeting.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		createdAt := time.Now()
		responseMeeting := Meeting{
			ID:          "test-meeting-id",
			Title:       meeting.Title,
			Agenda:      meeting.Agenda,
			Start:       meeting.Start,
			End:         meeting.End,
			MeetingType: "meetingSeries",
			State:       "active",
			Created:     &createdAt,
		}

		_ = json.NewEncoder(w).Encode(responseMeeting)
	})
	defer server.Close()

	meeting := &Meeting{
		Title:  "Updated Meeting",
		Agenda: "Updated agenda",
		Start:  "2026-02-01T10:00:00Z",
		End:    "2026-02-01T11:00:00Z",
	}

	result, err := meetingsPlugin.Update("test-meeting-id", meeting)
	if err != nil {
		t.Fatalf("Failed to update meeting: %v", err)
	}

	if result.ID != "test-meeting-id" {
		t.Errorf("Expected ID 'test-meeting-id', got '%s'", result.ID)
	}
	if result.Title != "Updated Meeting" {
		t.Errorf("Expected title 'Updated Meeting', got '%s'", result.Title)
	}
}

func TestUpdateValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := meetingsPlugin.Update("", &Meeting{Title: "Test"})
	if err == nil {
		t.Error("Expected error for empty meetingID")
	}

	_, err = meetingsPlugin.Update("test-id", &Meeting{})
	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestDelete(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings/test-meeting-id" {
			t.Errorf("Expected path '/meetings/test-meeting-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := meetingsPlugin.Delete("test-meeting-id")
	if err != nil {
		t.Fatalf("Failed to delete meeting: %v", err)
	}
}

func TestDeleteValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	err := meetingsPlugin.Delete("")
	if err == nil {
		t.Error("Expected error for empty meetingID")
	}
}

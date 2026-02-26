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

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
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
				ID:               "meeting-1",
				MeetingSeriesID:  "series-1",
				Title:            "Meeting 1",
				Start:            "2026-02-01T10:00:00Z",
				End:              "2026-02-01T11:00:00Z",
				MeetingType:      "meetingSeries",
				State:            "active",
				ScheduledType:    "meeting",
				MeetingNumber:    "1111111111",
				SiteURL:          "example.webex.com",
				HasRecording:     false,
				HasTranscription: false,
				Created:          &createdAt,
			},
			{
				ID:               "meeting-2",
				MeetingSeriesID:  "series-2",
				Title:            "Meeting 2",
				Start:            "2026-02-02T14:00:00Z",
				End:              "2026-02-02T15:00:00Z",
				MeetingType:      "meetingSeries",
				State:            "active",
				ScheduledType:    "meeting",
				MeetingNumber:    "2222222222",
				SiteURL:          "example.webex.com",
				HasRecording:     true,
				HasTranscription: true,
				Created:          &createdAt,
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
	if page.Items[0].MeetingSeriesID != "series-1" {
		t.Errorf("Expected meetingSeriesId 'series-1', got '%s'", page.Items[0].MeetingSeriesID)
	}
	if page.Items[0].ScheduledType != "meeting" {
		t.Errorf("Expected scheduledType 'meeting', got '%s'", page.Items[0].ScheduledType)
	}
	if page.Items[1].ID != "meeting-2" {
		t.Errorf("Expected ID 'meeting-2', got '%s'", page.Items[1].ID)
	}
	if !page.Items[1].HasRecording {
		t.Error("Expected meeting-2 hasRecording to be true")
	}
	if !page.Items[1].HasTranscription {
		t.Error("Expected meeting-2 hasTranscription to be true")
	}
}

func TestListStateRequiresMeetingType(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	// state without meetingType should return an error
	_, err := meetingsPlugin.List(&ListOptions{
		State: "ended",
	})
	if err == nil {
		t.Error("Expected error when state is set without meetingType")
	}

	// state with meetingType should not error (at the validation level)
	// We don't check the API call here since newTestClient doesn't serve proper responses for this
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

func TestPatch(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetings/test-meeting-id" {
			t.Errorf("Expected path '/meetings/test-meeting-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("Expected method PATCH, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if body["title"] != "Patched Title" {
			t.Errorf("Expected title 'Patched Title', got '%v'", body["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Meeting{
			ID:    "test-meeting-id",
			Title: "Patched Title",
		})
	})
	defer server.Close()

	patch := map[string]interface{}{"title": "Patched Title"}
	result, err := meetingsPlugin.Patch("test-meeting-id", patch)
	if err != nil {
		t.Fatalf("Failed to patch meeting: %v", err)
	}

	if result.Title != "Patched Title" {
		t.Errorf("Expected title 'Patched Title', got '%s'", result.Title)
	}
}

func TestPatchValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := meetingsPlugin.Patch("", map[string]interface{}{"title": "Test"})
	if err == nil {
		t.Error("Expected error for empty meetingID")
	}
}

func TestListParticipants(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingParticipants" {
			t.Errorf("Expected path '/meetingParticipants', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Query().Get("meetingId") != "meeting-instance-123" {
			t.Errorf("Expected meetingId 'meeting-instance-123', got '%s'", r.URL.Query().Get("meetingId"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		participants := []Participant{
			{
				ID:          "part-1",
				Email:       "host@example.com",
				DisplayName: "Host User",
				Host:        true,
				State:       "end",
				JoinedTime:  "2026-02-01T10:00:00Z",
				LeftTime:    "2026-02-01T11:00:00Z",
			},
			{
				ID:          "part-2",
				Email:       "guest@example.com",
				DisplayName: "Guest User",
				Host:        false,
				CoHost:      false,
				State:       "end",
				JoinedTime:  "2026-02-01T10:05:00Z",
				LeftTime:    "2026-02-01T10:55:00Z",
				Devices: []ParticipantDevice{
					{
						DeviceType: "tp",
						AudioType:  "voip",
					},
				},
			},
		}

		response := struct {
			Items []Participant `json:"items"`
		}{
			Items: participants,
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	page, err := meetingsPlugin.ListParticipants(&ParticipantListOptions{
		MeetingID: "meeting-instance-123",
	})
	if err != nil {
		t.Fatalf("Failed to list participants: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(page.Items))
	}

	if page.Items[0].ID != "part-1" {
		t.Errorf("Expected ID 'part-1', got '%s'", page.Items[0].ID)
	}
	if !page.Items[0].Host {
		t.Error("Expected participant 1 to be host")
	}
	if page.Items[1].Email != "guest@example.com" {
		t.Errorf("Expected email 'guest@example.com', got '%s'", page.Items[1].Email)
	}
	if len(page.Items[1].Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(page.Items[1].Devices))
	}
	if page.Items[1].Devices[0].DeviceType != "tp" {
		t.Errorf("Expected deviceType 'tp', got '%s'", page.Items[1].Devices[0].DeviceType)
	}
}

func TestListParticipantsValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := meetingsPlugin.ListParticipants(nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	_, err = meetingsPlugin.ListParticipants(&ParticipantListOptions{})
	if err == nil {
		t.Error("Expected error for empty meetingId")
	}
}

func TestGetParticipant(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/meetingParticipants/part-1" {
			t.Errorf("Expected path '/meetingParticipants/part-1', got '%s'", r.URL.Path)
		}
		if r.URL.Query().Get("meetingId") != "meeting-123" {
			t.Errorf("Expected meetingId 'meeting-123', got '%s'", r.URL.Query().Get("meetingId"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Participant{
			ID:          "part-1",
			Email:       "user@example.com",
			DisplayName: "User Name",
			Host:        true,
			State:       "end",
		})
	})
	defer server.Close()

	participant, err := meetingsPlugin.GetParticipant("part-1", "meeting-123")
	if err != nil {
		t.Fatalf("Failed to get participant: %v", err)
	}

	if participant.ID != "part-1" {
		t.Errorf("Expected ID 'part-1', got '%s'", participant.ID)
	}
	if participant.Email != "user@example.com" {
		t.Errorf("Expected email 'user@example.com', got '%s'", participant.Email)
	}
	if !participant.Host {
		t.Error("Expected participant to be host")
	}
}

func TestGetParticipantValidation(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("Request should not have been sent")
	})
	defer server.Close()

	_, err := meetingsPlugin.GetParticipant("", "meeting-123")
	if err == nil {
		t.Error("Expected error for empty participantID")
	}

	_, err = meetingsPlugin.GetParticipant("part-1", "")
	if err == nil {
		t.Error("Expected error for empty meetingID")
	}
}

func TestMeetingTelephonyDeserialization(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "meeting-with-telephony",
			"title": "Telephony Meeting",
			"start": "2026-02-01T10:00:00Z",
			"end": "2026-02-01T11:00:00Z",
			"telephony": {
				"accessCode": "1234567890",
				"callInNumbers": [
					{
						"label": "US Toll Free",
						"callInNumber": "+1-800-555-1234",
						"tollType": "tollFree"
					},
					{
						"label": "US Toll",
						"callInNumber": "+1-555-555-5678",
						"tollType": "toll"
					}
				],
				"links": [
					{
						"globalCallinNumbers": "https://example.com/call",
						"telephonyTopic": "Meeting Telephony"
					}
				]
			},
			"audioConnectionOptions": {
				"audioConnectionType": "webexAudio",
				"enabledTollFreeCallIn": true,
				"enabledGlobalCallIn": true,
				"entryAndExitTone": "beep",
				"muteAttendeeUponEntry": true
			}
		}`))
	})
	defer server.Close()

	meeting, err := meetingsPlugin.Get("meeting-with-telephony")
	if err != nil {
		t.Fatalf("Failed to get meeting: %v", err)
	}

	if meeting.Telephony == nil {
		t.Fatal("Expected telephony to be non-nil")
	}
	if meeting.Telephony.AccessCode != "1234567890" {
		t.Errorf("Expected accessCode '1234567890', got '%s'", meeting.Telephony.AccessCode)
	}
	if len(meeting.Telephony.CallInNumbers) != 2 {
		t.Fatalf("Expected 2 call-in numbers, got %d", len(meeting.Telephony.CallInNumbers))
	}
	if meeting.Telephony.CallInNumbers[0].TollType != "tollFree" {
		t.Errorf("Expected tollType 'tollFree', got '%s'", meeting.Telephony.CallInNumbers[0].TollType)
	}
	if meeting.Telephony.Links == nil {
		t.Fatal("Expected telephony links to be non-nil")
	}
	if len(*meeting.Telephony.Links) == 0 {
		t.Fatal("Expected telephony links to have at least one item")
	}
	if (*meeting.Telephony.Links)[0].GlobalCallinNumbers != "https://example.com/call" {
		t.Errorf("Expected globalCallinNumbers link")
	}

	if meeting.AudioConnectionOptions == nil {
		t.Fatal("Expected audioConnectionOptions to be non-nil")
	}
	if meeting.AudioConnectionOptions.AudioConnectionType != "webexAudio" {
		t.Errorf("Expected audioConnectionType 'webexAudio', got '%s'", meeting.AudioConnectionOptions.AudioConnectionType)
	}
	if !meeting.AudioConnectionOptions.EnabledTollFreeCallIn {
		t.Error("Expected enabledTollFreeCallIn to be true")
	}
	if !meeting.AudioConnectionOptions.MuteAttendeeUponEntry {
		t.Error("Expected muteAttendeeUponEntry to be true")
	}
}

func TestListWithNewFilters(t *testing.T) {
	meetingsPlugin, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("siteUrl") != "cisco.webex.com" {
			t.Errorf("Expected siteUrl 'cisco.webex.com', got '%s'", r.URL.Query().Get("siteUrl"))
		}
		if r.URL.Query().Get("integrationTag") != "my-app" {
			t.Errorf("Expected integrationTag 'my-app', got '%s'", r.URL.Query().Get("integrationTag"))
		}
		if r.URL.Query().Get("current") != "true" {
			t.Errorf("Expected current 'true', got '%s'", r.URL.Query().Get("current"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Items []Meeting `json:"items"`
		}{Items: []Meeting{}})
	})
	defer server.Close()

	_, err := meetingsPlugin.List(&ListOptions{
		MeetingType:    "meeting",
		SiteURL:        "cisco.webex.com",
		IntegrationTag: "my-app",
		Current:        true,
	})
	if err != nil {
		t.Fatalf("Failed to list with new filters: %v", err)
	}
}

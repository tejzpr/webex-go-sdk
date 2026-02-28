/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestNew(t *testing.T) {
	core, _ := webexsdk.NewClient("test-token", nil)

	t.Run("with nil config uses defaults", func(t *testing.T) {
		client := New(core, nil)
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
		if client.config.BaseURL != "https://webexapis.com/v1" {
			t.Errorf("Expected default base URL, got %q", client.config.BaseURL)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{BaseURL: "https://custom.api.com/v1"}
		client := New(core, cfg)
		if client.config.BaseURL != "https://custom.api.com/v1" {
			t.Errorf("Expected custom base URL, got %q", client.config.BaseURL)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.BaseURL != "https://webexapis.com/v1" {
		t.Errorf("Expected default base URL, got %q", cfg.BaseURL)
	}
}

func TestSubClientAccessors(t *testing.T) {
	core, _ := webexsdk.NewClient("test-token", nil)
	client := New(core, nil)

	t.Run("CallHistory returns non-nil", func(t *testing.T) {
		if client.CallHistory() == nil {
			t.Error("Expected non-nil CallHistory client")
		}
	})

	t.Run("CallHistory returns same instance", func(t *testing.T) {
		ch1 := client.CallHistory()
		ch2 := client.CallHistory()
		if ch1 != ch2 {
			t.Error("Expected same CallHistory instance")
		}
	})

	t.Run("CallSettings returns non-nil", func(t *testing.T) {
		if client.CallSettings() == nil {
			t.Error("Expected non-nil CallSettings client")
		}
	})

	t.Run("Voicemail returns non-nil", func(t *testing.T) {
		if client.Voicemail() == nil {
			t.Error("Expected non-nil Voicemail client")
		}
	})

	t.Run("Contacts returns non-nil", func(t *testing.T) {
		if client.Contacts() == nil {
			t.Error("Expected non-nil Contacts client")
		}
	})
}

// ---- Call History Tests ----

func TestCallHistoryGetCallHistoryData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/telephony/callHistory" {
				t.Errorf("Unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("days") != "7" {
				t.Errorf("Expected days=7, got %s", r.URL.Query().Get("days"))
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"userSessions": []map[string]interface{}{
					{"id": "session-1", "sessionId": "s1", "disposition": "Answered", "direction": "outbound"},
					{"id": "session-2", "sessionId": "s2", "disposition": "MISSED", "direction": "inbound"},
				},
			})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		ch := newCallHistoryClient(core, &Config{BaseURL: server.URL})

		resp, err := ch.GetCallHistoryData(7, 10, SortDESC, SortByEndTime)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
		if len(resp.Data.UserSessions) != 2 {
			t.Errorf("Expected 2 sessions, got %d", len(resp.Data.UserSessions))
		}
		if resp.Message != "SUCCESS" {
			t.Errorf("Expected SUCCESS, got %q", resp.Message)
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized"})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("bad-token", nil)
		ch := newCallHistoryClient(core, &Config{BaseURL: server.URL})

		resp, err := ch.GetCallHistoryData(7, 10, SortDESC, SortByEndTime)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 401 {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
		if resp.Message != "FAILURE" {
			t.Errorf("Expected FAILURE, got %q", resp.Message)
		}
	})
}

func TestCallHistoryUpdateMissedCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	ch := newCallHistoryClient(core, &Config{BaseURL: server.URL})

	resp, err := ch.UpdateMissedCalls([]EndTimeSessionID{
		{EndTime: "2025-01-01T00:00:00Z", SessionID: "s1"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestCallHistoryDeleteRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	ch := newCallHistoryClient(core, &Config{BaseURL: server.URL})

	resp, err := ch.DeleteCallHistoryRecords([]EndTimeSessionID{
		{EndTime: "2025-01-01T00:00:00Z", SessionID: "s1"},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

// ---- Call Settings Tests ----

func TestCallSettingsGetCallWaiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/me/features/callWaiting" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":           true,
			"ringSplashEnabled": false,
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.GetCallWaitingSetting()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestCallSettingsGetDND(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/me/features/doNotDisturb" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.GetDoNotDisturbSetting()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsSetDND(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		var body ToggleSetting
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if !body.Enabled {
			t.Error("Expected enabled=true")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.SetDoNotDisturbSetting(true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsGetCallForward(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/me/features/callForwarding" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"callForwarding": map[string]interface{}{
				"always": map[string]interface{}{"enabled": false},
			},
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.GetCallForwardSetting()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsSetCallForward(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	setting := CallForwardSetting{
		CallForwarding: CallForwardingConfig{
			Always: CallForwardAlwaysSetting{Enabled: true, Destination: "+15551234567"},
		},
	}

	resp, err := cs.SetCallForwardSetting(setting)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsGetVoicemail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/me/features/voicemail" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.GetVoicemailSetting()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsSetVoicemail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	setting := VoicemailSettingConfig{Enabled: true}
	resp, err := cs.SetVoicemailSetting(setting)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestCallSettingsGetCallForwardAlways(t *testing.T) {
	t.Run("without directory number", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/people/me/features/callForwarding/always" {
				t.Errorf("Unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("directoryNumber") != "" {
				t.Error("Did not expect directoryNumber query param")
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": false})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

		resp, err := cs.GetCallForwardAlwaysSetting("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("with directory number", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("directoryNumber") != "1234" {
				t.Errorf("Expected directoryNumber=1234, got %s", r.URL.Query().Get("directoryNumber"))
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"enabled": true})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

		resp, err := cs.GetCallForwardAlwaysSetting("1234")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestCallSettingsErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Forbidden"})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cs := newCallSettingsClient(core, &Config{BaseURL: server.URL})

	resp, err := cs.GetCallWaitingSetting()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
	if resp.Message != "FAILURE" {
		t.Errorf("Expected FAILURE, got %q", resp.Message)
	}
}

// ---- Voicemail Tests ----

func TestVoicemailGetList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/telephony/voiceMessages" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("offset") != "0" {
			t.Errorf("Expected offset=0, got %s", r.URL.Query().Get("offset"))
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{"messageId": "msg-1", "duration": "30"},
				{"messageId": "msg-2", "duration": "45"},
			},
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.GetVoicemailList(0, 10, SortDESC)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if len(resp.Data.VoicemailList) != 2 {
		t.Errorf("Expected 2 voicemails, got %d", len(resp.Data.VoicemailList))
	}
}

func TestVoicemailGetContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/telephony/voiceMessages/msg-1/content" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type":    "audio/wav",
			"content": "base64-encoded-audio",
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.GetVoicemailContent("msg-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if resp.Data.VoicemailContent == nil {
		t.Fatal("Expected non-nil voicemail content")
	}
}

func TestVoicemailGetSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"newMessages":       3,
			"oldMessages":       10,
			"newUrgentMessages": 1,
			"oldUrgentMessages": 0,
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.GetVoicemailSummary()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Data.VoicemailSummary == nil {
		t.Fatal("Expected non-nil summary")
	}
	if resp.Data.VoicemailSummary.NewMessages != 3 {
		t.Errorf("Expected 3 new messages, got %d", resp.Data.VoicemailSummary.NewMessages)
	}
	if resp.Data.VoicemailSummary.OldMessages != 10 {
		t.Errorf("Expected 10 old messages, got %d", resp.Data.VoicemailSummary.OldMessages)
	}
}

func TestVoicemailMarkAsRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/telephony/voiceMessages/msg-1/markAsRead" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.MarkAsRead("msg-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestVoicemailMarkAsUnread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/telephony/voiceMessages/msg-1/markAsUnread" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.MarkAsUnread("msg-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestVoicemailDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/telephony/voiceMessages/msg-1" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.Delete("msg-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestVoicemailGetTranscript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/telephony/voiceMessages/msg-1/transcript" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"transcript": "Hello, this is a test voicemail.",
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.GetTranscript("msg-1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Data.VoicemailTranscript == nil {
		t.Fatal("Expected non-nil transcript")
	}
	if *resp.Data.VoicemailTranscript != "Hello, this is a test voicemail." {
		t.Errorf("Unexpected transcript: %q", *resp.Data.VoicemailTranscript)
	}
}

func TestVoicemailErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Service unavailable"})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	vm := newVoicemailClient(core, &Config{BaseURL: server.URL})

	resp, err := vm.GetVoicemailSummary()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Errorf("Expected 503, got %d", resp.StatusCode)
	}
	if resp.Message != "FAILURE" {
		t.Errorf("Expected FAILURE, got %q", resp.Message)
	}
}

// ---- Contacts Tests ----

func TestContactsGetContacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/me/contacts" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"contacts": []map[string]interface{}{
				{"contactId": "c1", "displayName": "Alice", "contactType": "CLOUD", "encryptionKeyUrl": "kms://key1", "groups": []string{}, "resolved": true},
			},
			"groups": []map[string]interface{}{
				{"groupId": "g1", "displayName": "Team", "encryptionKeyUrl": "kms://key2", "groupType": "NORMAL"},
			},
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	resp, err := cc.GetContacts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
	if len(resp.Data.Contacts) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(resp.Data.Contacts))
	}
	if len(resp.Data.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(resp.Data.Groups))
	}
}

func TestContactsCreateContactGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/people/me/contactGroups" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		var body struct {
			DisplayName string `json:"displayName"`
			GroupType   string `json:"groupType"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.DisplayName != "My Group" {
			t.Errorf("Expected displayName 'My Group', got %q", body.DisplayName)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"group": map[string]interface{}{
				"groupId":          "g-new",
				"displayName":      "My Group",
				"encryptionKeyUrl": "",
				"groupType":        "NORMAL",
			},
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	resp, err := cc.CreateContactGroup("My Group", "", GroupTypeNormal)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestContactsDeleteContactGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/people/me/contactGroups/g1" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	resp, err := cc.DeleteContactGroup("g1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestContactsCreateContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"contact": map[string]interface{}{
				"contactId":        "c-new",
				"displayName":      "Bob",
				"contactType":      "CUSTOM",
				"encryptionKeyUrl": "",
				"groups":           []string{},
				"resolved":         true,
			},
		})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	contact := Contact{
		ContactID:        "c-new",
		DisplayName:      "Bob",
		ContactType:      ContactTypeCustom,
		EncryptionKeyURL: "",
		Groups:           []string{},
		Resolved:         true,
	}

	resp, err := cc.CreateContact(contact)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}
}

func TestContactsDeleteContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/people/me/contacts/c1" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	resp, err := cc.DeleteContact("c1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.Message != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %q", resp.Message)
	}
}

func TestContactsErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not found"})
	}))
	defer server.Close()

	core, _ := webexsdk.NewClient("test-token", nil)
	cc := newContactsClient(core, &Config{BaseURL: server.URL})

	resp, err := cc.GetContacts()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
	if resp.Message != "FAILURE" {
		t.Errorf("Expected FAILURE, got %q", resp.Message)
	}
}

// ---- Type Constants Tests ----

func TestTypeConstants(t *testing.T) {
	if SortASC != "ASC" {
		t.Errorf("Expected ASC, got %q", SortASC)
	}
	if SortDESC != "DESC" {
		t.Errorf("Expected DESC, got %q", SortDESC)
	}
	if SortByEndTime != "endTime" {
		t.Errorf("Expected endTime, got %q", SortByEndTime)
	}
	if SortByStartTime != "startTime" {
		t.Errorf("Expected startTime, got %q", SortByStartTime)
	}
	if CallDirectionInbound != "inbound" {
		t.Errorf("Expected inbound, got %q", CallDirectionInbound)
	}
	if CallDirectionOutbound != "outbound" {
		t.Errorf("Expected outbound, got %q", CallDirectionOutbound)
	}
	if ContactTypeCustom != "CUSTOM" {
		t.Errorf("Expected CUSTOM, got %q", ContactTypeCustom)
	}
	if ContactTypeCloud != "CLOUD" {
		t.Errorf("Expected CLOUD, got %q", ContactTypeCloud)
	}
	if GroupTypeNormal != "NORMAL" {
		t.Errorf("Expected NORMAL, got %q", GroupTypeNormal)
	}
	if GroupTypeExternal != "EXTERNAL" {
		t.Errorf("Expected EXTERNAL, got %q", GroupTypeExternal)
	}
	if DispositionAnswered != "Answered" {
		t.Errorf("Expected Answered, got %q", DispositionAnswered)
	}
	if SessionTypeSpark != "SPARK" {
		t.Errorf("Expected SPARK, got %q", SessionTypeSpark)
	}
}

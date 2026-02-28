//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func functionalClient(t *testing.T) *webexsdk.Client {
	t.Helper()
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
	return client
}

// TestFunctionalCallHistoryList tests retrieving call history data
// This is a read-only operation that lists recent call history
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallHistoryList -v ./calling/
func TestFunctionalCallHistoryList(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	historyClient := callingClient.CallHistory()

	// Fetch last 7 days of call history, up to 20 records
	resp, err := historyClient.GetCallHistoryData(7, 20, SortDESC, SortByEndTime)
	if err != nil {
		t.Fatalf("GetCallHistoryData failed: %v", err)
	}

	t.Logf("Call history response: status=%d message=%s", resp.StatusCode, resp.Message)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Found %d call session(s)", len(resp.Data.UserSessions))
		for i, session := range resp.Data.UserSessions {
			if i >= 5 {
				t.Logf("  ... and %d more", len(resp.Data.UserSessions)-5)
				break
			}
			t.Logf("  Session: ID=%s Direction=%s Disposition=%s Duration=%ds Other=%s",
				session.ID, session.Direction, session.Disposition,
				session.DurationSeconds, session.Other.Name)
		}
	} else {
		// Calling features may not be enabled for all accounts
		t.Logf("Call history not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Call history API not available for this account")
	}
}

// TestFunctionalCallSettingsGetDND tests retrieving the Do Not Disturb setting
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallSettingsGetDND -v ./calling/
func TestFunctionalCallSettingsGetDND(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	settingsClient := callingClient.CallSettings()

	resp, err := settingsClient.GetDoNotDisturbSetting()
	if err != nil {
		t.Fatalf("GetDoNotDisturbSetting failed: %v", err)
	}

	t.Logf("DND setting response: status=%d message=%s", resp.StatusCode, resp.Message)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("DND setting data: %v", resp.Data.CallSetting)
	} else {
		t.Logf("DND setting not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Call settings API not available for this account")
	}
}

// TestFunctionalCallSettingsToggleDND tests toggling DND on and restoring it
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallSettingsToggleDND -v ./calling/
func TestFunctionalCallSettingsToggleDND(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	settingsClient := callingClient.CallSettings()

	// Get current DND setting
	currentResp, err := settingsClient.GetDoNotDisturbSetting()
	if err != nil {
		t.Fatalf("GetDoNotDisturbSetting failed: %v", err)
	}
	if currentResp.StatusCode < 200 || currentResp.StatusCode >= 300 {
		t.Skipf("DND setting not available (status %d)", currentResp.StatusCode)
	}

	// Enable DND
	enableResp, err := settingsClient.SetDoNotDisturbSetting(true)
	if err != nil {
		t.Fatalf("SetDoNotDisturbSetting(true) failed: %v", err)
	}
	t.Logf("Enable DND: status=%d message=%s", enableResp.StatusCode, enableResp.Message)

	// Verify it's enabled
	verifyResp, err := settingsClient.GetDoNotDisturbSetting()
	if err != nil {
		t.Fatalf("Verify DND enabled failed: %v", err)
	}
	t.Logf("After enable: %v", verifyResp.Data.CallSetting)

	// Restore — disable DND
	disableResp, err := settingsClient.SetDoNotDisturbSetting(false)
	if err != nil {
		t.Fatalf("SetDoNotDisturbSetting(false) failed: %v", err)
	}
	t.Logf("Disable DND: status=%d message=%s", disableResp.StatusCode, disableResp.Message)
}

// TestFunctionalCallSettingsGetCallWaiting tests retrieving call waiting setting
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallSettingsGetCallWaiting -v ./calling/
func TestFunctionalCallSettingsGetCallWaiting(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	settingsClient := callingClient.CallSettings()

	resp, err := settingsClient.GetCallWaitingSetting()
	if err != nil {
		t.Fatalf("GetCallWaitingSetting failed: %v", err)
	}

	t.Logf("Call waiting setting: status=%d message=%s", resp.StatusCode, resp.Message)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Call waiting data: %v", resp.Data.CallSetting)
	} else {
		t.Logf("Not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Call waiting setting not available for this account")
	}
}

// TestFunctionalCallSettingsGetCallForward tests retrieving call forwarding settings
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallSettingsGetCallForward -v ./calling/
func TestFunctionalCallSettingsGetCallForward(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	settingsClient := callingClient.CallSettings()

	resp, err := settingsClient.GetCallForwardSetting()
	if err != nil {
		t.Fatalf("GetCallForwardSetting failed: %v", err)
	}

	t.Logf("Call forward setting: status=%d message=%s", resp.StatusCode, resp.Message)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Call forward data: %v", resp.Data.CallSetting)
	} else {
		t.Logf("Not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Call forwarding setting not available for this account")
	}
}

// TestFunctionalCallSettingsGetVoicemail tests retrieving voicemail settings
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalCallSettingsGetVoicemail -v ./calling/
func TestFunctionalCallSettingsGetVoicemail(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	settingsClient := callingClient.CallSettings()

	resp, err := settingsClient.GetVoicemailSetting()
	if err != nil {
		t.Fatalf("GetVoicemailSetting failed: %v", err)
	}

	t.Logf("Voicemail setting: status=%d message=%s", resp.StatusCode, resp.Message)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Voicemail data: %v", resp.Data.CallSetting)
	} else {
		t.Logf("Not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Voicemail setting not available for this account")
	}
}

// TestFunctionalVoicemailList tests retrieving voicemail list
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalVoicemailList -v ./calling/
func TestFunctionalVoicemailList(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	vmClient := callingClient.Voicemail()

	resp, err := vmClient.GetVoicemailList(0, 10, SortDESC)
	if err != nil {
		t.Fatalf("GetVoicemailList failed: %v", err)
	}

	t.Logf("Voicemail list response: status=%d message=%s", resp.StatusCode, resp.Message)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Found %d voicemail(s)", len(resp.Data.VoicemailList))
		for i, vm := range resp.Data.VoicemailList {
			if i >= 5 {
				t.Logf("  ... and %d more", len(resp.Data.VoicemailList)-5)
				break
			}
			t.Logf("  Voicemail: Duration=%s From=%s Read=%v",
				vm.Duration, vm.CallingPartyInfo.Name, vm.Read)
		}
	} else {
		t.Logf("Voicemail list not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Voicemail API not available for this account")
	}
}

// TestFunctionalContactsList tests retrieving contacts
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContactsList -v ./calling/
func TestFunctionalContactsList(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	contactsClient := callingClient.Contacts()

	resp, err := contactsClient.GetContacts()
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}

	t.Logf("Contacts response: status=%d message=%s", resp.StatusCode, resp.Message)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Logf("Contacts data: groups=%d contacts=%d",
			len(resp.Data.Groups), len(resp.Data.Contacts))
		for i, cg := range resp.Data.Groups {
			if i >= 3 {
				break
			}
			t.Logf("  Group: ID=%s Name=%s Type=%s", cg.GroupID, cg.DisplayName, cg.GroupType)
		}
	} else {
		t.Logf("Contacts not available (status %d): %s", resp.StatusCode, resp.Data.Error)
		t.Skip("Contacts API not available for this account")
	}
}

// TestFunctionalContactsGroupCRUD tests creating and deleting a contact group
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalContactsGroupCRUD -v ./calling/
func TestFunctionalContactsGroupCRUD(t *testing.T) {
	client := functionalClient(t)
	callingClient := New(client, nil)
	contactsClient := callingClient.Contacts()

	// Check if contacts API is available
	checkResp, err := contactsClient.GetContacts()
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}
	if checkResp.StatusCode < 200 || checkResp.StatusCode >= 300 {
		t.Skipf("Contacts API not available (status %d)", checkResp.StatusCode)
	}

	// Create a contact group
	groupName := "SDK Test Group"
	createResp, err := contactsClient.CreateContactGroup(groupName, "", GroupTypeNormal)
	if err != nil {
		t.Fatalf("CreateContactGroup failed: %v", err)
	}

	t.Logf("Create group response: status=%d message=%s", createResp.StatusCode, createResp.Message)

	if createResp.StatusCode >= 200 && createResp.StatusCode < 300 {
		// Find the created group via listing
		listResp, err := contactsClient.GetContacts()
		if err != nil {
			t.Fatalf("GetContacts after create failed: %v", err)
		}

		var createdGroupID string
		for _, cg := range listResp.Data.Groups {
			if cg.DisplayName == groupName {
				createdGroupID = cg.GroupID
				t.Logf("Found created group: ID=%s", createdGroupID)
				break
			}
		}

		if createdGroupID != "" {
			// Delete the group
			delResp, err := contactsClient.DeleteContactGroup(createdGroupID)
			if err != nil {
				t.Fatalf("DeleteContactGroup failed: %v", err)
			}
			t.Logf("Delete group: status=%d message=%s", delResp.StatusCode, delResp.Message)
		} else {
			t.Log("Could not find created group in listing — creation may have been processed differently")
		}
	} else {
		t.Logf("Create group not available: %s", createResp.Data.Error)
		t.Skip("Contact group creation not available for this account")
	}
}

// TestFunctionalNormalizeAddress tests the address normalization utility
// This is a pure unit utility test but included here for completeness
func TestFunctionalNormalizeAddress(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		want    string
	}{
		{"sip:user@example.com", false, "sip:user@example.com"},
		{"sips:user@example.com", false, "sips:user@example.com"},
		{"tel:+14155551234", false, "tel:+14155551234"},
		{"+1 (415) 555-1234", false, "tel:+14155551234"},
		{"4155551234", false, "tel:4155551234"},
		{"", true, ""},
	}

	for _, tt := range tests {
		addr, _, err := NormalizeAddress(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("NormalizeAddress(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && addr != tt.want {
			t.Errorf("NormalizeAddress(%q) = %q, want %q", tt.input, addr, tt.want)
		}
	}
}

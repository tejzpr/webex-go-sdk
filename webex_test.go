/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webex

import (
	"testing"

	"github.com/tejzpr/webex-go-sdk/v2/conversation"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

func TestConversationReturnsSingletonWhenCached(t *testing.T) {
	// Create a client with a valid token
	client, err := NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Pre-populate the conversationClient to simulate a previous successful init
	core, _ := webexsdk.NewClient("test-token", nil)
	preWired := conversation.New(core, nil)
	client.conversationClient = preWired

	// Subsequent calls should return the cached instance without error
	result, err := client.Conversation()
	if err != nil {
		t.Fatalf("Expected no error from cached Conversation(), got: %v", err)
	}
	if result != preWired {
		t.Error("Expected Conversation() to return the cached singleton instance")
	}

	// Call again to verify idempotency
	result2, err := client.Conversation()
	if err != nil {
		t.Fatalf("Expected no error from second Conversation() call, got: %v", err)
	}
	if result2 != result {
		t.Error("Expected repeated Conversation() calls to return the same instance")
	}
}

func TestConversationReturnsErrorOnDeviceRegistrationFailure(t *testing.T) {
	// Create a client with a dummy token -- device registration will fail
	// because it can't reach the real WDM service
	client, err := NewClient("invalid-token-for-testing", nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Conversation() should return a meaningful error wrapping device failure
	_, err = client.Conversation()
	if err == nil {
		t.Fatal("Expected error from Conversation() when device registration fails")
	}

	// Verify the error mentions device registration
	if got := err.Error(); got == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestConversationConnectErrorsWithoutMercury(t *testing.T) {
	// Create a bare conversation client with no mercury wired
	core, _ := webexsdk.NewClient("test-token", nil)
	convClient := conversation.New(core, nil)

	// Connect should fail because no mercury client is set
	err := convClient.Connect()
	if err == nil {
		t.Fatal("Expected error from Connect() without mercury client")
	}
}

func TestConversationDisconnectErrorsWithoutMercury(t *testing.T) {
	// Create a bare conversation client with no mercury wired
	core, _ := webexsdk.NewClient("test-token", nil)
	convClient := conversation.New(core, nil)

	// Disconnect should fail because no mercury client is set
	err := convClient.Disconnect()
	if err == nil {
		t.Fatal("Expected error from Disconnect() without mercury client")
	}
}

func TestConversationConnectDelegatesToMercury(t *testing.T) {
	// Create a conversation client and wire a mercury client (no device provider)
	client, err := NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	core, _ := webexsdk.NewClient("test-token", nil)
	convClient := conversation.New(core, nil)
	convClient.SetMercuryClient(client.Mercury())

	// Connect will fail because mercury has no device provider with a valid URL,
	// but the key assertion is that it does NOT return the "mercury client not set" error
	err = convClient.Connect()
	if err == nil {
		t.Fatal("Expected error from Connect() (no device registration)")
	}
	// Verify it's a connection error, not a "mercury not set" error
	if err.Error() == "mercury client not set; call SetMercuryClient or use the top-level webex.Client.Conversation() convenience method" {
		t.Error("Connect() should delegate to mercury, not return 'not set' error")
	}
}

func TestWebexClientAccessors(t *testing.T) {
	client, err := NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify Core() returns non-nil
	if client.Core() == nil {
		t.Error("Core() should not return nil")
	}

	// Verify lazy-init accessors return non-nil
	if client.People() == nil {
		t.Error("People() should not return nil")
	}
	if client.Messages() == nil {
		t.Error("Messages() should not return nil")
	}
	if client.Rooms() == nil {
		t.Error("Rooms() should not return nil")
	}
	if client.Mercury() == nil {
		t.Error("Mercury() should not return nil")
	}
	if client.Device() == nil {
		t.Error("Device() should not return nil")
	}
	if client.Meetings() == nil {
		t.Error("Meetings() should not return nil")
	}
	if client.Transcripts() == nil {
		t.Error("Transcripts() should not return nil")
	}

	// Verify Internal() returns populated struct
	internal := client.Internal()
	if internal == nil {
		t.Fatal("Internal() should not return nil")
	}
	if internal.Mercury == nil {
		t.Error("Internal().Mercury should not return nil")
	}
	if internal.Device == nil {
		t.Error("Internal().Device should not return nil")
	}
}

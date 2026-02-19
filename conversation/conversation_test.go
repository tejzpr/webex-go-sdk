/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package conversation

import (
	"encoding/json"
	"testing"

	"github.com/WebexCommunity/webex-go-sdk/v2/mercury"
	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestNew(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)

	t.Run("with default config", func(t *testing.T) {
		convClient := New(client, nil)
		if convClient == nil {
			t.Fatal("Expected non-nil conversation client")
		}
		if convClient.handlers == nil {
			t.Error("Expected non-nil handlers map")
		}
		if convClient.encryptionClient == nil {
			t.Error("Expected non-nil encryption client")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{}
		convClient := New(client, cfg)
		if convClient == nil {
			t.Fatal("Expected non-nil conversation client")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("Expected non-nil default config")
	}
}

func TestOnAndOff(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	convClient := New(client, nil)

	t.Run("register handler", func(t *testing.T) {
		handler := func(activity *Activity) {}
		convClient.On("post", handler)

		convClient.mu.RLock()
		handlers := convClient.handlers["post"]
		convClient.mu.RUnlock()

		if len(handlers) != 1 {
			t.Errorf("Expected 1 handler, got %d", len(handlers))
		}
	})

	t.Run("nil handler ignored", func(t *testing.T) {
		convClient.On("share", nil)

		convClient.mu.RLock()
		handlers := convClient.handlers["share"]
		convClient.mu.RUnlock()

		if len(handlers) != 0 {
			t.Errorf("Expected 0 handlers for nil handler, got %d", len(handlers))
		}
	})

	t.Run("unregister handler", func(t *testing.T) {
		myHandler := func(activity *Activity) {}
		convClient.On("acknowledge", myHandler)

		convClient.mu.RLock()
		before := len(convClient.handlers["acknowledge"])
		convClient.mu.RUnlock()
		if before != 1 {
			t.Fatalf("Expected 1 handler before Off, got %d", before)
		}

		convClient.Off("acknowledge", myHandler)

		convClient.mu.RLock()
		after := len(convClient.handlers["acknowledge"])
		convClient.mu.RUnlock()
		if after != 0 {
			t.Errorf("Expected 0 handlers after Off, got %d", after)
		}
	})

	t.Run("off with nil handler ignored", func(t *testing.T) {
		convClient.Off("post", nil)

		convClient.mu.RLock()
		handlers := convClient.handlers["post"]
		convClient.mu.RUnlock()

		if len(handlers) != 1 {
			t.Errorf("Expected 1 handler to remain, got %d", len(handlers))
		}
	})

	t.Run("wildcard handler", func(t *testing.T) {
		handler := func(activity *Activity) {}
		convClient.On(WildcardHandler, handler)

		convClient.mu.RLock()
		handlers := convClient.handlers[WildcardHandler]
		convClient.mu.RUnlock()

		if len(handlers) != 1 {
			t.Errorf("Expected 1 wildcard handler, got %d", len(handlers))
		}
	})
}

func TestProcessActivityEvent(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	convClient := New(client, nil)

	t.Run("nil event data", func(t *testing.T) {
		event := &mercury.Event{Data: nil}
		_, err := convClient.ProcessActivityEvent(event)
		if err == nil {
			t.Error("Expected error for nil event data")
		}
	})

	t.Run("missing activity in data", func(t *testing.T) {
		event := &mercury.Event{
			Data: map[string]interface{}{
				"eventType": "conversation.activity",
			},
		}
		_, err := convClient.ProcessActivityEvent(event)
		if err == nil {
			t.Error("Expected error for missing activity data")
		}
	})

	t.Run("valid activity event", func(t *testing.T) {
		event := &mercury.Event{
			Data: map[string]interface{}{
				"eventType": "conversation.activity",
				"activity": map[string]interface{}{
					"id":         "activity-123",
					"objectType": "activity",
					"verb":       "post",
					"actor": map[string]interface{}{
						"id":          "actor-123",
						"displayName": "Test User",
					},
					"target": map[string]interface{}{
						"id":         "room-123",
						"objectType": "conversation",
					},
					"object": map[string]interface{}{
						"objectType":  "comment",
						"displayName": "Hello, World!",
					},
					"encryptionKeyUrl": "kms://key-url",
				},
			},
		}

		activity, err := convClient.ProcessActivityEvent(event)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if activity.ID != "activity-123" {
			t.Errorf("Expected ID 'activity-123', got %q", activity.ID)
		}
		if activity.Verb != "post" {
			t.Errorf("Expected Verb 'post', got %q", activity.Verb)
		}
		if activity.MessageType != MessageTypePost {
			t.Errorf("Expected MessageType 'post', got %q", activity.MessageType)
		}
		if activity.EncryptionKeyURL != "kms://key-url" {
			t.Errorf("Expected EncryptionKeyURL 'kms://key-url', got %q", activity.EncryptionKeyURL)
		}
		if activity.Actor == nil || activity.Actor.ID != "actor-123" {
			t.Error("Expected actor with ID 'actor-123'")
		}
		if activity.Target == nil || activity.Target.ID != "room-123" {
			t.Error("Expected target with ID 'room-123'")
		}
		if activity.RawData == nil {
			t.Error("Expected non-nil RawData")
		}
	})
}

func TestGetMessageContent(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	convClient := New(client, nil)

	t.Run("nil activity", func(t *testing.T) {
		_, err := convClient.GetMessageContent(nil)
		if err == nil {
			t.Error("Expected error for nil activity")
		}
	})

	t.Run("activity with content already set", func(t *testing.T) {
		activity := &Activity{
			Content: "Hello, World!",
		}
		content, err := convClient.GetMessageContent(activity)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if content != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got %q", content)
		}
	})

	t.Run("activity with decrypted object content", func(t *testing.T) {
		activity := &Activity{
			DecryptedObject: &Object{
				Content: "Decrypted content",
			},
		}
		content, err := convClient.GetMessageContent(activity)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if content != "Decrypted content" {
			t.Errorf("Expected 'Decrypted content', got %q", content)
		}
	})

	t.Run("activity with nil object", func(t *testing.T) {
		activity := &Activity{}
		_, err := convClient.GetMessageContent(activity)
		if err == nil {
			t.Error("Expected error for nil object")
		}
	})

	t.Run("activity with displayName in object", func(t *testing.T) {
		activity := &Activity{
			Object: map[string]interface{}{
				"displayName": "Plain text message",
			},
		}
		content, err := convClient.GetMessageContent(activity)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if content != "Plain text message" {
			t.Errorf("Expected 'Plain text message', got %q", content)
		}
	})

	t.Run("activity with empty displayName", func(t *testing.T) {
		activity := &Activity{
			Object: map[string]interface{}{
				"displayName": "",
			},
		}
		_, err := convClient.GetMessageContent(activity)
		if err == nil {
			t.Error("Expected error for empty displayName")
		}
	})
}

func TestActivitySerialization(t *testing.T) {
	t.Run("marshal and unmarshal activity", func(t *testing.T) {
		activity := Activity{
			ID:         "act-123",
			ObjectType: "activity",
			Verb:       "post",
			Actor: &Actor{
				ID:          "user-123",
				DisplayName: "Test User",
			},
			Target: &Target{
				ID:         "room-456",
				ObjectType: "conversation",
			},
		}

		data, err := json.Marshal(activity)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var parsed Activity
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if parsed.ID != "act-123" {
			t.Errorf("Expected ID 'act-123', got %q", parsed.ID)
		}
		if parsed.Verb != "post" {
			t.Errorf("Expected Verb 'post', got %q", parsed.Verb)
		}
		if parsed.Actor.DisplayName != "Test User" {
			t.Errorf("Expected Actor.DisplayName 'Test User', got %q", parsed.Actor.DisplayName)
		}
	})
}

func TestMessageTypeConstants(t *testing.T) {
	if MessageTypePost != "post" {
		t.Errorf("Expected MessageTypePost 'post', got %q", MessageTypePost)
	}
	if MessageTypeShare != "share" {
		t.Errorf("Expected MessageTypeShare 'share', got %q", MessageTypeShare)
	}
	if MessageTypeAcknowledge != "acknowledge" {
		t.Errorf("Expected MessageTypeAcknowledge 'acknowledge', got %q", MessageTypeAcknowledge)
	}
	if WildcardHandler != "*" {
		t.Errorf("Expected WildcardHandler '*', got %q", WildcardHandler)
	}
}

func TestIsMessageActivity(t *testing.T) {
	tests := []struct {
		verb     string
		expected bool
	}{
		{"post", true},
		{"share", true},
		{"acknowledge", false},
		{"delete", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.verb, func(t *testing.T) {
			result := isMessageActivity(tc.verb)
			if result != tc.expected {
				t.Errorf("isMessageActivity(%q) = %v, want %v", tc.verb, result, tc.expected)
			}
		})
	}
}

func TestProcessMessageContent(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	convClient := New(client, nil)

	t.Run("nil object", func(t *testing.T) {
		activity := &Activity{Object: nil}
		err := convClient.processMessageContent(activity)
		if err != nil {
			t.Errorf("Expected nil error for nil object, got %v", err)
		}
	})

	t.Run("object with displayName", func(t *testing.T) {
		activity := &Activity{
			Object: map[string]interface{}{
				"objectType":  "comment",
				"displayName": "Hello",
			},
		}
		err := convClient.processMessageContent(activity)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if activity.Content != "Hello" {
			t.Errorf("Expected Content 'Hello', got %q", activity.Content)
		}
		if activity.DecryptedObject == nil {
			t.Error("Expected non-nil DecryptedObject")
		}
	})

	t.Run("object with empty displayName", func(t *testing.T) {
		activity := &Activity{
			Object: map[string]interface{}{
				"objectType": "comment",
			},
		}
		err := convClient.processMessageContent(activity)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if activity.Content != "" {
			t.Errorf("Expected empty Content, got %q", activity.Content)
		}
	})
}

func TestSetMercuryClient(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	convClient := New(client, nil)
	mercuryClient := mercury.New(client, nil)

	convClient.SetMercuryClient(mercuryClient)

	convClient.mu.RLock()
	mc := convClient.mercuryClient
	convClient.mu.RUnlock()

	if mc == nil {
		t.Error("Expected mercury client to be set")
	}
}

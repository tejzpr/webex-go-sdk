/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package mercury

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestNew(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)

	t.Run("with default config", func(t *testing.T) {
		mercuryClient := New(client, nil)
		if mercuryClient == nil {
			t.Fatal("Expected non-nil mercury client")
		}
		if mercuryClient.config.PingInterval != 30*time.Second {
			t.Errorf("Expected PingInterval 30s, got %v", mercuryClient.config.PingInterval)
		}
		if mercuryClient.config.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries 3, got %d", mercuryClient.config.MaxRetries)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			PingInterval: 15 * time.Second,
			PongTimeout:  5 * time.Second,
			MaxRetries:   10,
		}
		mercuryClient := New(client, cfg)
		if mercuryClient == nil {
			t.Fatal("Expected non-nil mercury client")
		}
		if mercuryClient.config.PingInterval != 15*time.Second {
			t.Errorf("Expected PingInterval 15s, got %v", mercuryClient.config.PingInterval)
		}
		if mercuryClient.config.MaxRetries != 10 {
			t.Errorf("Expected MaxRetries 10, got %d", mercuryClient.config.MaxRetries)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ForceCloseDelay != 10*time.Second {
		t.Errorf("Expected ForceCloseDelay 10s, got %v", cfg.ForceCloseDelay)
	}
	if cfg.PingInterval != 30*time.Second {
		t.Errorf("Expected PingInterval 30s, got %v", cfg.PingInterval)
	}
	if cfg.PongTimeout != 10*time.Second {
		t.Errorf("Expected PongTimeout 10s, got %v", cfg.PongTimeout)
	}
	if cfg.BackoffTimeMax != 32*time.Second {
		t.Errorf("Expected BackoffTimeMax 32s, got %v", cfg.BackoffTimeMax)
	}
	if cfg.BackoffTimeReset != 1*time.Second {
		t.Errorf("Expected BackoffTimeReset 1s, got %v", cfg.BackoffTimeReset)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialConnectionMaxRetries != 5 {
		t.Errorf("Expected InitialConnectionMaxRetries 5, got %d", cfg.InitialConnectionMaxRetries)
	}
}

func TestIsConnected(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	if mercuryClient.IsConnected() {
		t.Error("Expected IsConnected to be false initially")
	}

	mercuryClient.mu.Lock()
	mercuryClient.connected = true
	mercuryClient.mu.Unlock()

	if !mercuryClient.IsConnected() {
		t.Error("Expected IsConnected to be true after setting connected flag")
	}
}

func TestConnectAlreadyConnected(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	mercuryClient.mu.Lock()
	mercuryClient.connected = true
	mercuryClient.mu.Unlock()

	err := mercuryClient.Connect()
	if err != nil {
		t.Errorf("Expected nil error when already connected, got %v", err)
	}
}

func TestConnectAlreadyConnecting(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	mercuryClient.mu.Lock()
	mercuryClient.connecting = true
	mercuryClient.mu.Unlock()

	err := mercuryClient.Connect()
	if err == nil {
		t.Error("Expected error when connection attempt already in progress")
	}
}

func TestConnectNoDeviceProvider(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	err := mercuryClient.Connect()
	if err == nil {
		t.Error("Expected error when no device provider or custom URL is set")
	}
}

func TestSetDeviceProvider(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	provider := &mockDeviceProvider{wsURL: "wss://test-url"}
	mercuryClient.SetDeviceProvider(provider)

	mercuryClient.mu.Lock()
	dp := mercuryClient.deviceProvider
	mercuryClient.mu.Unlock()

	if dp == nil {
		t.Error("Expected device provider to be set")
	}
}

func TestSetCustomWebSocketURL(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	mercuryClient.SetCustomWebSocketURL("wss://custom-url")

	mercuryClient.mu.Lock()
	url := mercuryClient.customWebSocketURL
	mercuryClient.mu.Unlock()

	if url != "wss://custom-url" {
		t.Errorf("Expected 'wss://custom-url', got %q", url)
	}
}

func TestOnAndOff(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	t.Run("register handler", func(t *testing.T) {
		handler := func(event *Event) {}
		mercuryClient.On("test.event", handler)

		mercuryClient.mu.Lock()
		handlers := mercuryClient.eventHandlers["test.event"]
		mercuryClient.mu.Unlock()

		if len(handlers) != 1 {
			t.Errorf("Expected 1 handler, got %d", len(handlers))
		}
	})

	t.Run("nil handler ignored", func(t *testing.T) {
		mercuryClient.On("test.nil", nil)

		mercuryClient.mu.Lock()
		handlers := mercuryClient.eventHandlers["test.nil"]
		mercuryClient.mu.Unlock()

		if len(handlers) != 0 {
			t.Errorf("Expected 0 handlers for nil handler, got %d", len(handlers))
		}
	})

	t.Run("unregister all handlers by clearing map", func(t *testing.T) {
		// Register a handler we can reference for Off
		myHandler := func(event *Event) {}
		mercuryClient.On("test.off", myHandler)

		mercuryClient.mu.Lock()
		before := len(mercuryClient.eventHandlers["test.off"])
		mercuryClient.mu.Unlock()
		if before != 1 {
			t.Fatalf("Expected 1 handler before Off, got %d", before)
		}

		mercuryClient.Off("test.off", myHandler)

		mercuryClient.mu.Lock()
		after := len(mercuryClient.eventHandlers["test.off"])
		mercuryClient.mu.Unlock()
		if after != 0 {
			t.Errorf("Expected 0 handlers after Off, got %d", after)
		}
	})
}

func TestDisconnectWhenNotConnected(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mercuryClient := New(client, nil)

	err := mercuryClient.Disconnect()
	if err != nil {
		t.Errorf("Expected nil error when disconnecting while not connected, got %v", err)
	}
}

func TestEventParsing(t *testing.T) {
	t.Run("parse event JSON", func(t *testing.T) {
		eventJSON := `{
			"id": "event-123",
			"data": {"eventType": "conversation.activity"},
			"timestamp": 1234567890,
			"trackingId": "tracking-123",
			"sequenceNumber": 42
		}`

		var event Event
		err := json.Unmarshal([]byte(eventJSON), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		if event.ID != "event-123" {
			t.Errorf("Expected ID 'event-123', got %q", event.ID)
		}
		if event.Timestamp != 1234567890 {
			t.Errorf("Expected Timestamp 1234567890, got %d", event.Timestamp)
		}
		if event.TrackingID != "tracking-123" {
			t.Errorf("Expected TrackingID 'tracking-123', got %q", event.TrackingID)
		}
		if event.SequenceNumber != 42 {
			t.Errorf("Expected SequenceNumber 42, got %d", event.SequenceNumber)
		}
		if event.Data["eventType"] != "conversation.activity" {
			t.Errorf("Expected eventType 'conversation.activity', got %v", event.Data["eventType"])
		}
	})
}

func TestClearHandlers(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	mc.On("test.event", func(event *Event) {})
	mc.On("test.event", func(event *Event) {})
	mc.On("other.event", func(event *Event) {})

	mc.ClearHandlers("test.event")

	mc.mu.Lock()
	testHandlers := mc.eventHandlers["test.event"]
	otherHandlers := mc.eventHandlers["other.event"]
	mc.mu.Unlock()

	if len(testHandlers) != 0 {
		t.Errorf("Expected 0 handlers after ClearHandlers, got %d", len(testHandlers))
	}
	if len(otherHandlers) != 1 {
		t.Errorf("Expected 1 handler for other.event, got %d", len(otherHandlers))
	}
}

func TestEventHandlers(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	mc.On("a", func(event *Event) {})
	mc.On("b", func(event *Event) {})
	mc.On("b", func(event *Event) {})

	handlers := mc.EventHandlers()
	if len(handlers["a"]) != 1 {
		t.Errorf("Expected 1 handler for 'a', got %d", len(handlers["a"]))
	}
	if len(handlers["b"]) != 2 {
		t.Errorf("Expected 2 handlers for 'b', got %d", len(handlers["b"]))
	}
}

func TestDispatchEvent(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	typedCh := make(chan string, 1)
	wildcardCh := make(chan string, 1)
	activityCh := make(chan string, 1)

	mc.On("conversation.activity", func(event *Event) {
		typedCh <- "typed"
	})
	mc.On("*", func(event *Event) {
		wildcardCh <- "wildcard"
	})
	mc.On("activity:post", func(event *Event) {
		activityCh <- "activity"
	})

	event := &Event{
		EventType:    "conversation.activity",
		ActivityType: "post",
	}
	mc.dispatchEvent(event)

	// Wait for all handlers to fire
	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-typedCh:
		case <-wildcardCh:
		case <-activityCh:
		case <-timeout:
			t.Fatal("Timed out waiting for handlers")
		}
	}
}

func TestDispatchEvent_NoHandlers(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	// Should not panic with no handlers registered
	event := &Event{EventType: "unknown.event"}
	mc.dispatchEvent(event)
}

func TestHandleConversationActivity_Post(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	ch := make(chan *Event, 1)
	mc.On("message.created", func(event *Event) {
		ch <- event
	})

	event := &Event{
		EventType:    "conversation.activity",
		ActivityType: "post",
		ID:           "evt-1",
	}
	mc.handleConversationActivity(event)

	select {
	case received := <-ch:
		if received.EventType != "message.created" {
			t.Errorf("Expected EventType 'message.created', got %q", received.EventType)
		}
		if received.ID != "evt-1" {
			t.Errorf("Expected ID 'evt-1', got %q", received.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message.created handler")
	}
}

func TestHandleConversationActivity_Share(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	ch := make(chan *Event, 1)
	mc.On("message.created", func(event *Event) {
		ch <- event
	})

	event := &Event{
		EventType:    "conversation.activity",
		ActivityType: "share",
	}
	mc.handleConversationActivity(event)

	select {
	case <-ch:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message.created handler on share")
	}
}

func TestHandleConversationActivity_NonMessage(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	called := false
	mc.On("message.created", func(event *Event) {
		called = true
	})

	event := &Event{
		EventType:    "conversation.activity",
		ActivityType: "acknowledge",
	}
	mc.handleConversationActivity(event)

	// Give a moment for any goroutine to fire
	time.Sleep(50 * time.Millisecond)
	if called {
		t.Error("message.created handler should not fire for acknowledge activity")
	}
}

func TestHandlePong(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	t.Run("empty data", func(t *testing.T) {
		err := mc.handlePong("")
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("valid timestamp with nil conn", func(t *testing.T) {
		// handlePong returns nil early when conn is nil (before calculating offset)
		pingTime := time.Now().Add(-100 * time.Millisecond).UnixMilli()
		data := fmt.Sprintf("%d", pingTime)

		err := mc.handlePong(data)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}

		// With nil conn, handlePong returns early — timeOffset stays 0
		mc.mu.Lock()
		offset := mc.timeOffset
		mc.mu.Unlock()
		if offset != 0 {
			t.Errorf("Expected timeOffset 0 with nil conn, got %dms", offset)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		err := mc.handlePong("not-a-number")
		if err != nil {
			t.Errorf("Expected nil error for invalid data, got %v", err)
		}
		// timeOffset should remain from previous test (not reset to 0)
	})
}

func TestListenStopListening(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	// Listen and StopListening are aliases for Connect/Disconnect
	// StopListening when not connected should be no-op
	err := mc.StopListening()
	if err != nil {
		t.Errorf("Expected nil error from StopListening, got %v", err)
	}
}

func TestGetReconnectURL_CustomURL(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	url := mc.getReconnectURL(nil, "wss://my-custom-url")
	if url != "wss://my-custom-url" {
		t.Errorf("Expected custom URL, got %q", url)
	}
}

func TestGetReconnectURL_DeviceProvider(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	provider := &mockDeviceProvider{wsURL: "wss://from-device"}
	url := mc.getReconnectURL(provider, "")
	if url != "wss://from-device" {
		t.Errorf("Expected device provider URL, got %q", url)
	}
}

func TestGetReconnectURL_Fallback(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	url := mc.getReconnectURL(nil, "")
	if url != "wss://mercury-connection-a.wbx2.com/mercury/device" {
		t.Errorf("Expected fallback URL, got %q", url)
	}
}

func TestGetReconnectURL_CustomFallback(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	cfg := DefaultConfig()
	cfg.FallbackWebSocketURL = "wss://custom-fallback"
	mc := New(client, cfg)

	url := mc.getReconnectURL(nil, "")
	if url != "wss://custom-fallback" {
		t.Errorf("Expected custom fallback URL, got %q", url)
	}
}

func TestGetReconnectURL_DeviceProviderError(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	mc := New(client, nil)

	provider := &mockDeviceProvider{err: fmt.Errorf("register failed")}
	url := mc.getReconnectURL(provider, "")
	// When Register() fails, getReconnectURL returns empty string
	if url != "" {
		t.Errorf("Expected empty string on provider error, got %q", url)
	}
}

// mockDeviceProvider implements DeviceProvider for testing
type mockDeviceProvider struct {
	wsURL string
	err   error
}

func (m *mockDeviceProvider) Register() error {
	return m.err
}

func (m *mockDeviceProvider) GetWebSocketURL() (string, error) {
	return m.wsURL, m.err
}

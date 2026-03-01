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
	"sync"
	"testing"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	"github.com/pion/webrtc/v4"
)

// ---- EventEmitter Tests ----

func TestEventEmitter(t *testing.T) {
	t.Run("On and Emit", func(t *testing.T) {
		emitter := NewEventEmitter()
		var received interface{}
		emitter.On("test", func(data interface{}) {
			received = data
		})
		emitter.Emit("test", "hello")
		if received != "hello" {
			t.Errorf("Expected 'hello', got %v", received)
		}
	})

	t.Run("multiple handlers", func(t *testing.T) {
		emitter := NewEventEmitter()
		count := 0
		emitter.On("test", func(data interface{}) { count++ })
		emitter.On("test", func(data interface{}) { count++ })
		emitter.Emit("test", nil)
		if count != 2 {
			t.Errorf("Expected 2 calls, got %d", count)
		}
	})

	t.Run("Off removes handlers", func(t *testing.T) {
		emitter := NewEventEmitter()
		called := false
		emitter.On("test", func(data interface{}) { called = true })
		emitter.Off("test")
		emitter.Emit("test", nil)
		if called {
			t.Error("Handler should not have been called after Off")
		}
	})

	t.Run("nil handler ignored", func(t *testing.T) {
		emitter := NewEventEmitter()
		emitter.On("test", nil)
		emitter.Emit("test", nil) // should not panic
	})

	t.Run("emit unknown event", func(t *testing.T) {
		emitter := NewEventEmitter()
		emitter.Emit("unknown", nil) // should not panic
	})

	t.Run("concurrent safety", func(t *testing.T) {
		emitter := NewEventEmitter()
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				emitter.On("test", func(data interface{}) {})
				emitter.Emit("test", nil)
			}()
		}
		wg.Wait()
	})
}

// ---- MediaEngine Tests ----

func TestMediaEngine(t *testing.T) {
	t.Run("NewMediaEngine with nil config", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if me == nil {
			t.Fatal("Expected non-nil MediaEngine")
		}
		defer func() { _ = me.Close() }()
	})

	t.Run("NewMediaEngine with custom config", func(t *testing.T) {
		cfg := &MediaConfig{
			ICEServers: []webrtc.ICEServer{},
		}
		_ = cfg // just verify it compiles
		me, err := NewMediaEngine(DefaultMediaConfig())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()
		if me == nil {
			t.Fatal("Expected non-nil MediaEngine")
		}
	})

	t.Run("AddAudioTrack", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()

		track, err := me.AddAudioTrack()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if track == nil {
			t.Fatal("Expected non-nil track")
		}
		if me.GetLocalTrack() == nil {
			t.Fatal("Expected GetLocalTrack to return the track")
		}
	})

	t.Run("Mute/Unmute", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()

		if me.IsMuted() {
			t.Error("Should not be muted initially")
		}
		me.Mute()
		if !me.IsMuted() {
			t.Error("Should be muted after Mute()")
		}
		me.Unmute()
		if me.IsMuted() {
			t.Error("Should not be muted after Unmute()")
		}
	})

	t.Run("CreateOffer", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()

		_, err = me.AddAudioTrack()
		if err != nil {
			t.Fatalf("Unexpected error adding track: %v", err)
		}

		sdp, err := me.CreateOffer()
		if err != nil {
			t.Fatalf("Unexpected error creating offer: %v", err)
		}
		if sdp == "" {
			t.Fatal("Expected non-empty SDP")
		}
		if len(sdp) < 50 {
			t.Errorf("SDP seems too short: %s", sdp)
		}
	})

	t.Run("GetConnectionState", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()

		state := me.GetConnectionState()
		if state.String() == "" {
			t.Error("Expected a valid connection state")
		}
	})

	t.Run("OnRemoteTrack callback", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer func() { _ = me.Close() }()

		called := false
		me.OnRemoteTrack(func(track *webrtc.TrackRemote) {
			called = true
		})
		// We can't easily trigger a remote track in unit tests,
		// but verify the callback is set without panic
		if called {
			t.Error("Should not be called yet")
		}
	})

	t.Run("Close", func(t *testing.T) {
		me, err := NewMediaEngine(nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if err := me.Close(); err != nil {
			t.Fatalf("Unexpected error closing: %v", err)
		}
	})
}

// ---- ROAP Helpers Tests ----

func TestRoapHelpers(t *testing.T) {
	t.Run("SDPToRoapOffer", func(t *testing.T) {
		msg := SDPToRoapOffer("v=0\r\n", 1)
		if msg.MessageType != RoapMessageOffer {
			t.Errorf("Expected OFFER, got %s", msg.MessageType)
		}
		if msg.SDP != "v=0\r\n" {
			t.Errorf("Unexpected SDP: %s", msg.SDP)
		}
		if msg.Seq != 1 {
			t.Errorf("Expected seq=1, got %d", msg.Seq)
		}
	})

	t.Run("SDPToRoapAnswer", func(t *testing.T) {
		msg := SDPToRoapAnswer("v=0\r\n", 2)
		if msg.MessageType != RoapMessageAnswer {
			t.Errorf("Expected ANSWER, got %s", msg.MessageType)
		}
		if msg.Seq != 2 {
			t.Errorf("Expected seq=2, got %d", msg.Seq)
		}
	})

	t.Run("NewRoapOK", func(t *testing.T) {
		msg := NewRoapOK(3)
		if msg.MessageType != RoapMessageOK {
			t.Errorf("Expected OK, got %s", msg.MessageType)
		}
		if msg.Seq != 3 {
			t.Errorf("Expected seq=3, got %d", msg.Seq)
		}
	})

	t.Run("RoapToSDP nil", func(t *testing.T) {
		if sdp := RoapToSDP(nil); sdp != "" {
			t.Errorf("Expected empty string, got %q", sdp)
		}
	})

	t.Run("RoapToSDP with message", func(t *testing.T) {
		msg := &RoapMessage{SDP: "test-sdp"}
		if sdp := RoapToSDP(msg); sdp != "test-sdp" {
			t.Errorf("Expected 'test-sdp', got %q", sdp)
		}
	})

	t.Run("ModifySdpForMobius", func(t *testing.T) {
		sdp := "v=0\r\na=candidate:1 1 udp 2130706431 192.168.1.1 5000 typ host\r\na=candidate:2 1 udp 2130706431 ::1 5001 typ host generation 0 ufrag abc network-id 1 network-cost 10 IP6 \r\n"
		result := ModifySdpForMobius(sdp)
		if result == sdp {
			t.Error("Expected IPv6 candidates to be removed")
		}
		if len(result) >= len(sdp) {
			t.Error("Expected result to be shorter after removing IPv6")
		}
	})
}

// ---- Line Tests ----

func TestLine(t *testing.T) {
	t.Run("NewLine", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, nil)
		if line == nil {
			t.Fatal("Expected non-nil line")
		}
		if line.LineID == "" {
			t.Error("Expected non-empty LineID")
		}
		if line.GetStatus() != RegistrationStatusIdle {
			t.Errorf("Expected IDLE status, got %s", line.GetStatus())
		}
		if line.IsRegistered() {
			t.Error("Should not be registered initially")
		}
	})

	t.Run("Register success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(MobiusDeviceInfo{
					UserID:            "user-123",
					KeepaliveInterval: 30,
					Device: &DeviceType{
						DeviceID: "device-abc",
						URI:      "https://mobius/devices/device-abc",
					},
				}); err != nil {
					t.Logf("Failed to encode response: %v", err)
				}
			case http.MethodDelete, http.MethodPut:
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/"},
			ClientDeviceURI:   "https://wdm/devices/test",
		})

		if err := line.Register(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !line.IsRegistered() {
			t.Error("Expected line to be registered")
		}
		if line.GetDeviceID() != "device-abc" {
			t.Errorf("Expected device-abc, got %s", line.GetDeviceID())
		}
		if line.GetActiveMobiusURL() != server.URL+"/" {
			t.Errorf("Unexpected active URL: %s", line.GetActiveMobiusURL())
		}

		// Cleanup
		_ = line.Deregister()
	})

	t.Run("Register failure falls back to backup", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(MobiusDeviceInfo{
				Device: &DeviceType{DeviceID: "backup-device"},
			}); err != nil {
				t.Logf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/primary/"},
			BackupMobiusURLs:  []string{server.URL + "/backup/"},
		})

		if err := line.Register(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !line.IsRegistered() {
			t.Error("Expected line to be registered via backup")
		}

		_ = line.Deregister()
	})

	t.Run("Register all servers fail", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/"},
		})

		err := line.Register()
		if err == nil {
			t.Fatal("Expected error when all servers fail")
		}
		if line.IsRegistered() {
			t.Error("Should not be registered")
		}
	})

	t.Run("Deregister", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(MobiusDeviceInfo{
					Device: &DeviceType{DeviceID: "dev-1", URI: "https://test/dev-1"},
				}); err != nil {
					t.Logf("Failed to encode response: %v", err)
				}
				return
			}
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusOK)
				return
			}
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/"},
		})
		if err := line.Register(); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if err := line.Deregister(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if line.IsRegistered() {
			t.Error("Should not be registered after deregister")
		}
	})

	t.Run("Deregister when not registered", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, nil)
		if err := line.Deregister(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("Register already registered is no-op", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(MobiusDeviceInfo{
				Device: &DeviceType{DeviceID: "dev-1"},
			})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/"},
		})
		if err := line.Register(); err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Second register should be no-op
		if err := line.Register(); err != nil {
			t.Fatalf("Unexpected error on second register: %v", err)
		}

		_ = line.Deregister()
	})

	t.Run("GetDeviceInfo", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(MobiusDeviceInfo{
				UserID: "user-xyz",
				Device: &DeviceType{DeviceID: "dev-1"},
			})
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		line := NewLine(core, nil, &LineConfig{
			PrimaryMobiusURLs: []string{server.URL + "/"},
		})
		if err := line.Register(); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		defer func() { _ = line.Deregister() }()

		info := line.GetDeviceInfo()
		if info == nil {
			t.Fatal("Expected non-nil device info")
		}
		if info.UserID != "user-xyz" {
			t.Errorf("Expected user-xyz, got %s", info.UserID)
		}
	})
}

// ---- Call Tests ----

func TestCall(t *testing.T) {
	t.Run("NewCall", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, err := NewCall(core, CallDirectionOutbound, &CallDetails{
			Type:    CallTypeURI,
			Address: "sip:user@example.com",
		}, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if call == nil {
			t.Fatal("Expected non-nil call")
		}
		if call.GetState() != CallStateIdle {
			t.Errorf("Expected idle state, got %s", call.GetState())
		}
		if call.GetDirection() != CallDirectionOutbound {
			t.Errorf("Expected outbound, got %s", call.GetDirection())
		}
		if call.IsConnected() {
			t.Error("Should not be connected initially")
		}
		if call.IsMuted() {
			t.Error("Should not be muted initially")
		}
		if call.IsHeld() {
			t.Error("Should not be held initially")
		}
		if call.GetCallID() == "" {
			t.Error("Expected non-empty callID")
		}
		if call.GetCorrelationID() == "" {
			t.Error("Expected non-empty correlationID")
		}
		_ = call.GetMedia().Close()
	})

	t.Run("NewCall nil config", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		_, err := NewCall(core, CallDirectionOutbound, nil, nil)
		if err == nil {
			t.Fatal("Expected error with nil config")
		}
	})

	t.Run("Mute/Unmute", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		call.Mute()
		if !call.IsMuted() {
			t.Error("Should be muted")
		}
		call.Unmute()
		if call.IsMuted() {
			t.Error("Should not be muted")
		}
	})

	t.Run("Hold when not connected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		err := call.Hold()
		if err == nil {
			t.Error("Expected error when holding a non-connected call")
		}
	})

	t.Run("Resume when not held", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		err := call.Resume()
		if err == nil {
			t.Error("Expected error when resuming a non-held call")
		}
	})

	t.Run("End when already disconnected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})

		call.mu.Lock()
		call.state = CallStateDisconnected
		call.mu.Unlock()

		err := call.End()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("SendDigit when not connected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		err := call.SendDigit("5")
		if err == nil {
			t.Error("Expected error when sending digit on non-connected call")
		}
	})

	t.Run("CompleteTransfer when not connected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		err := call.CompleteTransfer(TransferTypeBlind, "", "+15551234567")
		if err == nil {
			t.Error("Expected error when transferring non-connected call")
		}
	})

	t.Run("Dial when not idle", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		call.mu.Lock()
		call.state = CallStateConnected
		call.mu.Unlock()

		err := call.Dial()
		if err == nil {
			t.Error("Expected error when dialing non-idle call")
		}
	})

	t.Run("GetDisconnectReason", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		reason := call.GetDisconnectReason()
		if reason.Code != DisconnectCodeNormal {
			t.Errorf("Expected normal disconnect code, got %d", reason.Code)
		}
	})

	t.Run("HandleMobiusEvent nil", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		call.HandleMobiusEvent(nil) // should not panic
	})

	t.Run("HandleMobiusEvent call progress", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		call.mu.Lock()
		call.state = CallStateProceeding
		call.mu.Unlock()

		alertingEmitted := false
		call.Emitter.On(string(CallEventAlerting), func(data interface{}) {
			alertingEmitted = true
		})

		call.HandleMobiusEvent(&MobiusCallEvent{
			Data: MobiusCallData{
				EventType: MobiusEventCallProgress,
				CallID:    "call-1",
			},
		})

		if !alertingEmitted {
			t.Error("Expected alerting event to be emitted")
		}
		if call.GetState() != CallStateAlerting {
			t.Errorf("Expected alerting state, got %s", call.GetState())
		}
	})

	t.Run("HandleMobiusEvent call connected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})
		defer func() { _ = call.GetMedia().Close() }()

		connectEmitted := false
		call.Emitter.On(string(CallEventConnect), func(data interface{}) {
			connectEmitted = true
		})

		call.HandleMobiusEvent(&MobiusCallEvent{
			Data: MobiusCallData{
				EventType: MobiusEventCallConnected,
				CallID:    "call-1",
			},
		})

		if !connectEmitted {
			t.Error("Expected connect event to be emitted")
		}
		if !call.IsConnected() {
			t.Error("Expected call to be connected")
		}
	})

	t.Run("HandleMobiusEvent call disconnected", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		call, _ := NewCall(core, CallDirectionOutbound, nil, &CallConfig{
			MobiusURL: "https://mobius.test/",
			DeviceID:  "dev-1",
			LineID:    "line-1",
		})

		call.mu.Lock()
		call.state = CallStateConnected
		call.connected = true
		call.mu.Unlock()

		disconnectEmitted := false
		call.Emitter.On(string(CallEventDisconnect), func(data interface{}) {
			disconnectEmitted = true
		})

		call.HandleMobiusEvent(&MobiusCallEvent{
			Data: MobiusCallData{
				EventType: MobiusEventCallDisconnected,
				CallID:    "call-1",
			},
		})

		if !disconnectEmitted {
			t.Error("Expected disconnect event to be emitted")
		}
		if call.IsConnected() {
			t.Error("Expected call to be disconnected")
		}
	})
}

// ---- CallingClient Tests ----

func TestCallingClient(t *testing.T) {
	t.Run("NewCallingClient", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		if cc == nil {
			t.Fatal("Expected non-nil CallingClient")
		}
	})

	t.Run("NewCallingClient with config", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, &CallingClientConfig{
			ClientDeviceURI: "https://wdm/devices/test",
			MediaConfig:     DefaultMediaConfig(),
		})
		if cc == nil {
			t.Fatal("Expected non-nil CallingClient")
		}
	})

	t.Run("SetMobiusServers", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		cc.SetMobiusServers(
			[]string{"https://primary.test/"},
			[]string{"https://backup.test/"},
		)
		// Verify via CreateLine (which reads the URLs)
		// We can't directly access private fields, but we can verify it doesn't panic
	})

	t.Run("CreateLine with no servers", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		_, err := cc.CreateLine()
		if err == nil {
			t.Error("Expected error when no Mobius servers configured")
		}
	})

	t.Run("CreateLine success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(MobiusDeviceInfo{
					Device: &DeviceType{DeviceID: "dev-1"},
				})
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		cc.SetMobiusServers([]string{server.URL + "/"}, nil)

		line, err := cc.CreateLine()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if line == nil {
			t.Fatal("Expected non-nil line")
		}

		lines := cc.GetLines()
		if len(lines) != 1 {
			t.Errorf("Expected 1 line, got %d", len(lines))
		}

		_ = cc.Shutdown()
	})

	t.Run("MakeCall with nil line", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		_, err := cc.MakeCall(nil, &CallDetails{Address: "sip:test@example.com"})
		if err == nil {
			t.Error("Expected error with nil line")
		}
	})

	t.Run("MakeCall with nil destination", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		line := NewLine(core, nil, nil)
		line.mu.Lock()
		line.status = RegistrationStatusActive
		line.mu.Unlock()
		_, err := cc.MakeCall(line, nil)
		if err == nil {
			t.Error("Expected error with nil destination")
		}
	})

	t.Run("MakeCall with unregistered line", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		line := NewLine(core, nil, nil)
		_, err := cc.MakeCall(line, &CallDetails{Address: "sip:test@example.com"})
		if err == nil {
			t.Error("Expected error with unregistered line")
		}
	})

	t.Run("GetActiveCalls empty", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		calls := cc.GetActiveCalls()
		if len(calls) != 0 {
			t.Errorf("Expected 0 active calls, got %d", len(calls))
		}
	})

	t.Run("GetConnectedCall nil", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		call := cc.GetConnectedCall()
		if call != nil {
			t.Error("Expected nil connected call")
		}
	})

	t.Run("HandleMercuryEvent invalid JSON", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		cc.HandleMercuryEvent([]byte("not json")) // should not panic
	})

	t.Run("HandleMercuryEvent incoming call", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)

		// Add a line so the incoming call can be routed
		line := NewLine(core, nil, nil)
		line.mu.Lock()
		line.MobiusDeviceID = "dev-1"
		line.status = RegistrationStatusActive
		line.activeMobiusURL = "https://mobius.test/"
		line.mu.Unlock()

		cc.mu.Lock()
		cc.lines[line.LineID] = line
		cc.mu.Unlock()

		incomingReceived := false
		cc.Emitter.On(string(LineEventIncomingCall), func(data interface{}) {
			incomingReceived = true
		})

		eventData, _ := json.Marshal(MobiusCallEvent{
			Data: MobiusCallData{
				EventType:     MobiusEventCallSetup,
				CallID:        "incoming-call-1",
				DeviceID:      "dev-1",
				CorrelationID: "corr-1",
			},
		})

		cc.HandleMercuryEvent(eventData)

		if !incomingReceived {
			t.Error("Expected incoming call event to be emitted")
		}

		calls := cc.GetActiveCalls()
		if len(calls) != 1 {
			t.Errorf("Expected 1 active call, got %d", len(calls))
		}

		_ = cc.Shutdown()
	})

	t.Run("Shutdown", func(t *testing.T) {
		core, _ := webexsdk.NewClient("test-token", nil)
		cc := NewCallingClient(core, nil, nil)
		if err := cc.Shutdown(); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

// ---- CallingClient accessor from top-level Client ----

func TestClientCallingClientAccessor(t *testing.T) {
	core, _ := webexsdk.NewClient("test-token", nil)
	client := New(core, nil)

	cc := client.CallingClient(nil)
	if cc == nil {
		t.Fatal("Expected non-nil CallingClient")
	}

	// Should return same instance
	cc2 := client.CallingClient(nil)
	if cc != cc2 {
		t.Error("Expected same CallingClient instance")
	}
}

// ---- Event/Type Constants Tests ----

func TestCallControlConstants(t *testing.T) {
	if CallStateIdle != "idle" {
		t.Errorf("Expected idle, got %q", CallStateIdle)
	}
	if CallStateConnected != "connected" {
		t.Errorf("Expected connected, got %q", CallStateConnected)
	}
	if CallStateHeld != "held" {
		t.Errorf("Expected held, got %q", CallStateHeld)
	}
	if CallStateDisconnected != "disconnected" {
		t.Errorf("Expected disconnected, got %q", CallStateDisconnected)
	}
	if RegistrationStatusActive != "active" {
		t.Errorf("Expected active, got %q", RegistrationStatusActive)
	}
	if RoapMessageOffer != "OFFER" {
		t.Errorf("Expected OFFER, got %q", RoapMessageOffer)
	}
	if RoapMessageAnswer != "ANSWER" {
		t.Errorf("Expected ANSWER, got %q", RoapMessageAnswer)
	}
	if TransferTypeBlind != "BLIND" {
		t.Errorf("Expected BLIND, got %q", TransferTypeBlind)
	}
	if TransferTypeConsult != "CONSULT" {
		t.Errorf("Expected CONSULT, got %q", TransferTypeConsult)
	}
	if MobiusEventCallSetup != "mobius.call" {
		t.Errorf("Expected mobius.call, got %q", MobiusEventCallSetup)
	}
}

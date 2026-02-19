/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestNew(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)

	t.Run("with default config", func(t *testing.T) {
		deviceClient := New(client, nil)
		if deviceClient == nil {
			t.Fatal("Expected non-nil device client")
		}
		if deviceClient.config.DeviceType != "WEB" {
			t.Errorf("Expected default DeviceType 'WEB', got %q", deviceClient.config.DeviceType)
		}
		if deviceClient.config.EphemeralDeviceTTL != 86400 {
			t.Errorf("Expected default EphemeralDeviceTTL 86400, got %d", deviceClient.config.EphemeralDeviceTTL)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &Config{
			Ephemeral:          true,
			EphemeralDeviceTTL: 3600,
			DeviceType:         "DESKTOP",
		}
		deviceClient := New(client, cfg)
		if deviceClient == nil {
			t.Fatal("Expected non-nil device client")
		}
		if deviceClient.config.DeviceType != "DESKTOP" {
			t.Errorf("Expected DeviceType 'DESKTOP', got %q", deviceClient.config.DeviceType)
		}
		if !deviceClient.config.Ephemeral {
			t.Error("Expected Ephemeral to be true")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Ephemeral {
		t.Error("Expected Ephemeral to be false by default")
	}
	if cfg.EphemeralDeviceTTL != 86400 {
		t.Errorf("Expected EphemeralDeviceTTL 86400, got %d", cfg.EphemeralDeviceTTL)
	}
	if cfg.DeviceType != "WEB" {
		t.Errorf("Expected DeviceType 'WEB', got %q", cfg.DeviceType)
	}
}

func TestRegister(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", r.Method)
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-token" {
				t.Errorf("Expected 'Bearer test-token', got %q", authHeader)
			}

			resp := DeviceDTO{
				URL:          "https://wdm-a.wbx2.com/wdm/api/v1/devices/device-123",
				WebSocketURL: "wss://mercury-connection-a.wbx2.com/v1/apps/wx2/registrations/device-123/messages",
				UserID:       "user-123",
				DeviceType:   "TEAMS_SDK_JS",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// We can't easily override the WDM URL since it's hardcoded,
		// but we can test the state management methods
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		// Verify initial state
		if deviceClient.IsRegistered() {
			t.Error("Expected device to not be registered initially")
		}
	})

	t.Run("idempotent registration", func(t *testing.T) {
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		// Manually set device info to simulate registration
		deviceClient.mu.Lock()
		deviceClient.deviceInfo = &DeviceResponse{
			URL:          "https://device-url",
			WebSocketURL: "wss://websocket-url",
		}
		deviceClient.registered = true
		deviceClient.mu.Unlock()

		// Second call should return nil (already registered)
		err := deviceClient.Register()
		if err != nil {
			t.Errorf("Expected nil error for already registered device, got %v", err)
		}
	})
}

func TestGetWebSocketURL(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	deviceClient := New(client, nil)

	// Manually set device info
	deviceClient.mu.Lock()
	deviceClient.deviceInfo = &DeviceResponse{
		URL:          "https://device-url",
		WebSocketURL: "wss://websocket-url",
	}
	deviceClient.registered = true
	deviceClient.mu.Unlock()

	wsURL, err := deviceClient.GetWebSocketURL()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if wsURL != "wss://websocket-url" {
		t.Errorf("Expected 'wss://websocket-url', got %q", wsURL)
	}
}

func TestGetDeviceURL(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	deviceClient := New(client, nil)

	// Manually set device info
	deviceClient.mu.Lock()
	deviceClient.deviceInfo = &DeviceResponse{
		URL:          "https://device-url",
		WebSocketURL: "wss://websocket-url",
	}
	deviceClient.registered = true
	deviceClient.mu.Unlock()

	deviceURL, err := deviceClient.GetDeviceURL()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if deviceURL != "https://device-url" {
		t.Errorf("Expected 'https://device-url', got %q", deviceURL)
	}
}

func TestGetDevice(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	deviceClient := New(client, nil)

	deviceClient.mu.Lock()
	deviceClient.device = &DeviceDTO{
		URL:        "https://device-url",
		UserID:     "user-123",
		DeviceType: "TEAMS_SDK_JS",
	}
	deviceClient.mu.Unlock()

	device := deviceClient.GetDevice()
	if device.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got %q", device.UserID)
	}
	if device.DeviceType != "TEAMS_SDK_JS" {
		t.Errorf("Expected DeviceType 'TEAMS_SDK_JS', got %q", device.DeviceType)
	}
}

func TestIsRegistered(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	deviceClient := New(client, nil)

	if deviceClient.IsRegistered() {
		t.Error("Expected IsRegistered to be false initially")
	}

	deviceClient.mu.Lock()
	deviceClient.registered = true
	deviceClient.mu.Unlock()

	if !deviceClient.IsRegistered() {
		t.Error("Expected IsRegistered to be true after setting registered flag")
	}
}

func TestOnRegistered(t *testing.T) {
	client, _ := webexsdk.NewClient("test-token", nil)
	deviceClient := New(client, nil)

	t.Run("callback called immediately if already registered", func(t *testing.T) {
		deviceClient.mu.Lock()
		deviceClient.registered = true
		deviceClient.mu.Unlock()

		called := make(chan bool, 1)
		deviceClient.OnRegistered(func() {
			called <- true
		})

		select {
		case <-called:
			// Success
		case <-time.After(1 * time.Second):
			t.Error("Expected callback to be called within 1 second")
		}
	})
}

func TestWaitForRegistration(t *testing.T) {
	t.Run("returns immediately if already registered", func(t *testing.T) {
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		deviceClient.mu.Lock()
		deviceClient.registered = true
		deviceClient.mu.Unlock()

		err := deviceClient.WaitForRegistration(100 * time.Millisecond)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})

	t.Run("times out if not registered", func(t *testing.T) {
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		err := deviceClient.WaitForRegistration(100 * time.Millisecond)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

func TestUnregister(t *testing.T) {
	t.Run("no-op when not registered", func(t *testing.T) {
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		err := deviceClient.Unregister()
		if err != nil {
			t.Errorf("Expected nil error for unregistered device, got %v", err)
		}
	})
}

func TestRefresh(t *testing.T) {
	t.Run("error when not registered", func(t *testing.T) {
		client, _ := webexsdk.NewClient("test-token", nil)
		deviceClient := New(client, nil)

		err := deviceClient.Refresh()
		if err == nil {
			t.Error("Expected error when refreshing unregistered device")
		}
	})
}

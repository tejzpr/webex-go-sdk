/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Line represents a registered telephony line with a Mobius server.
// A line must be registered before calls can be made or received.
type Line struct {
	mu sync.RWMutex

	core   *webexsdk.Client
	config *Config

	// Line properties
	LineID         string
	UserID         string
	MobiusDeviceID string
	PhoneNumber    string
	Extension      string
	SipAddresses   []string
	Voicemail      string

	// Registration
	status            RegistrationStatus
	activeMobiusURL   string
	primaryMobiusURLs []string
	backupMobiusURLs  []string
	deviceInfo        *MobiusDeviceInfo
	clientDeviceURI   string

	// Keepalive
	keepaliveInterval time.Duration
	keepaliveTicker   *time.Ticker
	keepaliveStop     chan struct{}

	// Events
	Emitter *EventEmitter
}

// LineConfig holds configuration for creating a line
type LineConfig struct {
	// PrimaryMobiusURLs is the list of primary Mobius server URLs
	PrimaryMobiusURLs []string
	// BackupMobiusURLs is the list of backup Mobius server URLs
	BackupMobiusURLs []string
	// ClientDeviceURI is the Webex device URL for this client
	ClientDeviceURI string
	// UserID is the Webex user ID (from device registration)
	UserID string
}

// NewLine creates a new Line instance
func NewLine(core *webexsdk.Client, config *Config, lineConfig *LineConfig) *Line {
	if config == nil {
		config = DefaultConfig()
	}

	l := &Line{
		core:              core,
		config:            config,
		LineID:            uuid.New().String(),
		status:            RegistrationStatusIdle,
		Emitter:           NewEventEmitter(),
		keepaliveInterval: 30 * time.Second,
	}

	if lineConfig != nil {
		l.primaryMobiusURLs = lineConfig.PrimaryMobiusURLs
		l.backupMobiusURLs = lineConfig.BackupMobiusURLs
		l.clientDeviceURI = lineConfig.ClientDeviceURI
		l.UserID = lineConfig.UserID
	}

	return l
}

// Register registers this line with the Mobius server.
// It attempts primary servers first, then falls back to backup servers.
func (l *Line) Register() error {
	l.mu.Lock()
	if l.status == RegistrationStatusActive {
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()

	l.Emitter.Emit(string(LineEventConnecting), nil)

	// Try primary servers first
	for _, url := range l.primaryMobiusURLs {
		if err := l.attemptRegistration(url); err != nil {
			log.Printf("Registration failed with primary %s: %v", url, err)
			continue
		}
		l.mu.Lock()
		l.activeMobiusURL = url
		l.mu.Unlock()
		l.onRegistered()
		return nil
	}

	// Fall back to backup servers
	for _, url := range l.backupMobiusURLs {
		if err := l.attemptRegistration(url); err != nil {
			log.Printf("Registration failed with backup %s: %v", url, err)
			continue
		}
		l.mu.Lock()
		l.activeMobiusURL = url
		l.mu.Unlock()
		l.onRegistered()
		return nil
	}

	l.mu.Lock()
	l.status = RegistrationStatusInactive
	l.mu.Unlock()
	l.Emitter.Emit(string(LineEventError), fmt.Errorf("registration failed with all servers"))

	return fmt.Errorf("registration failed with all Mobius servers")
}

// attemptRegistration sends a POST to register a device with a Mobius server
func (l *Line) attemptRegistration(mobiusURL string) error {
	payload := map[string]interface{}{
		"userId":          l.UserID,
		"clientDeviceUri": l.clientDeviceURI,
		"serviceData": map[string]string{
			"indicator": "calling",
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling registration payload: %w", err)
	}

	url := fmt.Sprintf("%sdevice", mobiusURL)
	log.Printf("Registration POST %s payload: %s", url, string(payloadBytes))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating registration request: %w", err)
	}

	trackingID := fmt.Sprintf("webex-go-sdk_%s", uuid.New().String())
	req.Header.Set("Authorization", "Bearer "+l.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/beta")
	req.Header.Set("trackingid", trackingID)
	if l.clientDeviceURI != "" {
		req.Header.Set("cisco-device-url", l.clientDeviceURI)
	}

	log.Printf("Registration: sending POST to %s ...", url)
	resp, err := l.core.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("error making registration request: %w", err)
	}
	defer resp.Body.Close()
	log.Printf("Registration: received response status %d from %s", resp.StatusCode, url)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading registration response: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		// 403 with errorCode 101 means device already registered — delete old and retry
		var errResp struct {
			ErrorCode int `json:"errorCode"`
			Devices   []struct {
				DeviceID string `json:"deviceId"`
				URI      string `json:"uri"`
			} `json:"devices"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.ErrorCode == 101 && len(errResp.Devices) > 0 {
			log.Printf("Device already registered (errorCode 101), deleting existing device %s and re-registering", errResp.Devices[0].DeviceID)
			if delErr := l.deleteDevice(mobiusURL, errResp.Devices[0].DeviceID); delErr != nil {
				log.Printf("Failed to delete existing device: %v", delErr)
			} else {
				// Retry registration after deleting old device
				return l.attemptRegistration(mobiusURL)
			}
		}
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceInfo MobiusDeviceInfo
	if err := json.Unmarshal(body, &deviceInfo); err != nil {
		return fmt.Errorf("error parsing registration response: %w", err)
	}

	l.mu.Lock()
	l.deviceInfo = &deviceInfo
	if deviceInfo.Device != nil {
		l.MobiusDeviceID = deviceInfo.Device.DeviceID
	}
	if deviceInfo.UserID != "" {
		l.UserID = deviceInfo.UserID
	}
	if deviceInfo.KeepaliveInterval > 0 {
		l.keepaliveInterval = time.Duration(deviceInfo.KeepaliveInterval) * time.Second
	}
	l.mu.Unlock()

	return nil
}

// deleteDevice deletes an existing Mobius device registration by deviceId.
// This is used when a 403 errorCode 101 is received (device already registered).
func (l *Line) deleteDevice(mobiusURL string, deviceID string) error {
	url := fmt.Sprintf("%sdevices/%s", mobiusURL, deviceID)
	log.Printf("Deleting existing Mobius device: DELETE %s", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error creating delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.core.GetAccessToken())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/beta")
	req.Header.Set("trackingid", fmt.Sprintf("webex-go-sdk_%s", uuid.New().String()))
	if l.clientDeviceURI != "" {
		req.Header.Set("cisco-device-url", l.clientDeviceURI)
	}

	resp, err := l.core.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("error making delete request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Successfully deleted existing device %s", deviceID)
		return nil
	}

	return fmt.Errorf("delete device failed with status %d: %s", resp.StatusCode, string(body))
}

// onRegistered is called after successful registration
func (l *Line) onRegistered() {
	l.mu.Lock()
	l.status = RegistrationStatusActive
	l.mu.Unlock()

	l.startKeepalive()
	l.Emitter.Emit(string(LineEventRegistered), l)
}

// Deregister deregisters this line from the Mobius server
func (l *Line) Deregister() error {
	l.mu.RLock()
	if l.status != RegistrationStatusActive {
		l.mu.RUnlock()
		return nil
	}
	mobiusURL := l.activeMobiusURL
	deviceID := l.MobiusDeviceID
	l.mu.RUnlock()

	l.stopKeepalive()

	if mobiusURL == "" || deviceID == "" {
		l.mu.Lock()
		l.status = RegistrationStatusInactive
		l.mu.Unlock()
		return nil
	}

	url := fmt.Sprintf("%sdevices/%s", mobiusURL, deviceID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error creating deregister request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.core.GetAccessToken())
	req.Header.Set("spark-user-agent", "webex-calling/beta")
	if l.clientDeviceURI != "" {
		req.Header.Set("cisco-device-url", l.clientDeviceURI)
	}
	req.Header.Set("trackingId", fmt.Sprintf("webex-web-client_%s", uuid.New().String()))

	resp, err := l.core.GetHTTPClient().Do(req)
	if err != nil {
		log.Printf("Deregister request failed: %v", err)
	} else {
		resp.Body.Close()
	}

	l.mu.Lock()
	l.status = RegistrationStatusInactive
	l.mu.Unlock()

	l.Emitter.Emit(string(LineEventUnregistered), nil)
	return nil
}

// GetStatus returns the current registration status
func (l *Line) GetStatus() RegistrationStatus {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status
}

// GetDeviceID returns the Mobius device ID
func (l *Line) GetDeviceID() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.MobiusDeviceID
}

// GetActiveMobiusURL returns the active Mobius server URL
func (l *Line) GetActiveMobiusURL() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.activeMobiusURL
}

// GetDeviceInfo returns the device info from Mobius registration
func (l *Line) GetDeviceInfo() *MobiusDeviceInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.deviceInfo
}

// IsRegistered returns true if the line is currently registered
func (l *Line) IsRegistered() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.status == RegistrationStatusActive
}

// startKeepalive starts the periodic keepalive timer
func (l *Line) startKeepalive() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.keepaliveTicker != nil {
		return
	}

	l.keepaliveStop = make(chan struct{})
	l.keepaliveTicker = time.NewTicker(l.keepaliveInterval)

	// Capture local references so the goroutine doesn't access
	// fields that may be nilled out by stopKeepalive.
	ticker := l.keepaliveTicker
	stop := l.keepaliveStop

	go func() {
		for {
			select {
			case <-ticker.C:
				l.sendKeepalive()
			case <-stop:
				return
			}
		}
	}()
}

// stopKeepalive stops the periodic keepalive timer
func (l *Line) stopKeepalive() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.keepaliveTicker != nil {
		l.keepaliveTicker.Stop()
		l.keepaliveTicker = nil
	}
	if l.keepaliveStop != nil {
		close(l.keepaliveStop)
		l.keepaliveStop = nil
	}
}

// sendKeepalive sends a keepalive PUT to the Mobius server
func (l *Line) sendKeepalive() {
	l.mu.RLock()
	mobiusURL := l.activeMobiusURL
	deviceID := l.MobiusDeviceID
	deviceURI := ""
	if l.deviceInfo != nil && l.deviceInfo.Device != nil {
		deviceURI = l.deviceInfo.Device.URI
	}
	l.mu.RUnlock()

	if mobiusURL == "" || deviceURI == "" {
		return
	}

	url := fmt.Sprintf("%sdevices/%s", mobiusURL, deviceID)

	payload := map[string]interface{}{
		"deviceId": deviceID,
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Error creating keepalive request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+l.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/beta")

	resp, err := l.core.GetHTTPClient().Do(req)
	if err != nil {
		log.Printf("Keepalive request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("Keepalive returned 404, device may have been deregistered")
		l.mu.Lock()
		l.status = RegistrationStatusInactive
		l.mu.Unlock()
		l.Emitter.Emit(string(LineEventUnregistered), nil)
	}
}

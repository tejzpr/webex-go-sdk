/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

// DeviceResponse represents the response from the device registration
type DeviceResponse struct {
	URL          string `json:"url"`
	WebSocketURL string `json:"webSocketUrl"`
}

// Config holds the configuration for the Device plugin
type Config struct {
	// Ephemeral determines if the device is temporary and should be refreshed
	Ephemeral bool
	// EphemeralDeviceTTL is the time to live for ephemeral devices in seconds
	EphemeralDeviceTTL int
	// DeviceType specifies the type of device
	DeviceType string
	// DefaultHeaders to include in requests
	DefaultHeaders map[string]string
	// DefaultBody to include in requests
	DefaultBody map[string]interface{}
}

// DefaultConfig returns the default configuration for the Device plugin
func DefaultConfig() *Config {
	return &Config{
		Ephemeral:          false,
		EphemeralDeviceTTL: 86400, // 24 hours
		DeviceType:         "WEB",
		DefaultHeaders:     make(map[string]string),
		DefaultBody:        make(map[string]interface{}),
	}
}

// DeviceDTO represents the response from the WDM service for a device
type DeviceDTO struct {
	URL                         string      `json:"url,omitempty"`
	WebSocketURL                string      `json:"webSocketUrl,omitempty"`
	UserID                      string      `json:"userId,omitempty"`
	DeviceType                  string      `json:"deviceType,omitempty"`
	IntranetInactivityDuration  int         `json:"intranetInactivityDuration,omitempty"`
	InNetworkInactivityDuration int         `json:"inNetworkInactivityDuration,omitempty"`
	ModificationTime            string      `json:"modificationTime,omitempty"`
	Services                    interface{} `json:"services,omitempty"`
	ServiceHostMap              interface{} `json:"serviceHostMap,omitempty"`
	WebFileShareControl         string      `json:"webFileShareControl,omitempty"`
	ClientMessagingGiphy        string      `json:"clientMessagingGiphy,omitempty"`
	ETag                        string      `json:"-"`
}

// Client is the Device API client
type Client struct {
	webexClient     *webexsdk.Client
	config          *Config
	device          *DeviceDTO
	deviceInfo      *DeviceResponse
	registered      bool
	registering     bool
	refreshTimer    *time.Timer
	mu              sync.Mutex
	registrationCbs []func()
}

// New creates a new Device plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
		device:      &DeviceDTO{},
	}
}

// Register registers a device with Webex to get a WebSocket URL
func (c *Client) Register() error {
	c.mu.Lock()
	if c.deviceInfo != nil {
		c.mu.Unlock()
		return nil
	}
	c.registering = true
	c.mu.Unlock()

	// Create the payload - keeping these values exactly as required
	payload := map[string]interface{}{
		"deviceType":     "TEAMS_SDK_JS",
		"name":           "Webex SDK",
		"model":          "Webex Go SDK",
		"localizedModel": "Cisco Webex Teams",
		"systemName":     "Webex Go SDK",
		"systemVersion":  "1.0.0",
	}

	// Convert payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	// Create the request
	req, err := http.NewRequest(http.MethodPost, "https://wdm-a.wbx2.com/wdm/api/v1/devices", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("TrackingID", fmt.Sprintf("GoSDK_%d", time.Now().UnixMilli()))

	// Add query parameters
	q := req.URL.Query()
	q.Add("includeUpstreamServices", "all")
	req.URL.RawQuery = q.Encode()

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Check for error
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error response: %s", respBody)
	}

	// Parse the response
	var deviceResp DeviceResponse
	if err := json.Unmarshal(respBody, &deviceResp); err != nil {
		return fmt.Errorf("error parsing response: %w", err)
	}

	// Store the device info and update status
	c.mu.Lock()
	c.deviceInfo = &deviceResp
	c.registered = true
	c.registering = false

	// Trigger registration callbacks
	callbacks := make([]func(), len(c.registrationCbs))
	copy(callbacks, c.registrationCbs)
	c.mu.Unlock()

	// Call registration callbacks outside the lock
	for _, cb := range callbacks {
		go cb()
	}

	return nil
}

// Unregister unregisters a device with Webex
func (c *Client) Unregister() error {
	c.mu.Lock()
	if c.deviceInfo == nil || c.deviceInfo.URL == "" {
		c.mu.Unlock()
		return nil
	}
	deviceURL := c.deviceInfo.URL
	c.mu.Unlock()

	// Create the request
	req, err := http.NewRequest(http.MethodDelete, deviceURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check for error
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response: %s", respBody)
	}

	// Clear the device info
	c.mu.Lock()
	c.deviceInfo = nil
	c.registered = false
	c.mu.Unlock()

	return nil
}

// GetWebSocketURL returns the WebSocket URL for Mercury connections
func (c *Client) GetWebSocketURL() (string, error) {
	c.mu.Lock()
	deviceInfo := c.deviceInfo
	c.mu.Unlock()

	// Ensure we have device info
	if deviceInfo == nil {
		if err := c.Register(); err != nil {
			return "", fmt.Errorf("failed to register device: %w", err)
		}
		c.mu.Lock()
		deviceInfo = c.deviceInfo
		c.mu.Unlock()
	}

	// Return the WebSocket URL
	return deviceInfo.WebSocketURL, nil
}

// GetDeviceURL returns the device URL
func (c *Client) GetDeviceURL() (string, error) {
	c.mu.Lock()
	deviceInfo := c.deviceInfo
	c.mu.Unlock()

	// Ensure we have device info
	if deviceInfo == nil {
		if err := c.Register(); err != nil {
			return "", fmt.Errorf("failed to register device: %w", err)
		}
		c.mu.Lock()
		deviceInfo = c.deviceInfo
		c.mu.Unlock()
	}

	// Return the device URL
	return deviceInfo.URL, nil
}

// OnRegistered registers a callback function to be called when the device is registered
func (c *Client) OnRegistered(callback func()) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.registrationCbs = append(c.registrationCbs, callback)

	// If already registered, call the callback immediately
	if c.registered {
		go callback()
	}
}

// IsRegistered returns true if the device is registered
func (c *Client) IsRegistered() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.registered
}

// WaitForRegistration waits for the device to be registered with a timeout
func (c *Client) WaitForRegistration(timeout time.Duration) error {
	if c.IsRegistered() {
		return nil
	}

	waitCh := make(chan struct{})
	timeoutCh := time.After(timeout)

	// Register callback
	c.OnRegistered(func() {
		close(waitCh)
	})

	// Wait for either registration or timeout
	select {
	case <-waitCh:
		return nil
	case <-timeoutCh:
		return fmt.Errorf("timeout waiting for device registration")
	}
}

// GetDevice returns a copy of the current device data
func (c *Client) GetDevice() DeviceDTO {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to prevent modification of the original
	return *c.device
}

// setupRefreshTimer creates or resets the refresh timer
func (c *Client) setupRefreshTimer() {
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
	}

	refreshTime := time.Duration(c.config.EphemeralDeviceTTL/2+60) * time.Second
	c.refreshTimer = time.AfterFunc(refreshTime, func() {
		if err := c.Refresh(); err != nil {
			// Log error but don't stop the timer
			// Using fmt.Printf since we don't want to add a logger dependency
			fmt.Printf("Error refreshing device: %v\n", err)
		}
	})
}

// Refresh refreshes the device registration with the Webex service
func (c *Client) Refresh() error {
	c.mu.Lock()
	if !c.registered {
		c.mu.Unlock()
		return fmt.Errorf("device not registered, cannot refresh")
	}

	deviceURL := c.device.URL
	etag := c.device.ETag
	c.mu.Unlock()

	// Prepare headers
	headers := make(map[string]string)
	for k, v := range c.config.DefaultHeaders {
		headers[k] = v
	}

	// Add If-None-Match header if we have an ETag
	if etag != "" {
		headers["If-None-Match"] = etag
	}

	// Make the refresh request
	resp, err := c.webexClient.Request(http.MethodPut, deviceURL, nil, nil)
	if err != nil {
		return err
	}

	// Check if there was no change (304 Not Modified)
	if resp.StatusCode == http.StatusNotModified {
		// Reset refresh timer for ephemeral devices
		if c.config.Ephemeral && c.config.EphemeralDeviceTTL > 0 {
			c.mu.Lock()
			c.setupRefreshTimer()
			c.mu.Unlock()
		}
		return nil
	}

	// Parse the response body
	var deviceDTO DeviceDTO
	if err := webexsdk.ParseResponse(resp, &deviceDTO); err != nil {
		return err
	}

	// Extract ETag from headers if present
	if etag := resp.Header.Get("ETag"); etag != "" {
		deviceDTO.ETag = etag
	}

	// Update device data
	c.mu.Lock()
	c.device = &deviceDTO

	// Reset refresh timer for ephemeral devices
	if c.config.Ephemeral && c.config.EphemeralDeviceTTL > 0 {
		c.setupRefreshTimer()
	}
	c.mu.Unlock()

	return nil
}

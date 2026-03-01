/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/WebexCommunity/webex-go-sdk/v2/mercury"
	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	"github.com/google/uuid"
)

// CallingClient is the main orchestrator for Webex Calling call control.
// It manages line registration with Mobius servers, call creation, and
// incoming call handling via Mercury WebSocket events.
type CallingClient struct {
	mu sync.RWMutex

	core   *webexsdk.Client
	config *Config

	// Lines
	lines map[string]*Line

	// Active calls keyed by correlationID
	activeCalls map[string]*Call

	// Mobius discovery
	primaryMobiusURLs []string
	backupMobiusURLs  []string

	// Client device URI (from Webex internal device registration)
	clientDeviceURI string

	// User ID (from Webex internal device registration)
	userID string

	// WDM WebSocket URL (for Mercury connection to same device)
	wdmWebSocketURL string

	// Media config
	mediaConfig *MediaConfig

	// Mercury client for Mobius event delivery
	mercuryClient *mercury.Client

	// Audio bridge for automatic call↔bridge binding
	audioBridge *AudioBridge

	// Events
	Emitter *EventEmitter
}

// CallingClientConfig holds configuration for the CallingClient
type CallingClientConfig struct {
	// ClientDeviceURI is the Webex device URL (from device registration)
	ClientDeviceURI string
	// MediaConfig is the WebRTC media configuration
	MediaConfig *MediaConfig
	// DiscoveryRegion overrides the region for Mobius server discovery
	DiscoveryRegion string
	// DiscoveryCountry overrides the country for Mobius server discovery
	DiscoveryCountry string
}

// NewCallingClient creates a new CallingClient for managing lines and calls
func NewCallingClient(core *webexsdk.Client, config *Config, clientConfig *CallingClientConfig) *CallingClient {
	if config == nil {
		config = DefaultConfig()
	}

	cc := &CallingClient{
		core:        core,
		config:      config,
		lines:       make(map[string]*Line),
		activeCalls: make(map[string]*Call),
		Emitter:     NewEventEmitter(),
		mediaConfig: DefaultMediaConfig(),
	}

	if clientConfig != nil {
		cc.clientDeviceURI = clientConfig.ClientDeviceURI
		if clientConfig.MediaConfig != nil {
			cc.mediaConfig = clientConfig.MediaConfig
		}
	}

	return cc
}

// DiscoverMobiusServers discovers the Mobius servers for the user's region.
// It follows the same flow as the JS SDK:
//  1. Register a WDM device to get clientDeviceUri and serviceHostMap (Mobius hosts)
//  2. Get region info from the Webex region discovery service
//  3. Query each Mobius host with region/country to get primary/backup URIs
func (cc *CallingClient) DiscoverMobiusServers() error {
	// Step 1: Register WDM device to get clientDeviceUri and Mobius hosts
	mobiusHosts, err := cc.registerWDMDevice()
	if err != nil {
		log.Printf("WDM device registration failed: %v", err)
	}
	if len(mobiusHosts) == 0 {
		// Fallback: use the well-known EU Mobius host (which resolves)
		mobiusHosts = []string{"mobius-eu-central-1.prod.infra.webex.com"}
	}

	// Step 2: Get region info
	regionInfo, err := cc.getRegionInfo()
	if err != nil {
		log.Printf("Region discovery failed: %v, will query Mobius without region", err)
	}

	// Step 3: Query Mobius discovery endpoint for primary/backup URIs
	for _, host := range mobiusHosts {
		mobiusBase := fmt.Sprintf("https://%s/api/v1", host)
		discoveryURL := fmt.Sprintf("%s/calling/web/", mobiusBase)

		if regionInfo != nil && regionInfo.ClientRegion != "" && regionInfo.CountryCode != "" {
			discoveryURL = fmt.Sprintf("%s?regionCode=%s&countryCode=%s",
				discoveryURL, regionInfo.ClientRegion, regionInfo.CountryCode)
		}

		req, err := http.NewRequest(http.MethodGet, discoveryURL, nil)
		if err != nil {
			continue
		}
		trackingID := fmt.Sprintf("webex-go-sdk_%s", uuid.New().String())
		req.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("spark-user-agent", "webex-calling/beta")
		req.Header.Set("trackingid", trackingID)
		if cc.clientDeviceURI != "" {
			req.Header.Set("cisco-device-url", cc.clientDeviceURI)
		}

		log.Printf("Mobius discovery request: GET %s (cisco-device-url=%s, trackingId=%s)", discoveryURL, cc.clientDeviceURI, trackingID)

		resp, err := cc.core.GetHTTPClient().Do(req)
		if err != nil {
			log.Printf("Mobius discovery failed for host %s: %v", host, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Mobius discovery returned %d for host %s: %s", resp.StatusCode, host, string(body))
			continue
		}

		log.Printf("Mobius discovery raw response: %s", string(body))

		// Parse the Mobius discovery response.
		// Format: { "primary": { "region": "...", "uris": ["https://..."] },
		//           "backup":  { "region": "...", "uris": ["https://..."] } }
		var mobiusServers struct {
			Primary struct {
				Region string   `json:"region"`
				URIs   []string `json:"uris"`
			} `json:"primary"`
			Backup struct {
				Region string   `json:"region"`
				URIs   []string `json:"uris"`
			} `json:"backup"`
		}
		if err := json.Unmarshal(body, &mobiusServers); err != nil {
			log.Printf("Failed to parse Mobius discovery response: %v, body: %s", err, string(body))
			// Use this host directly as the Mobius URL
			cc.mu.Lock()
			cc.primaryMobiusURLs = []string{mobiusBase + "/calling/web/"}
			cc.mu.Unlock()
			log.Printf("Using Mobius host directly: %s", mobiusBase)
			return nil
		}

		cc.mu.Lock()
		// Build primary URLs: each URI + /calling/web/
		cc.primaryMobiusURLs = make([]string, 0, len(mobiusServers.Primary.URIs))
		for _, uri := range mobiusServers.Primary.URIs {
			cc.primaryMobiusURLs = append(cc.primaryMobiusURLs, uri+"/calling/web/")
		}
		// Build backup URLs
		cc.backupMobiusURLs = make([]string, 0, len(mobiusServers.Backup.URIs))
		for _, uri := range mobiusServers.Backup.URIs {
			cc.backupMobiusURLs = append(cc.backupMobiusURLs, uri+"/calling/web/")
		}
		// If discovery returned empty lists, use this host directly
		if len(cc.primaryMobiusURLs) == 0 {
			cc.primaryMobiusURLs = []string{mobiusBase + "/calling/web/"}
		}
		cc.mu.Unlock()

		log.Printf("Discovered Mobius servers: primary=%v backup=%v", cc.primaryMobiusURLs, cc.backupMobiusURLs)
		return nil
	}

	// Final fallback: use the first host directly
	if len(mobiusHosts) > 0 {
		cc.mu.Lock()
		cc.primaryMobiusURLs = []string{
			fmt.Sprintf("https://%s/api/v1/calling/web/", mobiusHosts[0]),
		}
		cc.mu.Unlock()
		log.Printf("Using fallback Mobius URL: %s", cc.primaryMobiusURLs[0])
		return nil
	}

	return fmt.Errorf("failed to discover any Mobius servers")
}

// regionInfoResult holds the result of region discovery
type regionInfoResult struct {
	CountryCode  string `json:"countryCode"`
	ClientRegion string `json:"clientRegion"`
	RegionCode   string `json:"regionCode"`
}

// getRegionInfo fetches the client's region from the Webex region discovery service
func (cc *CallingClient) getRegionInfo() (*regionInfoResult, error) {
	url := cc.config.RegionDiscoveryURL
	if url == "" {
		url = "https://ds.ciscospark.com/v1/region"
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
	req.Header.Set("spark-user-agent", "webex-calling/go-sdk (web)")

	resp, err := cc.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("region discovery returned status %d: %s", resp.StatusCode, string(body))
	}

	var info regionInfoResult
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	// Use regionCode if clientRegion is empty
	if info.ClientRegion == "" && info.RegionCode != "" {
		info.ClientRegion = info.RegionCode
	}

	log.Printf("Region discovered: region=%s country=%s", info.ClientRegion, info.CountryCode)
	return &info, nil
}

// registerWDMDevice registers a WDM device and extracts the clientDeviceUri and
// Mobius hosts from the serviceHostMap in the WDM response.
// This is the same approach the JS SDK uses: webex.internal.device.url provides
// the clientDeviceUri, and webex.internal.services._hostCatalog provides Mobius hosts.
func (cc *CallingClient) registerWDMDevice() ([]string, error) {
	// If clientDeviceURI is already set, skip WDM registration
	if cc.clientDeviceURI != "" {
		log.Printf("Using provided clientDeviceUri: %s", cc.clientDeviceURI)
		return nil, nil
	}

	payload := map[string]interface{}{
		"deviceType":     "TEAMS_SDK_JS",
		"name":           "Webex Go SDK Calling",
		"model":          "Webex Go SDK",
		"localizedModel": "Cisco Webex Teams",
		"systemName":     "Webex Go SDK",
		"systemVersion":  "1.0.0",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling WDM payload: %w", err)
	}

	baseWDMURL := cc.config.WDMURL
	if baseWDMURL == "" {
		baseWDMURL = "https://wdm-a.wbx2.com/wdm/api/v1/devices"
	}
	wdmURL := baseWDMURL + "?includeUpstreamServices=all"
	req, err := http.NewRequest(http.MethodPost, wdmURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating WDM request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/go-sdk (web)")

	resp, err := cc.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making WDM request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading WDM response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("WDM registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the WDM response to get device URL and serviceHostMap
	var wdmResp struct {
		URL            string `json:"url"`
		WebSocketURL   string `json:"webSocketUrl"`
		UserID         string `json:"userId"`
		ServiceHostMap struct {
			HostCatalog map[string][]struct {
				Host     string `json:"host"`
				ID       string `json:"id"`
				TTL      int    `json:"ttl"`
				Priority int    `json:"priority"`
			} `json:"hostCatalog"`
			ServiceLinks map[string]string `json:"serviceLinks"`
		} `json:"serviceHostMap"`
		// V2 services - can be array or object depending on WDM version
		Services json.RawMessage `json:"services"`
	}
	if err := json.Unmarshal(body, &wdmResp); err != nil {
		return nil, fmt.Errorf("error parsing WDM response: %w", err)
	}

	// Set clientDeviceURI from the WDM device URL
	if wdmResp.URL != "" {
		cc.mu.Lock()
		cc.clientDeviceURI = wdmResp.URL
		if wdmResp.UserID != "" {
			cc.userID = wdmResp.UserID
		}
		if wdmResp.WebSocketURL != "" {
			cc.wdmWebSocketURL = wdmResp.WebSocketURL
		}
		cc.mu.Unlock()
		log.Printf("WDM device registered: url=%s userId=%s wsUrl=%s", wdmResp.URL, wdmResp.UserID, wdmResp.WebSocketURL)
	}

	// Debug: log all hostCatalog keys to find Mobius entries
	if len(wdmResp.ServiceHostMap.HostCatalog) > 0 {
		var keys []string
		for k := range wdmResp.ServiceHostMap.HostCatalog {
			keys = append(keys, k)
		}
		log.Printf("WDM serviceHostMap.hostCatalog keys (%d): %v", len(keys), keys)
	} else {
		log.Printf("WDM serviceHostMap.hostCatalog is empty")
	}

	// Debug: log serviceLinks
	if len(wdmResp.ServiceHostMap.ServiceLinks) > 0 {
		for k, v := range wdmResp.ServiceHostMap.ServiceLinks {
			if contains(k, "mobius") || contains(k, "call") {
				log.Printf("WDM serviceLink: %s = %s", k, v)
			}
		}
	}

	// Extract Mobius hosts from serviceHostMap.hostCatalog.
	// The JS SDK filters by entry.id ending with ":mobius".
	var mobiusHosts []string
	seen := make(map[string]bool)

	for key, entries := range wdmResp.ServiceHostMap.HostCatalog {
		for _, entry := range entries {
			// Match the JS SDK: entry.id.endsWith(':mobius')
			if len(entry.ID) >= 7 && entry.ID[len(entry.ID)-7:] == ":mobius" {
				// Prefer entry.host if present, else use the map key
				host := entry.Host
				if host == "" {
					host = key
				}
				if host != "" && !seen[host] {
					seen[host] = true
					mobiusHosts = append(mobiusHosts, host)
				}
			}
		}
	}

	// Also check v2 services for serviceName=mobius (may be array or object)
	if len(wdmResp.Services) > 0 {
		var v2Services []struct {
			ServiceName string `json:"serviceName"`
			ServiceURLs []struct {
				BaseURL  string `json:"baseUrl"`
				Priority int    `json:"priority"`
			} `json:"serviceUrls"`
		}
		if json.Unmarshal(wdmResp.Services, &v2Services) == nil {
			for _, svc := range v2Services {
				if svc.ServiceName == "mobius" {
					for _, su := range svc.ServiceURLs {
						if su.BaseURL != "" {
							host := su.BaseURL
							host = strings.TrimPrefix(host, "https://")
							host = strings.TrimPrefix(host, "http://")
							if idx := strings.Index(host, "/"); idx > 0 {
								host = host[:idx]
							}
							if host != "" && !seen[host] {
								seen[host] = true
								mobiusHosts = append(mobiusHosts, host)
							}
						}
					}
				}
			}
		}
	}

	// Also check serviceLinks for a direct mobius URL
	if mobiusLink, ok := wdmResp.ServiceHostMap.ServiceLinks["mobius"]; ok && mobiusLink != "" {
		host := mobiusLink
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		if idx := strings.Index(host, "/"); idx > 0 {
			host = host[:idx]
		}
		if host != "" && !seen[host] {
			seen[host] = true
			mobiusHosts = append(mobiusHosts, host)
		}
	}

	if len(mobiusHosts) > 0 {
		log.Printf("Found %d Mobius hosts from WDM: %v", len(mobiusHosts), mobiusHosts)
	} else {
		log.Printf("No Mobius hosts found in WDM response, will use defaults")
	}

	return mobiusHosts, nil
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// DeregisterAllDevices attempts to deregister all existing Mobius devices for this user.
// It does this by attempting a registration POST — if a 403 errorCode 101 is returned,
// it extracts the existing device list and deletes each one.
func (cc *CallingClient) DeregisterAllDevices() (int, error) {
	cc.mu.RLock()
	primary := cc.primaryMobiusURLs
	deviceURI := cc.clientDeviceURI
	userID := cc.userID
	cc.mu.RUnlock()

	if len(primary) == 0 {
		return 0, fmt.Errorf("no Mobius servers configured; call DiscoverMobiusServers() first")
	}

	mobiusURL := primary[0]
	deleted := 0

	// Attempt registration to provoke a 403 with existing device list
	payload := map[string]interface{}{
		"userId":          userID,
		"clientDeviceUri": deviceURI,
		"serviceData": map[string]string{
			"indicator": "calling",
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	url := fmt.Sprintf("%sdevice", mobiusURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, err
	}

	trackingID := fmt.Sprintf("webex-go-sdk_%s", uuid.New().String())
	req.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/beta")
	req.Header.Set("trackingid", trackingID)
	if deviceURI != "" {
		req.Header.Set("cisco-device-url", deviceURI)
	}

	resp, err := cc.core.GetHTTPClient().Do(req)
	if err != nil {
		return 0, err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		var errResp struct {
			ErrorCode int `json:"errorCode"`
			Devices   []struct {
				DeviceID string `json:"deviceId"`
			} `json:"devices"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.ErrorCode == 101 {
			for _, dev := range errResp.Devices {
				delURL := fmt.Sprintf("%sdevices/%s", mobiusURL, dev.DeviceID)
				log.Printf("Deregister all: deleting device %s", dev.DeviceID)
				delReq, err := http.NewRequest(http.MethodDelete, delURL, nil)
				if err != nil {
					continue
				}
				delReq.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
				delReq.Header.Set("Accept", "application/json")
				delReq.Header.Set("spark-user-agent", "webex-calling/beta")
				delReq.Header.Set("trackingid", fmt.Sprintf("webex-go-sdk_%s", uuid.New().String()))
				if deviceURI != "" {
					delReq.Header.Set("cisco-device-url", deviceURI)
				}
				delResp, err := cc.core.GetHTTPClient().Do(delReq)
				if err != nil {
					log.Printf("Failed to delete device %s: %v", dev.DeviceID, err)
					continue
				}
				_ = delResp.Body.Close()
				log.Printf("Deleted device %s (status %d)", dev.DeviceID, delResp.StatusCode)
				deleted++
			}
		}
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Registration succeeded — no stale devices. Delete the one we just created.
		var regResp struct {
			Device struct {
				DeviceID string `json:"deviceId"`
			} `json:"device"`
		}
		if json.Unmarshal(body, &regResp) == nil && regResp.Device.DeviceID != "" {
			delURL := fmt.Sprintf("%sdevices/%s", mobiusURL, regResp.Device.DeviceID)
			delReq, _ := http.NewRequest(http.MethodDelete, delURL, nil)
			delReq.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
			delReq.Header.Set("spark-user-agent", "webex-calling/beta")
			delReq.Header.Set("trackingId", fmt.Sprintf("webex-go-sdk_%s", uuid.New().String()))
			if deviceURI != "" {
				delReq.Header.Set("cisco-device-url", deviceURI)
			}
			delResp, err := cc.core.GetHTTPClient().Do(delReq)
			if err == nil {
				_ = delResp.Body.Close()
				deleted++
			}
		}
	}

	log.Printf("Deregister all: deleted %d devices", deleted)
	return deleted, nil
}

// SetMobiusServers manually sets the primary and backup Mobius server URLs
func (cc *CallingClient) SetMobiusServers(primary, backup []string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.primaryMobiusURLs = primary
	cc.backupMobiusURLs = backup
}

// CreateLine creates and registers a new line with the Mobius servers.
// Returns the Line object which can be used to make and receive calls.
func (cc *CallingClient) CreateLine() (*Line, error) {
	cc.mu.RLock()
	primary := cc.primaryMobiusURLs
	backup := cc.backupMobiusURLs
	deviceURI := cc.clientDeviceURI
	cc.mu.RUnlock()

	if len(primary) == 0 && len(backup) == 0 {
		return nil, fmt.Errorf("no Mobius servers configured; call DiscoverMobiusServers() or SetMobiusServers() first")
	}

	cc.mu.RLock()
	userID := cc.userID
	cc.mu.RUnlock()

	line := NewLine(cc.core, cc.config, &LineConfig{
		PrimaryMobiusURLs: primary,
		BackupMobiusURLs:  backup,
		ClientDeviceURI:   deviceURI,
		UserID:            userID,
	})

	if err := line.Register(); err != nil {
		return nil, fmt.Errorf("line registration failed: %w", err)
	}

	cc.mu.Lock()
	cc.lines[line.LineID] = line
	cc.mu.Unlock()

	// Listen for incoming calls on this line
	line.Emitter.On(string(LineEventIncomingCall), func(data interface{}) {
		if call, ok := data.(*Call); ok {
			cc.mu.Lock()
			cc.activeCalls[call.GetCorrelationID()] = call
			cc.mu.Unlock()
			cc.Emitter.Emit(string(LineEventIncomingCall), call)
		}
	})

	return line, nil
}

// GetLines returns all registered lines
func (cc *CallingClient) GetLines() map[string]*Line {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	result := make(map[string]*Line, len(cc.lines))
	for k, v := range cc.lines {
		result[k] = v
	}
	return result
}

// MakeCall creates and dials an outbound call on the specified line
func (cc *CallingClient) MakeCall(line *Line, destination *CallDetails) (*Call, error) {
	if line == nil {
		return nil, fmt.Errorf("line is required")
	}
	if !line.IsRegistered() {
		return nil, fmt.Errorf("line is not registered")
	}
	if destination == nil {
		return nil, fmt.Errorf("destination is required")
	}

	call, err := NewCall(cc.core, CallDirectionOutbound, destination, &CallConfig{
		MobiusURL:       line.GetActiveMobiusURL(),
		DeviceID:        line.GetDeviceID(),
		LineID:          line.LineID,
		ClientDeviceURI: cc.clientDeviceURI,
		MediaConfig:     cc.mediaConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	cc.mu.Lock()
	cc.activeCalls[call.GetCorrelationID()] = call
	cc.mu.Unlock()

	if err := call.Dial(); err != nil {
		cc.mu.Lock()
		delete(cc.activeCalls, call.GetCorrelationID())
		cc.mu.Unlock()
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	// Auto-attach AudioBridge if one is registered
	cc.mu.RLock()
	bridge := cc.audioBridge
	cc.mu.RUnlock()
	if bridge != nil {
		bridge.AttachCall(call)
	}

	// Listen for disconnect to clean up call and detach bridge
	call.Emitter.On(string(CallEventDisconnect), func(data interface{}) {
		cc.mu.Lock()
		delete(cc.activeCalls, call.GetCorrelationID())
		ab := cc.audioBridge
		cc.mu.Unlock()
		if ab != nil {
			ab.DetachCall()
		}
	})

	return call, nil
}

// GetActiveCalls returns all active calls
func (cc *CallingClient) GetActiveCalls() map[string]*Call {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	result := make(map[string]*Call, len(cc.activeCalls))
	for k, v := range cc.activeCalls {
		result[k] = v
	}
	return result
}

// GetWDMWebSocketURL returns the WebSocket URL from the WDM device registration.
// This should be used for Mercury connections to ensure events are received
// on the same device used for Mobius registration.
func (cc *CallingClient) GetWDMWebSocketURL() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.wdmWebSocketURL
}

// ConnectMercury connects a Mercury WebSocket client and wires it to receive
// Mobius call events (ROAP answers, call progress, etc.). It uses the WDM
// WebSocket URL from line registration so events arrive on the correct device.
//
// This is the idiomatic way to set up Mercury for calling — replaces the
// manual wildcard handler + event routing that consumers previously had to do.
func (cc *CallingClient) ConnectMercury(merc *mercury.Client) error {
	// Use the same WDM device's WebSocket URL that was used for Mobius registration
	wsURL := cc.GetWDMWebSocketURL()
	if wsURL != "" {
		log.Printf("CallingClient: Mercury using WDM WebSocket URL: %s", wsURL)
		merc.SetCustomWebSocketURL(wsURL)
	} else {
		log.Println("CallingClient: WARNING - no WDM WebSocket URL, Mercury using default device")
	}

	// Clear existing wildcard handlers to prevent duplicates on re-registration
	merc.ClearHandlers("*")

	// Register a wildcard handler to route Mobius events to this CallingClient
	merc.On("*", func(event *mercury.Event) {
		if event == nil || event.Data == nil {
			return
		}
		eventType, _ := event.Data["eventType"].(string)
		log.Printf("CallingClient: Mercury event: eventType=%s id=%s", eventType, event.ID)
		if strings.HasPrefix(eventType, "mobius.") {
			log.Printf("CallingClient: routing Mobius event: %s", eventType)
			eventBytes, err := json.Marshal(event)
			if err != nil {
				log.Printf("CallingClient: failed to marshal Mercury event: %v", err)
				return
			}
			cc.HandleMercuryEvent(eventBytes)
		}
	})

	log.Println("CallingClient: connecting Mercury WebSocket...")
	if err := merc.Connect(); err != nil {
		return fmt.Errorf("mercury connection failed: %w", err)
	}
	log.Println("CallingClient: Mercury WebSocket connected")

	cc.mu.Lock()
	cc.mercuryClient = merc
	cc.mu.Unlock()

	return nil
}

// DisconnectMercury disconnects the Mercury WebSocket client if connected.
func (cc *CallingClient) DisconnectMercury() {
	cc.mu.Lock()
	merc := cc.mercuryClient
	cc.mercuryClient = nil
	cc.mu.Unlock()

	if merc != nil {
		if err := merc.Disconnect(); err != nil {
			log.Printf("CallingClient: Mercury disconnect error: %v", err)
		}
		log.Println("CallingClient: Mercury disconnected")
	}
}

// IsMercuryConnected returns whether the Mercury client is connected.
func (cc *CallingClient) IsMercuryConnected() bool {
	cc.mu.RLock()
	merc := cc.mercuryClient
	cc.mu.RUnlock()
	return merc != nil && merc.IsConnected()
}

// SetAudioBridge registers an AudioBridge with this CallingClient. When set,
// MakeCall() will automatically attach the bridge to new calls and detach it
// when calls disconnect. This eliminates manual bridge↔call wiring.
func (cc *CallingClient) SetAudioBridge(bridge *AudioBridge) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.audioBridge = bridge
	log.Println("CallingClient: audio bridge set")
}

// ClearAudioBridge removes the AudioBridge from this CallingClient.
func (cc *CallingClient) ClearAudioBridge() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.audioBridge = nil
	log.Println("CallingClient: audio bridge cleared")
}

// GetAudioBridge returns the currently registered AudioBridge, or nil.
func (cc *CallingClient) GetAudioBridge() *AudioBridge {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.audioBridge
}

// GetConnectedCall returns the currently connected (not held) call, if any
func (cc *CallingClient) GetConnectedCall() *Call {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	for _, call := range cc.activeCalls {
		if call.IsConnected() && !call.IsHeld() {
			return call
		}
	}
	return nil
}

// HandleMercuryEvent processes a Mercury WebSocket event that may contain
// Mobius call signaling data. This should be called when the Mercury client
// receives a "mobius" scoped event.
func (cc *CallingClient) HandleMercuryEvent(eventData []byte) {
	var event MobiusCallEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		log.Printf("Failed to parse Mobius event: %v (raw: %s)", err, string(eventData[:min(len(eventData), 200)]))
		return
	}

	data := event.Data
	log.Printf("HandleMercuryEvent: eventType=%s callId=%s correlationId=%s hasMessage=%v",
		data.EventType, data.CallID, data.CorrelationID, data.Message != nil)

	// Route to existing call if we have one
	cc.mu.RLock()
	for _, call := range cc.activeCalls {
		matchByCallID := data.CallID != "" && call.GetCallID() == data.CallID
		matchByCorrelation := data.CorrelationID != "" && call.GetCorrelationID() == data.CorrelationID
		if matchByCallID || matchByCorrelation {
			cc.mu.RUnlock()
			log.Printf("HandleMercuryEvent: routing to existing call %s", call.GetCallID())
			call.HandleMobiusEvent(&event)
			return
		}
	}
	cc.mu.RUnlock()

	// New incoming call
	if data.EventType == MobiusEventCallSetup {
		cc.handleIncomingCall(&event)
	}
}

// handleIncomingCall creates a new Call object for an incoming call
func (cc *CallingClient) handleIncomingCall(event *MobiusCallEvent) {
	data := event.Data

	// Skip incoming call if we already have an active outbound call on the same
	// device. This handles the self-call scenario where BroadWorks sends both
	// an outbound and inbound leg — we should only process the outbound one.
	cc.mu.RLock()
	for _, call := range cc.activeCalls {
		if call.GetDirection() == CallDirectionOutbound && call.GetState() != CallStateDisconnected {
			cc.mu.RUnlock()
			log.Printf("Ignoring incoming call %s — active outbound call exists on this device", data.CallID)
			return
		}
	}
	cc.mu.RUnlock()

	// Find the line this call belongs to
	cc.mu.RLock()
	var targetLine *Line
	for _, line := range cc.lines {
		if line.GetDeviceID() == data.DeviceID {
			targetLine = line
			break
		}
	}
	cc.mu.RUnlock()

	if targetLine == nil {
		log.Printf("Received incoming call for unknown device %s", data.DeviceID)
		return
	}

	call, err := NewCall(cc.core, CallDirectionInbound, nil, &CallConfig{
		MobiusURL:       targetLine.GetActiveMobiusURL(),
		DeviceID:        data.DeviceID,
		LineID:          targetLine.LineID,
		ClientDeviceURI: cc.clientDeviceURI,
		MediaConfig:     cc.mediaConfig,
	})
	if err != nil {
		log.Printf("Failed to create incoming call: %v", err)
		return
	}

	// Set call ID from the event
	call.mu.Lock()
	call.callID = data.CallID
	call.correlationID = data.CorrelationID
	if call.correlationID == "" {
		call.correlationID = uuid.New().String()
	}
	call.state = CallStateAlerting
	call.mu.Unlock()

	cc.mu.Lock()
	cc.activeCalls[call.GetCorrelationID()] = call
	cc.mu.Unlock()

	// Clean up on disconnect
	call.Emitter.On(string(CallEventDisconnect), func(d interface{}) {
		cc.mu.Lock()
		delete(cc.activeCalls, call.GetCorrelationID())
		cc.mu.Unlock()
	})

	// Process the initial ROAP message if present
	call.HandleMobiusEvent(event)

	// Emit incoming call event
	targetLine.Emitter.Emit(string(LineEventIncomingCall), call)
	cc.Emitter.Emit(string(LineEventIncomingCall), call)
}

// Shutdown deregisters all lines, disconnects Mercury, and cleans up resources
func (cc *CallingClient) Shutdown() error {
	// Disconnect Mercury first so no new events arrive during teardown
	cc.DisconnectMercury()
	cc.ClearAudioBridge()

	cc.mu.Lock()
	calls := make([]*Call, 0, len(cc.activeCalls))
	for _, call := range cc.activeCalls {
		calls = append(calls, call)
	}
	lines := make([]*Line, 0, len(cc.lines))
	for _, line := range cc.lines {
		lines = append(lines, line)
	}
	cc.mu.Unlock()

	// End all active calls
	for _, call := range calls {
		if err := call.End(); err != nil {
			log.Printf("Error ending call %s: %v", call.GetCallID(), err)
		}
	}

	// Deregister all lines
	for _, line := range lines {
		if err := line.Deregister(); err != nil {
			log.Printf("Error deregistering line %s: %v", line.LineID, err)
		}
	}

	cc.mu.Lock()
	cc.activeCalls = make(map[string]*Call)
	cc.lines = make(map[string]*Line)
	cc.mu.Unlock()

	return nil
}

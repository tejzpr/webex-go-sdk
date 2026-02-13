/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
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

	// Media config
	mediaConfig *MediaConfig

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
// This calls the Webex discovery endpoint to find the appropriate Mobius cluster.
func (cc *CallingClient) DiscoverMobiusServers() error {
	url := "https://ds.ciscospark.com/v1/region"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating discovery request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cc.core.GetAccessToken())
	req.Header.Set("spark-user-agent", "webex-calling/beta")

	resp, err := cc.core.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("error making discovery request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fall back to default Mobius URLs
		cc.mu.Lock()
		cc.primaryMobiusURLs = []string{
			"https://mobius-us-east-1.prod.infra.webex.com/api/v1/calling/web/",
		}
		cc.backupMobiusURLs = []string{
			"https://mobius-eu-central-1.prod.infra.webex.com/api/v1/calling/web/",
		}
		cc.mu.Unlock()
		log.Printf("Discovery returned %d, using default Mobius URLs", resp.StatusCode)
		return nil
	}

	var regionInfo struct {
		CountryCode  string `json:"countryCode"`
		ClientRegion string `json:"clientRegion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regionInfo); err != nil {
		return fmt.Errorf("error parsing discovery response: %w", err)
	}

	// Build Mobius URLs based on region
	cc.mu.Lock()
	if regionInfo.ClientRegion == "EU" || regionInfo.ClientRegion == "eu" {
		cc.primaryMobiusURLs = []string{
			"https://mobius-eu-central-1.prod.infra.webex.com/api/v1/calling/web/",
		}
		cc.backupMobiusURLs = []string{
			"https://mobius-us-east-1.prod.infra.webex.com/api/v1/calling/web/",
		}
	} else {
		cc.primaryMobiusURLs = []string{
			"https://mobius-us-east-1.prod.infra.webex.com/api/v1/calling/web/",
		}
		cc.backupMobiusURLs = []string{
			"https://mobius-eu-central-1.prod.infra.webex.com/api/v1/calling/web/",
		}
	}
	cc.mu.Unlock()

	log.Printf("Discovered Mobius servers for region=%s country=%s", regionInfo.ClientRegion, regionInfo.CountryCode)
	return nil
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

	line := NewLine(cc.core, cc.config, &LineConfig{
		PrimaryMobiusURLs: primary,
		BackupMobiusURLs:  backup,
		ClientDeviceURI:   deviceURI,
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

	// Listen for disconnect to clean up
	call.Emitter.On(string(CallEventDisconnect), func(data interface{}) {
		cc.mu.Lock()
		delete(cc.activeCalls, call.GetCorrelationID())
		cc.mu.Unlock()
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
		log.Printf("Failed to parse Mobius event: %v", err)
		return
	}

	data := event.Data

	// Route to existing call if we have one
	cc.mu.RLock()
	for _, call := range cc.activeCalls {
		if call.GetCallID() == data.CallID || call.GetCorrelationID() == data.CorrelationID {
			cc.mu.RUnlock()
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

// Shutdown deregisters all lines and cleans up resources
func (cc *CallingClient) Shutdown() error {
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

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
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Call represents an active or pending call with full call control.
// It manages the Mobius signaling and Pion WebRTC media layer.
type Call struct {
	mu sync.RWMutex

	core *webexsdk.Client

	// Call identifiers
	callID        string
	correlationID string
	lineID        string
	deviceID      string

	// Call properties
	direction        CallDirection
	destination      *CallDetails
	state            CallState
	disconnectReason DisconnectReason
	muted            bool
	held             bool
	connected        bool

	// Mobius signaling
	mobiusURL       string
	clientDeviceURI string
	seq             int

	// Media
	media *MediaEngine

	// Events
	Emitter *EventEmitter
}

// CallDetails contains the destination for an outbound call
type CallDetails struct {
	Type    CallType `json:"type"`
	Address string   `json:"address"`
}

// CallConfig holds configuration for creating a call
type CallConfig struct {
	// MobiusURL is the active Mobius server URL
	MobiusURL string
	// DeviceID is the registered Mobius device ID
	DeviceID string
	// LineID is the line ID this call belongs to
	LineID string
	// ClientDeviceURI is the Webex device URL
	ClientDeviceURI string
	// MediaConfig is the WebRTC media configuration
	MediaConfig *MediaConfig
}

// NewCall creates a new Call instance
func NewCall(core *webexsdk.Client, direction CallDirection, destination *CallDetails, config *CallConfig) (*Call, error) {
	if config == nil {
		return nil, fmt.Errorf("call config is required")
	}

	mediaEngine, err := NewMediaEngine(config.MediaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create media engine: %w", err)
	}

	c := &Call{
		core:            core,
		callID:          fmt.Sprintf("DefaultLocalId_%s", uuid.New().String()),
		correlationID:   uuid.New().String(),
		lineID:          config.LineID,
		deviceID:        config.DeviceID,
		direction:       direction,
		destination:     destination,
		state:           CallStateIdle,
		mobiusURL:       config.MobiusURL,
		clientDeviceURI: config.ClientDeviceURI,
		seq:             1,
		media:           mediaEngine,
		Emitter:         NewEventEmitter(),
		disconnectReason: DisconnectReason{
			Code:  DisconnectCodeNormal,
			Cause: "Normal Disconnect.",
		},
	}

	return c, nil
}

// GetCallID returns the call ID
func (c *Call) GetCallID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.callID
}

// GetCorrelationID returns the correlation ID
func (c *Call) GetCorrelationID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.correlationID
}

// GetDirection returns the call direction
func (c *Call) GetDirection() CallDirection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.direction
}

// GetState returns the current call state
func (c *Call) GetState() CallState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsConnected returns true if the call is connected
func (c *Call) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// IsMuted returns true if the call is muted
func (c *Call) IsMuted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.muted
}

// IsHeld returns true if the call is on hold
func (c *Call) IsHeld() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.held
}

// GetDisconnectReason returns the disconnect reason
func (c *Call) GetDisconnectReason() DisconnectReason {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.disconnectReason
}

// GetMedia returns the media engine for direct RTP access
func (c *Call) GetMedia() *MediaEngine {
	return c.media
}

// ---- Call Control Methods ----

// Dial initiates an outbound call.
// It creates a WebRTC offer, wraps it in ROAP, and POSTs to Mobius.
func (c *Call) Dial() error {
	c.mu.Lock()
	if c.state != CallStateIdle {
		c.mu.Unlock()
		return fmt.Errorf("cannot dial: call is in state %s", c.state)
	}
	c.state = CallStateProceeding
	c.mu.Unlock()

	// Add audio track
	if _, err := c.media.AddAudioTrack(); err != nil {
		return fmt.Errorf("failed to add audio track: %w", err)
	}

	// Create SDP offer
	sdp, err := c.media.CreateOffer()
	if err != nil {
		return fmt.Errorf("failed to create SDP offer: %w", err)
	}

	// Filter IPv6 for compatibility
	sdp = ModifySdpForMobius(sdp)
	log.Printf("Outgoing SDP offer:\n--- Our SDP Offer ---\n%s\n--- End SDP ---", sdp)

	// Set up remote track handler BEFORE postCall so we don't miss the track
	c.media.OnRemoteTrack(func(track *webrtc.TrackRemote) {
		log.Printf("Mobius remote track received: codec=%s", track.Codec().MimeType)
		c.Emitter.Emit(string(CallEventRemoteMedia), track)
	})

	// Wrap in ROAP and POST to Mobius
	roapMsg := SDPToRoapOffer(sdp, c.seq)
	resp, err := c.postCall(roapMsg)
	if err != nil {
		c.mu.Lock()
		c.state = CallStateDisconnected
		c.mu.Unlock()
		c.Emitter.Emit(string(CallEventError), err)
		return fmt.Errorf("failed to post call to Mobius: %w", err)
	}

	c.mu.Lock()
	c.callID = resp.Body.CallID
	c.seq++
	c.mu.Unlock()

	log.Printf("Call setup successful, callId=%s correlationId=%s", resp.Body.CallID, c.correlationID)

	// Process the ROAP answer from Mobius to complete the WebRTC handshake
	if resp.Body.LocalMedia != nil && resp.Body.LocalMedia.Roap != nil {
		roap := resp.Body.LocalMedia.Roap
		log.Printf("Received ROAP %s from Mobius (seq=%d, sdp length=%d)", roap.MessageType, roap.Seq, len(roap.SDP))
		if roap.MessageType == RoapMessageAnswer && roap.SDP != "" {
			if err := c.media.SetRemoteAnswer(roap.SDP); err != nil {
				log.Printf("Failed to set remote answer from Mobius: %v", err)
			} else {
				log.Printf("Set remote SDP answer from Mobius, WebRTC handshake complete")
				// Send ROAP OK back to Mobius
				okMsg := NewRoapOK(roap.Seq)
				if err := c.postMedia(okMsg); err != nil {
					log.Printf("Failed to send ROAP OK: %v", err)
				}
			}
		}
	} else {
		log.Printf("No ROAP answer in Mobius call response — media negotiation may happen via events")
	}

	c.Emitter.Emit(string(CallEventProgress), c.callID)
	return nil
}

// Answer answers an incoming call.
// It sets the remote SDP offer, creates an answer, and sends it via ROAP.
func (c *Call) Answer(remoteOffer string) error {
	c.mu.Lock()
	if c.state != CallStateAlerting && c.state != CallStateIdle {
		c.mu.Unlock()
		return fmt.Errorf("cannot answer: call is in state %s", c.state)
	}
	c.mu.Unlock()

	// Add audio track
	if _, err := c.media.AddAudioTrack(); err != nil {
		return fmt.Errorf("failed to add audio track: %w", err)
	}

	// Set remote offer
	if err := c.media.SetRemoteOffer(remoteOffer); err != nil {
		return fmt.Errorf("failed to set remote offer: %w", err)
	}

	// Create SDP answer
	sdp, err := c.media.CreateAnswer()
	if err != nil {
		return fmt.Errorf("failed to create SDP answer: %w", err)
	}

	sdp = ModifySdpForMobius(sdp)

	// Send ROAP answer to Mobius via media endpoint
	roapMsg := SDPToRoapAnswer(sdp, c.seq)
	if err := c.postMedia(roapMsg); err != nil {
		return fmt.Errorf("failed to send answer to Mobius: %w", err)
	}

	c.mu.Lock()
	c.state = CallStateConnected
	c.connected = true
	c.seq++
	c.mu.Unlock()

	// Set up remote track handler
	c.media.OnRemoteTrack(func(track *webrtc.TrackRemote) {
		c.Emitter.Emit(string(CallEventRemoteMedia), track)
	})

	c.Emitter.Emit(string(CallEventConnect), c.callID)
	c.Emitter.Emit(string(CallEventEstablished), c.callID)
	return nil
}

// End disconnects the call
func (c *Call) End() error {
	c.mu.Lock()
	if c.state == CallStateDisconnected {
		c.mu.Unlock()
		return nil
	}
	prevState := c.state
	c.state = CallStateDisconnected
	c.connected = false
	c.mu.Unlock()

	// Send disconnect to Mobius if we were connected
	if prevState == CallStateConnected || prevState == CallStateHeld || prevState == CallStateProceeding || prevState == CallStateAlerting {
		if err := c.deleteCall(); err != nil {
			log.Printf("Error sending disconnect to Mobius: %v", err)
		}
	}

	// Close media
	if c.media != nil {
		if err := c.media.Close(); err != nil {
			log.Printf("Error closing media: %v", err)
		}
	}

	c.Emitter.Emit(string(CallEventDisconnect), c.callID)
	return nil
}

// Mute mutes the local audio
func (c *Call) Mute() {
	c.mu.Lock()
	c.muted = true
	c.mu.Unlock()
	if c.media != nil {
		c.media.Mute()
	}
}

// Unmute unmutes the local audio
func (c *Call) Unmute() {
	c.mu.Lock()
	c.muted = false
	c.mu.Unlock()
	if c.media != nil {
		c.media.Unmute()
	}
}

// Hold puts the call on hold
func (c *Call) Hold() error {
	c.mu.Lock()
	if c.state != CallStateConnected {
		c.mu.Unlock()
		return fmt.Errorf("cannot hold: call is in state %s", c.state)
	}
	c.mu.Unlock()

	if err := c.postSupplementaryService("callhold", "hold"); err != nil {
		c.Emitter.Emit(string(CallEventHoldError), err)
		return fmt.Errorf("hold request failed: %w", err)
	}

	c.mu.Lock()
	c.state = CallStateHeld
	c.held = true
	c.mu.Unlock()

	c.Emitter.Emit(string(CallEventHeld), c.callID)
	return nil
}

// Resume resumes a held call
func (c *Call) Resume() error {
	c.mu.Lock()
	if c.state != CallStateHeld {
		c.mu.Unlock()
		return fmt.Errorf("cannot resume: call is in state %s", c.state)
	}
	c.mu.Unlock()

	if err := c.postSupplementaryService("callhold", "resume"); err != nil {
		c.Emitter.Emit(string(CallEventResumeError), err)
		return fmt.Errorf("resume request failed: %w", err)
	}

	c.mu.Lock()
	c.state = CallStateConnected
	c.held = false
	c.mu.Unlock()

	c.Emitter.Emit(string(CallEventResumed), c.callID)
	return nil
}

// DoHoldResume toggles hold/resume
func (c *Call) DoHoldResume() error {
	if c.IsHeld() {
		return c.Resume()
	}
	return c.Hold()
}

// SendDigit sends a DTMF digit during the call
func (c *Call) SendDigit(tone string) error {
	c.mu.RLock()
	if c.state != CallStateConnected {
		c.mu.RUnlock()
		return fmt.Errorf("cannot send digit: call is not connected")
	}
	c.mu.RUnlock()

	payload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"callId": c.callID,
		"dtmf": map[string]string{
			"digit": tone,
		},
	}

	return c.postToMobius(
		fmt.Sprintf("%sdevices/%s/calls/%s/dtmf", c.mobiusURL, c.deviceID, c.callID),
		payload,
	)
}

// CompleteTransfer completes a call transfer (blind or consult)
func (c *Call) CompleteTransfer(transferType TransferType, transferCallID, transferTarget string) error {
	c.mu.RLock()
	if c.state != CallStateConnected && c.state != CallStateHeld {
		c.mu.RUnlock()
		return fmt.Errorf("cannot transfer: call is in state %s", c.state)
	}
	c.mu.RUnlock()

	payload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"callId": c.callID,
	}

	switch transferType {
	case TransferTypeBlind:
		payload["blindTransferContext"] = map[string]string{
			"transferorCallId": c.callID,
			"destination":      transferTarget,
		}
		payload["transferType"] = string(TransferTypeBlind)
	case TransferTypeConsult:
		payload["consultTransferContext"] = map[string]string{
			"transferorCallId": c.callID,
			"transferToCallId": transferCallID,
		}
		payload["transferType"] = string(TransferTypeConsult)
	default:
		return fmt.Errorf("unknown transfer type: %s", transferType)
	}

	url := fmt.Sprintf("%sservices/calltransfer/commit", c.mobiusURL)
	if err := c.postToMobius(url, payload); err != nil {
		c.Emitter.Emit(string(CallEventTransferError), err)
		return fmt.Errorf("transfer failed: %w", err)
	}

	return nil
}

// PostStatus sends a call keepalive/status to Mobius
func (c *Call) PostStatus() error {
	payload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"callId": c.callID,
	}

	url := fmt.Sprintf("%sdevices/%s/calls/%s/status", c.mobiusURL, c.deviceID, c.callID)
	return c.postToMobius(url, payload)
}

// HandleMobiusEvent processes an incoming Mobius WebSocket event for this call
func (c *Call) HandleMobiusEvent(event *MobiusCallEvent) {
	if event == nil {
		return
	}

	data := event.Data

	switch data.EventType {
	case MobiusEventCallProgress:
		c.mu.Lock()
		if c.state == CallStateProceeding {
			c.state = CallStateAlerting
		}
		c.mu.Unlock()
		c.Emitter.Emit(string(CallEventAlerting), c.callID)

		// Handle ROAP message in progress
		if data.Message != nil {
			c.handleRoapMessage(data.Message)
		}

	case MobiusEventCallConnected:
		c.mu.Lock()
		c.state = CallStateConnected
		c.connected = true
		c.mu.Unlock()
		c.Emitter.Emit(string(CallEventConnect), c.callID)
		c.Emitter.Emit(string(CallEventEstablished), c.callID)

	case MobiusEventCallMedia:
		if data.Message != nil {
			c.handleRoapMessage(data.Message)
		}

	case MobiusEventCallDisconnected:
		c.mu.Lock()
		c.state = CallStateDisconnected
		c.connected = false
		c.mu.Unlock()
		if c.media != nil {
			c.media.Close()
		}
		c.Emitter.Emit(string(CallEventDisconnect), c.callID)

	case MobiusEventCallSetup:
		// Incoming call setup
		c.mu.Lock()
		c.callID = data.CallID
		c.state = CallStateAlerting
		c.mu.Unlock()
		c.Emitter.Emit(string(CallEventAlerting), c.callID)
	}

	// Handle caller ID
	if data.CallerID != nil {
		c.Emitter.Emit(string(CallEventCallerID), data.CallerID)
	}

	// Handle mid-call events (hold/resume from remote)
	for _, midCall := range data.MidCallService {
		if midCall.EventType == "callState" {
			// Check for remote hold/resume
			c.Emitter.Emit(string(CallEventEstablished), c.callID)
		}
	}
}

// handleRoapMessage processes a ROAP message from Mobius
func (c *Call) handleRoapMessage(msg *RoapMessage) {
	if msg == nil {
		return
	}

	switch msg.MessageType {
	case RoapMessageAnswer:
		// Remote sent an answer to our offer
		log.Printf("ROAP ANSWER received (seq=%d, sdp length=%d) for callId=%s\n--- Mobius SDP Answer ---\n%s\n--- End SDP ---", msg.Seq, len(msg.SDP), c.callID, msg.SDP)
		if err := c.media.SetRemoteAnswer(msg.SDP); err != nil {
			log.Printf("Failed to set remote answer: %v", err)
			c.Emitter.Emit(string(CallEventError), err)
			return
		}
		log.Printf("Remote SDP answer set successfully, Mobius WebRTC handshake completing for callId=%s", c.callID)
		// Send ROAP OK
		okMsg := NewRoapOK(msg.Seq)
		if err := c.postMedia(okMsg); err != nil {
			log.Printf("Failed to send ROAP OK: %v", err)
		} else {
			log.Printf("ROAP OK sent successfully for callId=%s seq=%d", c.callID, msg.Seq)
		}

	case RoapMessageOffer:
		// Remote sent a new offer (renegotiation)
		if err := c.media.SetRemoteOffer(msg.SDP); err != nil {
			log.Printf("Failed to set remote offer: %v", err)
			return
		}
		sdp, err := c.media.CreateAnswer()
		if err != nil {
			log.Printf("Failed to create answer: %v", err)
			return
		}
		answerMsg := SDPToRoapAnswer(sdp, msg.Seq)
		if err := c.postMedia(answerMsg); err != nil {
			log.Printf("Failed to send ROAP answer: %v", err)
		}

	case RoapMessageOK:
		// Media negotiation complete
		log.Printf("ROAP OK received, media negotiation complete for callId=%s", c.callID)

	case RoapMessageOfferRequest:
		// Server requests a new offer
		sdp, err := c.media.CreateOffer()
		if err != nil {
			log.Printf("Failed to create offer for offer request: %v", err)
			return
		}
		offerMsg := SDPToRoapOffer(sdp, msg.Seq)
		if err := c.postMedia(offerMsg); err != nil {
			log.Printf("Failed to send ROAP offer: %v", err)
		}
	}
}

// ---- Mobius HTTP API Methods ----

// postCall sends a POST to create a new call with Mobius
func (c *Call) postCall(roapMsg *RoapMessage) (*MobiusCallResponse, error) {
	basePayload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"localMedia": map[string]interface{}{
			"roap":    roapMsg,
			"mediaId": uuid.New().String(),
		},
	}

	if c.destination != nil {
		basePayload["callee"] = map[string]string{
			"type":    string(c.destination.Type),
			"address": c.destination.Address,
		}
	}

	url := fmt.Sprintf("%sdevices/%s/call", c.mobiusURL, c.deviceID)

	payloadBytes, err := json.Marshal(basePayload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling call payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating call request: %w", err)
	}

	c.setMobiusHeaders(req)

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making call request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading call response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("call request failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Mobius call response (status %d): %s", resp.StatusCode, string(body))

	var callResp MobiusCallResponse
	callResp.StatusCode = resp.StatusCode
	if err := json.Unmarshal(body, &callResp.Body); err != nil {
		return nil, fmt.Errorf("error parsing call response: %w", err)
	}

	return &callResp, nil
}

// postMedia sends a ROAP message to the media endpoint
func (c *Call) postMedia(roapMsg *RoapMessage) error {
	payload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"callId": c.callID,
		"localMedia": map[string]interface{}{
			"roap":    roapMsg,
			"mediaId": uuid.New().String(),
		},
	}

	url := fmt.Sprintf("%sdevices/%s/calls/%s/media", c.mobiusURL, c.deviceID, c.callID)
	return c.postToMobius(url, payload)
}

// deleteCall sends a DELETE to disconnect the call
func (c *Call) deleteCall() error {
	url := fmt.Sprintf("%sdevices/%s/calls/%s", c.mobiusURL, c.deviceID, c.callID)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error creating delete request: %w", err)
	}

	c.setMobiusHeaders(req)

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("error making delete request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// postSupplementaryService sends a supplementary service request (hold/resume/transfer)
func (c *Call) postSupplementaryService(service, action string) error {
	payload := map[string]interface{}{
		"device": map[string]string{
			"deviceId":      c.deviceID,
			"correlationId": c.correlationID,
		},
		"callId": c.callID,
	}

	url := fmt.Sprintf("%sservices/%s/%s", c.mobiusURL, service, action)
	return c.postToMobius(url, payload)
}

// postToMobius is a generic helper for POST requests to Mobius
func (c *Call) postToMobius(url string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.setMobiusHeaders(req)

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// setMobiusHeaders sets the standard Mobius API headers
func (c *Call) setMobiusHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("spark-user-agent", "webex-calling/beta")
	if c.clientDeviceURI != "" {
		req.Header.Set("cisco-device-url", c.clientDeviceURI)
	}
	req.Header.Set("trackingid", fmt.Sprintf("webex-go-sdk_%s", uuid.New().String()))
}

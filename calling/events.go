/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import "sync"

// ---- Call State & Event Enums ----

// CallState represents the state of a call in the state machine
type CallState string

const (
	CallStateIdle         CallState = "idle"
	CallStateProceeding   CallState = "proceeding"
	CallStateAlerting     CallState = "alerting"
	CallStateConnected    CallState = "connected"
	CallStateHeld         CallState = "held"
	CallStateDisconnected CallState = "disconnected"
)

// CallEventKey identifies the type of call event
type CallEventKey string

const (
	CallEventAlerting      CallEventKey = "alerting"
	CallEventProgress      CallEventKey = "progress"
	CallEventConnect       CallEventKey = "connect"
	CallEventEstablished   CallEventKey = "established"
	CallEventDisconnect    CallEventKey = "disconnect"
	CallEventHeld          CallEventKey = "held"
	CallEventResumed       CallEventKey = "resumed"
	CallEventRemoteMedia   CallEventKey = "remote_media"
	CallEventCallerID      CallEventKey = "caller_id"
	CallEventError         CallEventKey = "call_error"
	CallEventHoldError     CallEventKey = "hold_error"
	CallEventResumeError   CallEventKey = "resume_error"
	CallEventTransferError CallEventKey = "transfer_error"
)

// LineEventKey identifies the type of line event
type LineEventKey string

const (
	LineEventConnecting   LineEventKey = "connecting"
	LineEventRegistered   LineEventKey = "registered"
	LineEventUnregistered LineEventKey = "unregistered"
	LineEventReconnecting LineEventKey = "reconnecting"
	LineEventReconnected  LineEventKey = "reconnected"
	LineEventIncomingCall LineEventKey = "incoming_call"
	LineEventError        LineEventKey = "error"
)

// RegistrationStatus represents the registration state of a line
type RegistrationStatus string

const (
	RegistrationStatusIdle     RegistrationStatus = "IDLE"
	RegistrationStatusActive   RegistrationStatus = "active"
	RegistrationStatusInactive RegistrationStatus = "inactive"
)

// ---- Event Emitter ----

// EventHandler is a callback function for events
type EventHandler func(data interface{})

// EventEmitter provides a simple event pub/sub system
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// NewEventEmitter creates a new EventEmitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		handlers: make(map[string][]EventHandler),
	}
}

// On registers an event handler for a specific event type
func (e *EventEmitter) On(event string, handler EventHandler) {
	if handler == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[event] = append(e.handlers[event], handler)
}

// Off removes all handlers for a specific event type
func (e *EventEmitter) Off(event string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.handlers, event)
}

// Emit fires an event, calling all registered handlers
func (e *EventEmitter) Emit(event string, data interface{}) {
	e.mu.RLock()
	handlers := make([]EventHandler, len(e.handlers[event]))
	copy(handlers, e.handlers[event])
	e.mu.RUnlock()

	for _, handler := range handlers {
		handler(data)
	}
}

// ---- Mobius Event Types (WebSocket events from Mobius) ----

// MobiusEventType identifies the type of Mobius WebSocket event
type MobiusEventType string

const (
	MobiusEventCallSetup        MobiusEventType = "mobius.call"
	MobiusEventCallProgress     MobiusEventType = "mobius.callprogress"
	MobiusEventCallConnected    MobiusEventType = "mobius.callconnected"
	MobiusEventCallMedia        MobiusEventType = "mobius.media"
	MobiusEventCallDisconnected MobiusEventType = "mobius.calldisconnected"
)

// RoapMessageType identifies the type of ROAP message
type RoapMessageType string

const (
	RoapMessageOffer        RoapMessageType = "OFFER"
	RoapMessageAnswer       RoapMessageType = "ANSWER"
	RoapMessageOK           RoapMessageType = "OK"
	RoapMessageError        RoapMessageType = "ERROR"
	RoapMessageOfferRequest RoapMessageType = "OFFER_REQUEST"
)

// RoapMessage represents a ROAP signaling message exchanged with Mobius
type RoapMessage struct {
	Seq               int             `json:"seq"`
	MessageType       RoapMessageType `json:"messageType"`
	SDP               string          `json:"sdp,omitempty"`
	OffererSessionID  string          `json:"offererSessionId,omitempty"`
	AnswererSessionID string          `json:"answererSessionId,omitempty"`
	Version           string          `json:"version,omitempty"`
	TieBreaker        string          `json:"tieBreaker,omitempty"`
	ErrorType         string          `json:"errorType,omitempty"`
}

// MobiusCallData represents the data payload in a Mobius call event
type MobiusCallData struct {
	CallProgressData *struct {
		Alerting    bool `json:"alerting"`
		InbandMedia bool `json:"inbandMedia"`
	} `json:"callProgressData,omitempty"`
	Message        *RoapMessage    `json:"message,omitempty"`
	CallerID       *CallerIDInfo   `json:"callerId,omitempty"`
	MidCallService []MidCallEvent  `json:"midCallService,omitempty"`
	CallID         string          `json:"callId"`
	CallURL        string          `json:"callUrl"`
	DeviceID       string          `json:"deviceId"`
	CorrelationID  string          `json:"correlationId"`
	EventType      MobiusEventType `json:"eventType"`
}

// CallerIDInfo contains caller identification info from Mobius
type CallerIDInfo struct {
	From                       string `json:"from,omitempty"`
	PAssertedIdentity          string `json:"p-asserted-identity,omitempty"`
	XBroadworksRemotePartyInfo string `json:"x-broadworks-remote-party-info,omitempty"`
}

// MidCallEvent represents a mid-call event (e.g., hold state change)
type MidCallEvent struct {
	EventType string      `json:"eventType"`
	EventData interface{} `json:"eventData"`
}

// MobiusCallEvent is a full Mobius WebSocket event for calls
type MobiusCallEvent struct {
	ID         string         `json:"id"`
	Data       MobiusCallData `json:"data"`
	Timestamp  int64          `json:"timestamp"`
	TrackingID string         `json:"trackingId"`
}

// ---- Mobius API Response Types ----

// MobiusDeviceInfo represents device info returned from Mobius registration
type MobiusDeviceInfo struct {
	UserID                string      `json:"userId,omitempty"`
	Device                *DeviceType `json:"device,omitempty"`
	KeepaliveInterval     int         `json:"keepaliveInterval,omitempty"`
	CallKeepaliveInterval int         `json:"callKeepaliveInterval,omitempty"`
	VoicePortalNumber     int         `json:"voicePortalNumber,omitempty"`
	VoicePortalExtension  int         `json:"voicePortalExtension,omitempty"`
	RehomingIntervalMin   int         `json:"rehomingIntervalMin,omitempty"`
	RehomingIntervalMax   int         `json:"rehomingIntervalMax,omitempty"`
}

// DeviceType represents a registered Mobius device
type DeviceType struct {
	DeviceID        string   `json:"deviceId"`
	URI             string   `json:"uri"`
	Status          string   `json:"status"`
	LastSeen        string   `json:"lastSeen"`
	Addresses       []string `json:"addresses"`
	ClientDeviceURI string   `json:"clientDeviceUri"`
}

// MobiusCallResponse is the response from Mobius when creating a call
type MobiusCallResponse struct {
	StatusCode int `json:"statusCode"`
	Body       struct {
		Device struct {
			DeviceID      string `json:"deviceId"`
			CorrelationID string `json:"correlationId"`
		} `json:"device"`
		CallID   string `json:"callId"`
		CallData *struct {
			CallState string `json:"callState"`
		} `json:"callData,omitempty"`
		LocalMedia *struct {
			Roap *RoapMessage `json:"roap,omitempty"`
		} `json:"localMedia,omitempty"`
	} `json:"body"`
}

// DisconnectCode represents the reason code for a call disconnect
type DisconnectCode int

const (
	DisconnectCodeNormal          DisconnectCode = 0
	DisconnectCodeBusy            DisconnectCode = 115
	DisconnectCodeMediaInactivity DisconnectCode = 131
)

// DisconnectReason contains the code and cause for a call disconnect
type DisconnectReason struct {
	Code  DisconnectCode `json:"code"`
	Cause string         `json:"cause"`
}

// TransferType indicates the type of call transfer
type TransferType string

const (
	TransferTypeBlind   TransferType = "BLIND"
	TransferTypeConsult TransferType = "CONSULT"
)

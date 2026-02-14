/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

// Package main provides a web-based example application for the Webex Calling API.
// It demonstrates call history, call settings, voicemail, contacts, and real-time
// call control (register, dial, hold, transfer, DTMF) using the Webex Go SDK.
//
// Usage:
//
//	go run main.go
//
// Then open http://localhost:8090 in your browser.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"

	webex "github.com/tejzpr/webex-go-sdk/v2"
	"github.com/tejzpr/webex-go-sdk/v2/calling"
	"github.com/tejzpr/webex-go-sdk/v2/mercury"
)

//go:embed static/*
var staticFiles embed.FS

// appState holds the server-side state
type appState struct {
	mu               sync.RWMutex
	client           *webex.WebexClient
	callingClient    *calling.CallingClient
	line             *calling.Line
	activeCall       *calling.Call
	accessToken      string
	mercuryClient    *mercury.Client
	mercuryConnected bool

	// Audio bridge: browser-side Pion PeerConnection
	browserPC         *webrtc.PeerConnection
	browserLocalTrack *webrtc.TrackLocalStaticRTP
}

var state = &appState{}

func main() {
	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API endpoints
	mux.HandleFunc("/api/connect", handleConnect)
	mux.HandleFunc("/api/disconnect", handleDisconnect)
	mux.HandleFunc("/api/status", handleStatus)

	// REST API endpoints
	mux.HandleFunc("/api/call-history", handleCallHistory)
	mux.HandleFunc("/api/call-settings/dnd", handleDND)
	mux.HandleFunc("/api/call-settings/call-waiting", handleCallWaiting)
	mux.HandleFunc("/api/call-settings/call-forward", handleCallForward)
	mux.HandleFunc("/api/voicemail/list", handleVoicemailList)
	mux.HandleFunc("/api/voicemail/summary", handleVoicemailSummary)
	mux.HandleFunc("/api/contacts", handleContacts)

	// Call control endpoints
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/deregister", handleDeregister)
	mux.HandleFunc("/api/deregister-all", handleDeregisterAll)
	mux.HandleFunc("/api/dial", handleDial)
	mux.HandleFunc("/api/end", handleEnd)
	mux.HandleFunc("/api/hold", handleHold)
	mux.HandleFunc("/api/resume", handleResume)
	mux.HandleFunc("/api/mute", handleMute)
	mux.HandleFunc("/api/unmute", handleUnmute)
	mux.HandleFunc("/api/dtmf", handleDTMF)
	mux.HandleFunc("/api/transfer", handleTransfer)

	// Audio bridge WebSocket
	mux.HandleFunc("/ws/audio", handleAudioWS)

	addr := ":8095"
	server := &http.Server{Addr: addr, Handler: mux}

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %v, shutting down...", sig)

		// Auto-deregister all lines and devices
		state.mu.Lock()
		if state.callingClient != nil {
			log.Println("Auto-deregistering calling client...")
			state.callingClient.Shutdown()
			log.Println("Calling client shut down.")
		}
		state.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("Webex Calling Example running at http://localhost%s", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Println("Server stopped.")
}

// ---- Helper ----

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

func requireClient(w http.ResponseWriter) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	if state.client == nil {
		jsonError(w, http.StatusBadRequest, "Not connected. Enter your access token first.")
		return false
	}
	return true
}

// ---- Connection ----

func handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AccessToken == "" {
		jsonError(w, http.StatusBadRequest, "accessToken is required")
		return
	}

	client, err := webex.NewClient(req.AccessToken, nil)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create client: %v", err))
		return
	}

	state.mu.Lock()
	state.client = client
	state.accessToken = req.AccessToken
	state.callingClient = nil
	state.line = nil
	state.activeCall = nil
	state.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]string{"status": "connected"})
}

func handleDisconnect(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	if state.callingClient != nil {
		state.callingClient.Shutdown()
	}
	state.client = nil
	state.callingClient = nil
	state.line = nil
	state.activeCall = nil
	state.accessToken = ""
	state.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	resp := map[string]interface{}{
		"connected":  state.client != nil,
		"registered": false,
		"callActive": false,
	}

	if state.line != nil {
		resp["registered"] = state.line.IsRegistered()
		resp["lineId"] = state.line.LineID
		resp["deviceId"] = state.line.GetDeviceID()
	}

	if state.activeCall != nil {
		resp["callActive"] = true
		resp["callId"] = state.activeCall.GetCallID()
		resp["callState"] = string(state.activeCall.GetState())
		resp["callDirection"] = string(state.activeCall.GetDirection())
		resp["muted"] = state.activeCall.IsMuted()
		resp["held"] = state.activeCall.IsHeld()
	}

	jsonResponse(w, http.StatusOK, resp)
}

// ---- REST APIs ----

func handleCallHistory(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().CallHistory().GetCallHistoryData(7, 10, calling.SortDESC, calling.SortByStartTime)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func handleDND(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	if r.Method == http.MethodGet {
		data, err := client.Calling().CallSettings().GetDoNotDisturbSetting()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
			return
		}
		jsonResponse(w, http.StatusOK, data)
		return
	}

	if r.Method == http.MethodPut {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		data, err := client.Calling().CallSettings().SetDoNotDisturbSetting(req.Enabled)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
			return
		}
		jsonResponse(w, http.StatusOK, data)
		return
	}

	jsonError(w, http.StatusMethodNotAllowed, "GET or PUT required")
}

func handleCallWaiting(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().CallSettings().GetCallWaitingSetting()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func handleCallForward(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().CallSettings().GetCallForwardSetting()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func handleVoicemailList(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().Voicemail().GetVoicemailList(0, 20, calling.SortDESC)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func handleVoicemailSummary(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().Voicemail().GetVoicemailSummary()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func handleContacts(w http.ResponseWriter, r *http.Request) {
	if !requireClient(w) {
		return
	}
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	data, err := client.Calling().Contacts().GetContacts()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Error: %v", err))
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

// ---- Call Control ----

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !requireClient(w) {
		return
	}

	var req struct {
		PrimaryMobiusURL string `json:"primaryMobiusUrl"`
		ClientDeviceURI  string `json:"clientDeviceUri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	state.mu.Lock()
	client := state.client
	state.mu.Unlock()

	cc := client.Calling().CallingClient(&calling.CallingClientConfig{
		ClientDeviceURI: req.ClientDeviceURI,
	})

	if req.PrimaryMobiusURL != "" {
		cc.SetMobiusServers([]string{req.PrimaryMobiusURL}, nil)
	} else {
		if err := cc.DiscoverMobiusServers(); err != nil {
			jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Discovery failed: %v", err))
			return
		}
	}

	line, err := cc.CreateLine()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Registration failed: %v", err))
		return
	}

	state.mu.Lock()
	state.callingClient = cc
	state.line = line
	state.mu.Unlock()

	// Connect Mercury WebSocket for Mobius call events (ROAP answers, call progress, etc.)
	// IMPORTANT: Use the WDM WebSocket URL from CallingClient so Mercury connects
	// to the same device that Mobius registered with. Otherwise Mobius events go
	// to a different device and never reach Mercury.
	go func() {
		merc := client.Mercury()

		// Use the same WDM device's WebSocket URL that was used for Mobius registration
		wsURL := cc.GetWDMWebSocketURL()
		if wsURL != "" {
			log.Printf("Mercury: using WDM WebSocket URL from calling registration: %s", wsURL)
			merc.SetCustomWebSocketURL(wsURL)
		} else {
			log.Println("Mercury: WARNING - no WDM WebSocket URL from calling, using default device")
		}

		// Clear all wildcard handlers to prevent duplicates on re-registration
		merc.ClearHandlers("*")

		// Register a wildcard handler to route Mobius events to CallingClient
		merc.On("*", func(event *mercury.Event) {
			if event == nil || event.Data == nil {
				return
			}
			// Log ALL events for debugging
			eventType, _ := event.Data["eventType"].(string)
			log.Printf("Mercury event received: eventType=%s id=%s", eventType, event.ID)
			// Check if this is a Mobius call event
			if strings.HasPrefix(eventType, "mobius.") {
				log.Printf("Mercury: received Mobius event: %s", eventType)
				// Re-marshal the event data as a MobiusCallEvent for HandleMercuryEvent
				eventBytes, err := json.Marshal(event)
				if err != nil {
					log.Printf("Mercury: failed to marshal event: %v", err)
					return
				}
				state.mu.RLock()
				callingCl := state.callingClient
				state.mu.RUnlock()
				if callingCl != nil {
					callingCl.HandleMercuryEvent(eventBytes)
				}
			}
		})

		log.Println("Connecting Mercury WebSocket...")
		if err := merc.Connect(); err != nil {
			log.Printf("Mercury connection failed: %v", err)
			return
		}
		log.Println("Mercury WebSocket connected!")

		state.mu.Lock()
		state.mercuryClient = merc
		state.mercuryConnected = true
		state.mu.Unlock()
	}()

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "registered",
		"lineId":   line.LineID,
		"deviceId": line.GetDeviceID(),
	})
}

func handleDeregister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.Lock()
	if state.mercuryClient != nil {
		state.mercuryClient.Disconnect()
		state.mercuryClient = nil
		state.mercuryConnected = false
	}
	if state.callingClient != nil {
		state.callingClient.Shutdown()
	}
	state.callingClient = nil
	state.line = nil
	state.activeCall = nil
	state.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deregistered"})
}

func handleDeregisterAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !requireClient(w) {
		return
	}

	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	// Create a temporary CallingClient to discover Mobius and clean up devices
	cc := client.Calling().CallingClient(&calling.CallingClientConfig{})

	if err := cc.DiscoverMobiusServers(); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Discovery failed: %v", err))
		return
	}

	deleted, err := cc.DeregisterAllDevices()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Deregister all failed: %v", err))
		return
	}

	// Also clean up local state
	state.mu.Lock()
	if state.callingClient != nil {
		state.callingClient.Shutdown()
	}
	state.callingClient = nil
	state.line = nil
	state.activeCall = nil
	state.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "deregistered_all",
		"deleted": deleted,
	})
}

func handleDial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	if !requireClient(w) {
		return
	}

	var req struct {
		Address  string `json:"address"`
		CallType string `json:"callType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		jsonError(w, http.StatusBadRequest, "address is required")
		return
	}

	state.mu.RLock()
	cc := state.callingClient
	line := state.line
	state.mu.RUnlock()

	if cc == nil || line == nil {
		jsonError(w, http.StatusBadRequest, "Not registered. Register a line first.")
		return
	}

	// Normalize address:
	// - SIP URIs → pass through as-is (works for some BroadWorks destinations)
	// - Phone numbers → sanitize and add tel: prefix (matching JS SDK behavior)
	address := strings.TrimSpace(req.Address)
	ct := calling.CallTypeURI

	if strings.HasPrefix(address, "sip:") || strings.HasPrefix(address, "sips:") {
		// SIP URI — pass through as-is, BroadWorks may or may not route it
		log.Printf("Dial: using SIP URI as-is: %s", address)
	} else if strings.HasPrefix(address, "tel:") {
		// Already tel: formatted
		log.Printf("Dial: using tel URI as-is: %s", address)
	} else {
		// Assume phone number — sanitize and add tel: prefix (JS SDK behavior)
		sanitized := sanitizePhoneNumber(address)
		if sanitized == "" {
			jsonError(w, http.StatusBadRequest, "Invalid phone number. Use E.164 format (e.g. +14085551234)")
			return
		}
		address = "tel:" + sanitized
		log.Printf("Dial: normalized phone number to: %s", address)
	}

	call, err := cc.MakeCall(line, &calling.CallDetails{
		Type:    ct,
		Address: address,
	})
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Dial failed: %v", err))
		return
	}

	state.mu.Lock()
	state.activeCall = call
	state.mu.Unlock()

	// Clear activeCall when remote party disconnects
	call.Emitter.On(string(calling.CallEventDisconnect), func(d interface{}) {
		log.Printf("Call disconnected by remote party, clearing activeCall")
		state.mu.Lock()
		if state.activeCall == call {
			state.activeCall = nil
		}
		state.mu.Unlock()
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":        "dialing",
		"callId":        call.GetCallID(),
		"correlationId": call.GetCorrelationID(),
	})
}

func handleEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.Lock()
	call := state.activeCall
	state.mu.Unlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	if err := call.End(); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("End failed: %v", err))
		return
	}

	state.mu.Lock()
	state.activeCall = nil
	state.mu.Unlock()

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ended"})
}

func handleHold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	if err := call.Hold(); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Hold failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "held"})
}

func handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	if err := call.Resume(); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Resume failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func handleMute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	call.Mute()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "muted"})
}

func handleUnmute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	call.Unmute()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "unmuted"})
}

func handleDTMF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		Digit string `json:"digit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Digit == "" {
		jsonError(w, http.StatusBadRequest, "digit is required")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	if err := call.SendDigit(req.Digit); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("DTMF failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "sent", "digit": req.Digit})
}

// ---- Audio Bridge (WebSocket + WebRTC relay) ----

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleAudioWS creates a browser-side Pion PeerConnection, exchanges SDP/ICE
// with the browser over WebSocket, and relays RTP packets between the Mobius
// PeerConnection and the browser PeerConnection.
func handleAudioWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Audio WS upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	log.Println("Audio WebSocket connected")

	// Create a MediaEngine with only PCMU/PCMA so the browser is forced to
	// negotiate the same codec Mobius uses (PCMU). This allows direct RTP relay.
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMU,
			ClockRate: 8000,
			Channels:  1,
		},
		PayloadType: 0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("Audio bridge: failed to register PCMU: %v", err)
		return
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMA,
			ClockRate: 8000,
			Channels:  1,
		},
		PayloadType: 8,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("Audio bridge: failed to register PCMA: %v", err)
		return
	}

	// Register default interceptors (RTCP reports, SRTP) — required for
	// the browser bridge PC to keep the DTLS/SRTP session alive.
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		log.Printf("Audio bridge: failed to register interceptors: %v", err)
		return
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		log.Printf("Audio bridge: failed to create PC: %v", err)
		return
	}
	defer pc.Close()

	// Create a local audio track with PCMU (to send Mobius remote audio to browser)
	browserLocalTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU},
		"audio-from-mobius",
		"webex-bridge",
	)
	if err != nil {
		log.Printf("Audio bridge: failed to create local track: %v", err)
		return
	}
	if _, err := pc.AddTrack(browserLocalTrack); err != nil {
		log.Printf("Audio bridge: failed to add track: %v", err)
		return
	}

	state.mu.Lock()
	state.browserPC = pc
	state.browserLocalTrack = browserLocalTrack
	state.mu.Unlock()

	// When browser sends audio (mic), relay to Mobius local track
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Audio bridge: received browser track codec=%s", track.Codec().MimeType)
		go func() {
			buf := make([]byte, 1500)
			var pktCount int
			for {
				n, _, readErr := track.Read(buf)
				if readErr != nil {
					log.Printf("Audio bridge: browser track read ended: %v (relayed %d packets to Mobius)", readErr, pktCount)
					return
				}

				state.mu.RLock()
				call := state.activeCall
				state.mu.RUnlock()

				if call == nil {
					continue
				}

				media := call.GetMedia()
				if !media.IsConnected() {
					continue // Wait for Mobius PC to finish ICE/DTLS handshake
				}

				mobiusLocalTrack := media.GetLocalTrack()
				if mobiusLocalTrack == nil {
					continue
				}

				// Parse RTP and write to Mobius local track
				pkt := &rtp.Packet{}
				if err := pkt.Unmarshal(buf[:n]); err != nil {
					continue
				}
				if writeErr := mobiusLocalTrack.WriteRTP(pkt); writeErr != nil {
					if pktCount == 0 {
						log.Printf("Audio bridge: first write to mobius failed: %v", writeErr)
					}
				} else {
					pktCount++
					if pktCount == 1 {
						log.Printf("Audio bridge: first RTP packet relayed browser→Mobius (pt=%d ssrc=%d)", pkt.PayloadType, pkt.SSRC)
					} else if pktCount%500 == 0 {
						log.Printf("Audio bridge: relayed %d packets browser→Mobius", pktCount)
					}
				}
			}
		}()
	})

	// Send ICE candidates to browser
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candidateJSON, _ := json.Marshal(c.ToJSON())
		msg := map[string]interface{}{
			"type":      "ice-candidate",
			"candidate": json.RawMessage(candidateJSON),
		}
		msgBytes, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, msgBytes)
	})

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Audio bridge: PC state=%s", s.String())
	})

	// Mobius→Browser relay: silence keepalive until real audio, then direct write.
	// Key: write Mobius RTP directly to browserLocalTrack (no channel) to
	// preserve packet timing and avoid jitter from buffering/interleaving.
	stopRelay := make(chan struct{})
	stopSilence := make(chan struct{})

	// Silence keepalive goroutine — sends PCMU silence every 20ms until
	// real Mobius audio arrives, then stops to avoid interleaving.
	go func() {
		silenceBuf := make([]byte, 160)
		for i := range silenceBuf {
			silenceBuf[i] = 0xFF
		}
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopRelay:
				return
			case <-stopSilence:
				return
			case <-ticker.C:
				if writeErr := browserLocalTrack.WriteRTP(&rtp.Packet{
					Header:  rtp.Header{Version: 2, PayloadType: 0},
					Payload: silenceBuf,
				}); writeErr != nil {
					return
				}
			}
		}
	}()

	// Relay goroutine: poll for Mobius remote track, then relay RTP directly.
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopRelay:
				return
			case <-ticker.C:
				state.mu.RLock()
				call := state.activeCall
				state.mu.RUnlock()
				if call == nil {
					continue
				}
				remoteTrack := call.GetMedia().GetRemoteTrack()
				if remoteTrack == nil {
					continue
				}
				ticker.Stop()
				log.Println("Audio bridge: starting Mobius→Browser relay")
				silenceStopped := false
				buf := make([]byte, 1500)
				for {
					n, _, readErr := remoteTrack.Read(buf)
					if readErr != nil {
						log.Printf("Audio bridge: Mobius remote track read ended: %v", readErr)
						return
					}
					pkt := &rtp.Packet{}
					if err := pkt.Unmarshal(buf[:n]); err != nil {
						continue
					}
					// Write directly — preserves original packet timing
					if writeErr := browserLocalTrack.WriteRTP(pkt); writeErr != nil {
						log.Printf("Audio bridge: Mobius→Browser write error: %v", writeErr)
						return
					}
					// Stop silence only after first real packet is written
					if !silenceStopped {
						close(stopSilence)
						silenceStopped = true
						log.Println("Audio bridge: silence keepalive stopped, real audio flowing")
					}
				}
			}
		}
	}()

	// Read signaling messages from browser
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Audio WS read error: %v", err)
			break
		}

		var msg struct {
			Type      string          `json:"type"`
			SDP       string          `json:"sdp"`
			Candidate json.RawMessage `json:"candidate"`
		}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("Audio WS: invalid message: %v", err)
			continue
		}

		switch msg.Type {
		case "offer":
			log.Println("Audio bridge: received browser SDP offer")
			if err := pc.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  msg.SDP,
			}); err != nil {
				log.Printf("Audio bridge: set remote desc failed: %v", err)
				continue
			}

			answer, err := pc.CreateAnswer(nil)
			if err != nil {
				log.Printf("Audio bridge: create answer failed: %v", err)
				continue
			}
			if err := pc.SetLocalDescription(answer); err != nil {
				log.Printf("Audio bridge: set local desc failed: %v", err)
				continue
			}

			// Wait for ICE gathering
			gatherComplete := webrtc.GatheringCompletePromise(pc)
			<-gatherComplete

			localDesc := pc.LocalDescription()
			resp := map[string]string{
				"type": "answer",
				"sdp":  localDesc.SDP,
			}
			respBytes, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, respBytes)
			log.Println("Audio bridge: sent SDP answer to browser")

		case "ice-candidate":
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal(msg.Candidate, &candidate); err != nil {
				log.Printf("Audio bridge: invalid ICE candidate: %v", err)
				continue
			}
			if err := pc.AddICECandidate(candidate); err != nil {
				log.Printf("Audio bridge: add ICE candidate failed: %v", err)
			}
		}
	}

	close(stopRelay)
	state.mu.Lock()
	state.browserPC = nil
	state.browserLocalTrack = nil
	state.mu.Unlock()
	log.Println("Audio WebSocket disconnected")
}

func handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		TransferType   string `json:"transferType"`
		TransferTarget string `json:"transferTarget"`
		TransferCallID string `json:"transferCallId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	state.mu.RLock()
	call := state.activeCall
	state.mu.RUnlock()

	if call == nil {
		jsonError(w, http.StatusBadRequest, "No active call")
		return
	}

	tt := calling.TransferTypeBlind
	if req.TransferType == "CONSULT" {
		tt = calling.TransferTypeConsult
	}

	if err := call.CompleteTransfer(tt, req.TransferCallID, req.TransferTarget); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Transfer failed: %v", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "transferred"})
}

// sanitizePhoneNumber strips non-phone characters from a string,
// matching the JS SDK's VALID_PHONE_REGEX /[\d\s()*#+.-]+/ behavior.
// Returns the sanitized number or empty string if invalid.
func sanitizePhoneNumber(input string) string {
	var b strings.Builder
	for _, r := range input {
		if (r >= '0' && r <= '9') || r == '+' || r == '*' || r == '#' {
			b.WriteRune(r)
		}
		// Skip spaces, parens, dots, dashes (JS SDK strips these)
	}
	return b.String()
}

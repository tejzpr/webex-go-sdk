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
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	webex "github.com/WebexCommunity/webex-go-sdk/v2"
	"github.com/WebexCommunity/webex-go-sdk/v2/calling"
)

//go:embed static/*
var staticFiles embed.FS

// appState holds the server-side state
type appState struct {
	mu            sync.RWMutex
	client        *webex.WebexClient
	callingClient *calling.CallingClient
	line          *calling.Line
	activeCall    *calling.Call
	accessToken   string
	activeWSConn  *websocket.Conn // audio bridge WS, closed on shutdown
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

		// Close the audio bridge WebSocket so HandleSignaling unblocks
		state.mu.Lock()
		if state.activeWSConn != nil {
			state.activeWSConn.Close()
		}
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

		// Force exit if goroutines are still hanging
		time.Sleep(2 * time.Second)
		log.Println("Force exiting.")
		os.Exit(0)
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
	// The SDK handles WDM URL wiring, event filtering, and routing internally.
	go func() {
		if err := cc.ConnectMercury(client.Mercury()); err != nil {
			log.Printf("Mercury connection failed: %v", err)
		}
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

	address, ct, err := calling.NormalizeAddress(req.Address)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid address: %v", err))
		return
	}
	log.Printf("Dial: normalized address=%s type=%s", address, ct)

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
	// (AudioBridge detach is handled automatically by CallingClient)
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

// wsTransport adapts gorilla/websocket to calling.SignalingTransport.
type wsTransport struct{ conn *websocket.Conn }

func (t *wsTransport) ReadMessage() ([]byte, error) {
	_, data, err := t.conn.ReadMessage()
	return data, err
}
func (t *wsTransport) WriteMessage(data []byte) error {
	return t.conn.WriteMessage(websocket.TextMessage, data)
}

// handleAudioWS creates an AudioBridge and delegates signaling to the SDK.
func handleAudioWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Audio WS upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	log.Println("Audio WebSocket connected")

	state.mu.Lock()
	state.activeWSConn = conn
	state.mu.Unlock()
	defer func() {
		state.mu.Lock()
		state.activeWSConn = nil
		state.mu.Unlock()
	}()

	bridge, err := calling.NewAudioBridge(nil)
	if err != nil {
		log.Printf("Audio bridge: failed to create: %v", err)
		return
	}
	defer bridge.Close()

	// Register bridge with CallingClient for automatic call↔bridge binding
	state.mu.RLock()
	cc := state.callingClient
	state.mu.RUnlock()
	if cc != nil {
		cc.SetAudioBridge(bridge)
	}

	// Blocks until the WebSocket closes
	if err := bridge.HandleSignaling(&wsTransport{conn: conn}); err != nil {
		log.Printf("Audio bridge signaling ended: %v", err)
	}

	if cc != nil {
		cc.ClearAudioBridge()
	}
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

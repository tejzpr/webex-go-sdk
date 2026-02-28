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
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

// SignalingTransport is a transport-agnostic interface for exchanging
// WebRTC signaling messages (SDP offers/answers, ICE candidates) with
// a browser. Implement this over WebSocket, gRPC, HTTP polling, etc.
type SignalingTransport interface {
	// ReadMessage blocks until a signaling message arrives from the browser.
	// Returns the raw JSON bytes or an error (e.g. connection closed).
	ReadMessage() ([]byte, error)

	// WriteMessage sends a signaling message to the browser.
	WriteMessage(data []byte) error
}

// SignalingMessage is the JSON structure exchanged between the AudioBridge
// and the browser for WebRTC signaling.
type SignalingMessage struct {
	Type      string          `json:"type"`
	SDP       string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
}

// AudioBridgeConfig holds configuration for creating an AudioBridge.
type AudioBridgeConfig struct {
	// ICEServers for the browser-facing PeerConnection.
	// Default: Google STUN server.
	ICEServers []webrtc.ICEServer
}

// DefaultAudioBridgeConfig returns an AudioBridgeConfig with sensible defaults.
func DefaultAudioBridgeConfig() *AudioBridgeConfig {
	return &AudioBridgeConfig{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
}

// AudioBridge manages a browser-facing WebRTC PeerConnection and
// bidirectional RTP relay between the browser and a Mobius Call.
//
// Usage:
//
//	bridge, err := calling.NewAudioBridge(nil)
//	// Use bridge.PeerConnection() for WebRTC signaling with the browser.
//	// Call bridge.AttachCall(call) when a Call is active.
//	// Call bridge.Close() when done.
type AudioBridge struct {
	mu sync.RWMutex

	pc         *webrtc.PeerConnection
	localTrack *webrtc.TrackLocalStaticRTP // sends Mobius audio to browser

	call        *Call
	stopRelay   chan struct{}
	stopSilence chan struct{}
	closed      bool

	// Callbacks
	onICECandidate          func(candidate *webrtc.ICECandidate)
	onConnectionStateChange func(state webrtc.PeerConnectionState)
}

// NewAudioBridge creates a new AudioBridge with a browser-facing PeerConnection.
// The PeerConnection is configured with only PCMU/PCMA codecs to match Mobius.
func NewAudioBridge(config *AudioBridgeConfig) (*AudioBridge, error) {
	if config == nil {
		config = DefaultAudioBridgeConfig()
	}

	// Register only PCMU and PCMA so the browser negotiates the same
	// codec Mobius uses, enabling direct RTP relay without transcoding.
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMU,
			ClockRate: 8000,
			Channels:  1,
		},
		PayloadType: 0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMA,
			ClockRate: 8000,
			Channels:  1,
		},
		PayloadType: 8,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, err
	}

	// Register default interceptors (RTCP reports, SRTP) — required for
	// the browser bridge PC to keep the DTLS/SRTP session alive.
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: config.ICEServers,
	})
	if err != nil {
		return nil, err
	}

	// Create a local audio track with PCMU (to send Mobius remote audio to browser)
	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU},
		"audio-from-mobius",
		"webex-bridge",
	)
	if err != nil {
		pc.Close()
		return nil, err
	}
	if _, err := pc.AddTrack(localTrack); err != nil {
		pc.Close()
		return nil, err
	}

	ab := &AudioBridge{
		pc:          pc,
		localTrack:  localTrack,
		stopRelay:   make(chan struct{}),
		stopSilence: make(chan struct{}),
	}

	// Wire up browser→Mobius relay on incoming browser track
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("AudioBridge: received browser track codec=%s", track.Codec().MimeType)
		go ab.relayBrowserToMobius(track)
	})

	// Wire up ICE candidate forwarding
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		ab.mu.RLock()
		handler := ab.onICECandidate
		ab.mu.RUnlock()
		if handler != nil {
			handler(c)
		}
	})

	// Wire up connection state change forwarding
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("AudioBridge: browser PC state=%s", s.String())
		ab.mu.RLock()
		handler := ab.onConnectionStateChange
		ab.mu.RUnlock()
		if handler != nil {
			handler(s)
		}
	})

	// Start silence keepalive and Mobius→Browser relay goroutines
	go ab.silenceKeepalive()
	go ab.relayMobiusToBrowser()

	return ab, nil
}

// PeerConnection returns the browser-facing PeerConnection for signaling.
func (ab *AudioBridge) PeerConnection() *webrtc.PeerConnection {
	return ab.pc
}

// LocalTrack returns the local track used to send Mobius audio to the browser.
func (ab *AudioBridge) LocalTrack() *webrtc.TrackLocalStaticRTP {
	return ab.localTrack
}

// OnICECandidate sets the callback for when an ICE candidate is gathered
// on the browser-facing PeerConnection.
func (ab *AudioBridge) OnICECandidate(handler func(candidate *webrtc.ICECandidate)) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.onICECandidate = handler
}

// OnConnectionStateChange sets the callback for browser PC connection state changes.
func (ab *AudioBridge) OnConnectionStateChange(handler func(state webrtc.PeerConnectionState)) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.onConnectionStateChange = handler
}

// AttachCall attaches a Call to the bridge, enabling bidirectional audio relay.
// Can be called before or after the call is connected — the relay goroutines
// will wait for the call's Mobius PC to reach connected state.
func (ab *AudioBridge) AttachCall(call *Call) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.call = call
	log.Printf("AudioBridge: call attached (callId=%s)", call.GetCallID())
}

// DetachCall removes the current call from the bridge.
func (ab *AudioBridge) DetachCall() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.call = nil
	log.Println("AudioBridge: call detached")
}

// GetCall returns the currently attached call, or nil.
func (ab *AudioBridge) GetCall() *Call {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.call
}

// HandleSignaling runs the WebRTC signaling loop over the given transport.
// It forwards ICE candidates to the browser, processes incoming SDP offers
// and ICE candidates, and blocks until the transport returns an error
// (e.g. connection closed). Call this from your WebSocket handler.
//
// Example with gorilla/websocket:
//
//	bridge.HandleSignaling(&gorillaTransport{conn: wsConn})
func (ab *AudioBridge) HandleSignaling(transport SignalingTransport) error {
	// Forward ICE candidates from browser PC to the remote browser
	ab.OnICECandidate(func(c *webrtc.ICECandidate) {
		candidateJSON, _ := json.Marshal(c.ToJSON())
		msg := SignalingMessage{
			Type:      "ice-candidate",
			Candidate: candidateJSON,
		}
		msgBytes, _ := json.Marshal(msg)
		if err := transport.WriteMessage(msgBytes); err != nil {
			log.Printf("AudioBridge: failed to send ICE candidate: %v", err)
		}
	})

	pc := ab.pc

	// Read signaling messages from browser
	for {
		msgBytes, err := transport.ReadMessage()
		if err != nil {
			return fmt.Errorf("signaling transport read: %w", err)
		}

		var msg SignalingMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("AudioBridge: invalid signaling message: %v", err)
			continue
		}

		switch msg.Type {
		case "offer":
			log.Println("AudioBridge: received browser SDP offer")
			if err := pc.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  msg.SDP,
			}); err != nil {
				log.Printf("AudioBridge: set remote desc failed: %v", err)
				continue
			}

			answer, err := pc.CreateAnswer(nil)
			if err != nil {
				log.Printf("AudioBridge: create answer failed: %v", err)
				continue
			}
			if err := pc.SetLocalDescription(answer); err != nil {
				log.Printf("AudioBridge: set local desc failed: %v", err)
				continue
			}

			// Wait for ICE gathering
			gatherComplete := webrtc.GatheringCompletePromise(pc)
			<-gatherComplete

			localDesc := pc.LocalDescription()
			resp := SignalingMessage{
				Type: "answer",
				SDP:  localDesc.SDP,
			}
			respBytes, _ := json.Marshal(resp)
			if err := transport.WriteMessage(respBytes); err != nil {
				return fmt.Errorf("signaling transport write: %w", err)
			}
			log.Println("AudioBridge: sent SDP answer to browser")

		case "ice-candidate":
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal(msg.Candidate, &candidate); err != nil {
				log.Printf("AudioBridge: invalid ICE candidate: %v", err)
				continue
			}
			if err := pc.AddICECandidate(candidate); err != nil {
				log.Printf("AudioBridge: add ICE candidate failed: %v", err)
			}
		}
	}
}

// Close stops all relay goroutines and closes the browser PeerConnection.
func (ab *AudioBridge) Close() error {
	ab.mu.Lock()
	if ab.closed {
		ab.mu.Unlock()
		return nil
	}
	ab.closed = true
	ab.mu.Unlock()

	close(ab.stopRelay)
	if ab.pc != nil {
		return ab.pc.Close()
	}
	return nil
}

// relayBrowserToMobius reads RTP packets from the browser track and writes
// them directly to the Mobius local track. Each packet is written exactly once
// to preserve the browser's natural ~20ms pacing.
func (ab *AudioBridge) relayBrowserToMobius(track *webrtc.TrackRemote) {
	buf := make([]byte, 1500)
	var pktCount int
	var mobiusLocalTrack *webrtc.TrackLocalStaticRTP
	var connectedCh <-chan struct{}

	for {
		n, _, readErr := track.Read(buf)
		if readErr != nil {
			log.Printf("AudioBridge: browser track read ended: %v (relayed %d packets to Mobius)", readErr, pktCount)
			return
		}

		// Phase 1: wait for call and local track
		if mobiusLocalTrack == nil {
			ab.mu.RLock()
			call := ab.call
			ab.mu.RUnlock()
			if call == nil {
				continue
			}
			mobiusLocalTrack = call.GetMedia().GetLocalTrack()
			if mobiusLocalTrack == nil {
				continue
			}
			connectedCh = call.GetMedia().ConnectedCh()
		}

		// Phase 2: wait for Mobius PC to be connected
		select {
		case <-connectedCh:
			// Connected — relay this packet and all future ones
		default:
			continue // Not connected yet — discard packet
		}

		// Phase 3: relay packet directly (exactly once)
		pkt := &rtp.Packet{}
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}
		if writeErr := mobiusLocalTrack.WriteRTP(pkt); writeErr != nil {
			if pktCount == 0 || pktCount%500 == 0 {
				log.Printf("AudioBridge: write to Mobius failed (pkt %d): %v", pktCount, writeErr)
			}
		} else {
			pktCount++
			if pktCount == 1 {
				log.Printf("AudioBridge: first RTP packet relayed browser→Mobius (pt=%d ssrc=%d)", pkt.PayloadType, pkt.SSRC)
			} else if pktCount%500 == 0 {
				log.Printf("AudioBridge: relayed %d packets browser→Mobius", pktCount)
			}
		}
	}
}

// silenceKeepalive sends PCMU silence every 20ms to the browser local track
// until real Mobius audio arrives (stopSilence is closed) or the bridge is closed.
func (ab *AudioBridge) silenceKeepalive() {
	silenceBuf := make([]byte, 160)
	for i := range silenceBuf {
		silenceBuf[i] = 0xFF
	}
	var seq uint16
	var ts uint32
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var silenceCount int
	for {
		select {
		case <-ab.stopRelay:
			return
		case <-ab.stopSilence:
			return
		case <-ticker.C:
			seq++
			ts += 160 // 160 samples = 20ms at 8kHz
			if writeErr := ab.localTrack.WriteRTP(&rtp.Packet{
				Header: rtp.Header{
					Version:        2,
					PayloadType:    0,
					SequenceNumber: seq,
					Timestamp:      ts,
					Marker:         seq == 1, // Mark first packet
				},
				Payload: silenceBuf,
			}); writeErr != nil {
				log.Printf("AudioBridge: silence write error: %v (after %d packets)", writeErr, silenceCount)
				return
			}
			silenceCount++
			if silenceCount == 1 {
				log.Println("AudioBridge: silence keepalive started")
			}
		}
	}
}

// relayMobiusToBrowser polls for the Mobius remote track, then relays RTP
// directly to the browser local track. Stops silence keepalive after the
// first real packet is written.
func (ab *AudioBridge) relayMobiusToBrowser() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ab.stopRelay:
			return
		case <-ticker.C:
			ab.mu.RLock()
			call := ab.call
			ab.mu.RUnlock()
			if call == nil {
				continue
			}
			remoteTrack := call.GetMedia().GetRemoteTrack()
			if remoteTrack == nil {
				continue
			}
			ticker.Stop()
			log.Println("AudioBridge: starting Mobius→Browser relay")
			silenceStopped := false
			buf := make([]byte, 1500)
			for {
				n, _, readErr := remoteTrack.Read(buf)
				if readErr != nil {
					log.Printf("AudioBridge: Mobius remote track read ended: %v", readErr)
					return
				}
				pkt := &rtp.Packet{}
				if err := pkt.Unmarshal(buf[:n]); err != nil {
					continue
				}
				// Write directly — preserves original packet timing
				if writeErr := ab.localTrack.WriteRTP(pkt); writeErr != nil {
					log.Printf("AudioBridge: Mobius→Browser write error: %v", writeErr)
					return
				}
				// Stop silence only after first real packet is written
				if !silenceStopped {
					close(ab.stopSilence)
					silenceStopped = true
					log.Println("AudioBridge: silence keepalive stopped, real audio flowing")
				}
			}
		}
	}
}

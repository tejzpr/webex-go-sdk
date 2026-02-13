/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/pion/webrtc/v4"
)

// AudioTrackWriter is an interface for writing audio samples to a track
type AudioTrackWriter interface {
	WriteSample(sample []byte, duration uint32) error
}

// MediaEngine manages the WebRTC peer connection and media tracks for a call.
type MediaEngine struct {
	mu              sync.Mutex
	peerConnection  *webrtc.PeerConnection
	localTrack      *webrtc.TrackLocalStaticRTP
	remoteTrack     *webrtc.TrackRemote
	muted           bool
	onRemoteTrack   func(track *webrtc.TrackRemote)
	onICECandidate  func(candidate *webrtc.ICECandidate)
	api             *webrtc.API
}

// MediaConfig holds configuration for the media engine
type MediaConfig struct {
	// ICEServers is the list of ICE servers (STUN/TURN) to use
	ICEServers []webrtc.ICEServer
	// AudioCodecs is the list of audio codecs to use (default: opus, PCMU, PCMA)
	AudioCodecs []string
}

// DefaultMediaConfig returns a MediaConfig with sensible defaults
func DefaultMediaConfig() *MediaConfig {
	return &MediaConfig{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
}

// NewMediaEngine creates a new WebRTC media engine for a call
func NewMediaEngine(config *MediaConfig) (*MediaEngine, error) {
	if config == nil {
		config = DefaultMediaConfig()
	}

	// Create a MediaEngine and register codecs
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("failed to register codecs: %w", err)
	}

	// Create the API with the MediaEngine
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Create PeerConnection configuration
	pcConfig := webrtc.Configuration{
		ICEServers: config.ICEServers,
	}

	// Create the PeerConnection
	pc, err := api.NewPeerConnection(pcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	engine := &MediaEngine{
		peerConnection: pc,
		api:            api,
	}

	// Set up ICE candidate handler
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil && engine.onICECandidate != nil {
			engine.onICECandidate(c)
		}
	})

	// Set up remote track handler
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		engine.mu.Lock()
		engine.remoteTrack = track
		handler := engine.onRemoteTrack
		engine.mu.Unlock()

		if handler != nil {
			handler(track)
		}
	})

	return engine, nil
}

// OnRemoteTrack sets the callback for when a remote audio track is received
func (me *MediaEngine) OnRemoteTrack(handler func(track *webrtc.TrackRemote)) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.onRemoteTrack = handler
}

// OnICECandidate sets the callback for when an ICE candidate is gathered
func (me *MediaEngine) OnICECandidate(handler func(candidate *webrtc.ICECandidate)) {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.onICECandidate = handler
}

// AddAudioTrack adds a local audio track to the peer connection.
// Returns the track so the caller can write RTP packets to it.
func (me *MediaEngine) AddAudioTrack() (*webrtc.TrackLocalStaticRTP, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Create an audio track with Opus codec
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"webex-calling",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	_, err = me.peerConnection.AddTrack(track)
	if err != nil {
		return nil, fmt.Errorf("failed to add audio track: %w", err)
	}

	me.localTrack = track
	return track, nil
}

// CreateOffer creates an SDP offer for the peer connection
func (me *MediaEngine) CreateOffer() (string, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	offer, err := me.peerConnection.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create offer: %w", err)
	}

	// Set the local description
	if err := me.peerConnection.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(me.peerConnection)
	<-gatherComplete

	// Return the SDP with ICE candidates included
	localDesc := me.peerConnection.LocalDescription()
	if localDesc == nil {
		return "", fmt.Errorf("local description is nil after gathering")
	}

	return localDesc.SDP, nil
}

// CreateAnswer creates an SDP answer for the peer connection
func (me *MediaEngine) CreateAnswer() (string, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	answer, err := me.peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create answer: %w", err)
	}

	if err := me.peerConnection.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	gatherComplete := webrtc.GatheringCompletePromise(me.peerConnection)
	<-gatherComplete

	localDesc := me.peerConnection.LocalDescription()
	if localDesc == nil {
		return "", fmt.Errorf("local description is nil after gathering")
	}

	return localDesc.SDP, nil
}

// SetRemoteOffer sets the remote SDP offer on the peer connection
func (me *MediaEngine) SetRemoteOffer(sdp string) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	return me.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	})
}

// SetRemoteAnswer sets the remote SDP answer on the peer connection
func (me *MediaEngine) SetRemoteAnswer(sdp string) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	return me.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	})
}

// Mute disables the local audio track
func (me *MediaEngine) Mute() {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.muted = true
}

// Unmute enables the local audio track
func (me *MediaEngine) Unmute() {
	me.mu.Lock()
	defer me.mu.Unlock()
	me.muted = false
}

// IsMuted returns whether the local audio is muted
func (me *MediaEngine) IsMuted() bool {
	me.mu.Lock()
	defer me.mu.Unlock()
	return me.muted
}

// GetLocalTrack returns the local audio track
func (me *MediaEngine) GetLocalTrack() *webrtc.TrackLocalStaticRTP {
	me.mu.Lock()
	defer me.mu.Unlock()
	return me.localTrack
}

// GetRemoteTrack returns the remote audio track
func (me *MediaEngine) GetRemoteTrack() *webrtc.TrackRemote {
	me.mu.Lock()
	defer me.mu.Unlock()
	return me.remoteTrack
}

// GetConnectionState returns the current peer connection state
func (me *MediaEngine) GetConnectionState() webrtc.PeerConnectionState {
	return me.peerConnection.ConnectionState()
}

// Close closes the peer connection and releases resources
func (me *MediaEngine) Close() error {
	me.mu.Lock()
	defer me.mu.Unlock()

	if me.peerConnection != nil {
		if err := me.peerConnection.Close(); err != nil {
			return fmt.Errorf("failed to close peer connection: %w", err)
		}
	}
	return nil
}

// ---- ROAP Protocol Helpers ----

// RoapToSDP extracts the SDP from a ROAP message received from Mobius
func RoapToSDP(roap *RoapMessage) string {
	if roap == nil {
		return ""
	}
	return roap.SDP
}

// SDPToRoapOffer wraps an SDP string into a ROAP OFFER message
func SDPToRoapOffer(sdp string, seq int) *RoapMessage {
	return &RoapMessage{
		Seq:         seq,
		MessageType: RoapMessageOffer,
		SDP:         sdp,
	}
}

// SDPToRoapAnswer wraps an SDP string into a ROAP ANSWER message
func SDPToRoapAnswer(sdp string, seq int) *RoapMessage {
	return &RoapMessage{
		Seq:         seq,
		MessageType: RoapMessageAnswer,
		SDP:         sdp,
	}
}

// NewRoapOK creates a ROAP OK message
func NewRoapOK(seq int) *RoapMessage {
	return &RoapMessage{
		Seq:         seq,
		MessageType: RoapMessageOK,
	}
}

// ModifySdpForIPv4 removes IPv6 candidates from SDP for compatibility
func ModifySdpForIPv4(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		// Skip IPv6 candidates
		if strings.Contains(line, "a=candidate:") && strings.Contains(line, " IP6 ") {
			log.Printf("Removing IPv6 candidate: %s", line)
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\r\n")
}

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

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

// AudioTrackWriter is an interface for writing audio samples to a track
type AudioTrackWriter interface {
	WriteSample(sample []byte, duration uint32) error
}

// MediaEngine manages the WebRTC peer connection and media tracks for a call.
type MediaEngine struct {
	mu             sync.Mutex
	peerConnection *webrtc.PeerConnection
	localTrack     *webrtc.TrackLocalStaticRTP
	remoteTrack    *webrtc.TrackRemote
	muted          bool
	onRemoteTrack  func(track *webrtc.TrackRemote)
	onICECandidate func(candidate *webrtc.ICECandidate)
	api            *webrtc.API
}

// MediaConfig holds configuration for the media engine
type MediaConfig struct {
	// ICEServers is the list of ICE servers (STUN/TURN) to use
	ICEServers []webrtc.ICEServer
	// AudioCodecs is the list of audio codecs to use (default: opus, PCMU, PCMA)
	AudioCodecs []string
}

// DefaultMediaConfig returns a MediaConfig with sensible defaults.
// STUN is required because the Go server is typically behind NAT and
// BroadWorks (ice-lite) needs a public srflx candidate to reach us.
// The JS SDK uses iceServers:[] because the browser handles NAT traversal
// via ICE connectivity checks, but Pion needs an explicit public candidate.
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

	// Register only PCMU and PCMA — BroadWorks/Mobius consistently selects PCMU.
	// Avoid RegisterDefaultCodecs which adds Opus/G722/video codecs that
	// BroadWorks doesn't support and can cause negotiation issues.
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000},
		PayloadType:        0,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("failed to register PCMU: %w", err)
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000},
		PayloadType:        8,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("failed to register PCMA: %w", err)
	}

	// BroadWorks (ice-lite) sends RTP before Pion finishes processing the SDP answer.
	// Enable undeclared SSRC handling so OnTrack fires for early media.
	settings := webrtc.SettingEngine{}
	settings.SetHandleUndeclaredSSRCWithoutAnswer(true)

	// Register default interceptors (RTCP reports, NACK, TWCC) — required when
	// using a custom MediaEngine/SettingEngine, otherwise Pion won't process
	// incoming SRTP properly and OnTrack may not fire.
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, fmt.Errorf("failed to register default interceptors: %w", err)
	}

	// Create the API with the MediaEngine, SettingEngine, and interceptors
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(m),
		webrtc.WithSettingEngine(settings),
		webrtc.WithInterceptorRegistry(i),
	)

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
		if c != nil {
			log.Printf("Mobius PC: ICE candidate gathered: %s", c.String())
			if engine.onICECandidate != nil {
				engine.onICECandidate(c)
			}
		}
	})

	// Log connection state changes for debugging
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Mobius PC: connection state → %s", s.String())
	})
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Printf("Mobius PC: ICE connection state → %s", s.String())
	})

	// Set up remote track handler
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Mobius PC: OnTrack fired! codec=%s ssrc=%d", track.Codec().MimeType, track.SSRC())
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

// IsConnected returns true if the Mobius PeerConnection is in the connected state.
func (me *MediaEngine) IsConnected() bool {
	me.mu.Lock()
	defer me.mu.Unlock()
	if me.peerConnection == nil {
		return false
	}
	s := me.peerConnection.ConnectionState()
	return s == webrtc.PeerConnectionStateConnected
}

// AddAudioTrack adds a local audio track to the peer connection.
// Uses PCMU codec since BroadWorks/Mobius consistently selects it.
func (me *MediaEngine) AddAudioTrack() (*webrtc.TrackLocalStaticRTP, error) {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Create a PCMU track — Mobius/BroadWorks always negotiates PCMU.
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000},
		"audio",
		"webex-calling",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Use AddTransceiverFromTrack with sendrecv so Pion creates a proper
	// bidirectional transceiver. This is required for OnTrack to fire when
	// BroadWorks sends RTP back to us.
	transceiver, err := me.peerConnection.AddTransceiverFromTrack(track,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendrecv},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add audio transceiver: %w", err)
	}

	// Read RTCP from the sender to keep the connection alive
	go func() {
		sender := transceiver.Sender()
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := sender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

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
		SDP:  fixIncomingSdp(sdp),
	})
}

// SetRemoteAnswer sets the remote SDP answer on the peer connection.
// If the PC is already in stable state (answer already applied), this is a no-op.
func (me *MediaEngine) SetRemoteAnswer(sdp string) error {
	me.mu.Lock()
	defer me.mu.Unlock()

	// Guard against duplicate answers (Mercury may deliver the same ROAP answer
	// more than once due to reconnection or duplicate event delivery).
	if me.peerConnection.SignalingState() == webrtc.SignalingStateStable {
		log.Printf("Mobius PC: ignoring duplicate SDP answer (signaling state already stable)")
		return nil
	}

	return me.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  fixIncomingSdp(sdp),
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

// GetPeerConnection returns the underlying Pion PeerConnection for advanced use (e.g. RTP relay)
func (me *MediaEngine) GetPeerConnection() *webrtc.PeerConnection {
	return me.peerConnection
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

// fixIncomingSdp patches incoming BroadWorks SDP for Pion v4 compatibility:
// - Injects a=mid:0 after the first m= line if missing (Pion v4 requires mid)
// - Adds a=group:BUNDLE 0 at session level if missing
func fixIncomingSdp(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	result := make([]string, 0, len(lines)+2)
	hasMid := false
	hasBundle := false
	inMedia := false

	// First pass: check what's present
	for _, line := range lines {
		if strings.HasPrefix(line, "a=mid:") {
			hasMid = true
		}
		if strings.HasPrefix(line, "a=group:BUNDLE") {
			hasBundle = true
		}
	}

	// Second pass: inject missing attributes
	for _, line := range lines {
		if strings.HasPrefix(line, "m=") {
			// Before the first m= line, inject BUNDLE if missing
			if !inMedia && !hasBundle {
				result = append(result, "a=group:BUNDLE 0")
			}
			inMedia = true
			result = append(result, line)
			// After m= line, inject mid if missing
			if !hasMid {
				result = append(result, "a=mid:0")
			}
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\r\n")
}

// ModifySdpForMobius cleans up the SDP offer for BroadWorks/Mobius compatibility:
// - Removes IPv6 candidates (BroadWorks only supports IPv4)
// - Converts port 9 to 0 in m= line (JS SDK: convertPort9to0)
// - Removes rtcp-fb lines (BroadWorks doesn't support transport-cc)
// - Removes extmap lines (BroadWorks doesn't support RTP header extensions)
// - Removes extmap-allow-mixed
func ModifySdpForMobius(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		// Skip IPv6 candidates (address contains ":")
		if strings.HasPrefix(line, "a=candidate:") {
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				addr := parts[4]
				if strings.Contains(addr, ":") {
					continue
				}
			}
		}
		// Skip rtcp-fb lines (transport-cc not supported by BroadWorks)
		if strings.HasPrefix(line, "a=rtcp-fb:") {
			continue
		}
		// Skip extmap lines
		if strings.HasPrefix(line, "a=extmap:") {
			continue
		}
		// Skip extmap-allow-mixed
		if strings.HasPrefix(line, "a=extmap-allow-mixed") {
			continue
		}
		// Note: port 9 in SDP is a placeholder (Pion default). Do NOT convert to 0
		// as port 0 means "reject this media stream" in SDP.
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\r\n")
}

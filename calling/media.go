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
	connectedCh    chan struct{} // closed when PC reaches connected state
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

	// Register audio codecs for BroadWorks/Mobius compatibility.
	m := &webrtc.MediaEngine{}
	for _, codec := range []webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU, ClockRate: 8000}, PayloadType: 0},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMA, ClockRate: 8000}, PayloadType: 8},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/telephone-event", ClockRate: 8000, SDPFmtpLine: "0-15"}, PayloadType: 101},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/CN", ClockRate: 8000}, PayloadType: 19},
	} {
		if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
			return nil, fmt.Errorf("failed to register codec %s: %w", codec.MimeType, err)
		}
	}

	// BroadWorks (ice-lite) sends RTP before Pion finishes processing the SDP answer.
	// Enable undeclared SSRC handling so OnTrack fires for early media.
	settings := webrtc.SettingEngine{}
	settings.SetHandleUndeclaredSSRCWithoutAnswer(true)
	// BroadWorks is ice-lite and offers a=setup:actpass. It will NOT initiate
	// DTLS (it only acts as server when we're active, or client when we're passive).
	// With ice-lite, we must be the DTLS client (active). Pion defaults to passive
	// when answering, which causes a DTLS deadlock — nobody initiates the handshake.
	if err := settings.SetAnsweringDTLSRole(webrtc.DTLSRoleClient); err != nil {
		return nil, fmt.Errorf("failed to set DTLS role: %w", err)
	}

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
		connectedCh:    make(chan struct{}),
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

	// Log connection state changes and signal when connected
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Mobius PC: connection state → %s", s.String())
		if s == webrtc.PeerConnectionStateConnected {
			engine.mu.Lock()
			select {
			case <-engine.connectedCh:
			default:
				close(engine.connectedCh)
			}
			engine.mu.Unlock()
		}
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

// OnRemoteTrack sets the callback for when a remote audio track is received.
// If a track was already received before the handler was set, the handler is
// called immediately with that track.
func (me *MediaEngine) OnRemoteTrack(handler func(track *webrtc.TrackRemote)) {
	me.mu.Lock()
	me.onRemoteTrack = handler
	existingTrack := me.remoteTrack
	me.mu.Unlock()

	if existingTrack != nil && handler != nil {
		log.Printf("OnRemoteTrack: track already available, calling handler immediately")
		handler(existingTrack)
	}
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

// ConnectedCh returns a channel that is closed when the Mobius PC reaches connected state.
func (me *MediaEngine) ConnectedCh() <-chan struct{} {
	return me.connectedCh
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

	fixed := fixIncomingSdp(sdp)
	return me.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  fixed,
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
// - Normalises line endings to \r\n (SDP spec)
func fixIncomingSdp(sdp string) string {
	// Normalize line endings: BroadWorks may send \n only
	sdp = strings.ReplaceAll(sdp, "\r\n", "\n")
	lines := strings.Split(sdp, "\n")

	result := make([]string, 0, len(lines)+4)
	hasMid := false
	hasBundle := false
	hasDirection := false
	inMedia := false

	// First pass: check what's present
	for _, line := range lines {
		if strings.HasPrefix(line, "a=mid:") {
			hasMid = true
		}
		if strings.HasPrefix(line, "a=group:BUNDLE") {
			hasBundle = true
		}
		if strings.HasPrefix(line, "a=sendrecv") || strings.HasPrefix(line, "a=sendonly") ||
			strings.HasPrefix(line, "a=recvonly") || strings.HasPrefix(line, "a=inactive") {
			hasDirection = true
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
			// After m= line, inject mid and direction if missing
			if !hasMid {
				result = append(result, "a=mid:0")
			}
			// Pion v4 requires an explicit direction attribute; BroadWorks omits it
			// (SDP spec says default is sendrecv, but Pion doesn't apply the default)
			if !hasDirection {
				result = append(result, "a=sendrecv")
			}
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\r\n")
}

// ModifySdpForMobius cleans up the SDP offer/answer for BroadWorks/Mobius compatibility:
// - Removes IPv6 candidates (BroadWorks only supports IPv4)
// - Removes rtcp-fb lines (BroadWorks doesn't support transport-cc)
// - Removes extmap lines (BroadWorks doesn't support RTP header extensions)
// - Removes extmap-allow-mixed
// - Copies c= line from media level to session level (copyClineToSessionLevel)
func ModifySdpForMobius(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	filtered := make([]string, 0, len(lines))

	// First pass: collect info we need
	// - Find first IPv4 srflx candidate (preferred) or host candidate for c= line
	// - Find media-level c= line for copyClineToSessionLevel
	var bestCandidateIP, bestCandidatePort string
	var mediaCline string
	hasSessionCline := false
	inMedia := false
	for _, line := range lines {
		if strings.HasPrefix(line, "m=") {
			inMedia = true
		}
		if strings.HasPrefix(line, "c=") {
			if inMedia {
				mediaCline = line
			} else {
				hasSessionCline = true
			}
		}
		if strings.HasPrefix(line, "a=candidate:") {
			parts := strings.Fields(line)
			if len(parts) >= 8 {
				addr := parts[4]
				port := parts[5]
				// Skip IPv6
				if strings.Contains(addr, ":") {
					continue
				}
				candidateType := ""
				if len(parts) >= 8 {
					candidateType = parts[7] // "host" or "srflx"
				}
				// Prefer srflx (public IP), fall back to host
				if candidateType == "srflx" || bestCandidateIP == "" {
					bestCandidateIP = addr
					bestCandidatePort = port
				}
			}
		}
	}

	for _, line := range lines {
		// Copy c= line to session level (before first m= line)
		// Use candidate IP if the c= line has 0.0.0.0
		if strings.HasPrefix(line, "m=") && !hasSessionCline {
			if bestCandidateIP != "" {
				filtered = append(filtered, fmt.Sprintf("c=IN IP4 %s", bestCandidateIP))
			} else if mediaCline != "" {
				filtered = append(filtered, mediaCline)
			}
			hasSessionCline = true
		}

		// Replace c=IN IP4 0.0.0.0 with real candidate IP
		if strings.HasPrefix(line, "c=") && strings.Contains(line, "0.0.0.0") && bestCandidateIP != "" {
			filtered = append(filtered, fmt.Sprintf("c=IN IP4 %s", bestCandidateIP))
			continue
		}

		// Replace port 9 in m= line with real candidate port
		if strings.HasPrefix(line, "m=audio 9 ") && bestCandidatePort != "" {
			line = strings.Replace(line, "m=audio 9 ", "m=audio "+bestCandidatePort+" ", 1)
		}

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
		// Skip rtcp-rsize (not supported by BroadWorks)
		if strings.HasPrefix(line, "a=rtcp-rsize") {
			continue
		}
		// BroadWorks ice-lite + actpass offer: we must be DTLS client (active).
		// Pion defaults to passive when answering, but ice-lite won't initiate DTLS.
		if line == "a=setup:passive" {
			filtered = append(filtered, "a=setup:active")
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\r\n")
}

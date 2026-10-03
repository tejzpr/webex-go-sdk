/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package mercury

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	"github.com/gorilla/websocket"
)

func newGenerationTestClient(t *testing.T, cfg *Config) *Client {
	t.Helper()
	sdk, err := webexsdk.NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	return New(sdk, cfg)
}

// dialPair returns a client-side websocket connected to a server that keeps
// the socket open until the test ends.
func dialPair(t *testing.T) *websocket.Conn {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// A listener for a replaced connection must only tear down its own
// generation: it closes its own done channel and leaves the current
// connection's state and done channel alone.
func TestStaleListenerDoesNotTouchCurrentGeneration(t *testing.T) {
	c := newGenerationTestClient(t, nil)
	oldConn := dialPair(t)
	newConn := dialPair(t)

	oldDone := make(chan struct{})
	oldCloseCh := make(chan struct{})

	// Simulate the state right after a reconnect installed newConn.
	c.mu.Lock()
	c.conn = newConn
	c.connected = true
	newDone := c.done
	c.mu.Unlock()

	go c.listen(oldConn, oldDone, oldCloseCh)
	_ = oldConn.Close()

	select {
	case <-oldDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stale listener did not exit")
	}

	if isClosed(newDone) {
		t.Error("stale listener closed the current generation's done channel")
	}
	if !c.IsConnected() {
		t.Error("stale listener marked the current connection disconnected")
	}
	if c.reconnecting.Load() {
		t.Error("stale listener started a reconnect for a replaced connection")
	}
}

// After a read error, handleConnectionError sets connected = false but leaves
// the failed connection installed while it schedules a reconnect. Disconnect
// in that window must still cancel the generation, not return early.
func TestDisconnectAfterConnectionErrorCancelsGeneration(t *testing.T) {
	c := newGenerationTestClient(t, nil)
	conn := dialPair(t)

	c.mu.Lock()
	c.conn = conn
	c.connected = false
	c.connecting = false
	oldCloseCh := c.closeCh
	c.mu.Unlock()

	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	if !isClosed(oldCloseCh) {
		t.Error("Disconnect returned early and left the generation running")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		t.Error("Disconnect left the failed connection installed")
	}
}

// A connect loop from a cancelled generation must not clear the connecting
// flag of a newer attempt.
func TestStaleAttemptDoesNotClearNewerConnecting(t *testing.T) {
	c := newGenerationTestClient(t, nil)

	c.mu.Lock()
	staleCloseCh := c.closeCh
	c.closeCh = make(chan struct{}) // a Disconnect + Connect happened
	c.connecting = true
	c.mu.Unlock()

	c.finishConnecting(staleCloseCh)

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connecting {
		t.Error("stale attempt cleared the newer attempt's connecting flag")
	}
}

// Disconnect while a reconnect is dialing/authenticating must leave the
// client disconnected, and the in-flight socket must be closed.
func TestDisconnectDuringReconnectStaysDisconnected(t *testing.T) {
	var connectionNumber atomic.Int32
	reconnectDialed := make(chan struct{})
	releaseAuth := make(chan struct{})
	reconnectClosed := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		n := connectionNumber.Add(1)
		if n == 2 {
			// Hold the reconnect in authentication until the test has
			// called Disconnect.
			close(reconnectDialed)
			<-releaseAuth
		}
		if err := conn.WriteJSON(map[string]any{"data": map[string]string{"eventType": "mercury.buffer_state"}}); err != nil {
			return
		}
		if n == 1 {
			return // drop the first connection to trigger a reconnect
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				if n == 2 {
					close(reconnectClosed)
				}
				return
			}
		}
	}))
	defer server.Close()

	c := newGenerationTestClient(t, &Config{
		ForceCloseDelay:             time.Second,
		PingInterval:                time.Second,
		PongTimeout:                 time.Second,
		BackoffTimeMax:              10 * time.Millisecond,
		BackoffTimeReset:            time.Millisecond,
		MaxRetries:                  3,
		InitialConnectionMaxRetries: 3,
	})
	c.SetCustomWebSocketURL("ws" + strings.TrimPrefix(server.URL, "http"))
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	select {
	case <-reconnectDialed:
	case <-time.After(2 * time.Second):
		t.Fatal("Mercury did not start reconnecting")
	}

	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	close(releaseAuth)

	select {
	case <-reconnectClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("the reconnect socket was not closed after Disconnect")
	}
	if c.IsConnected() {
		t.Fatal("client is connected after Disconnect")
	}
	if got := connectionNumber.Load(); got != 2 {
		t.Fatalf("connection count = %d, want 2 (no reconnect after Disconnect)", got)
	}
}

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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	"github.com/gorilla/websocket"
)

func TestReconnectKeepsNewConnectionHeartbeatAlive(t *testing.T) {
	var connectionNumber atomic.Int32
	secondConnected := make(chan struct{})
	secondPing := make(chan struct{})
	var connectedOnce sync.Once
	var pingOnce sync.Once
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade WebSocket: %v", err)
			return
		}
		defer conn.Close()

		n := connectionNumber.Add(1)
		conn.SetPingHandler(func(appData string) error {
			if n == 2 {
				pingOnce.Do(func() { close(secondPing) })
			}
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})

		if err := conn.WriteJSON(map[string]any{"data": map[string]string{"eventType": "mercury.buffer_state"}}); err != nil {
			t.Errorf("write authorization confirmation: %v", err)
			return
		}
		if n == 1 {
			time.Sleep(40 * time.Millisecond)
			return
		}

		connectedOnce.Do(func() { close(secondConnected) })
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	sdk, err := webexsdk.NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	config := &Config{
		ForceCloseDelay:             time.Second,
		PingInterval:                20 * time.Millisecond,
		PongTimeout:                 100 * time.Millisecond,
		BackoffTimeMax:              20 * time.Millisecond,
		BackoffTimeReset:            5 * time.Millisecond,
		MaxRetries:                  3,
		InitialConnectionMaxRetries: 3,
	}
	client := New(sdk, config)
	client.SetCustomWebSocketURL("ws" + strings.TrimPrefix(server.URL, "http"))
	if err := client.Connect(); err != nil {
		t.Fatalf("connect Mercury: %v", err)
	}
	defer client.Disconnect()

	select {
	case <-secondConnected:
	case <-time.After(2 * time.Second):
		t.Fatal("Mercury did not reconnect")
	}
	select {
	case <-secondPing:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("reconnected Mercury socket did not send a heartbeat")
	}

	if got := connectionNumber.Load(); got < 2 {
		t.Fatalf("connection count = %d, want at least 2", got)
	}
	if !client.IsConnected() {
		t.Fatal("replacement connection was marked disconnected by the stale connection")
	}
}

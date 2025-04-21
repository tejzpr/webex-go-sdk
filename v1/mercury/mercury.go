/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package mercury

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

// Config holds the configuration for the Mercury plugin
type Config struct {
	ForceCloseDelay             time.Duration // Delay after which to force close a websocket connection if no close event is received
	PingInterval                time.Duration // Interval between ping messages
	PongTimeout                 time.Duration // Timeout for receiving a pong response
	BackoffTimeMax              time.Duration // Maximum time between connection attempts
	BackoffTimeReset            time.Duration // Initial time before the first retry
	MaxRetries                  int           // Number of times to retry before giving up
	InitialConnectionMaxRetries int           // Number of times to retry before giving up on the initial connection
}

// DefaultConfig returns the default configuration for the Mercury plugin
func DefaultConfig() *Config {
	return &Config{
		ForceCloseDelay:             10 * time.Second,
		PingInterval:                30 * time.Second,
		PongTimeout:                 10 * time.Second,
		BackoffTimeMax:              32 * time.Second,
		BackoffTimeReset:            1 * time.Second,
		MaxRetries:                  3,
		InitialConnectionMaxRetries: 5,
	}
}

// DeviceProvider is an interface for getting the websocket URL from a device
type DeviceProvider interface {
	Register() error
	GetWebSocketURL() (string, error)
}

// EventHandler is a function that handles a websocket event
type EventHandler func(event *Event)

// Event represents a Mercury websocket event
type Event struct {
	// JSON fields from the websocket message
	ID               string                 `json:"id,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
	Timestamp        int64                  `json:"timestamp,omitempty"`
	TrackingID       string                 `json:"trackingId,omitempty"`
	AlertType        string                 `json:"alertType,omitempty"`
	SequenceNumber   int64                  `json:"sequenceNumber,omitempty"`
	FilterMessage    bool                   `json:"filterMessage,omitempty"`
	WsWriteTimestamp int64                  `json:"wsWriteTimestamp,omitempty"`
	Headers          map[string]interface{} `json:"headers,omitempty"`

	// Derived fields populated during processing
	EventType      string                 `json:"-"` // Populated from data.eventType
	ActivityType   string                 `json:"-"` // Populated from data.activity.verb
	WebSocketError string                 `json:"-"` // Internal field for connection errors
	ResourceType   string                 `json:"-"` // Type of resource in the event
	ActorID        string                 `json:"-"` // ID of the actor who performed the activity
	OrgID          string                 `json:"-"` // Organization ID associated with the activity
	Resource       map[string]interface{} `json:"-"` // Resource data from the event
}

// Client is the Mercury API client for websocket communication
type Client struct {
	webexClient        *webexsdk.Client
	config             *Config
	conn               *websocket.Conn
	listening          bool
	connected          bool
	connecting         bool
	hasConnected       bool
	mu                 sync.Mutex
	eventHandlers      map[string][]EventHandler
	closeCh            chan struct{}
	done               chan struct{}
	timeOffset         int64
	retryCount         int
	currentBackoff     time.Duration
	deviceProvider     DeviceProvider
	customWebSocketURL string
}

// New creates a new Mercury plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient:    webexClient,
		config:         config,
		eventHandlers:  make(map[string][]EventHandler),
		closeCh:        make(chan struct{}),
		done:           make(chan struct{}),
		currentBackoff: config.BackoffTimeReset,
	}
}

// SetDeviceProvider sets a device provider to use for getting websocket URLs
func (c *Client) SetDeviceProvider(provider DeviceProvider) {
	c.mu.Lock()
	c.deviceProvider = provider
	c.mu.Unlock()
}

// SetCustomWebSocketURL sets a custom WebSocket URL for Mercury connection
func (c *Client) SetCustomWebSocketURL(url string) {
	c.mu.Lock()
	c.customWebSocketURL = url
	c.mu.Unlock()
}

// Connect establishes a websocket connection to the Mercury service
func (c *Client) Connect() error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}

	if c.connecting {
		c.mu.Unlock()
		return fmt.Errorf("connection attempt already in progress")
	}

	c.connecting = true
	deviceProvider := c.deviceProvider
	customURL := c.customWebSocketURL
	c.mu.Unlock()

	// If we have a custom URL, use it directly
	if customURL != "" {
		return c.connectWithBackoff(customURL)
	}

	// Try to get the websocket URL from the device provider
	if deviceProvider == nil {
		c.mu.Lock()
		c.connecting = false
		c.mu.Unlock()
		return fmt.Errorf("no device provider or custom URL available")
	}

	// Register the device and get WebSocket URL
	if err := deviceProvider.Register(); err != nil {
		c.mu.Lock()
		c.connecting = false
		c.mu.Unlock()
		return fmt.Errorf("failed to register device: %v", err)
	}

	wsURL, err := deviceProvider.GetWebSocketURL()
	if err != nil || wsURL == "" {
		c.mu.Lock()
		c.connecting = false
		c.mu.Unlock()
		if err != nil {
			return fmt.Errorf("failed to get WebSocket URL from device: %v", err)
		}
		return fmt.Errorf("device provider returned empty WebSocket URL")
	}

	return c.connectWithBackoff(wsURL)
}

// Disconnect closes the websocket connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	if !c.connected && !c.connecting {
		c.mu.Unlock()
		return nil
	}

	// Signal all goroutines to stop
	close(c.closeCh)

	// Create new channels for future connections
	c.closeCh = make(chan struct{})
	c.done = make(chan struct{})

	conn := c.conn
	c.conn = nil
	c.connected = false
	c.connecting = false
	c.mu.Unlock()

	if conn != nil {
		// Send close message and close the connection
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Disconnected by client"))
		_ = conn.Close()
	}

	return nil
}

// Listen is an alias for Connect, maintained for compatibility with JS SDK
func (c *Client) Listen() error {
	return c.Connect()
}

// StopListening is an alias for Disconnect, maintained for compatibility with JS SDK
func (c *Client) StopListening() error {
	return c.Disconnect()
}

// On registers an event handler for a specific event type
func (c *Client) On(eventType string, handler EventHandler) {
	if handler == nil {
		return
	}

	c.mu.Lock()
	handlers, ok := c.eventHandlers[eventType]
	if !ok {
		handlers = []EventHandler{}
	}
	c.eventHandlers[eventType] = append(handlers, handler)
	c.mu.Unlock()
}

// Off removes an event handler for a specific event type
func (c *Client) Off(eventType string, handler EventHandler) {
	if handler == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	handlers, ok := c.eventHandlers[eventType]
	if !ok {
		return
	}

	// Find the handler by comparing function pointers
	handlerPtr := fmt.Sprintf("%p", handler)
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == handlerPtr {
			// Remove handler by preserving order
			c.eventHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// Clean up empty handler slices
	if len(c.eventHandlers[eventType]) == 0 {
		delete(c.eventHandlers, eventType)
	}
}

// IsConnected returns whether the client is connected to the Mercury service
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// EventHandlers returns a copy of the event handlers map (for testing)
func (c *Client) EventHandlers() map[string][]EventHandler {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a copy to avoid race conditions
	result := make(map[string][]EventHandler, len(c.eventHandlers))
	for k, v := range c.eventHandlers {
		handlers := make([]EventHandler, len(v))
		copy(handlers, v)
		result[k] = handlers
	}

	return result
}

// connectWithBackoff attempts to connect to the Mercury service with exponential backoff
func (c *Client) connectWithBackoff(wsURL string) error {
	// Reset retry count on new connection attempt
	c.retryCount = 0
	c.currentBackoff = c.config.BackoffTimeReset

	maxRetries := c.config.MaxRetries
	if !c.hasConnected {
		maxRetries = c.config.InitialConnectionMaxRetries
	}

	var err error
	for c.retryCount <= maxRetries {
		err = c.attemptConnection(wsURL)
		if err == nil {
			return nil // Connection successful
		}

		// Increment retry count
		c.retryCount++
		if c.retryCount > maxRetries {
			break // Exceeded max retries
		}

		// Wait for backoff time or until connection is closed
		select {
		case <-time.After(c.currentBackoff):
			// Double the backoff time, up to max
			c.currentBackoff *= 2
			if c.currentBackoff > c.config.BackoffTimeMax {
				c.currentBackoff = c.config.BackoffTimeMax
			}
		case <-c.closeCh:
			return nil // Stopped by user
		}
	}

	// Couldn't connect after all retries
	c.mu.Lock()
	c.connecting = false
	c.mu.Unlock()
	return fmt.Errorf("failed to connect after %d attempts: %v", c.retryCount, err)
}

// attemptConnection makes a single connection attempt to the Mercury service
func (c *Client) attemptConnection(wsURL string) error {
	// Get auth token and prepare URL
	token := c.webexClient.AccessToken
	parsedURL, err := c.prepareWebSocketURL(wsURL)
	if err != nil {
		return err
	}

	// Connect to websocket
	conn, err := c.dialWebSocket(parsedURL.String(), token)
	if err != nil {
		return err
	}

	// Set up pong handler
	conn.SetPongHandler(func(data string) error {
		return c.handlePong(data)
	})

	// Authenticate the connection
	if err = c.authenticateConnection(conn, token); err != nil {
		conn.Close()
		return err
	}

	// Connection successful, update client state
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.connecting = false
	c.hasConnected = true
	c.mu.Unlock()

	// Start ping/pong cycle and message listener
	go c.startPingPong()
	go c.listen()

	return nil
}

// prepareWebSocketURL adds necessary query parameters to the WebSocket URL
func (c *Client) prepareWebSocketURL(wsURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("invalid WebSocket URL: %v", err)
	}

	query := parsedURL.Query()
	query.Set("outboundWireFormat", "text")
	query.Set("bufferStates", "true")
	query.Set("aliasHttpStatus", "true")
	query.Set("clientTimestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
	parsedURL.RawQuery = query.Encode()

	return parsedURL, nil
}

// dialWebSocket establishes a WebSocket connection with proper headers
func (c *Client) dialWebSocket(url string, token string) (*websocket.Conn, error) {
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer " + token}
	headers["TrackingID"] = []string{fmt.Sprintf("go-sdk_%d", time.Now().UnixMilli())}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Only use the client's transport if it exists
	if c.webexClient != nil && c.webexClient.HttpClient != nil &&
		c.webexClient.HttpClient.Transport != nil {
		if transport, ok := c.webexClient.HttpClient.Transport.(*http.Transport); ok {
			dialer.NetDialContext = transport.DialContext
		}
	}

	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}

	return conn, nil
}

// authenticateConnection sends authentication messages and waits for confirmation
func (c *Client) authenticateConnection(conn *websocket.Conn, token string) error {
	// Send authorization message
	authID := fmt.Sprintf("%d", time.Now().UnixMilli())
	trackingID := fmt.Sprintf("go-sdk_%d", time.Now().UnixMilli())

	authMsg := map[string]interface{}{
		"id":   authID,
		"type": "authorization",
		"data": map[string]interface{}{
			"token": token,
		},
		"trackingId": trackingID,
	}

	authMsgJSON, err := json.Marshal(authMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal auth message: %v", err)
	}

	if err = conn.WriteMessage(websocket.TextMessage, authMsgJSON); err != nil {
		return fmt.Errorf("failed to send auth message: %v", err)
	}

	// Wait for buffer state message to confirm authorization
	authChan := make(chan error, 1)
	go c.waitForAuthConfirmation(conn, authChan)

	// Wait for auth to complete with timeout
	select {
	case err := <-authChan:
		return err
	case <-time.After(30 * time.Second):
		return fmt.Errorf("authorization timed out after 30 seconds")
	}
}

// waitForAuthConfirmation waits for authorization confirmation messages
func (c *Client) waitForAuthConfirmation(conn *websocket.Conn, authChan chan<- error) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			authChan <- fmt.Errorf("error reading auth response: %v", err)
			return
		}

		var event map[string]interface{}
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		// Check for buffer state or registration status message
		data, ok := event["data"].(map[string]interface{})
		if ok {
			eventType, ok := data["eventType"].(string)
			if ok && (eventType == "mercury.buffer_state" || eventType == "mercury.registration_status") {
				// Send initial ping immediately, matching JavaScript SDK behavior
				_ = c.sendInitialPing(conn)
				authChan <- nil // Authorization successful
				return
			}
		}

		// Check for error messages
		if eventType, ok := event["type"].(string); ok && eventType == "error" {
			authChan <- fmt.Errorf("authorization failed: %v", event)
			return
		}
	}
}

// sendInitialPing sends the first ping after successful authentication
func (c *Client) sendInitialPing(conn *websocket.Conn) error {
	pingID := fmt.Sprintf("%d", time.Now().UnixMilli())
	pingMsg := map[string]interface{}{
		"id":   pingID,
		"type": "ping",
	}
	pingJSON, err := json.Marshal(pingMsg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, pingJSON)
}

// listen reads messages from the websocket
func (c *Client) listen() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		close(c.done)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// Connection closed or error occurred
			c.handleConnectionError(err)
			return
		}

		// Parse and process the message
		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		// Process event
		c.processEvent(&event)
	}
}

// handleConnectionError logs the connection error and triggers reconnection if needed
func (c *Client) handleConnectionError(err error) {
	c.mu.Lock()
	wasConnected := c.connected
	c.connected = false
	c.mu.Unlock()

	// If we were connected and not deliberately disconnected, attempt to reconnect
	if wasConnected {
		select {
		case <-c.closeCh:
			// Client was deliberately disconnected, don't reconnect
		default:
			// Connection error, try to reconnect
			go c.reconnect()
		}
	}
}

// processEvent processes an incoming Mercury event
func (c *Client) processEvent(event *Event) {
	// Apply header overrides if present
	c.applyHeaderOverrides(event)

	// Extract event metadata
	c.extractEventMetadata(event)

	// Skip internal events
	if event.EventType == "mercury.buffer_state" || event.EventType == "mercury.registration_status" {
		return
	}

	// Create derived events for compatibility (like message.created)
	if event.EventType == "conversation.activity" {
		c.handleConversationActivity(event)
	}

	// Dispatch to all relevant handlers
	c.dispatchEvent(event)
}

// applyHeaderOverrides applies any header overrides to the event
func (c *Client) applyHeaderOverrides(event *Event) {
	if event.Headers == nil {
		return
	}

	if trackingID, ok := event.Headers["trackingId"].(string); ok {
		event.TrackingID = trackingID
	}
	if id, ok := event.Headers["id"].(string); ok {
		event.ID = id
	}
}

// extractEventMetadata extracts metadata from the event data
func (c *Client) extractEventMetadata(event *Event) {
	if event.Data == nil {
		return
	}

	// Extract event type
	if eventType, ok := event.Data["eventType"].(string); ok {
		event.EventType = eventType
	}

	// For conversation activity events, extract additional metadata
	if event.EventType == "conversation.activity" {
		activity, ok := event.Data["activity"].(map[string]interface{})
		if !ok {
			return
		}

		// Extract verb/activity type
		if verb, ok := activity["verb"].(string); ok {
			event.ActivityType = verb
		}

		// Extract actor information
		if actor, ok := activity["actor"].(map[string]interface{}); ok {
			if actorID, ok := actor["id"].(string); ok {
				event.ActorID = actorID
			}
			if orgID, ok := actor["orgId"].(string); ok {
				event.OrgID = orgID
			}
		}

		// Extract resource/object information
		if object, ok := activity["object"].(map[string]interface{}); ok {
			event.Resource = object
			if objectType, ok := object["objectType"].(string); ok {
				event.ResourceType = objectType
			}
		}
	}
}

// handleConversationActivity creates derived events for compatibility
func (c *Client) handleConversationActivity(event *Event) {
	// For message-related activities, create a derived message.created event
	if event.ActivityType == "post" || event.ActivityType == "share" {
		messageEvent := *event
		messageEvent.EventType = "message.created"

		c.mu.Lock()
		handlers, ok := c.eventHandlers["message.created"]
		c.mu.Unlock()

		if ok {
			for _, handler := range handlers {
				go handler(&messageEvent)
			}
		}
	}
}

// dispatchEvent dispatches an event to all relevant handlers
func (c *Client) dispatchEvent(event *Event) {
	c.mu.Lock()
	// Get all relevant handlers
	handlers, hasHandlers := c.eventHandlers[event.EventType]

	var activityHandlers []EventHandler
	var hasActivityHandlers bool
	if event.EventType == "conversation.activity" && event.ActivityType != "" {
		activityHandlers, hasActivityHandlers = c.eventHandlers["activity:"+event.ActivityType]
	}

	wildcardHandlers, hasWildcardHandlers := c.eventHandlers["*"]
	c.mu.Unlock()

	// Call handlers concurrently
	if hasHandlers {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	if hasActivityHandlers {
		for _, handler := range activityHandlers {
			go handler(event)
		}
	}

	if hasWildcardHandlers {
		for _, handler := range wildcardHandlers {
			go handler(event)
		}
	}
}

// startPingPong begins the ping/pong cycle to keep the connection alive
func (c *Client) startPingPong() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.ping(); err != nil {
				// Connection error, reconnect
				c.reconnect()
				return
			}
		case <-c.closeCh:
			// Connection closed by user
			return
		case <-c.done:
			// Connection closed unexpectedly
			return
		}
	}
}

// ping sends a ping message
func (c *Client) ping() error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("websocket connection is nil")
	}

	// Create ping message with timestamp
	pingData := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Set a deadline for the pong
	if err := conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout)); err != nil {
		return err
	}

	// Send ping
	return conn.WriteMessage(websocket.PingMessage, []byte(pingData))
}

// handlePong handles a pong response
func (c *Client) handlePong(data string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	// Calculate time offset if pong contains timestamp
	if data != "" {
		pingTime, err := time.Parse(time.RFC3339, data)
		if err == nil {
			c.timeOffset = time.Now().UnixMilli() - pingTime.UnixMilli()
		}
	}

	// Reset the read deadline
	return c.conn.SetReadDeadline(time.Time{})
}

// reconnect attempts to reconnect to the Mercury service
func (c *Client) reconnect() {
	c.mu.Lock()
	// If we're already trying to reconnect or already disconnected, do nothing
	if !c.connected || c.connecting {
		c.mu.Unlock()
		return
	}

	c.connected = false
	c.connecting = true
	conn := c.conn
	deviceProvider := c.deviceProvider
	customURL := c.customWebSocketURL
	c.conn = nil
	c.mu.Unlock()

	// Close the old connection if it exists
	if conn != nil {
		conn.Close()
	}

	// Try to reconnect
	go func() {
		wsURL := c.getReconnectURL(deviceProvider, customURL)
		if wsURL == "" {
			c.mu.Lock()
			c.connecting = false
			c.mu.Unlock()
			return
		}

		// Try to connect with backoff
		_ = c.connectWithBackoff(wsURL)
	}()
}

// getReconnectURL gets the WebSocket URL for reconnection
func (c *Client) getReconnectURL(deviceProvider DeviceProvider, customURL string) string {
	// Use custom URL if available
	if customURL != "" {
		return customURL
	}

	if deviceProvider != nil {
		// Try to get URL from device provider
		if err := deviceProvider.Register(); err != nil {
			return ""
		}

		wsURL, err := deviceProvider.GetWebSocketURL()
		if err == nil && wsURL != "" {
			return wsURL
		}
	}

	// Default fallback URL
	return "wss://mercury-connection-a.wbx2.com/mercury/device"
}

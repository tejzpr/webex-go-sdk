/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package conversation

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tejzpr/webex-go-sdk/v2/encryption"
	"github.com/tejzpr/webex-go-sdk/v2/mercury"
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// ActivityHandler is a function that handles conversation activities
type ActivityHandler func(activity *Activity)

// MessageType represents the type of message
type MessageType string

const (
	// MessageTypePost represents a post message
	MessageTypePost MessageType = "post"
	// MessageTypeShare represents a share message
	MessageTypeShare MessageType = "share"
	// MessageTypeAcknowledge represents an acknowledge message
	MessageTypeAcknowledge MessageType = "acknowledge"

	// WildcardHandler is used for handling all activity types
	WildcardHandler = "*"
)

// Activity represents a conversation activity
type Activity struct {
	ID           string                 `json:"id,omitempty"`
	ObjectType   string                 `json:"objectType,omitempty"`
	URL          string                 `json:"url,omitempty"`
	Published    string                 `json:"published,omitempty"`
	Verb         string                 `json:"verb,omitempty"`
	Actor        *Actor                 `json:"actor,omitempty"`
	Object       map[string]interface{} `json:"object,omitempty"`
	Target       *Target                `json:"target,omitempty"`
	ClientTempID string                 `json:"clientTempId,omitempty"`

	// Additional fields that might be in the data
	EncryptionKeyURL string `json:"encryptionKeyUrl,omitempty"`

	// Parsed content after decryption (if applicable)
	Content         string      `json:"-"`
	DecryptedObject *Object     `json:"-"`
	MessageType     MessageType `json:"-"`

	// Raw data for debugging
	RawData map[string]interface{} `json:"-"`
}

// Actor represents the person who performed the activity
type Actor struct {
	ID           string `json:"id,omitempty"`
	ObjectType   string `json:"objectType,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	OrgID        string `json:"orgId,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	EntryUUID    string `json:"entryUUID,omitempty"`
	Type         string `json:"type,omitempty"`
}

// Target represents the conversation where the activity occurred
type Target struct {
	ID           string        `json:"id,omitempty"`
	ObjectType   string        `json:"objectType,omitempty"`
	URL          string        `json:"url,omitempty"`
	Published    string        `json:"published,omitempty"`
	Participants *Participants `json:"participants,omitempty"`
	Activities   *Activities   `json:"activities,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	GlobalID     string        `json:"globalId,omitempty"`
}

// Participants represents the participants in a conversation
type Participants struct {
	Items []interface{} `json:"items,omitempty"`
}

// Activities represents the activities in a conversation
type Activities struct {
	Items []interface{} `json:"items,omitempty"`
}

// Object represents the content of a message
type Object struct {
	ObjectType  string `json:"objectType,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Content     string `json:"content,omitempty"`
	ContentType string `json:"contentType,omitempty"`

	// For reference to another activity (used in acknowledge)
	ID        string `json:"id,omitempty"`
	URL       string `json:"url,omitempty"`
	Published string `json:"published,omitempty"`
}

// Config holds the configuration for the Conversation plugin
type Config struct {
	// Configuration options
}

// DefaultConfig returns the default configuration for the Conversation plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the Conversation API client
type Client struct {
	webexClient      *webexsdk.Client
	config           *Config
	mercuryClient    *mercury.Client
	encryptionClient *encryption.Client
	mu               sync.RWMutex
	handlers         map[string][]ActivityHandler
}

// New creates a new Conversation plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	// Create encryption client
	encryptionClient := encryption.New(webexClient, nil)

	return &Client{
		webexClient:      webexClient,
		config:           config,
		encryptionClient: encryptionClient,
		handlers:         make(map[string][]ActivityHandler),
	}
}

// ProcessActivityEvent processes a conversation activity event from Mercury
func (c *Client) ProcessActivityEvent(event *mercury.Event) (*Activity, error) {
	// Check if we have the necessary data
	if event.Data == nil {
		return nil, fmt.Errorf("event data is nil")
	}

	// Extract the activity from the event
	activityData, ok := event.Data["activity"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("activity data is missing or invalid")
	}

	// Store the raw data for debugging
	rawData := event.Data

	// Get encryption key URL if present
	encryptionKeyURL, _ := activityData["encryptionKeyUrl"].(string)

	// Convert the activity data to JSON
	activityJSON, err := json.Marshal(activityData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity data: %v", err)
	}

	// Parse the activity
	var activity Activity
	if err := json.Unmarshal(activityJSON, &activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity: %v", err)
	}

	// Set raw data for debugging
	activity.RawData = rawData
	activity.EncryptionKeyURL = encryptionKeyURL
	activity.MessageType = MessageType(activity.Verb)

	// Process message content if needed
	if isMessageActivity(activity.Verb) {
		if err := c.processMessageContent(&activity); err != nil {
			// Log error but continue with processing
			fmt.Printf("Error processing message content: %v\n", err)
		}
	}

	return &activity, nil
}

// isMessageActivity checks if the verb is a message-related activity
func isMessageActivity(verb string) bool {
	return verb == string(MessageTypePost) || verb == string(MessageTypeShare)
}

// processMessageContent extracts and decrypts message content if possible
func (c *Client) processMessageContent(activity *Activity) error {
	if activity.Object == nil {
		return nil
	}

	objectJSON, err := json.Marshal(activity.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %v", err)
	}

	var obj Object
	if err := json.Unmarshal(objectJSON, &obj); err != nil {
		return fmt.Errorf("failed to unmarshal object: %v", err)
	}

	activity.DecryptedObject = &obj

	// Try to decrypt the displayName if we have encryption client and key URL
	if obj.DisplayName == "" {
		return nil
	}

	activity.Content = obj.DisplayName // Set default content

	// Try to decrypt if possible
	if c.encryptionClient != nil && activity.EncryptionKeyURL != "" {
		decryptedContent, err := c.encryptionClient.DecryptMessageContent(activity.EncryptionKeyURL, obj.DisplayName)
		if err == nil {
			activity.Content = decryptedContent
		}
	}

	return nil
}

// GetMessageContent attempts to extract the message content from an activity.
// If the message is encrypted, it will be decrypted using the KMS encryption key.
func (c *Client) GetMessageContent(activity *Activity) (string, error) {
	if activity == nil {
		return "", fmt.Errorf("activity is nil")
	}

	// If we've already extracted the content
	if activity.Content != "" {
		return activity.Content, nil
	}

	// If we have a decrypted object, use its content
	if activity.DecryptedObject != nil && activity.DecryptedObject.Content != "" {
		return activity.DecryptedObject.Content, nil
	}

	// If we have a displayName in the object, try to decrypt it
	if activity.Object == nil {
		return "", fmt.Errorf("no content found in activity")
	}

	displayName, ok := activity.Object["displayName"].(string)
	if !ok || displayName == "" {
		return "", fmt.Errorf("no displayName found in activity")
	}

	// Try to decrypt if possible
	if c.encryptionClient != nil && activity.EncryptionKeyURL != "" {
		decryptedContent, err := c.encryptionClient.DecryptMessageContent(activity.EncryptionKeyURL, displayName)
		if err == nil {
			return decryptedContent, nil
		}
		// Log error but continue with encrypted content
		fmt.Printf("Error decrypting message content: %v\n", err)
	}

	return displayName, nil
}

// On registers a handler for a specific activity verb
func (c *Client) On(verb string, handler ActivityHandler) {
	if handler == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	handlers, ok := c.handlers[verb]
	if !ok {
		handlers = []ActivityHandler{}
	}

	c.handlers[verb] = append(handlers, handler)
}

// Off removes a handler for a specific activity verb
func (c *Client) Off(verb string, handler ActivityHandler) {
	if handler == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	handlers, ok := c.handlers[verb]
	if !ok {
		return
	}

	handlerPtr := fmt.Sprintf("%p", handler)
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == handlerPtr {
			c.handlers[verb] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// Clean up empty handler slices
	if len(c.handlers[verb]) == 0 {
		delete(c.handlers, verb)
	}
}

// SetMercuryClient sets the Mercury client to use for WebSocket events
func (c *Client) SetMercuryClient(mercuryClient *mercury.Client) {
	c.mu.Lock()
	c.mercuryClient = mercuryClient
	c.mu.Unlock()

	// Register handlers for conversation activity events
	mercuryClient.On("conversation.activity", func(event *mercury.Event) {
		// Process any KMS messages that arrive with the event
		c.processEventKMSMessages(event)

		activity, err := c.ProcessActivityEvent(event)
		if err != nil {
			fmt.Printf("Error processing activity event: %v\n", err)
			return
		}

		c.dispatchActivity(activity)
	})

	// Register handler for encryption KMS message events
	mercuryClient.On("encryption.kms_message", func(event *mercury.Event) {
		c.processEventKMSMessages(event)
	})
}

// SetEncryptionDeviceInfo passes device information to the encryption client
// so it can authenticate with the KMS service. Call this after device registration.
func (c *Client) SetEncryptionDeviceInfo(deviceURL, userID string) {
	if c.encryptionClient != nil {
		c.encryptionClient.SetDeviceInfo(deviceURL, userID)
	}
}

// EncryptionClient returns the encryption client for direct access.
func (c *Client) EncryptionClient() *encryption.Client {
	return c.encryptionClient
}

// processEventKMSMessages extracts and processes KMS messages from a Mercury event.
// KMS messages can arrive in the event data as "encryption.kmsMessages" and contain
// key updates or rotations that need to be decrypted and cached.
func (c *Client) processEventKMSMessages(event *mercury.Event) {
	if event.Data == nil || c.encryptionClient == nil {
		return
	}

	// Check for kmsMessages in the event data
	kmsMessages, ok := event.Data["encryption.kmsMessages"].([]interface{})
	if !ok {
		// Also check nested under "encryption" key
		if encData, ok := event.Data["encryption"].(map[string]interface{}); ok {
			kmsMessages, _ = encData["kmsMessages"].([]interface{})
		}
	}

	if len(kmsMessages) == 0 {
		return
	}

	jweStrings := make([]string, 0, len(kmsMessages))
	for _, msg := range kmsMessages {
		if s, ok := msg.(string); ok {
			jweStrings = append(jweStrings, s)
		}
	}

	if len(jweStrings) > 0 {
		c.encryptionClient.ProcessKMSMessages(jweStrings)
	}
}

// dispatchActivity dispatches an activity to all registered handlers
func (c *Client) dispatchActivity(activity *Activity) {
	c.mu.RLock()
	// Get verb-specific handlers
	handlers, hasHandlers := c.handlers[activity.Verb]
	// Get wildcard handlers
	wildcardHandlers, hasWildcardHandlers := c.handlers[WildcardHandler]
	c.mu.RUnlock()

	// Call verb-specific handlers
	if hasHandlers {
		for _, handler := range handlers {
			go handler(activity)
		}
	}

	// Call wildcard handlers
	if hasWildcardHandlers {
		for _, handler := range wildcardHandlers {
			go handler(activity)
		}
	}
}

// InitializeFromMercuryEvent initializes an activity from a Mercury event
func (c *Client) InitializeFromMercuryEvent(event *mercury.Event) (*Activity, error) {
	return c.ProcessActivityEvent(event)
}

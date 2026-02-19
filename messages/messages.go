/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package messages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v1/conversation"
	"github.com/WebexCommunity/webex-go-sdk/v1/device"
	"github.com/WebexCommunity/webex-go-sdk/v1/mercury"
	"github.com/WebexCommunity/webex-go-sdk/v1/webexsdk"
)

// Message represents a Webex message
type Message struct {
	ID              string       `json:"id,omitempty"`
	RoomID          string       `json:"roomId,omitempty"`
	ParentID        string       `json:"parentId,omitempty"`
	ToPersonID      string       `json:"toPersonId,omitempty"`
	ToPersonEmail   string       `json:"toPersonEmail,omitempty"`
	Text            string       `json:"text,omitempty"`
	Markdown        string       `json:"markdown,omitempty"`
	HTML            string       `json:"html,omitempty"`
	Files           []string     `json:"files,omitempty"`
	PersonID        string       `json:"personId,omitempty"`
	PersonEmail     string       `json:"personEmail,omitempty"`
	Created         *time.Time   `json:"created,omitempty"`
	Updated         *time.Time   `json:"updated,omitempty"`
	MentionedPeople []string     `json:"mentionedPeople,omitempty"`
	MentionedGroups []string     `json:"mentionedGroups,omitempty"`
	Attachments     []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a message attachment, such as an adaptive card
type Attachment struct {
	ContentType string      `json:"contentType,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}

// ListOptions contains the options for listing messages
type ListOptions struct {
	RoomID          string `url:"roomId,omitempty"`
	MentionedPeople string `url:"mentionedPeople,omitempty"`
	Before          string `url:"before,omitempty"`
	BeforeMessage   string `url:"beforeMessage,omitempty"`
	Max             int    `url:"max,omitempty"`
	ThreadID        string `url:"threadId,omitempty"`
}

// MessagesPage represents a paginated list of messages
type MessagesPage struct {
	Items []Message `json:"items"`
	*webexsdk.Page
}

// MessageHandler is a function that handles a message event
type MessageHandler func(message *Message)

// Config holds the configuration for the Messages plugin
type Config struct {
	// Any configuration settings for the messages plugin can go here
	MercuryConfig *mercury.Config
}

// DefaultConfig returns the default configuration for the Messages plugin
func DefaultConfig() *Config {
	return &Config{
		MercuryConfig: mercury.DefaultConfig(),
	}
}

// Client is the messages API client
type Client struct {
	webexClient        *webexsdk.Client
	config             *Config
	mercury            *mercury.Client
	mu                 sync.Mutex
	listeningActive    bool
	conversationClient *conversation.Client
}

// New creates a new Messages plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		webexClient: webexClient,
		config:      config,
	}

	// Create conversation client
	client.conversationClient = conversation.New(webexClient, nil)

	return client
}

// Create posts a new message and/or media content into a room
func (c *Client) Create(message *Message) (*Message, error) {
	if message.RoomID == "" && message.ToPersonID == "" && message.ToPersonEmail == "" {
		return nil, fmt.Errorf("message must contain either roomId, toPersonId, or toPersonEmail")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "messages", nil, message)
	if err != nil {
		return nil, err
	}

	var result Message
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns a single message by ID
func (c *Client) Get(messageID string) (*Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("messageID is required")
	}

	path := fmt.Sprintf("messages/%s", messageID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var message Message
	if err := webexsdk.ParseResponse(resp, &message); err != nil {
		return nil, err
	}

	return &message, nil
}

// List returns a list of messages in a room
func (c *Client) List(options *ListOptions) (*MessagesPage, error) {
	if options == nil || options.RoomID == "" {
		return nil, fmt.Errorf("roomId is required")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("roomId", options.RoomID)

	if options.MentionedPeople != "" {
		params.Set("mentionedPeople", options.MentionedPeople)
	}

	if options.Before != "" {
		params.Set("before", options.Before)
	}

	if options.BeforeMessage != "" {
		params.Set("beforeMessage", options.BeforeMessage)
	}

	if options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	if options.ThreadID != "" {
		params.Set("threadId", options.ThreadID)
	}

	resp, err := c.webexClient.Request(http.MethodGet, "messages", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "messages")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Messages
	messagesPage := &MessagesPage{
		Page:  page,
		Items: make([]Message, len(page.Items)),
	}

	for i, item := range page.Items {
		var message Message
		if err := json.Unmarshal(item, &message); err != nil {
			return nil, err
		}
		messagesPage.Items[i] = message
	}

	return messagesPage, nil
}

// Update updates an existing message
func (c *Client) Update(messageID string, message *Message) (*Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("messageID is required")
	}

	path := fmt.Sprintf("messages/%s", messageID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, message)
	if err != nil {
		return nil, err
	}

	var result Message
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a message
func (c *Client) Delete(messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID is required")
	}

	path := fmt.Sprintf("messages/%s", messageID)
	resp, err := c.webexClient.Request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}

	// For DELETE operations, we just check the status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Listen starts a real-time stream of message events
// The provided handler will be called for each new message event
func (c *Client) Listen(handler MessageHandler) error {
	c.mu.Lock()
	if c.listeningActive {
		c.mu.Unlock()
		return fmt.Errorf("already listening for messages")
	}
	c.listeningActive = true
	c.mu.Unlock()

	// Initialize Mercury client if not already initialized
	if c.mercury == nil {
		c.mercury = mercury.New(c.webexClient, c.config.MercuryConfig)

		// Create a Device client to use as the device provider
		deviceClient := device.New(c.webexClient, nil)
		c.mercury.SetDeviceProvider(deviceClient)
	}

	// Create and initialize the Conversation plugin
	c.conversationClient.SetMercuryClient(c.mercury)

	// Register handlers for different message types
	c.conversationClient.On("post", func(activity *conversation.Activity) {
		// Extract message data and convert to a Message
		message, err := c.activityToMessage(activity)
		if err != nil {
			fmt.Printf("Error converting post activity to message: %v\n", err)
			return
		}

		// Call the handler with the message
		handler(message)
	})

	c.conversationClient.On("share", func(activity *conversation.Activity) {
		// Extract message data and convert to a Message
		message, err := c.activityToMessage(activity)
		if err != nil {
			fmt.Printf("Error converting share activity to message: %v\n", err)
			return
		}

		// Call the handler with the message
		handler(message)
	})

	c.conversationClient.On("acknowledge", func(activity *conversation.Activity) {
		// For acknowledge events, we need to fetch the referenced message
		if activity.Object == nil {
			return
		}

		// Extract message ID from the referenced object
		objectID, ok := activity.Object["id"].(string)
		if !ok || objectID == "" {
			return
		}

		// Fetch the actual message using the Get method
		message, err := c.Get(objectID)
		if err != nil {
			fmt.Printf("Error fetching message %s: %v\n", objectID, err)
			return
		}

		// Call handler with fetched message
		handler(message)
	})

	// Start Mercury connection
	return c.mercury.Connect()
}

// activityToMessage converts a conversation Activity to a Message
func (c *Client) activityToMessage(activity *conversation.Activity) (*Message, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}

	// Extract basic message properties
	message := &Message{
		ID:       activity.ID,
		PersonID: activity.Actor.ID,
		Created:  parseTime(activity.Published),
	}

	// Extract room ID from target
	if activity.Target != nil {
		message.RoomID = activity.Target.ID
	}

	// Extract message content - this will use decrypted content if available
	content, err := c.conversationClient.GetMessageContent(activity)
	if err == nil && content != "" {
		message.Text = content
	} else if activity.Content != "" {
		message.Text = activity.Content
	}

	// Extract additional properties from actor
	if activity.Actor != nil {
		message.PersonEmail = activity.Actor.EmailAddress
	}

	return message, nil
}

// parseTime converts a time string to a *time.Time
func parseTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil
	}

	return &t
}

// StopListening stops the real-time stream of message events
func (c *Client) StopListening() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.listeningActive {
		return nil
	}

	if c.mercury != nil {
		// Disconnect Mercury
		if err := c.mercury.Disconnect(); err != nil {
			return fmt.Errorf("error disconnecting Mercury: %v", err)
		}
	}

	c.listeningActive = false
	return nil
}

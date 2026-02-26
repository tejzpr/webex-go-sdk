/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webex

import (
	"fmt"
	"sync"

	"github.com/WebexCommunity/webex-go-sdk/v2/attachmentactions"
	"github.com/WebexCommunity/webex-go-sdk/v2/calling"
	"github.com/WebexCommunity/webex-go-sdk/v2/contents"
	"github.com/WebexCommunity/webex-go-sdk/v2/conversation"
	"github.com/WebexCommunity/webex-go-sdk/v2/device"
	"github.com/WebexCommunity/webex-go-sdk/v2/events"
	"github.com/WebexCommunity/webex-go-sdk/v2/meetings"
	"github.com/WebexCommunity/webex-go-sdk/v2/memberships"
	"github.com/WebexCommunity/webex-go-sdk/v2/mercury"
	"github.com/WebexCommunity/webex-go-sdk/v2/messages"
	"github.com/WebexCommunity/webex-go-sdk/v2/people"
	"github.com/WebexCommunity/webex-go-sdk/v2/recordings"
	"github.com/WebexCommunity/webex-go-sdk/v2/rooms"
	"github.com/WebexCommunity/webex-go-sdk/v2/roomtabs"
	"github.com/WebexCommunity/webex-go-sdk/v2/teammemberships"
	"github.com/WebexCommunity/webex-go-sdk/v2/teams"
	"github.com/WebexCommunity/webex-go-sdk/v2/transcripts"
	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	"github.com/WebexCommunity/webex-go-sdk/v2/webhooks"
)

// WebexClient is the top-level client for the Webex API
type WebexClient struct {
	// Core client for the Webex API
	core *webexsdk.Client

	// Plugins
	peopleClient            *people.Client
	messagesClient          *messages.Client
	attachmentActionsClient *attachmentactions.Client
	membershipsClient       *memberships.Client
	roomsClient             *rooms.Client
	roomTabsClient          *roomtabs.Client
	teamMembershipsClient   *teammemberships.Client
	teamsClient             *teams.Client
	webhooksClient          *webhooks.Client
	eventsClient            *events.Client
	meetingsClient          *meetings.Client
	recordingsClient        *recordings.Client
	transcriptsClient       *transcripts.Client
	conversationClient      *conversation.Client
	callingClient           *calling.Client
	contentsClient          *contents.Client

	// Internal plugins
	mercuryClient *mercury.Client
	deviceClient  *device.Client

	// Mutex for thread-safe lazy initialization of conversation client
	convMu sync.Mutex
}

// NewClient creates a new Webex client with the given access token and optional configuration
func NewClient(accessToken string, config *webexsdk.Config) (*WebexClient, error) {
	core, err := webexsdk.NewClient(accessToken, config)
	if err != nil {
		return nil, err
	}

	client := &WebexClient{
		core: core,
	}

	return client, nil
}

// People returns the People plugin
func (c *WebexClient) People() *people.Client {
	if c.peopleClient == nil {
		c.peopleClient = people.New(c.core, nil)
	}
	return c.peopleClient
}

// Messages returns the Messages plugin
func (c *WebexClient) Messages() *messages.Client {
	if c.messagesClient == nil {
		c.messagesClient = messages.New(c.core, nil)
	}
	return c.messagesClient
}

// AttachmentActions returns the AttachmentActions plugin
func (c *WebexClient) AttachmentActions() *attachmentactions.Client {
	if c.attachmentActionsClient == nil {
		c.attachmentActionsClient = attachmentactions.New(c.core, nil)
	}
	return c.attachmentActionsClient
}

// Memberships returns the Memberships plugin
func (c *WebexClient) Memberships() *memberships.Client {
	if c.membershipsClient == nil {
		c.membershipsClient = memberships.New(c.core, nil)
	}
	return c.membershipsClient
}

// Rooms returns the Rooms plugin
func (c *WebexClient) Rooms() *rooms.Client {
	if c.roomsClient == nil {
		c.roomsClient = rooms.New(c.core, nil)
	}
	return c.roomsClient
}

// RoomTabs returns the RoomTabs plugin
func (c *WebexClient) RoomTabs() *roomtabs.Client {
	if c.roomTabsClient == nil {
		c.roomTabsClient = roomtabs.New(c.core, nil)
	}
	return c.roomTabsClient
}

// TeamMemberships returns the TeamMemberships plugin
func (c *WebexClient) TeamMemberships() *teammemberships.Client {
	if c.teamMembershipsClient == nil {
		c.teamMembershipsClient = teammemberships.New(c.core, nil)
	}
	return c.teamMembershipsClient
}

// Teams returns the Teams plugin
func (c *WebexClient) Teams() *teams.Client {
	if c.teamsClient == nil {
		c.teamsClient = teams.New(c.core, nil)
	}
	return c.teamsClient
}

// Webhooks returns the Webhooks plugin
func (c *WebexClient) Webhooks() *webhooks.Client {
	if c.webhooksClient == nil {
		c.webhooksClient = webhooks.New(c.core, nil)
	}
	return c.webhooksClient
}

// Contents returns the Contents plugin
func (c *WebexClient) Contents() *contents.Client {
	if c.contentsClient == nil {
		c.contentsClient = contents.New(c.core, nil)
	}
	return c.contentsClient
}

// Events returns the Events plugin
func (c *WebexClient) Events() *events.Client {
	if c.eventsClient == nil {
		c.eventsClient = events.New(c.core, nil)
	}
	return c.eventsClient
}

// Meetings returns the Meetings plugin
func (c *WebexClient) Meetings() *meetings.Client {
	if c.meetingsClient == nil {
		c.meetingsClient = meetings.New(c.core, nil)
	}
	return c.meetingsClient
}

// Recordings returns the Recordings plugin
func (c *WebexClient) Recordings() *recordings.Client {
	if c.recordingsClient == nil {
		c.recordingsClient = recordings.New(c.core, nil)
	}
	return c.recordingsClient
}

// Transcripts returns the Transcripts plugin
func (c *WebexClient) Transcripts() *transcripts.Client {
	if c.transcriptsClient == nil {
		c.transcriptsClient = transcripts.New(c.core, nil)
	}
	return c.transcriptsClient
}

// Calling returns the Calling plugin for Webex Calling APIs
// (Call History, Call Settings, Voicemail, Contacts).
func (c *WebexClient) Calling() *calling.Client {
	if c.callingClient == nil {
		c.callingClient = calling.New(c.core, nil)
	}
	return c.callingClient
}

// Conversation returns a fully-wired Conversation client for real-time
// WebSocket message listening with automatic decryption.
//
// This is a convenience method that abstracts away the manual setup of
// Device registration, Mercury WebSocket wiring, and encryption (KMS)
// authentication. The client is lazily initialized on first call and
// cached for subsequent calls.
//
// Simple usage:
//
//	conv, err := client.Conversation()
//	conv.On("post", handler)
//	conv.Connect()
//	defer conv.Disconnect()
//
// For advanced control over Device, Mercury, or Encryption configuration,
// use the lower-level APIs directly (device.New, mercury.New, conversation.New).
func (c *WebexClient) Conversation() (*conversation.Client, error) {
	c.convMu.Lock()
	defer c.convMu.Unlock()

	if c.conversationClient != nil {
		return c.conversationClient, nil
	}

	// 1. Ensure device is registered (network call)
	deviceClient := c.Device()
	if err := deviceClient.Register(); err != nil {
		return nil, fmt.Errorf("device registration failed: %w", err)
	}

	// 2. Get device info for encryption wiring
	deviceURL, err := deviceClient.GetDeviceURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get device URL: %w", err)
	}
	deviceInfo := deviceClient.GetDevice()

	// 3. Create conversation client
	convClient := conversation.New(c.core, nil)

	// 4. Wire Mercury (WebSocket event routing)
	convClient.SetMercuryClient(c.Mercury())

	// 5. Wire encryption device info (ECDH/KMS authentication)
	convClient.SetEncryptionDeviceInfo(deviceURL, deviceInfo.UserID)

	c.conversationClient = convClient
	return c.conversationClient, nil
}

// Internal returns a struct containing internal plugins
func (c *WebexClient) Internal() *InternalPlugins {
	return &InternalPlugins{
		Mercury: c.Mercury(),
		Device:  c.Device(),
	}
}

// Mercury returns the Mercury plugin (internal)
func (c *WebexClient) Mercury() *mercury.Client {
	if c.mercuryClient == nil {
		c.mercuryClient = mercury.New(c.core, nil)
		// Set the Device plugin as the DeviceProvider for Mercury
		c.mercuryClient.SetDeviceProvider(c.Device())
	}
	return c.mercuryClient
}

// Device returns the Device plugin (internal)
func (c *WebexClient) Device() *device.Client {
	if c.deviceClient == nil {
		c.deviceClient = device.New(c.core, nil)
	}
	return c.deviceClient
}

// InternalPlugins holds internal plugins that aren't part of the public API
type InternalPlugins struct {
	Mercury *mercury.Client
	Device  *device.Client
}

// Core returns the core Webex client
func (c *WebexClient) Core() *webexsdk.Client {
	return c.core
}

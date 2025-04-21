/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package webex

import (
	"github.com/tejzpr/webex-go-sdk/v1/attachmentactions"
	"github.com/tejzpr/webex-go-sdk/v1/device"
	"github.com/tejzpr/webex-go-sdk/v1/events"
	"github.com/tejzpr/webex-go-sdk/v1/memberships"
	"github.com/tejzpr/webex-go-sdk/v1/mercury"
	"github.com/tejzpr/webex-go-sdk/v1/messages"
	"github.com/tejzpr/webex-go-sdk/v1/people"
	"github.com/tejzpr/webex-go-sdk/v1/rooms"
	"github.com/tejzpr/webex-go-sdk/v1/roomtabs"
	"github.com/tejzpr/webex-go-sdk/v1/teammemberships"
	"github.com/tejzpr/webex-go-sdk/v1/teams"
	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
	"github.com/tejzpr/webex-go-sdk/v1/webhooks"
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

	// Internal plugins
	mercuryClient *mercury.Client
	deviceClient  *device.Client
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

// Events returns the Events plugin
func (c *WebexClient) Events() *events.Client {
	if c.eventsClient == nil {
		c.eventsClient = events.New(c.core, nil)
	}
	return c.eventsClient
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

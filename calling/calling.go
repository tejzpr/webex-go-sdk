/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

// Package calling provides a client for the Webex Calling APIs.
// It includes sub-clients for Call History, Call Settings, Voicemail, and Contacts.
package calling

import (
	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Client is the top-level Calling client that aggregates all calling sub-clients.
type Client struct {
	core   *webexsdk.Client
	config *Config

	callHistory  *CallHistoryClient
	callSettings *CallSettingsClient
	voicemail    *VoicemailClient
	contacts     *ContactsClient
}

// New creates a new Calling client.
func New(core *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		core:   core,
		config: config,
	}
}

// CallHistory returns the Call History sub-client for retrieving and managing call history records.
func (c *Client) CallHistory() *CallHistoryClient {
	if c.callHistory == nil {
		c.callHistory = newCallHistoryClient(c.core, c.config)
	}
	return c.callHistory
}

// CallSettings returns the Call Settings sub-client for managing DND, call waiting, call forwarding, and voicemail settings.
func (c *Client) CallSettings() *CallSettingsClient {
	if c.callSettings == nil {
		c.callSettings = newCallSettingsClient(c.core, c.config)
	}
	return c.callSettings
}

// Voicemail returns the Voicemail sub-client for retrieving and managing voicemail messages.
func (c *Client) Voicemail() *VoicemailClient {
	if c.voicemail == nil {
		c.voicemail = newVoicemailClient(c.core, c.config)
	}
	return c.voicemail
}

// Contacts returns the Contacts sub-client for managing contacts and contact groups.
func (c *Client) Contacts() *ContactsClient {
	if c.contacts == nil {
		c.contacts = newContactsClient(c.core, c.config)
	}
	return c.contacts
}

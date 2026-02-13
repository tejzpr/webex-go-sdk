/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Event represents a Webex event
type Event struct {
	ID       string    `json:"id,omitempty"`
	Resource string    `json:"resource,omitempty"`
	Type     string    `json:"type,omitempty"`
	AppID    string    `json:"appId,omitempty"`
	ActorID  string    `json:"actorId,omitempty"`
	OrgID    string    `json:"orgId,omitempty"`
	Created  time.Time `json:"created,omitempty"`
	Data     EventData `json:"data,omitempty"`
}

// EventData represents the data field of an event
type EventData struct {
	ID                   string          `json:"id,omitempty"`
	RoomID               string          `json:"roomId,omitempty"`
	RoomType             string          `json:"roomType,omitempty"`
	OrgID                string          `json:"orgId,omitempty"`
	Text                 string          `json:"text,omitempty"`
	PersonID             string          `json:"personId,omitempty"`
	PersonEmail          string          `json:"personEmail,omitempty"`
	MeetingID            string          `json:"meetingId,omitempty"`
	CreatorID            string          `json:"creatorId,omitempty"`
	Host                 json.RawMessage `json:"host,omitempty"`
	Attendees            json.RawMessage `json:"attendees,omitempty"`
	TranscriptionEnabled string          `json:"transcriptionEnabled,omitempty"`
	RecordingEnabled     string          `json:"recordingEnabled,omitempty"`
	HasPostMeetingsChat  string          `json:"hasPostMeetingsChat,omitempty"`
	// Telephony-related fields
	CorrelationID         string `json:"corelationId,omitempty"`
	CallType              string `json:"callType,omitempty"`
	UserID                string `json:"userId,omitempty"`
	UserType              string `json:"userType,omitempty"`
	CallDirection         string `json:"callDirection,omitempty"`
	IsCallAnswered        string `json:"isCallAnswered,omitempty"`
	CallDurationSeconds   string `json:"callDurationSeconds,omitempty"`
	CallStartTime         string `json:"callStartTime,omitempty"`
	CallAnswerTime        string `json:"callAnswerTime,omitempty"`
	CallTransferTime      string `json:"callTransferTime,omitempty"`
	CallingNumber         string `json:"callingNumber,omitempty"`
	CallingLineID         string `json:"callingLineId,omitempty"`
	CalledNumber          string `json:"calledNumber,omitempty"`
	CalledLineID          string `json:"calledLineId,omitempty"`
	DialedDigits          string `json:"dialedDigits,omitempty"`
	CallRedirectingNumber string `json:"callRedirectingNumber,omitempty"`
	CallRedirectedReason  string `json:"callRedirectedReason,omitempty"`
	Created               string `json:"created,omitempty"`
}

// ListOptions contains the options for listing events
type ListOptions struct {
	Resource string `url:"resource,omitempty"`
	Type     string `url:"type,omitempty"`
	ActorID  string `url:"actorId,omitempty"`
	From     string `url:"from,omitempty"`
	To       string `url:"to,omitempty"`
	Max      int    `url:"max,omitempty"`
}

// EventsPage represents a paginated list of events
type EventsPage struct {
	Items []Event `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Events plugin
type Config struct {
	// Any configuration settings for the events plugin can go here
}

// DefaultConfig returns the default configuration for the Events plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the events API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Events plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of events with optional filters
func (c *Client) List(options *ListOptions) (*EventsPage, error) {
	params := url.Values{}
	if options != nil {
		if options.Resource != "" {
			params.Set("resource", options.Resource)
		}
		if options.Type != "" {
			params.Set("type", options.Type)
		}
		if options.ActorID != "" {
			params.Set("actorId", options.ActorID)
		}
		if options.From != "" {
			params.Set("from", options.From)
		}
		if options.To != "" {
			params.Set("to", options.To)
		}
		if options.Max > 0 {
			params.Set("max", fmt.Sprintf("%d", options.Max))
		}
	}

	resp, err := c.webexClient.Request(http.MethodGet, "events", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "events")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Events
	eventsPage := &EventsPage{
		Page:  page,
		Items: make([]Event, len(page.Items)),
	}

	for i, item := range page.Items {
		var event Event
		if err := json.Unmarshal(item, &event); err != nil {
			return nil, err
		}
		eventsPage.Items[i] = event
	}

	return eventsPage, nil
}

// Get returns details for an event by ID
func (c *Client) Get(eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("eventID is required")
	}

	path := fmt.Sprintf("events/%s", eventID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := webexsdk.ParseResponse(resp, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

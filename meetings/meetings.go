/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package meetings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Meeting represents a Webex meeting
type Meeting struct {
	ID                           string     `json:"id,omitempty"`
	MeetingSeriesID              string     `json:"meetingSeriesId,omitempty"`
	ScheduledMeetingID           string     `json:"scheduledMeetingId,omitempty"`
	Title                        string     `json:"title,omitempty"`
	Agenda                       string     `json:"agenda,omitempty"`
	Password                     string     `json:"password,omitempty"`
	Start                        string     `json:"start,omitempty"`
	End                          string     `json:"end,omitempty"`
	Timezone                     string     `json:"timezone,omitempty"`
	Recurrence                   string     `json:"recurrence,omitempty"`
	EnabledAutoRecordMeeting     bool       `json:"enabledAutoRecordMeeting,omitempty"`
	AllowAnyUserToBeCoHost       bool       `json:"allowAnyUserToBeCoHost,omitempty"`
	EnabledJoinBeforeHost        bool       `json:"enabledJoinBeforeHost,omitempty"`
	EnableConnectAudioBeforeHost bool       `json:"enableConnectAudioBeforeHost,omitempty"`
	JoinBeforeHostMinutes        int        `json:"joinBeforeHostMinutes,omitempty"`
	ExcludePassword              bool       `json:"excludePassword,omitempty"`
	PublicMeeting                bool       `json:"publicMeeting,omitempty"`
	MeetingType                  string     `json:"meetingType,omitempty"`
	State                        string     `json:"state,omitempty"`
	ScheduledType                string     `json:"scheduledType,omitempty"`
	HostUserID                   string     `json:"hostUserId,omitempty"`
	HostDisplayName              string     `json:"hostDisplayName,omitempty"`
	HostEmail                    string     `json:"hostEmail,omitempty"`
	SipAddress                   string     `json:"sipAddress,omitempty"`
	WebLink                      string     `json:"webLink,omitempty"`
	MeetingNumber                string     `json:"meetingNumber,omitempty"`
	PhoneAndVideoSystemPassword  string     `json:"phoneAndVideoSystemPassword,omitempty"`
	SiteURL                      string     `json:"siteUrl,omitempty"`
	EnabledBreakoutSessions      bool       `json:"enabledBreakoutSessions,omitempty"`
	IntegrationTags              []string   `json:"integrationTags,omitempty"`
	HasChat                      bool       `json:"hasChat,omitempty"`
	HasRecording                 bool       `json:"hasRecording,omitempty"`
	HasTranscription             bool       `json:"hasTranscription,omitempty"`
	HasSummary                   bool       `json:"hasSummary,omitempty"`
	HasClosedCaption             bool       `json:"hasClosedCaption,omitempty"`
	HasPolls                     bool       `json:"hasPolls,omitempty"`
	HasQA                        bool       `json:"hasQA,omitempty"`
	HasRegistration              bool       `json:"hasRegistration,omitempty"`
	HasRegistrants               bool       `json:"hasRegistrants,omitempty"`
	Created                      *time.Time `json:"created,omitempty"`
}

// Invitee represents a meeting invitee
type Invitee struct {
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	CoHost      bool   `json:"coHost,omitempty"`
}

// ListOptions contains the options for listing meetings
type ListOptions struct {
	MeetingNumber string `url:"meetingNumber,omitempty"`
	MeetingType   string `url:"meetingType,omitempty"`
	State         string `url:"state,omitempty"`
	ScheduledType string `url:"scheduledType,omitempty"`
	HostEmail     string `url:"hostEmail,omitempty"`
	From          string `url:"from,omitempty"`
	To            string `url:"to,omitempty"`
	Max           int    `url:"max,omitempty"`
}

// MeetingsPage represents a paginated list of meetings
type MeetingsPage struct {
	Items []Meeting `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Meetings plugin
type Config struct {
	// Any configuration settings for the meetings plugin can go here
}

// DefaultConfig returns the default configuration for the Meetings plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the meetings API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Meetings plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of meetings.
// Note: The Webex API requires meetingType to be set when state is used as a filter.
// Without meetingType specified, the API returns meeting series (recurring definitions)
// rather than actual meeting instances. Use meetingType="meeting" with state="ended"
// and a from/to date range to list past meeting instances.
func (c *Client) List(options *ListOptions) (*MeetingsPage, error) {
	params := url.Values{}

	if options != nil {
		// Validate: the Webex API requires meetingType when state is specified
		if options.State != "" && options.MeetingType == "" {
			return nil, fmt.Errorf("meetingType is required when state filter is used (Webex API requirement)")
		}

		if options.MeetingNumber != "" {
			params.Set("meetingNumber", options.MeetingNumber)
		}
		if options.MeetingType != "" {
			params.Set("meetingType", options.MeetingType)
		}
		if options.State != "" {
			params.Set("state", options.State)
		}
		if options.ScheduledType != "" {
			params.Set("scheduledType", options.ScheduledType)
		}
		if options.HostEmail != "" {
			params.Set("hostEmail", options.HostEmail)
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

	resp, err := c.webexClient.Request(http.MethodGet, "meetings", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "meetings")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Meetings
	meetingsPage := &MeetingsPage{
		Page:  page,
		Items: make([]Meeting, len(page.Items)),
	}

	for i, item := range page.Items {
		var meeting Meeting
		if err := json.Unmarshal(item, &meeting); err != nil {
			return nil, err
		}
		meetingsPage.Items[i] = meeting
	}

	return meetingsPage, nil
}

// Create creates a new meeting
func (c *Client) Create(meeting *Meeting) (*Meeting, error) {
	if meeting.Title == "" {
		return nil, fmt.Errorf("meeting title is required")
	}
	if meeting.Start == "" {
		return nil, fmt.Errorf("meeting start time is required")
	}
	if meeting.End == "" {
		return nil, fmt.Errorf("meeting end time is required")
	}

	resp, err := c.webexClient.Request(http.MethodPost, "meetings", nil, meeting)
	if err != nil {
		return nil, err
	}

	var result Meeting
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get returns details for a meeting
func (c *Client) Get(meetingID string) (*Meeting, error) {
	if meetingID == "" {
		return nil, fmt.Errorf("meetingID is required")
	}

	path := fmt.Sprintf("meetings/%s", meetingID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var meeting Meeting
	if err := webexsdk.ParseResponse(resp, &meeting); err != nil {
		return nil, err
	}

	return &meeting, nil
}

// Update updates an existing meeting
func (c *Client) Update(meetingID string, meeting *Meeting) (*Meeting, error) {
	if meetingID == "" {
		return nil, fmt.Errorf("meetingID is required")
	}
	if meeting.Title == "" {
		return nil, fmt.Errorf("meeting title is required")
	}

	path := fmt.Sprintf("meetings/%s", meetingID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, meeting)
	if err != nil {
		return nil, err
	}

	var result Meeting
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Delete deletes a meeting
func (c *Client) Delete(meetingID string) error {
	if meetingID == "" {
		return fmt.Errorf("meetingID is required")
	}

	path := fmt.Sprintf("meetings/%s", meetingID)
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

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

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// Meeting represents a Webex meeting
type Meeting struct {
	ID                           string                      `json:"id,omitempty"`
	MeetingSeriesID              string                      `json:"meetingSeriesId,omitempty"`
	ScheduledMeetingID           string                      `json:"scheduledMeetingId,omitempty"`
	Title                        string                      `json:"title,omitempty"`
	Agenda                       string                      `json:"agenda,omitempty"`
	Password                     string                      `json:"password,omitempty"`
	Start                        string                      `json:"start,omitempty"`
	End                          string                      `json:"end,omitempty"`
	Timezone                     string                      `json:"timezone,omitempty"`
	Recurrence                   string                      `json:"recurrence,omitempty"`
	EnabledAutoRecordMeeting     bool                        `json:"enabledAutoRecordMeeting,omitempty"`
	AllowAnyUserToBeCoHost       bool                        `json:"allowAnyUserToBeCoHost,omitempty"`
	EnabledJoinBeforeHost        bool                        `json:"enabledJoinBeforeHost,omitempty"`
	EnableConnectAudioBeforeHost bool                        `json:"enableConnectAudioBeforeHost,omitempty"`
	JoinBeforeHostMinutes        int                         `json:"joinBeforeHostMinutes,omitempty"`
	ExcludePassword              bool                        `json:"excludePassword,omitempty"`
	PublicMeeting                bool                        `json:"publicMeeting,omitempty"`
	MeetingType                  string                      `json:"meetingType,omitempty"`
	State                        string                      `json:"state,omitempty"`
	ScheduledType                string                      `json:"scheduledType,omitempty"`
	HostUserID                   string                      `json:"hostUserId,omitempty"`
	HostDisplayName              string                      `json:"hostDisplayName,omitempty"`
	HostEmail                    string                      `json:"hostEmail,omitempty"`
	SipAddress                   string                      `json:"sipAddress,omitempty"`
	WebLink                      string                      `json:"webLink,omitempty"`
	MeetingNumber                string                      `json:"meetingNumber,omitempty"`
	PhoneAndVideoSystemPassword  string                      `json:"phoneAndVideoSystemPassword,omitempty"`
	SiteURL                      string                      `json:"siteUrl,omitempty"`
	EnabledBreakoutSessions      bool                        `json:"enabledBreakoutSessions,omitempty"`
	Invitees                     []Invitee                   `json:"invitees,omitempty"`
	IntegrationTags              []string                    `json:"integrationTags,omitempty"`
	Telephony                    *Telephony                  `json:"telephony,omitempty"`
	Registration                 *Registration               `json:"registration,omitempty"`
	SimultaneousInterpretation   *SimultaneousInterpretation `json:"simultaneousInterpretation,omitempty"`
	BreakoutSessions             []BreakoutSession           `json:"breakoutSessions,omitempty"`
	AudioConnectionOptions       *AudioConnectionOptions     `json:"audioConnectionOptions,omitempty"`
	HasChat                      bool                        `json:"hasChat,omitempty"`
	HasRecording                 bool                        `json:"hasRecording,omitempty"`
	HasTranscription             bool                        `json:"hasTranscription,omitempty"`
	HasSummary                   bool                        `json:"hasSummary,omitempty"`
	HasClosedCaption             bool                        `json:"hasClosedCaption,omitempty"`
	HasPolls                     bool                        `json:"hasPolls,omitempty"`
	HasQA                        bool                        `json:"hasQA,omitempty"`
	HasRegistration              bool                        `json:"hasRegistration,omitempty"`
	HasRegistrants               bool                        `json:"hasRegistrants,omitempty"`
	Created                      *time.Time                  `json:"created,omitempty"`
}

// Telephony contains telephony dial-in information for a meeting
type Telephony struct {
	AccessCode    string          `json:"accessCode,omitempty"`
	CallInNumbers []CallInNumber  `json:"callInNumbers,omitempty"`
	Links         *TelephonyLinks `json:"links,omitempty"`
}

// CallInNumber represents a phone number for dialing into a meeting
type CallInNumber struct {
	Label        string `json:"label,omitempty"`
	CallInNumber string `json:"callInNumber,omitempty"`
	TollType     string `json:"tollType,omitempty"`
}

// TelephonyLink contains global call-in URLs
type TelephonyLink struct {
	GlobalCallinNumbers string `json:"globalCallinNumbers,omitempty"`
	TelephonyTopic      string `json:"telephonyTopic,omitempty"`
}

// TelephonyLinks represents the telephony links structure which can be either a single object or an array
type TelephonyLinks []TelephonyLink

// Registration contains meeting registration settings
type Registration struct {
	AutoAcceptRequest  bool             `json:"autoAcceptRequest,omitempty"`
	RequireFirstName   bool             `json:"requireFirstName,omitempty"`
	RequireLastName    bool             `json:"requireLastName,omitempty"`
	RequireEmail       bool             `json:"requireEmail,omitempty"`
	RequireJobTitle    bool             `json:"requireJobTitle,omitempty"`
	RequireCompanyName bool             `json:"requireCompanyName,omitempty"`
	RequireAddress1    bool             `json:"requireAddress1,omitempty"`
	RequireAddress2    bool             `json:"requireAddress2,omitempty"`
	RequireCity        bool             `json:"requireCity,omitempty"`
	RequireState       bool             `json:"requireState,omitempty"`
	RequireZipCode     bool             `json:"requireZipCode,omitempty"`
	RequireCountry     bool             `json:"requireCountryRegion,omitempty"`
	RequirePhone       bool             `json:"requireWorkPhone,omitempty"`
	RequireFax         bool             `json:"requireFax,omitempty"`
	MaxRegisterNum     int              `json:"maxRegisterNum,omitempty"`
	CustomQuestions    []CustomQuestion `json:"customizedQuestions,omitempty"`
}

// CustomQuestion represents a custom registration question
type CustomQuestion struct {
	ID       int      `json:"id,omitempty"`
	Question string   `json:"question,omitempty"`
	Type     string   `json:"type,omitempty"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"`
}

// SimultaneousInterpretation contains interpretation settings
type SimultaneousInterpretation struct {
	Enabled      bool          `json:"enabled,omitempty"`
	Interpreters []Interpreter `json:"interpreters,omitempty"`
}

// Interpreter represents a meeting interpreter
type Interpreter struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	LanguageCode1 string `json:"languageCode1,omitempty"`
	LanguageCode2 string `json:"languageCode2,omitempty"`
}

// BreakoutSession represents a meeting breakout session
type BreakoutSession struct {
	Name  string   `json:"name,omitempty"`
	Users []string `json:"users,omitempty"`
}

// AudioConnectionOptions contains audio connection settings
type AudioConnectionOptions struct {
	AudioConnectionType           string `json:"audioConnectionType,omitempty"`
	EnabledTollFreeCallIn         bool   `json:"enabledTollFreeCallIn,omitempty"`
	EnabledGlobalCallIn           bool   `json:"enabledGlobalCallIn,omitempty"`
	EnabledAudienceCallBack       bool   `json:"enabledAudienceCallBack,omitempty"`
	EntryAndExitTone              string `json:"entryAndExitTone,omitempty"`
	AllowHostToUnmuteParticipants bool   `json:"allowHostToUnmuteParticipants,omitempty"`
	AllowAttendeeToUnmuteSelf     bool   `json:"allowAttendeeToUnmuteSelf,omitempty"`
	MuteAttendeeUponEntry         bool   `json:"muteAttendeeUponEntry,omitempty"`
}

// Invitee represents a meeting invitee
type Invitee struct {
	ID          string `json:"id,omitempty"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	CoHost      bool   `json:"coHost,omitempty"`
	MeetingID   string `json:"meetingId,omitempty"`
	Panelist    bool   `json:"panelist,omitempty"`
}

// Participant represents a meeting participant
type Participant struct {
	ID             string              `json:"id,omitempty"`
	OrgID          string              `json:"orgId,omitempty"`
	Host           bool                `json:"host,omitempty"`
	CoHost         bool                `json:"coHost,omitempty"`
	SpaceModerator bool                `json:"spaceModerator,omitempty"`
	Email          string              `json:"email,omitempty"`
	DisplayName    string              `json:"displayName,omitempty"`
	Invitee        bool                `json:"invitee,omitempty"`
	Muted          bool                `json:"muted,omitempty"`
	State          string              `json:"state,omitempty"`
	JoinedTime     string              `json:"joinedTime,omitempty"`
	LeftTime       string              `json:"leftTime,omitempty"`
	MeetingID      string              `json:"meetingId,omitempty"`
	HostEmail      string              `json:"hostEmail,omitempty"`
	Devices        []ParticipantDevice `json:"devices,omitempty"`
}

// ParticipantDevice represents a device used by a meeting participant
type ParticipantDevice struct {
	DeviceType   string `json:"deviceType,omitempty"`
	JoinedTime   string `json:"joinedTime,omitempty"`
	LeftTime     string `json:"leftTime,omitempty"`
	CallType     string `json:"callType,omitempty"`
	CallInNumber string `json:"callInNumber,omitempty"`
	AudioType    string `json:"audioType,omitempty"`
}

// ParticipantsPage represents a paginated list of meeting participants
type ParticipantsPage struct {
	Items []Participant `json:"items"`
	*webexsdk.Page
}

// ParticipantListOptions contains the options for listing meeting participants
type ParticipantListOptions struct {
	MeetingID string `url:"meetingId,omitempty"`
	HostEmail string `url:"hostEmail,omitempty"`
	Max       int    `url:"max,omitempty"`
}

// ListOptions contains the options for listing meetings
type ListOptions struct {
	MeetingNumber  string `url:"meetingNumber,omitempty"`
	MeetingType    string `url:"meetingType,omitempty"`
	State          string `url:"state,omitempty"`
	ScheduledType  string `url:"scheduledType,omitempty"`
	HostEmail      string `url:"hostEmail,omitempty"`
	SiteURL        string `url:"siteUrl,omitempty"`
	IntegrationTag string `url:"integrationTag,omitempty"`
	From           string `url:"from,omitempty"`
	To             string `url:"to,omitempty"`
	Max            int    `url:"max,omitempty"`
	Current        bool   `url:"current,omitempty"`
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
		if options.SiteURL != "" {
			params.Set("siteUrl", options.SiteURL)
		}
		if options.IntegrationTag != "" {
			params.Set("integrationTag", options.IntegrationTag)
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
		if options.Current {
			params.Set("current", "true")
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
	defer resp.Body.Close()

	// For DELETE operations, we just check the status code
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Patch partially updates a meeting (only the provided fields).
// Use this when you want to update specific fields without resending the entire meeting object.
func (c *Client) Patch(meetingID string, patch interface{}) (*Meeting, error) {
	if meetingID == "" {
		return nil, fmt.Errorf("meetingID is required")
	}

	path := fmt.Sprintf("meetings/%s", meetingID)
	resp, err := c.webexClient.Request(http.MethodPatch, path, nil, patch)
	if err != nil {
		return nil, err
	}

	var result Meeting
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListParticipants returns a list of participants for a meeting instance.
// Requires a meetingId that is a meeting instance ID (ended meetings).
func (c *Client) ListParticipants(options *ParticipantListOptions) (*ParticipantsPage, error) {
	if options == nil || options.MeetingID == "" {
		return nil, fmt.Errorf("meetingId is required")
	}

	params := url.Values{}
	params.Set("meetingId", options.MeetingID)

	if options.HostEmail != "" {
		params.Set("hostEmail", options.HostEmail)
	}
	if options.Max > 0 {
		params.Set("max", fmt.Sprintf("%d", options.Max))
	}

	resp, err := c.webexClient.Request(http.MethodGet, "meetingParticipants", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "meetingParticipants")
	if err != nil {
		return nil, err
	}

	participantsPage := &ParticipantsPage{
		Page:  page,
		Items: make([]Participant, len(page.Items)),
	}

	for i, item := range page.Items {
		var participant Participant
		if err := json.Unmarshal(item, &participant); err != nil {
			return nil, err
		}
		participantsPage.Items[i] = participant
	}

	return participantsPage, nil
}

// GetParticipant returns details for a specific meeting participant.
func (c *Client) GetParticipant(participantID string, meetingID string) (*Participant, error) {
	if participantID == "" {
		return nil, fmt.Errorf("participantID is required")
	}
	if meetingID == "" {
		return nil, fmt.Errorf("meetingID is required")
	}

	params := url.Values{}
	params.Set("meetingId", meetingID)

	path := fmt.Sprintf("meetingParticipants/%s", participantID)
	resp, err := c.webexClient.Request(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	var participant Participant
	if err := webexsdk.ParseResponse(resp, &participant); err != nil {
		return nil, err
	}

	return &participant, nil
}

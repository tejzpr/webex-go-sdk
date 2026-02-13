/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import "time"

// ---- Enums / Constants ----

// Sort order for list queries
type Sort string

const (
	SortASC  Sort = "ASC"
	SortDESC Sort = "DESC"
)

// SortBy field for call history queries
type SortBy string

const (
	SortByEndTime   SortBy = "endTime"
	SortByStartTime SortBy = "startTime"
)

// CallDirection indicates whether a call is inbound or outbound
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound"
)

// CallType indicates the type of call address
type CallType string

const (
	CallTypeURI CallType = "uri"
	CallTypeTEL CallType = "tel"
)

// Disposition indicates the outcome of a call session
type Disposition string

const (
	DispositionAnswered  Disposition = "Answered"
	DispositionCanceled  Disposition = "Canceled"
	DispositionInitiated Disposition = "Initiated"
	DispositionMissed    Disposition = "MISSED"
)

// SessionType indicates the type of call session
type SessionType string

const (
	SessionTypeSpark        SessionType = "SPARK"
	SessionTypeWebexCalling SessionType = "WEBEXCALLING"
)

// ContactType indicates whether a contact is custom or cloud-synced
type ContactType string

const (
	ContactTypeCustom ContactType = "CUSTOM"
	ContactTypeCloud  ContactType = "CLOUD"
)

// GroupType indicates the type of contact group
type GroupType string

const (
	GroupTypeNormal   GroupType = "NORMAL"
	GroupTypeExternal GroupType = "EXTERNAL"
)

// ---- Call History Types ----

// CallRecordSelf represents the user's own info in a call record
type CallRecordSelf struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	PhoneNumber   string `json:"phoneNumber,omitempty"`
	CucmDN        string `json:"cucmDN,omitempty"`
	UcmLineNumber int    `json:"ucmLineNumber,omitempty"`
}

// CallRecordOther represents the other party in a call record
type CallRecordOther struct {
	OwnerID                string `json:"ownerId,omitempty"`
	ID                     string `json:"id"`
	Name                   string `json:"name,omitempty"`
	SipURL                 string `json:"sipUrl,omitempty"`
	PrimaryDisplayString   string `json:"primaryDisplayString,omitempty"`
	SecondaryDisplayString string `json:"secondaryDisplayString,omitempty"`
	IsPrivate              bool   `json:"isPrivate"`
	CallbackAddress        string `json:"callbackAddress"`
	PhoneNumber            string `json:"phoneNumber,omitempty"`
	Contact                string `json:"contact,omitempty"`
	Email                  string `json:"email,omitempty"`
}

// CallRecordLink contains URLs associated with a call record
type CallRecordLink struct {
	LocusURL        string `json:"locusUrl,omitempty"`
	ConversationURL string `json:"conversationUrl,omitempty"`
	CallbackAddress string `json:"callbackAddress"`
}

// RedirectionDetails contains call redirection info
type RedirectionDetails struct {
	PhoneNumber string `json:"phoneNumber,omitempty"`
	SipURL      string `json:"sipUrl,omitempty"`
	Name        string `json:"name,omitempty"`
	Reason      string `json:"reason"`
	UserID      string `json:"userId,omitempty"`
	IsPrivate   bool   `json:"isPrivate"`
}

// CallingSpecifics contains calling-specific details for a session
type CallingSpecifics struct {
	RedirectionDetails RedirectionDetails `json:"redirectionDetails"`
}

// UserSession represents a single call history record
type UserSession struct {
	ID                    string            `json:"id"`
	SessionID             string            `json:"sessionId"`
	Disposition           Disposition       `json:"disposition"`
	StartTime             string            `json:"startTime"`
	EndTime               string            `json:"endTime"`
	URL                   string            `json:"url"`
	DurationSeconds       int               `json:"durationSeconds"`
	JoinedDurationSeconds int               `json:"joinedDurationSeconds"`
	ParticipantCount      int               `json:"participantCount"`
	IsDeleted             bool              `json:"isDeleted"`
	IsPMR                 bool              `json:"isPMR"`
	CorrelationIDs        []string          `json:"correlationIds"`
	Links                 CallRecordLink    `json:"links"`
	Self                  CallRecordSelf    `json:"self"`
	DurationSecs          int               `json:"durationSecs"`
	Other                 CallRecordOther   `json:"other"`
	SessionType           SessionType       `json:"sessionType"`
	Direction             string            `json:"direction"`
	CallingSpecifics      *CallingSpecifics `json:"callingSpecifics,omitempty"`
}

// EndTimeSessionID identifies a call history record by end time and session ID
type EndTimeSessionID struct {
	EndTime   string `json:"endTime"`
	SessionID string `json:"sessionId"`
}

// CallHistoryResponse is the response from call history APIs
type CallHistoryResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		UserSessions []UserSession `json:"userSessions,omitempty"`
		Error        string        `json:"error,omitempty"`
	} `json:"data"`
}

// UpdateMissedCallsResponse is the response from updating missed calls
type UpdateMissedCallsResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		ReadStatusMessage string `json:"readStatusMessage,omitempty"`
		Error             string `json:"error,omitempty"`
	} `json:"data"`
}

// DeleteCallHistoryResponse is the response from deleting call history records
type DeleteCallHistoryResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		DeleteStatusMessage string `json:"deleteStatusMessage,omitempty"`
		Error               string `json:"error,omitempty"`
	} `json:"data"`
}

// ---- Call Settings Types ----

// ToggleSetting represents a simple on/off setting with optional ring splash
type ToggleSetting struct {
	Enabled          bool `json:"enabled"`
	RingSplashEnabled *bool `json:"ringSplashEnabled,omitempty"`
}

// CallForwardAlwaysSetting configures always-on call forwarding
type CallForwardAlwaysSetting struct {
	Enabled                    bool   `json:"enabled"`
	RingReminderEnabled        *bool  `json:"ringReminderEnabled,omitempty"`
	DestinationVoicemailEnabled *bool  `json:"destinationVoicemailEnabled,omitempty"`
	Destination                string `json:"destination,omitempty"`
}

// CallForwardBusySetting configures call forwarding when the line is busy
type CallForwardBusySetting struct {
	Enabled                    bool   `json:"enabled"`
	DestinationVoicemailEnabled *bool  `json:"destinationVoicemailEnabled,omitempty"`
	Destination                string `json:"destination,omitempty"`
}

// CallForwardNoAnswerSetting configures call forwarding when unanswered
type CallForwardNoAnswerSetting struct {
	Enabled                    bool   `json:"enabled"`
	NumberOfRings              *int   `json:"numberOfRings,omitempty"`
	SystemMaxNumberOfRings     *int   `json:"systemMaxNumberOfRings,omitempty"`
	DestinationVoicemailEnabled *bool  `json:"destinationVoicemailEnabled,omitempty"`
	Destination                string `json:"destination,omitempty"`
}

// CallForwardingConfig groups all call forwarding rules
type CallForwardingConfig struct {
	Always   CallForwardAlwaysSetting   `json:"always"`
	Busy     CallForwardBusySetting     `json:"busy"`
	NoAnswer CallForwardNoAnswerSetting `json:"noAnswer"`
}

// BusinessContinuitySetting configures forwarding when the line is offline
type BusinessContinuitySetting struct {
	Enabled                    bool   `json:"enabled"`
	DestinationVoicemailEnabled *bool  `json:"destinationVoicemailEnabled,omitempty"`
	Destination                string `json:"destination,omitempty"`
}

// CallForwardSetting is the full call forward configuration
type CallForwardSetting struct {
	CallForwarding    CallForwardingConfig      `json:"callForwarding"`
	BusinessContinuity BusinessContinuitySetting `json:"businessContinuity"`
}

// VoicemailSettingConfig is the full voicemail configuration
type VoicemailSettingConfig struct {
	Enabled            bool `json:"enabled"`
	SendAllCalls       *struct {
		Enabled bool `json:"enabled"`
	} `json:"sendAllCalls,omitempty"`
	SendBusyCalls struct {
		Enabled          bool   `json:"enabled"`
		Greeting         string `json:"greeting,omitempty"`
		GreetingUploaded *bool  `json:"greetingUploaded,omitempty"`
	} `json:"sendBusyCalls"`
	SendUnansweredCalls struct {
		Enabled                bool   `json:"enabled"`
		Greeting               string `json:"greeting,omitempty"`
		GreetingUploaded       *bool  `json:"greetingUploaded,omitempty"`
		NumberOfRings          int    `json:"numberOfRings"`
		SystemMaxNumberOfRings *int   `json:"systemMaxNumberOfRings,omitempty"`
	} `json:"sendUnansweredCalls"`
	Notifications struct {
		Enabled     bool   `json:"enabled"`
		Destination string `json:"destination,omitempty"`
	} `json:"notifications"`
	TransferToNumber *struct {
		Enabled     bool   `json:"enabled"`
		Destination string `json:"destination"`
	} `json:"transferToNumber,omitempty"`
	EmailCopyOfMessage struct {
		Enabled bool   `json:"enabled"`
		EmailID string `json:"emailId,omitempty"`
	} `json:"emailCopyOfMessage"`
	MessageStorage struct {
		MWIEnabled    bool   `json:"mwiEnabled"`
		StorageType   string `json:"storageType"`
		ExternalEmail string `json:"externalEmail,omitempty"`
	} `json:"messageStorage"`
	FaxMessage *struct {
		Enabled     bool   `json:"enabled"`
		PhoneNumber string `json:"phoneNumber,omitempty"`
		Extension   string `json:"extension,omitempty"`
	} `json:"faxMessage,omitempty"`
	VoiceMessageForwardingEnabled *bool `json:"voiceMessageForwardingEnabled,omitempty"`
}

// CallSettingResponse is the generic response from call settings APIs
type CallSettingResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		CallSetting interface{} `json:"callSetting,omitempty"`
		Error       string      `json:"error,omitempty"`
	} `json:"data"`
}

// ---- Voicemail Types ----

// VoicemailCallingPartyInfo represents the caller info for a voicemail
type VoicemailCallingPartyInfo struct {
	Name           string `json:"name"`
	UserID         string `json:"userId,omitempty"`
	Address        string `json:"address"`
	UserExternalID string `json:"userExternalId,omitempty"`
}

// VoicemailSummary provides a count of voicemail messages
type VoicemailSummary struct {
	NewMessages       int `json:"newMessages"`
	OldMessages       int `json:"oldMessages"`
	NewUrgentMessages int `json:"newUrgentMessages"`
	OldUrgentMessages int `json:"oldUrgentMessages"`
}

// VoicemailMessage represents a single voicemail message
type VoicemailMessage struct {
	MessageID        string                    `json:"messageId"`
	Duration         string                    `json:"duration"`
	CallingPartyInfo VoicemailCallingPartyInfo  `json:"callingPartyInfo"`
	Time             int64                     `json:"time"`
	Read             bool                      `json:"read"`
}

// VoicemailContent represents the content of a voicemail
type VoicemailContent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// VoicemailResponse is the response from voicemail APIs
type VoicemailResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		VoicemailList       []VoicemailMessage `json:"voicemailList,omitempty"`
		VoicemailContent    *VoicemailContent  `json:"voicemailContent,omitempty"`
		VoicemailSummary    *VoicemailSummary  `json:"voicemailSummary,omitempty"`
		VoicemailTranscript *string            `json:"voicemailTranscript,omitempty"`
		Error               string             `json:"error,omitempty"`
	} `json:"data"`
}

// ---- Contact Types ----

// PhoneNumber represents a phone number with type and primary flag
type PhoneNumber struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary *bool  `json:"primary,omitempty"`
}

// URIAddress represents a URI-based address (email, SIP, etc.)
type URIAddress struct {
	Value   string `json:"value"`
	Type    string `json:"type"`
	Primary *bool  `json:"primary,omitempty"`
}

// Address represents a physical address
type Address struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	State   string `json:"state,omitempty"`
	Street  string `json:"street,omitempty"`
	ZipCode string `json:"zipCode,omitempty"`
}

// Contact represents a single contact
type Contact struct {
	AddressInfo          *Address     `json:"addressInfo,omitempty"`
	AvatarURL            string       `json:"avatarURL,omitempty"`
	AvatarURLDomain      string       `json:"avatarUrlDomain,omitempty"`
	CompanyName          string       `json:"companyName,omitempty"`
	ContactID            string       `json:"contactId"`
	ContactType          ContactType  `json:"contactType"`
	Department           string       `json:"department,omitempty"`
	DisplayName          string       `json:"displayName,omitempty"`
	Emails               []URIAddress `json:"emails,omitempty"`
	EncryptionKeyURL     string       `json:"encryptionKeyUrl"`
	FirstName            string       `json:"firstName,omitempty"`
	Groups               []string     `json:"groups"`
	KmsResourceObjectURL string       `json:"kmsResourceObjectUrl,omitempty"`
	LastName             string       `json:"lastName,omitempty"`
	Manager              string       `json:"manager,omitempty"`
	OwnerID              string       `json:"ownerId,omitempty"`
	PhoneNumbers         []PhoneNumber `json:"phoneNumbers,omitempty"`
	PrimaryContactMethod string       `json:"primaryContactMethod,omitempty"`
	Schemas              string       `json:"schemas,omitempty"`
	SipAddresses         []URIAddress `json:"sipAddresses,omitempty"`
	Resolved             bool         `json:"resolved"`
}

// ContactGroup represents a contact group
type ContactGroup struct {
	DisplayName      string    `json:"displayName"`
	EncryptionKeyURL string    `json:"encryptionKeyUrl"`
	GroupID          string    `json:"groupId"`
	GroupType        GroupType `json:"groupType"`
	Members          []string  `json:"members,omitempty"`
	OwnerID          string    `json:"ownerId,omitempty"`
}

// ContactResponse is the response from contacts APIs
type ContactResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
	Data       struct {
		Contacts []Contact      `json:"contacts,omitempty"`
		Groups   []ContactGroup `json:"groups,omitempty"`
		Contact  *Contact       `json:"contact,omitempty"`
		Group    *ContactGroup  `json:"group,omitempty"`
		Error    string         `json:"error,omitempty"`
	} `json:"data"`
}

// ---- Person / Display Types ----

// DisplayInformation represents resolved caller/contact display info
type DisplayInformation struct {
	AvatarSrc string `json:"avatarSrc,omitempty"`
	Name      string `json:"name,omitempty"`
	Num       string `json:"num,omitempty"`
	ID        string `json:"id,omitempty"`
}

// PersonInfo represents a Webex person
type PersonInfo struct {
	ID           string        `json:"id"`
	Emails       []string      `json:"emails"`
	PhoneNumbers []PhoneNumber `json:"phoneNumbers"`
	DisplayName  string        `json:"displayName"`
	NickName     string        `json:"nickName"`
	FirstName    string        `json:"firstName"`
	LastName     string        `json:"lastName"`
	Avatar       string        `json:"avatar"`
	OrgID        string        `json:"orgId"`
	Created      string        `json:"created"`
	LastModified string        `json:"lastModified"`
	LastActivity string        `json:"lastActivity"`
	Status       string        `json:"status"`
	Type         string        `json:"type"`
}

// ---- Config Types ----

// Config holds configuration for the Calling client
type Config struct {
	// BaseURL overrides the default Webex API base URL (default: https://webexapis.com/v1)
	BaseURL string

	// RequestTimeout overrides the default HTTP request timeout
	RequestTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		BaseURL:        "https://webexapis.com/v1",
		RequestTimeout: 30 * time.Second,
	}
}

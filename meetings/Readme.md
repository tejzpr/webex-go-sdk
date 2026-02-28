# Meetings

The Meetings module provides functionality for interacting with the Webex Meetings API. This module allows you to manage Webex Meetings — including creating, retrieving, updating, listing, and deleting meetings — as well as listing meeting participants and accessing telephony/audio configuration.

## Overview

Webex Meetings are virtual conferences where users can collaborate in real time using audio, video, content sharing, chat, online whiteboards, and more. This module allows you to:

1. Create new meetings (with invitees, telephony settings, breakout sessions, etc.)
2. Retrieve meeting details (including telephony dial-in info and audio options)
3. List meetings with filtering options
4. Update or patch existing meetings
5. Delete/cancel meetings
6. List meeting participants and get participant details

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the meetings module:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/meetings"
)
```

## Usage

### Initializing the Client

```go
// Create a new Webex client with your access token
client, err := webex.NewClient("your-access-token", nil)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}

// Access the Meetings API
meetingsClient := client.Meetings()
```

### Creating a Meeting

To schedule a new Webex meeting:

```go
newMeeting := &meetings.Meeting{
    Title:    "Weekly Standup",
    Start:    "2026-02-01T10:00:00Z",
    End:      "2026-02-01T10:30:00Z",
    Timezone: "America/New_York",
    Agenda:   "Weekly team sync",
}

createdMeeting, err := client.Meetings().Create(newMeeting)
if err != nil {
    log.Printf("Failed to create meeting: %v", err)
} else {
    fmt.Printf("Created meeting with ID: %s\n", createdMeeting.ID)
    fmt.Printf("Meeting Link: %s\n", createdMeeting.WebLink)
    fmt.Printf("Meeting Number: %s\n", createdMeeting.MeetingNumber)
}
```

The `Title`, `Start`, and `End` fields are required for creating a meeting.

#### Creating a Meeting with Invitees

```go
newMeeting := &meetings.Meeting{
    Title: "Project Kickoff",
    Start: "2026-02-10T14:00:00Z",
    End:   "2026-02-10T15:00:00Z",
    Invitees: []meetings.Invitee{
        {Email: "alice@example.com", CoHost: true},
        {Email: "bob@example.com"},
    },
}

createdMeeting, err := meetingsClient.Create(newMeeting)
```

### Getting a Meeting

Retrieve details of a specific meeting, including telephony and audio settings:

```go
meeting, err := meetingsClient.Get("meeting-id")
if err != nil {
    log.Printf("Failed to get meeting details: %v", err)
} else {
    fmt.Printf("Meeting Title: %s\n", meeting.Title)
    fmt.Printf("Start: %s\n", meeting.Start)
    fmt.Printf("State: %s\n", meeting.State)

    // Access telephony dial-in info
    if meeting.Telephony != nil {
        fmt.Printf("Access Code: %s\n", meeting.Telephony.AccessCode)
        for _, num := range meeting.Telephony.CallInNumbers {
            fmt.Printf("  %s: %s (%s)\n", num.Label, num.CallInNumber, num.TollType)
        }
    }

    // Access audio connection options
    if meeting.AudioConnectionOptions != nil {
        fmt.Printf("Audio Type: %s\n", meeting.AudioConnectionOptions.AudioConnectionType)
        fmt.Printf("Mute on Entry: %v\n", meeting.AudioConnectionOptions.MuteAttendeeUponEntry)
    }
}
```

### Listing Meetings

List meetings with various filter options:

```go
// List meeting series (recurring definitions)
meetingsPage, err := meetingsClient.List(&meetings.ListOptions{
    MeetingType: "meetingSeries",
    Max:         50,
})
if err != nil {
    log.Printf("Failed to list meetings: %v", err)
} else {
    for i, m := range meetingsPage.Items {
        fmt.Printf("%d. %s (State: %s)\n", i+1, m.Title, m.State)
    }
}
```

#### Listing Past Meeting Instances

To list actual meeting instances that have ended, you **must** specify both `MeetingType` and `State`. The Webex API requires `meetingType` whenever `state` is used as a filter.

```go
pastMeetings, err := meetingsClient.List(&meetings.ListOptions{
    MeetingType: "meeting",  // Required: "meeting" for actual instances
    State:       "ended",    // Requires meetingType to be set
    From:        "2026-01-01T00:00:00Z",
    To:          "2026-02-01T00:00:00Z",
    Max:         10,
})
```

#### Additional List Filters

```go
meetingsClient.List(&meetings.ListOptions{
    MeetingType:    "meeting",
    SiteURL:        "cisco.webex.com",  // Filter by Webex site
    IntegrationTag: "my-app",           // Filter by integration tag
    Current:        true,               // Only current meetings
})
```

> **Important:** If you set `State` without `MeetingType`, the SDK will return an error. This matches the Webex API requirement.

### Updating a Meeting

Full update of an existing meeting:

```go
updatedMeeting := &meetings.Meeting{
    Title:  "Updated Weekly Standup",
    Start:  "2026-02-01T11:00:00Z",
    End:    "2026-02-01T11:30:00Z",
    Agenda: "Updated agenda",
}

result, err := meetingsClient.Update("meeting-id", updatedMeeting)
```

The `Title` field is required when updating a meeting.

### Patching a Meeting

Partially update a meeting (only the provided fields):

```go
patch := map[string]interface{}{
    "title": "Quick Title Change",
}

result, err := meetingsClient.Patch("meeting-id", patch)
```

### Deleting a Meeting

Cancel/delete a meeting:

```go
err = meetingsClient.Delete("meeting-id")
```

### Listing Meeting Participants

List participants for an ended meeting instance:

```go
participantsPage, err := meetingsClient.ListParticipants(&meetings.ParticipantListOptions{
    MeetingID: "meeting-instance-id", // Required: must be a meeting instance ID
    Max:       50,
})
if err != nil {
    log.Printf("Failed to list participants: %v", err)
} else {
    for _, p := range participantsPage.Items {
        role := "attendee"
        if p.Host {
            role = "host"
        } else if p.CoHost {
            role = "co-host"
        }
        fmt.Printf("%s (%s) — %s, joined: %s\n",
            p.DisplayName, p.Email, role, p.JoinedTime)
    }
}
```

### Getting a Specific Participant

```go
participant, err := meetingsClient.GetParticipant("participant-id", "meeting-id")
if err != nil {
    log.Printf("Failed to get participant: %v", err)
} else {
    fmt.Printf("Name: %s\n", participant.DisplayName)
    fmt.Printf("Host: %v\n", participant.Host)
    fmt.Printf("Muted: %v\n", participant.Muted)
    for _, d := range participant.Devices {
        fmt.Printf("  Device: %s (%s)\n", d.DeviceType, d.AudioType)
    }
}
```

## Data Structures

### Meeting

```go
type Meeting struct {
    ID                           string
    MeetingSeriesID              string
    ScheduledMeetingID           string
    Title                        string
    Agenda                       string
    Password                     string
    Start                        string                      // ISO 8601
    End                          string                      // ISO 8601
    Timezone                     string
    Recurrence                   string                      // RFC 2445
    EnabledAutoRecordMeeting     bool
    AllowAnyUserToBeCoHost       bool
    EnabledJoinBeforeHost        bool
    EnableConnectAudioBeforeHost bool
    JoinBeforeHostMinutes        int
    ExcludePassword              bool
    PublicMeeting                bool
    MeetingType                  string                      // meetingSeries, scheduledMeeting, meeting
    State                        string                      // active, scheduled, ready, lobby, connected, started, ended, missed, expired
    ScheduledType                string                      // meeting, webinar, personalRoomMeeting
    HostUserID                   string
    HostDisplayName              string
    HostEmail                    string
    SipAddress                   string
    WebLink                      string
    MeetingNumber                string
    SiteURL                      string
    EnabledBreakoutSessions      bool
    Invitees                     []Invitee
    IntegrationTags              []string
    Telephony                    *Telephony                  // Dial-in numbers and access code
    Registration                 *Registration               // Registration form settings
    SimultaneousInterpretation   *SimultaneousInterpretation // Translation settings
    BreakoutSessions             []BreakoutSession           // Breakout session config
    AudioConnectionOptions       *AudioConnectionOptions     // Audio/mute settings
    HasChat                      bool
    HasRecording                 bool
    HasTranscription             bool
    HasSummary                   bool
    HasClosedCaption             bool
    HasPolls                     bool
    HasQA                        bool
    HasRegistration              bool
    HasRegistrants               bool
    Created                      *time.Time
}
```

### Telephony

```go
type Telephony struct {
    AccessCode    string         // Meeting access code
    CallInNumbers []CallInNumber // Phone numbers for dial-in
    Links         *TelephonyLink // Global call-in URLs
}

type CallInNumber struct {
    Label        string // e.g., "US Toll Free"
    CallInNumber string // e.g., "+1-800-555-1234"
    TollType     string // "toll" or "tollFree"
}
```

### AudioConnectionOptions

```go
type AudioConnectionOptions struct {
    AudioConnectionType           string // "webexAudio", "VoIP", "other"
    EnabledTollFreeCallIn         bool
    EnabledGlobalCallIn           bool
    EnabledAudienceCallBack       bool
    EntryAndExitTone              string // "beep", "announceName", etc.
    AllowHostToUnmuteParticipants bool
    AllowAttendeeToUnmuteSelf    bool
    MuteAttendeeUponEntry        bool
}
```

### Invitee

```go
type Invitee struct {
    ID          string // Invitee ID (response only)
    Email       string // Invitee email address
    DisplayName string // Display name
    CoHost      bool   // Whether this invitee is a co-host
    MeetingID   string // Associated meeting ID (response only)
    Panelist    bool   // Whether this invitee is a panelist
}
```

### Participant

```go
type Participant struct {
    ID             string              // Participant ID
    OrgID          string              // Organization ID
    Host           bool                // Whether this is the host
    CoHost         bool                // Whether this is a co-host
    SpaceModerator bool                // Space moderator flag
    Email          string              // Email address
    DisplayName    string              // Display name
    Invitee        bool                // Whether was an invitee
    Muted          bool                // Whether currently muted
    State          string              // "joined", "end", etc.
    JoinedTime     string              // When they joined (ISO 8601)
    LeftTime       string              // When they left (ISO 8601)
    MeetingID      string              // Associated meeting ID
    HostEmail      string              // Host email
    Devices        []ParticipantDevice // Devices used
}

type ParticipantDevice struct {
    DeviceType   string // "tp", "phone", etc.
    JoinedTime   string // Device join time
    LeftTime     string // Device leave time
    CallType     string // Call type
    CallInNumber string // Dial-in number used
    AudioType    string // "voip", "pstn", etc.
}
```

### ListOptions

```go
type ListOptions struct {
    MeetingNumber  string // Filter by meeting number
    MeetingType    string // meetingSeries, scheduledMeeting, meeting
    State          string // active, scheduled, ready, lobby, connected, started, ended, missed, expired
    ScheduledType  string // meeting, webinar, personalRoomMeeting
    HostEmail      string // Filter by host email (admin only)
    SiteURL        string // Filter by Webex site URL
    IntegrationTag string // Filter by integration tag
    From           string // Start date/time filter (ISO 8601)
    To             string // End date/time filter (ISO 8601)
    Max            int    // Maximum number of results
    Current        bool   // Only return current meetings
}
```

### ParticipantListOptions

```go
type ParticipantListOptions struct {
    MeetingID string // Required: meeting instance ID
    HostEmail string // Filter by host email
    Max       int    // Maximum number of results
}
```

## Limitations

- The `state` filter requires `meetingType` to also be specified (Webex API requirement)
- Without `meetingType` specified, the API returns meeting series (recurring definitions) rather than actual meeting instances
- Use `meetingType=meeting` with `state=ended` to list past meetings that have actually occurred
- Meeting participants can only be listed for ended meeting instances
- The `GetParticipant` endpoint requires both the participant ID and the meeting ID

## Error Handling

All methods return structured errors from the `webexsdk` package. Use the convenience functions to check error types:

```go
meeting, err := client.Meetings().Get("MEETING_ID")
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        log.Println("Meeting not found")
    case webexsdk.IsAuthError(err):
        log.Println("Invalid or expired access token")
    case webexsdk.IsRateLimited(err):
        log.Println("Rate limited — SDK retries automatically")
    default:
        log.Printf("Error: %v", err)
    }
}
```

See [webexsdk/Readme.md](../webexsdk/Readme.md) for the full error type reference.

## Related Resources

- [Webex Meetings API Documentation](https://developer.webex.com/docs/api/v1/meetings)
- [Webex Meeting Participants API](https://developer.webex.com/docs/api/v1/meeting-participants)
- [Webex Meetings Overview](https://developer.webex.com/docs/meetings)

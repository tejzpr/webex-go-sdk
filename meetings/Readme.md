# Meetings

The Meetings module provides functionality for interacting with the Webex Meetings API. This module allows you to manage Webex Meetings, including creating, retrieving, updating, listing, and deleting meetings.

## Overview

Webex Meetings are virtual conferences where users can collaborate in real time using audio, video, content sharing, chat, online whiteboards, and more. This module allows you to:

1. Create new meetings
2. Retrieve meeting details
3. List meetings with filtering options
4. Update existing meetings
5. Delete/cancel meetings

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the meetings module:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/meetings"
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

### Getting a Meeting

Retrieve details of a specific meeting:

```go
meeting, err := client.Meetings().Get("meeting-id")
if err != nil {
    log.Printf("Failed to get meeting details: %v", err)
} else {
    fmt.Printf("Meeting Title: %s\n", meeting.Title)
    fmt.Printf("Start: %s\n", meeting.Start)
    fmt.Printf("End: %s\n", meeting.End)
    fmt.Printf("State: %s\n", meeting.State)
}
```

### Listing Meetings

List meetings with various filter options:

```go
// List meeting series (recurring definitions)
meetingsPage, err := client.Meetings().List(&meetings.ListOptions{
    MeetingType: "meetingSeries",
    Max:         50,
})
if err != nil {
    log.Printf("Failed to list meetings: %v", err)
} else {
    fmt.Printf("Found %d meetings\n", len(meetingsPage.Items))
    for i, m := range meetingsPage.Items {
        fmt.Printf("%d. %s (State: %s)\n", i+1, m.Title, m.State)
    }
}
```

#### Listing Past Meeting Instances

To list actual meeting instances that have ended, you **must** specify both `MeetingType` and `State`. The Webex API requires `meetingType` whenever `state` is used as a filter.

```go
pastMeetings, err := client.Meetings().List(&meetings.ListOptions{
    MeetingType: "meeting",  // Required: "meeting" for actual instances
    State:       "ended",    // Requires meetingType to be set
    Max:         10,
})
if err != nil {
    log.Printf("Failed to list past meetings: %v", err)
} else {
    for _, m := range pastMeetings.Items {
        fmt.Printf("%s - Recording: %v, Transcript: %v\n",
            m.Title, m.HasRecording, m.HasTranscription)
    }
}
```

> **Important:** If you set `State` without `MeetingType`, the SDK will return an error. This matches the Webex API requirement.

### Updating a Meeting

Update an existing meeting:

```go
updatedMeeting := &meetings.Meeting{
    Title:  "Updated Weekly Standup",
    Start:  "2026-02-01T11:00:00Z",
    End:    "2026-02-01T11:30:00Z",
    Agenda: "Updated agenda",
}

result, err := client.Meetings().Update("meeting-id", updatedMeeting)
if err != nil {
    log.Printf("Failed to update meeting: %v", err)
} else {
    fmt.Printf("Updated meeting title to: %s\n", result.Title)
}
```

The `Title` field is required when updating a meeting.

### Deleting a Meeting

Cancel/delete a meeting:

```go
err = client.Meetings().Delete("meeting-id")
if err != nil {
    log.Printf("Failed to delete meeting: %v", err)
} else {
    fmt.Println("Successfully deleted meeting")
}
```

## Data Structures

### Meeting Structure

```go
type Meeting struct {
    ID                           string     // Unique identifier of the meeting
    MeetingSeriesID              string     // ID of the parent meeting series
    ScheduledMeetingID           string     // ID of the scheduled meeting occurrence
    Title                        string     // Title of the meeting
    Agenda                       string     // Agenda/description of the meeting
    Password                     string     // Meeting password
    Start                        string     // Start time in ISO 8601 format
    End                          string     // End time in ISO 8601 format
    Timezone                     string     // Timezone (e.g., "America/New_York")
    Recurrence                   string     // Recurrence pattern (RFC 2445)
    EnabledAutoRecordMeeting     bool       // Auto-record the meeting
    AllowAnyUserToBeCoHost       bool       // Allow any user to be co-host
    MeetingType                  string     // Type: meetingSeries, scheduledMeeting, meeting
    State                        string     // State: active, scheduled, ready, lobby, connected, started, ended, missed, expired
    ScheduledType                string     // Scheduled type: meeting, webinar, personalRoomMeeting
    HostUserID                   string     // Host user ID
    HostDisplayName              string     // Host display name
    HostEmail                    string     // Host email address
    SipAddress                   string     // SIP address for video systems
    WebLink                      string     // Link to join the meeting
    MeetingNumber                string     // Meeting number
    SiteURL                      string     // Webex site URL (e.g., "example.webex.com")
    EnabledBreakoutSessions      bool       // Breakout sessions enabled
    IntegrationTags              []string   // Integration metadata tags
    HasChat                      bool       // Whether the meeting had chat
    HasRecording                 bool       // Whether the meeting was recorded
    HasTranscription             bool       // Whether a transcript is available
    HasSummary                   bool       // Whether an AI summary is available
    HasClosedCaption             bool       // Whether closed captions were used
    HasPolls                     bool       // Whether polls were used
    HasQA                        bool       // Whether Q&A was used
    HasRegistration              bool       // Whether registration was enabled
    HasRegistrants               bool       // Whether there are registrants
    Created                      *time.Time // Time when the meeting was created
}
```

### ListOptions

```go
type ListOptions struct {
    MeetingNumber string // Filter by meeting number
    MeetingType   string // Filter by type: meetingSeries, scheduledMeeting, meeting
    State         string // Filter by state: active, scheduled, ready, lobby, connected, started, ended, missed, expired
    ScheduledType string // Filter by scheduled type: meeting, webinar, personalRoomMeeting
    HostEmail     string // Filter by host email (admin only)
    From          string // Start date/time filter (ISO 8601)
    To            string // End date/time filter (ISO 8601)
    Max           int    // Maximum number of results to return
}
```

## Limitations

- The `state` filter requires `meetingType` to also be specified (Webex API requirement)
- Without `meetingType` specified, the API returns meeting series (recurring definitions) rather than actual meeting instances
- Use `meetingType=meeting` with `state=ended` to list past meetings that have actually occurred

## Related Resources

- [Webex Meetings API Documentation](https://developer.webex.com/docs/api/v1/meetings)
- [Webex Meetings Overview](https://developer.webex.com/docs/meetings)

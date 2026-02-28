# Transcripts

The Transcripts module provides functionality for interacting with the Webex Meeting Transcripts API. This module allows you to list meeting transcripts, download transcript content, and work with individual transcript snippets.

## Overview

A meeting transcript provides a complete, AI-powered text record of what was discussed and decided during a meeting. A transcript snippet is a short segment spoken by a specific participant. This module allows you to:

1. List meeting transcripts
2. Download transcript content (VTT or TXT format)
3. List transcript snippets (with speaker and time filters)
4. Get individual snippet details
5. Update snippet text

**Note:** Meeting transcripts are generated only when:
- Meeting recording is turned on and either Webex Assistant or Closed Captions are enabled, OR
- Cisco AI Assistant for Webex is turned on during the meeting (since February 2024)

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the transcripts module:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/transcripts"
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

// Access the Transcripts API
transcriptsClient := client.Transcripts()
```

### Listing Transcripts

List transcripts with optional filters:

```go
transcriptsPage, err := client.Transcripts().List(&transcripts.ListOptions{
    MeetingID: "meeting-id",       // Optional: filter by meeting ID
    HostEmail: "host@example.com", // Optional: filter by host email
    From:      "2026-01-01T00:00:00Z",
    To:        "2026-01-31T23:59:59Z", // Must be within 30 days of From
    Max:       50,
})
if err != nil {
    log.Printf("Failed to list transcripts: %v", err)
} else {
    for i, t := range transcriptsPage.Items {
        fmt.Printf("%d. %s (Status: %s, Duration: %ds)\n",
            i+1, t.MeetingTopic, t.Status, t.Duration)
    }
}
```

> **Important:** The Webex API requires the `from`/`to` date range to be within 30 days. If you don't specify `From`/`To` (and no `MeetingID` is set), the SDK automatically defaults to the last 30 days to ensure results are returned.

### Downloading a Transcript

Download transcript content in plain text or VTT format:

```go
// Download as plain text (with optional meetingId for best results)
content, err := client.Transcripts().Download("transcript-id", "txt", &transcripts.DownloadOptions{
    MeetingID: "meeting-id", // Optional: include meetingId as provided by the API
})
if err != nil {
    log.Printf("Failed to download transcript: %v", err)
} else {
    fmt.Println(content)
}

// Download as VTT (WebVTT with timestamps) — meetingId is optional
vttContent, err := client.Transcripts().Download("transcript-id", "vtt")
if err != nil {
    log.Printf("Failed to download VTT transcript: %v", err)
} else {
    fmt.Println(vttContent)
}
```

### Listing Transcript Snippets

Get individual spoken segments from a transcript, with optional filters:

```go
snippetsPage, err := client.Transcripts().ListSnippets("transcript-id", &transcripts.SnippetListOptions{
    Max: 100,
})
if err != nil {
    log.Printf("Failed to list snippets: %v", err)
} else {
    for _, s := range snippetsPage.Items {
        fmt.Printf("[%s] %s: %s (confidence: %.2f)\n",
            s.StartTime, s.PersonName, s.Text, s.Confidence)
    }
}
```

#### Filtering Snippets by Speaker or Time

```go
snippetsPage, err := client.Transcripts().ListSnippets("transcript-id", &transcripts.SnippetListOptions{
    PersonEmail: "speaker@example.com",       // Filter by speaker email
    PeopleID:    "people-id",                 // Or filter by people ID
    From:        "2026-01-15T10:00:00Z",      // Time range start
    To:          "2026-01-15T10:30:00Z",      // Time range end
    Max:         50,
})
```

### Getting a Specific Snippet

```go
snippet, err := client.Transcripts().GetSnippet("transcript-id", "snippet-id")
if err != nil {
    log.Printf("Failed to get snippet: %v", err)
} else {
    fmt.Printf("Speaker: %s (%s)\n", snippet.PersonName, snippet.PersonEmail)
    fmt.Printf("Text: %s\n", snippet.Text)
    fmt.Printf("Duration: %.1f seconds\n", snippet.Duration)
    fmt.Printf("Confidence: %.2f\n", snippet.Confidence)
}
```

### Updating a Snippet

Correct the text of a transcript snippet:

```go
updated, err := client.Transcripts().UpdateSnippet("transcript-id", "snippet-id", &transcripts.Snippet{
    Text: "Corrected transcript text",
})
if err != nil {
    log.Printf("Failed to update snippet: %v", err)
} else {
    fmt.Printf("Updated snippet text to: %s\n", updated.Text)
}
```

## Data Structures

### Transcript

```go
type Transcript struct {
    ID                 string // Unique identifier
    MeetingID          string // Associated meeting instance ID
    MeetingTopic       string // Meeting topic/title
    SiteURL            string // Webex site URL (e.g., "example.webex.com")
    ScheduledMeetingID string // Scheduled meeting ID
    MeetingSeriesID    string // Meeting series ID
    HostUserID         string // Host user's unique ID
    HostEmail          string // Meeting host email (admin API only)
    StartTime          string // Meeting start time (ISO 8601)
    EndTime            string // Meeting end time (ISO 8601)
    Duration           int    // Duration in seconds
    Status             string // Transcript status ("available")
    VttDownloadLink    string // URL to download VTT format
    TxtDownloadLink    string // URL to download TXT format
    Created            string // When the transcript was created (ISO 8601)
    Updated            string // When the transcript was last updated (ISO 8601)
}
```

### Snippet

```go
type Snippet struct {
    ID                string  // Unique identifier
    TranscriptID      string  // Parent transcript ID
    Text              string  // Spoken text content
    PersonName        string  // Speaker's name
    PersonEmail       string  // Speaker's email
    PeopleID          string  // Speaker's people ID
    StartTime         string  // Snippet start time (ISO 8601)
    EndTime           string  // Snippet end time (ISO 8601)
    Duration          float64 // Duration in seconds
    OffsetMillisecond int     // Offset from meeting start in ms
    Language          string  // Detected language
    Confidence        float64 // Speech recognition confidence score (0.0–1.0)
}
```

### ListOptions

```go
type ListOptions struct {
    MeetingID string // Filter by meeting ID
    HostEmail string // Filter by host email
    SiteURL   string // Filter by Webex site URL
    From      string // Start date/time filter (ISO 8601)
    To        string // End date/time filter (ISO 8601)
    Max       int    // Maximum number of results
}
```

### SnippetListOptions

```go
type SnippetListOptions struct {
    Max         int    // Maximum number of results
    PersonEmail string // Filter by speaker email
    PeopleID    string // Filter by speaker people ID
    From        string // Time range start (ISO 8601)
    To          string // Time range end (ISO 8601)
}
```

## Limitations

- Meeting Transcripts are not supported for Webex for Government (FedRAMP)
- Listing, getting, or updating transcript snippets is not supported for Admin roles
- Transcripts generated by Cisco AI Assistant cannot be updated via API
- The `from`/`to` date range must be within 30 days when listing transcripts

## Error Handling

All methods return structured errors from the `webexsdk` package. Use the convenience functions to check error types:

```go
transcript, err := client.Transcripts().Get("TRANSCRIPT_ID")
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        log.Println("Transcript not found")
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

- [Webex Meeting Transcripts API Documentation](https://developer.webex.com/docs/api/v1/meeting-transcripts)
- [Webex Meetings Overview](https://developer.webex.com/docs/meetings)

# Transcripts

The Transcripts module provides functionality for interacting with the Webex Meeting Transcripts API. This module allows you to list meeting transcripts, download transcript content, and work with individual transcript snippets.

## Overview

A meeting transcript provides a complete, AI-powered text record of what was discussed and decided during a meeting. A transcript snippet is a short segment spoken by a specific participant. This module allows you to:

1. List meeting transcripts
2. Download transcript content (VTT or TXT format)
3. List transcript snippets
4. Get individual snippet details
5. Update snippet text

**Note:** Meeting transcripts are generated only when:
- Meeting recording is turned on and either Webex Assistant or Closed Captions are enabled, OR
- Cisco AI Assistant for Webex is turned on during the meeting (since February 2024)

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the transcripts module:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/transcripts"
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
    To:        "2026-12-31T23:59:59Z",
    Max:       50,
})
if err != nil {
    log.Printf("Failed to list transcripts: %v", err)
} else {
    fmt.Printf("Found %d transcripts\n", len(transcriptsPage.Items))
    for i, t := range transcriptsPage.Items {
        fmt.Printf("%d. %s (Status: %s)\n", i+1, t.Title, t.Status)
    }
}
```

### Downloading a Transcript

Download transcript content in plain text or VTT format:

```go
// Download as plain text
content, err := client.Transcripts().Download("transcript-id", "txt")
if err != nil {
    log.Printf("Failed to download transcript: %v", err)
} else {
    fmt.Println(content)
}

// Download as VTT (WebVTT with timestamps)
vttContent, err := client.Transcripts().Download("transcript-id", "vtt")
if err != nil {
    log.Printf("Failed to download VTT transcript: %v", err)
} else {
    fmt.Println(vttContent)
}
```

### Listing Transcript Snippets

Get individual spoken segments from a transcript:

```go
snippetsPage, err := client.Transcripts().ListSnippets("transcript-id", &transcripts.SnippetListOptions{
    Max: 100,
})
if err != nil {
    log.Printf("Failed to list snippets: %v", err)
} else {
    for _, s := range snippetsPage.Items {
        fmt.Printf("[%s] %s: %s\n", s.StartTime, s.PersonName, s.Text)
    }
}
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

### Transcript Structure

```go
type Transcript struct {
    ID              string // Unique identifier
    MeetingID       string // Associated meeting ID
    HostEmail       string // Meeting host email
    Title           string // Transcript title
    StartTime       string // Meeting start time (ISO 8601)
    Status          string // Transcript status
    VttDownloadLink string // URL to download VTT format
    TxtDownloadLink string // URL to download TXT format
}
```

### Snippet Structure

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

## Limitations

- Meeting Transcripts are not supported for Webex for Government (FedRAMP)
- Listing, getting, or updating transcript snippets is not supported for Admin roles
- Transcripts generated by Cisco AI Assistant cannot be updated via API

## Related Resources

- [Webex Meeting Transcripts API Documentation](https://developer.webex.com/docs/api/v1/meeting-transcripts)
- [Webex Meetings Overview](https://developer.webex.com/docs/meetings)

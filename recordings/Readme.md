# Recordings

The Recordings module provides functionality for interacting with the Webex Recordings API. This module allows you to list, retrieve, download, and delete meeting recordings — including direct download of **audio (MP3)**, **video (MP4)**, and **transcript** files.

## Overview

Recordings are meeting content captured during a Webex meeting. When the recording function is paused during a meeting, the pause is not included in the recording. If recording is stopped and restarted, multiple recordings are created but consolidated and made available together.

This module allows you to:

1. List recordings (with filters for meeting, host, date range, format, etc.)
2. Get recording details (including temporary direct download links)
3. Download audio (MP3), video (MP4), or transcript files
4. Delete recordings

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the recordings module:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/recordings"
)
```

## Usage

### Initializing the Client

```go
// Create a new Webex client with your access token
client, err := webexsdk.NewClient("your-access-token", nil)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}

// Create the Recordings client
recordingsClient := recordings.New(client, nil)
```

### Listing Recordings

List recordings with optional filters:

```go
page, err := recordingsClient.List(&recordings.ListOptions{
    MeetingID: "meeting-id",         // Optional: filter by meeting instance
    HostEmail: "host@example.com",   // Optional: filter by host
    SiteURL:   "cisco.webex.com",    // Optional: filter by site
    From:      "2026-01-01T00:00:00Z",
    To:        "2026-02-01T00:00:00Z",
    Format:    "MP4",                // Optional: "MP4" or "ARF"
    Status:    "available",          // Optional: recording status
    Max:       10,
})
if err != nil {
    log.Printf("Failed to list recordings: %v", err)
} else {
    for _, r := range page.Items {
        fmt.Printf("%s — %s (%d seconds, %.1f MB)\n",
            r.Topic, r.Format, r.DurationSeconds, float64(r.SizeBytes)/1e6)
    }
}
```

### Getting Recording Details

Get a single recording with temporary direct download links:

```go
recording, err := recordingsClient.Get("recording-id")
if err != nil {
    log.Fatalf("Failed to get recording: %v", err)
}

fmt.Printf("Topic: %s\n", recording.Topic)
fmt.Printf("Duration: %d seconds\n", recording.DurationSeconds)
fmt.Printf("Size: %d bytes\n", recording.SizeBytes)
fmt.Printf("Format: %s\n", recording.Format)

// Temporary direct download links (expire after ~3 hours)
if recording.TemporaryDirectDownloadLinks != nil {
    fmt.Printf("Audio:      %s\n", recording.TemporaryDirectDownloadLinks.AudioDownloadLink)
    fmt.Printf("Video:      %s\n", recording.TemporaryDirectDownloadLinks.RecordingDownloadLink)
    fmt.Printf("Transcript: %s\n", recording.TemporaryDirectDownloadLinks.TranscriptDownloadLink)
    fmt.Printf("Expires:    %s\n", recording.TemporaryDirectDownloadLinks.Expiration)
}
```

### Downloading Audio (MP3)

Download just the audio track of a recording:

```go
audio, err := recordingsClient.DownloadAudio("recording-id")
if err != nil {
    log.Fatalf("Failed to download audio: %v", err)
}

fmt.Printf("Content-Type: %s\n", audio.ContentType) // "audio/mpeg"
fmt.Printf("Size: %d bytes\n", len(audio.Data))

// Save to file
os.WriteFile("meeting-audio.mp3", audio.Data, 0644)
```

Or just get the download URL to stream externally:

```go
audioURL, recording, err := recordingsClient.GetAudioDownloadLink("recording-id")
if err != nil {
    log.Fatalf("Failed to get audio link: %v", err)
}
fmt.Printf("Stream from: %s\n", audioURL)
fmt.Printf("Expires: %s\n", recording.TemporaryDirectDownloadLinks.Expiration)
```

### Downloading Video (MP4)

```go
video, err := recordingsClient.DownloadRecording("recording-id")
if err != nil {
    log.Fatalf("Failed to download recording: %v", err)
}

os.WriteFile("meeting-recording.mp4", video.Data, 0644)
```

### Downloading Transcript

```go
transcript, err := recordingsClient.DownloadTranscript("recording-id")
if err != nil {
    log.Fatalf("Failed to download transcript: %v", err)
}

fmt.Println(string(transcript.Data))
```

### Deleting a Recording

```go
err := recordingsClient.Delete("recording-id")
if err != nil {
    log.Printf("Failed to delete recording: %v", err)
}
```

## Data Structures

### Recording

```go
type Recording struct {
    ID                           string                  // Unique identifier
    MeetingID                    string                  // Associated meeting instance ID
    ScheduledMeetingID           string                  // Scheduled meeting ID
    MeetingSeriesID              string                  // Meeting series ID
    Topic                        string                  // Meeting topic/title
    CreateTime                   string                  // When the recording was created (ISO 8601)
    TimeRecorded                 string                  // When the recording started (ISO 8601)
    HostEmail                    string                  // Host email address
    SiteURL                      string                  // Webex site URL
    DownloadURL                  string                  // Password-protected download URL
    PlaybackURL                  string                  // Password-protected playback URL
    Password                     string                  // Recording access password
    TemporaryDirectDownloadLinks *TemporaryDownloadLinks  // Time-limited direct download URLs
    Format                       string                  // "MP4" or "ARF"
    DurationSeconds              int                     // Duration in seconds
    SizeBytes                    int64                   // File size in bytes
    ShareToMe                    bool                    // Whether shared with current user
    ServiceType                  string                  // "MeetingCenter", "EventCenter", etc.
    Status                       string                  // "available", "deleted", etc.
}
```

### TemporaryDownloadLinks

```go
type TemporaryDownloadLinks struct {
    RecordingDownloadLink  string // Direct video (MP4) download URL
    AudioDownloadLink      string // Direct audio (MP3) download URL
    TranscriptDownloadLink string // Direct transcript download URL
    Expiration             string // When the links expire (ISO 8601, ~3 hours)
}
```

### DownloadedContent

```go
type DownloadedContent struct {
    ContentType        string // MIME type (e.g., "audio/mpeg", "video/mp4")
    ContentDisposition string // Content-Disposition header
    ContentLength      int64  // Size in bytes (-1 if unknown)
    Data               []byte // Raw file content
}
```

### ListOptions

```go
type ListOptions struct {
    MeetingID       string // Filter by meeting instance ID
    MeetingSeriesID string // Filter by meeting series ID
    HostEmail       string // Filter by host email
    SiteURL         string // Filter by Webex site URL
    ServiceType     string // Filter by service type
    From            string // Start date/time filter (ISO 8601)
    To              string // End date/time filter (ISO 8601)
    Max             int    // Maximum number of results
    Status          string // Filter by status ("available", etc.)
    Topic           string // Filter by topic keyword
    Format          string // Filter by format ("MP4", "ARF")
}
```

## Notes

- **Temporary download links** are only returned by the `Get` method (not `List`). They expire after approximately 3 hours.
- **Audio download** provides an MP3 file extracted from the meeting recording.
- Recording downloads can be large (hundreds of MB for long meetings). Consider streaming via the URL rather than loading into memory for large files.
- Refer to the [Meetings API Scopes](https://developer.webex.com/docs/meetings#user-level-authentication-and-scopes) for the required authentication scopes.

## Related Resources

- [Webex Recordings API Documentation](https://developer.webex.com/docs/api/v1/recordings)
- [Webex Meetings Overview](https://developer.webex.com/docs/meetings)

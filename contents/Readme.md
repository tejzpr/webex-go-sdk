# Contents

The Contents module provides functionality for downloading file attachments from the Webex API. File attachment URLs returned in `Message.Files` require Bearer token authentication — this module handles that transparently.

## Overview

When a user shares a file in a Webex space, the message contains one or more content URLs pointing to `GET /v1/contents/{contentId}`. This module allows you to:

1. Download a file by its content ID
2. Download a file from a full Webex content URL
3. Handle anti-malware scanning responses (423 Locked, 410 Gone, 428 Precondition Required)
4. Download unscannable files (e.g., encrypted) by opting in with `AllowUnscannable`

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the contents package:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/contents"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the contents client
contentsClient := client.Contents()
```

### Downloading a File by Content ID

```go
fileInfo, err := client.Contents().Download("Y2lzY29zcGFyazov...")
if err != nil {
    log.Fatalf("Error downloading file: %v", err)
}

fmt.Printf("Content-Type: %s\n", fileInfo.ContentType)
fmt.Printf("Disposition: %s\n", fileInfo.ContentDisposition)
fmt.Printf("Size: %d bytes\n", fileInfo.ContentLength)

// Save to disk
os.WriteFile("attachment.pdf", fileInfo.Data, 0644)
```

### Downloading from a Full URL

Message objects contain file URLs directly — pass them to `DownloadFromURL`:

```go
message, _ := client.Messages().Get("MESSAGE_ID")

for _, fileURL := range message.Files {
    fileInfo, err := client.Contents().DownloadFromURL(fileURL)
    if err != nil {
        log.Printf("Error downloading %s: %v", fileURL, err)
        continue
    }
    fmt.Printf("Downloaded: %s (%d bytes)\n", fileInfo.ContentType, fileInfo.ContentLength)
}
```

### Downloading Unscannable Files

Some files (e.g., password-protected archives) cannot be scanned for malware and return `428 Precondition Required`. Use `AllowUnscannable` to download them at your own risk:

```go
opts := &contents.DownloadOptions{
    AllowUnscannable: true,
}

fileInfo, err := client.Contents().DownloadWithOptions("CONTENT_ID", opts)
if err != nil {
    log.Fatalf("Error: %v", err)
}
```

Or with a full URL:

```go
fileInfo, err := client.Contents().DownloadFromURLWithOptions(fileURL, &contents.DownloadOptions{
    AllowUnscannable: true,
})
```

## Anti-Malware Scanning

Webex scans uploaded files for malware. The API may return the following status codes:

| Status | Meaning | SDK Behaviour |
|--------|---------|---------------|
| **200** | File is clean | Returns `FileInfo` with file data |
| **423 Locked** | File is still being scanned | **Automatically retried** by the SDK (respects `Retry-After` header) |
| **410 Gone** | File is infected and was deleted | Returns `*webexsdk.APIError` — use `webexsdk.IsGone(err)` to check |
| **428 Precondition Required** | File cannot be scanned (e.g., encrypted) | Returns `*webexsdk.APIError` — use `AllowUnscannable: true` to bypass |

### Automatic 423 Retry

The SDK automatically retries `423 Locked` responses using exponential backoff. The retry behaviour is controlled by `webexsdk.Config`:

```go
client, err := webex.NewClient(accessToken, &webexsdk.Config{
    MaxRetries:     5,              // Default: 3
    RetryBaseDelay: 2 * time.Second, // Default: 1s (exponential backoff: delay * 2^attempt)
})
```

If the `Retry-After` header is present, its value is used instead of exponential backoff.

## Data Structures

### FileInfo

```go
type FileInfo struct {
    ContentType        string // MIME type (e.g., "image/png", "application/pdf")
    ContentDisposition string // Original filename (e.g., `attachment; filename="report.pdf"`)
    ContentLength      int64  // Size in bytes (-1 if unknown)
    Data               []byte // Raw file content
}
```

### DownloadOptions

```go
type DownloadOptions struct {
    AllowUnscannable bool // When true, appends ?allow=unscannable to bypass 428
}
```

## Error Handling

All methods return structured errors from `webexsdk`. Use the convenience functions to inspect errors:

```go
fileInfo, err := client.Contents().Download("CONTENT_ID")
if err != nil {
    switch {
    case webexsdk.IsGone(err):
        log.Println("File was infected and has been removed")
    case webexsdk.IsLocked(err):
        log.Println("File scanning did not complete after all retries")
    case webexsdk.IsPreconditionRequired(err):
        log.Println("File is unscannable — retry with AllowUnscannable: true")
    case webexsdk.IsNotFound(err):
        log.Println("Content not found")
    case webexsdk.IsAuthError(err):
        log.Println("Invalid or expired access token")
    default:
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/contents"
    "github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func main() {
    accessToken := os.Getenv("WEBEX_ACCESS_TOKEN")
    if accessToken == "" {
        log.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
    }

    // Create client with generous retry settings for large files
    client, err := webex.NewClient(accessToken, &webexsdk.Config{
        MaxRetries:     5,
        RetryBaseDelay: 2 * time.Second,
    })
    if err != nil {
        log.Fatalf("Error creating client: %v", err)
    }

    // Get a message with file attachments
    messageID := os.Getenv("WEBEX_MESSAGE_ID")
    message, err := client.Messages().Get(messageID)
    if err != nil {
        log.Fatalf("Error getting message: %v", err)
    }

    // Download each attached file
    for i, fileURL := range message.Files {
        fileInfo, err := client.Contents().DownloadFromURL(fileURL)
        if err != nil {
            if webexsdk.IsGone(err) {
                fmt.Printf("File %d: infected, skipping\n", i+1)
                continue
            }
            if webexsdk.IsPreconditionRequired(err) {
                // Retry with unscannable bypass
                fileInfo, err = client.Contents().DownloadFromURLWithOptions(fileURL, &contents.DownloadOptions{
                    AllowUnscannable: true,
                })
                if err != nil {
                    log.Printf("File %d: still failed: %v", i+1, err)
                    continue
                }
            } else {
                log.Printf("File %d: download error: %v", i+1, err)
                continue
            }
        }

        filename := fmt.Sprintf("attachment_%d", i+1)
        fmt.Printf("File %d: %s (%d bytes)\n", i+1, fileInfo.ContentType, fileInfo.ContentLength)
        os.WriteFile(filename, fileInfo.Data, 0644)
    }
}
```

## Related Resources

- [Webex Messages API — File Attachments](https://developer.webex.com/docs/api/v1/messages)
- [Webex Content API](https://developer.webex.com/docs/api/v1/contents)
- [Anti-Malware Scanning Behaviour](https://developer.webex.com/docs/api/basics#message-attachments)

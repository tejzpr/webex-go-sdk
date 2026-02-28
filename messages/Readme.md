# Messages

The Messages module provides functionality to interact with the Webex Messages API, allowing you to send, retrieve, update, and delete messages in Webex spaces, as well as listen for new messages in real-time.

## Overview

Messages in Webex are the primary way users communicate within spaces (rooms). This module allows you to:

1. Send messages to spaces or directly to other users
2. Retrieve messages by ID
3. List messages in a space
4. Update existing messages
5. Delete messages
6. Listen for new messages in real-time (Retrieves conversation messages as an encrypted string)

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the messages package:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/messages"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the messages client
messagesClient := client.Messages()
```

### Sending a Message

To send a message to a space:

```go
message := &messages.Message{
    RoomID: "ROOM_ID",
    Text:   "Hello, World!",
}

createdMessage, err := client.Messages().Create(message)
if err != nil {
    log.Fatalf("Error sending message: %v", err)
}

fmt.Printf("Message sent with ID: %s\n", createdMessage.ID)
```

To send a direct message to a person (by ID or email):

```go
// By person ID
message := &messages.Message{
    ToPersonID: "PERSON_ID",
    Text:       "Hello, this is a direct message!",
}

// Or by email
message := &messages.Message{
    ToPersonEmail: "person@example.com",
    Text:          "Hello, this is a direct message!",
}
```

### Sending a Message with Markdown

You can use Markdown formatting for richer text formatting:

```go
message := &messages.Message{
    RoomID:   "ROOM_ID",
    Markdown: "# Hello\n**This** is a _formatted_ message.",
}
```

### Sending a Message with Attachments

You can include attachments like adaptive cards:

```go
message := &messages.Message{
    RoomID: "ROOM_ID",
    Text:   "This is a fallback text for clients that don't support cards",
    Attachments: []messages.Attachment{
        {
            ContentType: "application/vnd.microsoft.card.adaptive",
            Content: map[string]interface{}{
                "type":    "AdaptiveCard",
                "version": "1.0",
                "body": []map[string]interface{}{
                    {
                        "type": "TextBlock",
                        "text": "Card title",
                        "size": "large",
                    },
                    {
                        "type": "TextBlock",
                        "text": "Card content",
                    },
                },
                "actions": []map[string]interface{}{
                    {
                        "type":  "Action.Submit",
                        "title": "Submit",
                    },
                },
            },
        },
    },
}
```

### Retrieving a Message

To get details about a specific message:

```go
message, err := client.Messages().Get("MESSAGE_ID")
if err != nil {
    log.Fatalf("Error getting message: %v", err)
}

fmt.Printf("Message text: %s\n", message.Text)
fmt.Printf("Sent by: %s\n", message.PersonEmail)
fmt.Printf("Created at: %v\n", message.Created)
```

### Listing Messages in a Space

To retrieve messages from a space:

```go
options := &messages.ListOptions{
    RoomID: "ROOM_ID",
    Max:    10, // Optional: limit to 10 messages
}

page, err := client.Messages().List(options)
if err != nil {
    log.Fatalf("Error listing messages: %v", err)
}

for _, message := range page.Items {
    fmt.Printf("Message from %s: %s\n", message.PersonEmail, message.Text)
}
```

You can use additional options for more specific queries:

```go
options := &messages.ListOptions{
    RoomID:          "ROOM_ID",
    MentionedPeople: "me", // Messages that mention you
    BeforeMessage:   "MESSAGE_ID", // Messages before a specific message
    ThreadID:        "THREAD_ID", // Messages in a specific thread
}
```

### Updating a Message

To update an existing message:

```go
updatedMessage := &messages.Message{
    Text: "Updated text for the message",
}

result, err := client.Messages().Update("MESSAGE_ID", updatedMessage)
if err != nil {
    log.Fatalf("Error updating message: %v", err)
}

fmt.Printf("Message updated: %s\n", result.Text)
```

### Deleting a Message

To delete a message:

```go
err = client.Messages().Delete("MESSAGE_ID")
if err != nil {
    log.Fatalf("Error deleting message: %v", err)
}

fmt.Println("Message deleted successfully")
```

### Real-time Message Listening

One of the powerful features of this module is the ability to listen for new messages in real-time:

```go
// Define a message handler function
messageHandler := func(message *messages.Message) {
    fmt.Printf("New message from %s: %s\n", message.PersonEmail, message.Text)
    
    // You can respond to messages here
    if message.Text == "Hello bot" {
        response := &messages.Message{
            RoomID: message.RoomID,
            Text:   "Hello human!",
        }
        client.Messages().Create(response)
    }
}

// Start listening for messages
err = client.Messages().Listen(messageHandler)
if err != nil {
    log.Fatalf("Error starting message listener: %v", err)
}

// When you're done listening (e.g., when your program is shutting down)
client.Messages().StopListening()
```

## Data Structures

### Message Structure

The `Message` structure contains the following fields:

| Field             | Type                    | Description                                           |
|-------------------|-------------------------|-------------------------------------------------------|
| ID                | string                  | Unique identifier for the message                     |
| RoomID            | string                  | ID of the room where the message was sent             |
| ParentID          | string                  | ID of the parent message (for threaded replies)       |
| ToPersonID        | string                  | ID of the recipient (for direct messages)             |
| ToPersonEmail     | string                  | Email of the recipient (for direct messages)          |
| Text              | string                  | Plain text content of the message                     |
| Markdown          | string                  | Markdown-formatted content of the message             |
| HTML              | string                  | HTML-formatted content of the message                 |
| Files             | []string                | Array of file URLs attached to the message            |
| PersonID          | string                  | ID of the person who sent the message                 |
| PersonEmail       | string                  | Email of the person who sent the message              |
| Created           | *time.Time              | Timestamp when the message was created                |
| Updated           | *time.Time              | Timestamp when the message was last updated           |
| MentionedPeople   | []string                | Array of person IDs mentioned in the message          |
| MentionedGroups   | []string                | Array of group names mentioned in the message         |
| Attachments       | []Attachment            | Array of attachments (like adaptive cards)            |

### ListOptions

When listing messages, you can use the following filter options:

| Option          | Type    | Description                                           |
|----------------|---------|-------------------------------------------------------|
| RoomID         | string  | Filter messages by room ID (required)                 |
| MentionedPeople | string  | Filter messages where specific people are mentioned   |
| Before         | string  | Filter messages before a specific date (ISO8601)      |
| BeforeMessage  | string  | Filter messages before a specific message by ID       |
| Max            | int     | Maximum number of messages to return                  |
| ThreadID       | string  | Filter messages within a specific thread              |

## Complete Example

Here's a complete example that demonstrates the major features of the Messages module:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/messages"
)

func main() {
    // Get access token from environment
    accessToken := os.Getenv("WEBEX_ACCESS_TOKEN")
    if accessToken == "" {
        log.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
    }

    // Create client
    client, err := webex.NewClient(accessToken, nil)
    if err != nil {
        log.Fatalf("Error creating client: %v", err)
    }

    // Get a room ID to work with
    roomID := os.Getenv("WEBEX_ROOM_ID")
    if roomID == "" {
        log.Fatal("WEBEX_ROOM_ID environment variable is required")
    }

    // Send a message to the room
    message := &messages.Message{
        RoomID: roomID,
        Text:   "Hello from the Webex Go SDK!",
    }

    createdMessage, err := client.Messages().Create(message)
    if err != nil {
        log.Fatalf("Error sending message: %v", err)
    }
    fmt.Printf("Message sent: ID=%s\n", createdMessage.ID)

    // List recent messages in the room
    options := &messages.ListOptions{
        RoomID: roomID,
        Max:    5,
    }
    
    page, err := client.Messages().List(options)
    if err != nil {
        log.Fatalf("Error listing messages: %v", err)
    }

    fmt.Printf("Recent messages in the room:\n")
    for i, msg := range page.Items {
        fmt.Printf("%d. %s: %s\n", i+1, msg.PersonEmail, msg.Text)
    }

    // Set up a message listener
    fmt.Println("\nStarting message listener...")
    messageHandler := func(message *messages.Message) {
        fmt.Printf("New message from %s: %s\n", message.PersonEmail, message.Text)
        
        // Auto-respond to messages containing "hello"
        if client.Messages() != nil && message.PersonID != createdMessage.PersonID {
            lowercase := strings.ToLower(message.Text)
            if strings.Contains(lowercase, "hello") {
                response := &messages.Message{
                    RoomID: message.RoomID,
                    Text:   "Hello! I'm a bot using the Webex Go SDK.",
                }
                _, err := client.Messages().Create(response)
                if err != nil {
                    fmt.Printf("Error sending response: %v\n", err)
                }
            }
        }
    }

    err = client.Messages().Listen(messageHandler)
    if err != nil {
        log.Fatalf("Error starting message listener: %v", err)
    }

    fmt.Println("Listening for messages. Send a message in the space to test.")
    fmt.Println("Press Ctrl+C to exit.")

    // Wait for termination signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Clean up before exiting
    fmt.Println("Stopping message listener...")
    client.Messages().StopListening()

    // Delete the message we sent at the beginning
    fmt.Printf("Deleting message with ID %s...\n", createdMessage.ID)
    err = client.Messages().Delete(createdMessage.ID)
    if err != nil {
        log.Printf("Error deleting message: %v", err)
    } else {
        fmt.Println("Message deleted successfully")
    }
}
```

## Error Handling

All methods return structured errors from the `webexsdk` package. Use the convenience functions to check error types:

```go
message, err := client.Messages().Get("MESSAGE_ID")
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        log.Println("Message not found")
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

- [Webex Messages API Documentation](https://developer.webex.com/docs/api/v1/messages)
- [Markdown Support in Webex](https://developer.webex.com/docs/api/basics#formatting-messages)
- [Adaptive Cards Documentation](https://adaptivecards.io/)

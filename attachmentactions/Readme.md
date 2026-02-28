# Attachment Actions

The Attachment Actions module provides functionality to interact with Webex Attachment Actions API, which allows you to handle user inputs from adaptive cards in Webex messages.

## Overview

Attachment Actions in Webex are created when a user interacts with an adaptive card in a Webex message. When a user submits a form or clicks a button in an adaptive card, an attachment action is created with the user's inputs. This module allows you to programmatically:

1. Create attachment actions (simulate user input)
2. Retrieve attachment action details by ID

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the attachment actions package:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/attachmentactions"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the attachment actions client
attachmentActionsClient := client.AttachmentActions()
```

### Retrieving an Attachment Action

To retrieve details of an attachment action by its ID:

```go
action, err := client.AttachmentActions().Get("ATTACHMENT_ACTION_ID")
if err != nil {
    log.Fatalf("Error getting attachment action: %v", err)
}

// Access action properties
fmt.Printf("Action ID: %s\n", action.ID)
fmt.Printf("Type: %s\n", action.Type)
fmt.Printf("Message ID: %s\n", action.MessageID)
fmt.Printf("Room ID: %s\n", action.RoomID)
fmt.Printf("Person ID: %s\n", action.PersonID)
fmt.Printf("Created: %v\n", action.Created)
fmt.Printf("Inputs: %v\n", action.Inputs)
```

### Creating an Attachment Action

You can programmatically create attachment actions to simulate user input:

```go
action := &attachmentactions.AttachmentAction{
    Type:      "submit",
    MessageID: "MESSAGE_ID_WITH_ADAPTIVE_CARD",
    Inputs: map[string]interface{}{
        "inputFieldId1": "Value 1",
        "inputFieldId2": true,
    },
}

createdAction, err := client.AttachmentActions().Create(action)
if err != nil {
    log.Fatalf("Error creating attachment action: %v", err)
}
```

## Data Structures

### AttachmentAction Structure

The `AttachmentAction` structure contains the following fields:

| Field     | Type                    | Description                                           |
|-----------|-------------------------|-------------------------------------------------------|
| ID        | string                  | Unique identifier for the action                      |
| Type      | string                  | Type of action (usually "submit")                     |
| MessageID | string                  | ID of the message containing the adaptive card        |
| Inputs    | map[string]interface{}  | Map of user inputs from the adaptive card             |
| PersonID  | string                  | ID of the person who performed the action             |
| RoomID    | string                  | ID of the room where the action occurred              |
| Created   | *time.Time              | Timestamp when the action was created                 |

## Complete Example

Here's a complete example that demonstrates sending a message with an adaptive card and handling attachment actions:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/attachmentactions"
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

    // Send a message with an adaptive card
    roomID := os.Getenv("WEBEX_ROOM_ID")
    if roomID == "" {
        log.Fatal("WEBEX_ROOM_ID environment variable is required")
    }

    message := &messages.Message{
        RoomID: roomID,
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
                            "text": "Adaptive Card Example",
                            "size": "large",
                        },
                        {
                            "type":        "Input.Text",
                            "id":          "userInput",
                            "placeholder": "Enter some text",
                        },
                        {
                            "type":     "Input.Toggle",
                            "id":       "acceptTerms",
                            "title":    "I accept the terms and conditions",
                            "valueOn":  "true",
                            "valueOff": "false",
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

    createdMessage, err := client.Messages().Create(message)
    if err != nil {
        log.Fatalf("Error sending message: %v", err)
    }
    
    // Example: Create an attachment action programmatically
    action := &attachmentactions.AttachmentAction{
        Type:      "submit",
        MessageID: createdMessage.ID,
        Inputs: map[string]interface{}{
            "userInput":   "This is a test input",
            "acceptTerms": "true",
        },
    }

    createdAction, err := client.AttachmentActions().Create(action)
    if err != nil {
        log.Fatalf("Error creating attachment action: %v", err)
    }
    
    // Get the created action
    retrievedAction, err := client.AttachmentActions().Get(createdAction.ID)
    if err != nil {
        log.Fatalf("Error getting attachment action: %v", err)
    }
    
    fmt.Printf("Retrieved action inputs: %v\n", retrievedAction.Inputs)
}
```

## Error Handling

All methods return structured errors from the `webexsdk` package. Use the convenience functions to check error types:

```go
action, err := client.AttachmentActions().Get("ACTION_ID")
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        log.Println("Attachment action not found")
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

- [Webex Attachment Actions API Documentation](https://developer.webex.com/docs/api/v1/attachment-actions)
- [Adaptive Cards Documentation](https://adaptivecards.io/)

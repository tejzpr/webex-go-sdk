# Events

The Events module provides functionality to interact with the Webex Events API, allowing you to retrieve event data for activities that occur within your Webex organization.

## Overview

Webex Events provide a way to track and collect information about activities that occur within your Webex organization. This module allows compliance officers and administrators to:

1. List events with various filters (by resource type, event type, time range, etc.)
2. Retrieve detailed information about specific events by ID

The Events API is primarily intended for compliance and administrative purposes, enabling you to audit user activities across your Webex organization.

## Access Requirements

**Important:** The Events API is only available to Compliance Officers and administrators with the appropriate permissions in your Webex organization.

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the events package:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/events"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the events client
eventsClient := client.Events()
```

### Listing Events

To retrieve a list of events with optional filters:

```go
// Calculate time 7 days ago
fromTime := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)

eventsPage, err := client.Events().List(&events.ListOptions{
    Resource: "messages", // Filter for message events
    Type:     "created",  // Filter for created events
    From:     fromTime,   // Events from the last 7 days
    Max:      10,         // Limit to 10 events
})

if err != nil {
    log.Fatalf("Failed to list events: %v", err)
}

// Process the events
for _, event := range eventsPage.Items {
    fmt.Printf("Event: %s (%s)\n", event.Type, event.ID)
    fmt.Printf("Resource: %s\n", event.Resource)
    fmt.Printf("Created: %s\n", event.Created.Format(time.RFC3339))
    // ... process other event data
}
```

### Retrieving a Specific Event

To retrieve details of a specific event by its ID:

```go
eventDetails, err := client.Events().Get("EVENT_ID")
if err != nil {
    log.Fatalf("Failed to get event details: %v", err)
}

// Access event properties
fmt.Printf("Event ID: %s\n", eventDetails.ID)
fmt.Printf("Resource: %s\n", eventDetails.Resource)
fmt.Printf("Type: %s\n", eventDetails.Type)
// ... access other event properties
```

## Data Structures

### Event Structure

The `Event` structure contains the following fields:

| Field     | Type                    | Description                                           |
|-----------|-------------------------|-------------------------------------------------------|
| ID        | string                  | Unique identifier for the event                       |
| Resource  | string                  | The resource type the event is related to             |
| Type      | string                  | The type of event (e.g., "created", "updated")        |
| AppID     | string                  | The ID of the application that triggered the event    |
| ActorID   | string                  | The ID of the person who performed the action         |
| OrgID     | string                  | The organization ID where the event occurred          |
| Created   | time.Time              | Timestamp when the event occurred                     |
| Data      | EventData               | Resource-specific data about the event                |

The `EventData` structure contains resource-specific data that varies depending on the event type and resource. Common fields include:

- For message events: RoomID, RoomType, PersonID, PersonEmail, etc.
- For meeting events: MeetingID, CreatorID, RecordingEnabled, etc.
- For telephony events: CallType, CallDirection, CallDurationSeconds, etc.

### ListOptions

When listing events, you can use the following filter options:

| Option    | Type   | Description                                           |
|-----------|--------|-------------------------------------------------------|
| Resource  | string | Filter by resource type (e.g., "messages", "meetings") |
| Type      | string | Filter by event type (e.g., "created", "updated")     |
| ActorID   | string | Filter by the ID of the person who performed the action |
| From      | string | Filter events that occurred after this time (RFC3339 format) |
| To        | string | Filter events that occurred before this time (RFC3339 format) |
| Max       | int    | Maximum number of events to return                    |

### Resource Types

Common resource types that can be monitored with the Events API include:

- `messages`: Message-related events in spaces
- `meetings`: Meeting creation, updates, and deletion
- `rooms`: Space creation, updates, and deletion
- `memberships`: Space membership changes
- `teams`: Team creation, updates, and deletion
- `team.memberships`: Team membership changes
- `telephony`: Call events (for organizations with Webex Calling)

### Event Types

Common event types include:

- `created`: Resource creation events
- `updated`: Resource update events
- `deleted`: Resource deletion events

## Complete Example

Here's a complete example that demonstrates listing events and retrieving event details:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/events"
)

func main() {
    // Get access token from environment variable
    accessToken := os.Getenv("WEBEX_ACCESS_TOKEN")
    if accessToken == "" {
        log.Fatalf("WEBEX_ACCESS_TOKEN environment variable is required")
    }

    // Create a new Webex client
    client, err := webex.NewClient(accessToken, nil)
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }

    // List Events with filters
    fromTime := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)

    eventsPage, err := client.Events().List(&events.ListOptions{
        Resource: "messages", // Filter for message events
        Type:     "created",  // Filter for created events
        From:     fromTime,   // Events from the last 7 days
        Max:      10,         // Limit to 10 events
    })

    if err != nil {
        log.Fatalf("Failed to list events: %v", err)
    }

    fmt.Printf("Found %d events\n", len(eventsPage.Items))
    for i, event := range eventsPage.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, event.Type, event.ID)
        fmt.Printf("   Resource: %s\n", event.Resource)
        fmt.Printf("   Created: %s\n", event.Created.Format(time.RFC3339))
        fmt.Printf("   Actor ID: %s\n", event.ActorID)
        if event.Data.RoomID != "" {
            fmt.Printf("   Room ID: %s\n", event.Data.RoomID)
        }
        fmt.Println()
    }

    // Get Event Details
    if len(eventsPage.Items) > 0 {
        eventID := eventsPage.Items[0].ID
        eventDetails, err := client.Events().Get(eventID)
        if err != nil {
            log.Printf("Failed to get event details: %v\n", err)
        } else {
            fmt.Printf("Event Details:\n")
            fmt.Printf("  ID: %s\n", eventDetails.ID)
            fmt.Printf("  Resource: %s\n", eventDetails.Resource)
            fmt.Printf("  Type: %s\n", eventDetails.Type)
            fmt.Printf("  Actor ID: %s\n", eventDetails.ActorID)
            fmt.Printf("  Created: %s\n", eventDetails.Created.Format(time.RFC3339))

            // Process data fields based on resource type
            if eventDetails.Resource == "messages" {
                fmt.Printf("  Room ID: %s\n", eventDetails.Data.RoomID)
                fmt.Printf("  Person ID: %s\n", eventDetails.Data.PersonID)
            } else if eventDetails.Resource == "meetings" {
                fmt.Printf("  Meeting ID: %s\n", eventDetails.Data.MeetingID)
                fmt.Printf("  Creator ID: %s\n", eventDetails.Data.CreatorID)
            }
        }
    }
}
```

## Related Resources

- [Webex Events API Documentation](https://developer.webex.com/docs/api/v1/events)
- [Webex Compliance Guide](https://developer.webex.com/docs/compliance)

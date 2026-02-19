# Rooms

The Rooms module provides functionality to interact with the Webex Rooms API, allowing you to create, retrieve, update, and delete Webex spaces (rooms).

## Overview

Rooms (also called spaces) in Webex are virtual meeting places where people can message, share content, and meet. This module allows you to:

1. Create new rooms
2. Retrieve room details
3. List rooms with various filters
4. Update room properties
5. Delete rooms

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the rooms package:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/rooms"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the rooms client
roomsClient := client.Rooms()
```

### Creating a Room

To create a new room:

```go
room := &rooms.Room{
    Title: "My New Room",
    // Optional: Specify the team this room belongs to
    TeamID: "TEAM_ID",
}

createdRoom, err := client.Rooms().Create(room)
if err != nil {
    log.Fatalf("Error creating room: %v", err)
}

fmt.Printf("Room created: ID=%s, Title=%s\n", createdRoom.ID, createdRoom.Title)
```

### Retrieving a Room

To get details about a specific room:

```go
roomDetails, err := client.Rooms().Get("ROOM_ID")
if err != nil {
    log.Fatalf("Error getting room: %v", err)
}

fmt.Printf("Room: %s (Type: %s)\n", roomDetails.Title, roomDetails.Type)
fmt.Printf("Created: %v\n", roomDetails.Created)
```

### Listing Rooms

To retrieve a list of rooms:

```go
options := &rooms.ListOptions{
    Max: 10, // Optional: limit to 10 rooms
}

page, err := client.Rooms().List(options)
if err != nil {
    log.Fatalf("Error listing rooms: %v", err)
}

fmt.Printf("Found %d rooms:\n", len(page.Items))
for i, room := range page.Items {
    fmt.Printf("%d. %s (ID: %s)\n", i+1, room.Title, room.ID)
}
```

You can use additional filters when listing rooms:

```go
options := &rooms.ListOptions{
    TeamID: "TEAM_ID", // Filter rooms by team
    Type:   "group",   // Filter by type (direct or group)
    SortBy: "lastactivity", // Sort by last activity
    Max:    50,             // Limit to 50 rooms
}
```

### Updating a Room

To update a room's properties:

```go
updateRoom := &rooms.Room{
    Title:    "Updated Room Title",
    IsLocked: true, // Lock the room
}

updatedRoom, err := client.Rooms().Update("ROOM_ID", updateRoom)
if err != nil {
    log.Fatalf("Error updating room: %v", err)
}

fmt.Printf("Room updated: %s (Locked: %t)\n", updatedRoom.Title, updatedRoom.IsLocked)
```

### Deleting a Room

To delete a room:

```go
err = client.Rooms().Delete("ROOM_ID")
if err != nil {
    log.Fatalf("Error deleting room: %v", err)
}

fmt.Println("Room deleted successfully")
```

## Data Structures

### Room Structure

The `Room` structure contains the following fields:

| Field        | Type       | Description                                           |
|--------------|------------|-------------------------------------------------------|
| ID           | string     | Unique identifier for the room                        |
| Title        | string     | Title of the room                                     |
| TeamID       | string     | ID of the team the room belongs to (if applicable)    |
| IsLocked     | bool       | Whether the room is moderated/locked                  |
| Type         | string     | Type of room ("direct" or "group")                    |
| CreatorID    | string     | ID of the person who created the room                 |
| Created      | *time.Time | Timestamp when the room was created                   |
| LastActivity | *time.Time | Timestamp of the last activity in the room            |

There's also a `RoomWithReadStatus` structure that includes additional fields:

| Field            | Type       | Description                                          |
|------------------|------------|------------------------------------------------------|
| LastActivityDate | *time.Time | Timestamp of the last activity in the room           |
| LastSeenDate     | *time.Time | Timestamp when you last viewed the room              |

### ListOptions

When listing rooms, you can use the following filter options:

| Option  | Type    | Description                                                 |
|---------|---------|-------------------------------------------------------------|
| TeamID  | string  | Filter rooms by team ID                                     |
| Type    | string  | Filter by room type ("direct" or "group")                   |
| SortBy  | string  | Sort by: "id", "lastactivity", or "created"                 |
| Max     | int     | Maximum number of rooms to return                           |

### Room Types

Webex has two types of rooms:

- **Direct (1:1)**: Private spaces automatically created when you message someone directly
- **Group**: Spaces that can include multiple people, have titles, and can be moderated

## Complete Example

Here's a complete example that demonstrates the major operations with rooms:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/rooms"
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

    // Create a room
    room := &rooms.Room{
        Title: "Go SDK Room Example",
    }

    createdRoom, err := client.Rooms().Create(room)
    if err != nil {
        log.Fatalf("Error creating room: %v", err)
    }
    fmt.Printf("Room created: ID=%s, Title=%s\n", createdRoom.ID, createdRoom.Title)

    // Get room details
    roomDetails, err := client.Rooms().Get(createdRoom.ID)
    if err != nil {
        log.Fatalf("Error getting room: %v", err)
    }
    fmt.Printf("\nRoom details:\n")
    fmt.Printf("  ID: %s\n", roomDetails.ID)
    fmt.Printf("  Title: %s\n", roomDetails.Title)
    fmt.Printf("  Type: %s\n", roomDetails.Type)
    fmt.Printf("  IsLocked: %t\n", roomDetails.IsLocked)
    fmt.Printf("  CreatorID: %s\n", roomDetails.CreatorID)
    fmt.Printf("  Created: %v\n", roomDetails.Created)

    // List rooms
    options := &rooms.ListOptions{
        Max: 10,
    }

    page, err := client.Rooms().List(options)
    if err != nil {
        log.Fatalf("Error listing rooms: %v", err)
    }

    fmt.Printf("\nFound %d rooms:\n", len(page.Items))
    for i, r := range page.Items {
        fmt.Printf("%d. %s (ID: %s)\n", i+1, r.Title, r.ID)
    }

    // Update the room
    updateRoom := &rooms.Room{
        Title: "Go SDK Room Example (Updated)",
    }

    updatedRoom, err := client.Rooms().Update(createdRoom.ID, updateRoom)
    if err != nil {
        log.Fatalf("Error updating room: %v", err)
    }
    fmt.Printf("\nRoom updated: ID=%s, New Title=%s\n", updatedRoom.ID, updatedRoom.Title)

    // Delete the room
    fmt.Printf("\nDeleting room %s...\n", createdRoom.ID)
    err = client.Rooms().Delete(createdRoom.ID)
    if err != nil {
        log.Fatalf("Error deleting room: %v", err)
    }
    fmt.Println("Room deleted successfully")
}
```

## Related Resources

- [Webex Rooms API Documentation](https://developer.webex.com/docs/api/v1/rooms)

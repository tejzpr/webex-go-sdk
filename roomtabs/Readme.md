# RoomTabs

The RoomTabs module provides functionality for interacting with the Webex Room Tabs API. Room tabs allow you to add custom web content to Webex rooms, enabling integration of external applications, dashboards, or web tools directly within your Webex spaces.

## Overview

Room tabs in Webex allow you to embed web content inside a Webex space. This module allows you to:

1. Create tabs with custom web content in rooms
2. Retrieve tab details
3. List all tabs in a room
4. Update existing tabs
5. Delete tabs

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the roomtabs module:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/roomtabs"
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

// Access the RoomTabs API
roomTabsClient := client.RoomTabs()
```

### Creating a Room Tab

To create a new tab in a Webex room:

```go
newTab := &roomtabs.RoomTab{
    RoomID:      "room-id",
    DisplayName: "My Custom Tab",
    ContentURL:  "https://example.com/my-content",
}

createdTab, err := client.RoomTabs().Create(newTab)
if err != nil {
    log.Printf("Failed to create room tab: %v", err)
} else {
    fmt.Printf("Created tab with ID: %s\n", createdTab.ID)
}
```

Required fields:
- `RoomID`: The ID of the room where the tab will be created
- `DisplayName`: The name of the tab shown to users
- `ContentURL`: The URL of the content to display in the tab

### Getting a Room Tab

Retrieve details of a specific room tab:

```go
tabDetails, err := client.RoomTabs().Get("tab-id")
if err != nil {
    log.Printf("Failed to get room tab details: %v", err)
} else {
    fmt.Printf("Tab name: %s\n", tabDetails.DisplayName)
    fmt.Printf("Content URL: %s\n", tabDetails.ContentURL)
}
```

### Listing Room Tabs

List all tabs in a specific room:

```go
tabs, err := client.RoomTabs().List(&roomtabs.ListOptions{
    RoomID: "room-id",
})
if err != nil {
    log.Printf("Failed to list room tabs: %v", err)
} else {
    fmt.Printf("Found %d room tabs\n", len(tabs.Items))
    for i, tab := range tabs.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, tab.DisplayName, tab.ID)
    }
}
```

### Updating a Room Tab

Update the properties of an existing room tab:

```go
tab.DisplayName = "Updated Tab Name"
tab.ContentURL = "https://example.com/updated-content"

updatedTab, err := client.RoomTabs().Update(tab.ID, tab)
if err != nil {
    log.Printf("Failed to update room tab: %v", err)
} else {
    fmt.Printf("Updated tab display name to: %s\n", updatedTab.DisplayName)
}
```

### Deleting a Room Tab

Remove a tab from a room:

```go
err = client.RoomTabs().Delete("tab-id")
if err != nil {
    log.Printf("Failed to delete room tab: %v", err)
} else {
    fmt.Println("Successfully deleted room tab")
}
```

## Data Structures

### RoomTab Structure

The RoomTab structure represents a tab in a Webex room with the following fields:

```go
type RoomTab struct {
    ID          string     // Unique identifier of the tab
    RoomID      string     // ID of the room where the tab exists
    RoomType    string     // Type of the room (group or direct)
    DisplayName string     // Name of the tab shown to users
    ContentURL  string     // URL of the content to display in the tab
    CreatorID   string     // ID of the user who created the tab
    Created     *time.Time // Time when the tab was created
}
```

### ListOptions

When listing room tabs, you must specify the room ID:

```go
type ListOptions struct {
    RoomID string // Required: ID of the room to list tabs from
}
```

## Complete Example

Here's a complete example demonstrating the major operations with room tabs:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/roomtabs"
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

    // Get a room ID to work with
    roomID := os.Getenv("WEBEX_ROOM_ID")
    if roomID == "" {
        log.Fatalf("WEBEX_ROOM_ID environment variable is required")
    }

    // List existing tabs
    tabs, err := client.RoomTabs().List(&roomtabs.ListOptions{
        RoomID: roomID,
    })
    if err != nil {
        log.Fatalf("Failed to list room tabs: %v", err)
    }
    fmt.Printf("Found %d room tabs\n", len(tabs.Items))

    // Create a new tab
    newTab := &roomtabs.RoomTab{
        RoomID:      roomID,
        DisplayName: "API Demo Tab",
        ContentURL:  "https://example.com/demo",
    }
    createdTab, err := client.RoomTabs().Create(newTab)
    if err != nil {
        log.Printf("Failed to create room tab: %v\n", err)
        return
    }
    fmt.Printf("Created tab with ID: %s\n", createdTab.ID)

    // Get tab details
    tabDetails, err := client.RoomTabs().Get(createdTab.ID)
    if err != nil {
        log.Printf("Failed to get room tab details: %v\n", err)
        return
    }

    // Update the tab
    tabDetails.DisplayName = "Updated API Demo Tab"
    tabDetails.ContentURL = "https://example.com/updated-demo"
    updatedTab, err := client.RoomTabs().Update(tabDetails.ID, tabDetails)
    if err != nil {
        log.Printf("Failed to update room tab: %v\n", err)
        return
    }
    fmt.Printf("Updated tab to: %s\n", updatedTab.DisplayName)

    // Delete the tab
    err = client.RoomTabs().Delete(updatedTab.ID)
    if err != nil {
        log.Printf("Failed to delete room tab: %v\n", err)
        return
    }
    fmt.Printf("Successfully deleted room tab\n")
}
```

## Related Resources

- [Webex Room Tabs API Documentation](https://developer.webex.com/docs/api/v1/room-tabs)
- [Webex Rooms API Documentation](https://developer.webex.com/docs/api/v1/rooms)

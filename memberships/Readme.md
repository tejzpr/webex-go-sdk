# Memberships

The Memberships module provides functionality to interact with the Webex Memberships API, allowing you to manage user memberships in Webex spaces (rooms).

## Overview

Memberships in Webex represent a person's association with a space (room). This module allows you to:

1. List memberships for a person or in a space
2. Create memberships (add people to spaces)
3. Retrieve membership details
4. Update membership properties (e.g., moderator status)
5. Delete memberships (remove people from spaces)

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the memberships package:

```go
import (
    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/memberships"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the memberships client
membershipsClient := client.Memberships()
```

### Creating a Membership

To add a person to a room:

```go
membership := &memberships.Membership{
    RoomID:      "ROOM_ID",
    PersonEmail: "person@example.com", // or PersonID: "PERSON_ID"
    IsModerator: false, // Optional: make the person a moderator
}

createdMembership, err := client.Memberships().Create(membership)
if err != nil {
    log.Fatalf("Error creating membership: %v", err)
}

fmt.Printf("Added %s to the room with membership ID: %s\n",
    createdMembership.PersonEmail,
    createdMembership.ID)
```

### Retrieving a Membership

To get details about a specific membership:

```go
membership, err := client.Memberships().Get("MEMBERSHIP_ID")
if err != nil {
    log.Fatalf("Error getting membership: %v", err)
}

fmt.Printf("Membership details:\n")
fmt.Printf("  Person: %s (%s)\n", membership.PersonDisplayName, membership.PersonEmail)
fmt.Printf("  Room ID: %s\n", membership.RoomID)
fmt.Printf("  Is Moderator: %t\n", membership.IsModerator)
```

### Listing Memberships

To list memberships in a specific room:

```go
options := &memberships.ListOptions{
    RoomID: "ROOM_ID",
    Max:    10, // Limit results to 10 memberships (optional)
}

page, err := client.Memberships().List(options)
if err != nil {
    log.Fatalf("Error listing memberships: %v", err)
}

for _, membership := range page.Items {
    fmt.Printf("Person: %s (%s) - Moderator: %t\n",
        membership.PersonDisplayName,
        membership.PersonEmail,
        membership.IsModerator)
}
```

To list all rooms where a specific person is a member:

```go
options := &memberships.ListOptions{
    PersonID: "PERSON_ID", // or PersonEmail: "person@example.com"
}

page, err := client.Memberships().List(options)
```

### Updating a Membership

To update a membership (e.g., change moderator status):

```go
updatedMembership := &memberships.Membership{
    IsModerator: true, // Set moderator status
}

result, err := client.Memberships().Update("MEMBERSHIP_ID", updatedMembership)
if err != nil {
    log.Fatalf("Error updating membership: %v", err)
}

fmt.Printf("Updated %s to moderator status: %t\n",
    result.PersonEmail,
    result.IsModerator)
```

### Deleting a Membership

To remove a person from a room:

```go
err = client.Memberships().Delete("MEMBERSHIP_ID")
if err != nil {
    log.Fatalf("Error deleting membership: %v", err)
}

fmt.Println("Successfully removed the person from the room")
```

## Data Structures

### Membership Structure

The `Membership` structure contains the following fields:

| Field             | Type       | Description                                             |
|-------------------|------------|---------------------------------------------------------|
| ID                | string     | Unique identifier for the membership                    |
| RoomID            | string     | ID of the room                                          |
| PersonID          | string     | ID of the person                                        |
| PersonEmail       | string     | Email address of the person                             |
| PersonDisplayName | string     | Display name of the person                              |
| PersonOrgID       | string     | Organization ID of the person                           |
| IsModerator       | bool       | Whether the person is a moderator in the room           |
| IsMonitor         | bool       | Whether the person is a monitor in the room             |
| IsRoomHidden      | bool       | Whether the room is hidden in the person's client       |
| Created           | *time.Time | Timestamp when the membership was created               |
| RoomType          | string     | Type of room (e.g., "direct", "group")                  |
| LastSeenID        | string     | ID of the last seen message                             |
| LastSeenDate      | *time.Time | Timestamp when the person last saw a message in the room|

### ListOptions

When listing memberships, you can use the following filter options:

| Option      | Type   | Description                                           |
|-------------|--------|-------------------------------------------------------|
| RoomID      | string | Filter memberships by room ID                         |
| PersonID    | string | Filter memberships by person ID                       |
| PersonEmail | string | Filter memberships by person email                    |
| Max         | int    | Maximum number of memberships to return               |

## Complete Example

Here's a complete example that demonstrates the major operations with memberships:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/WebexCommunity/webex-go-sdk/v2"
    "github.com/WebexCommunity/webex-go-sdk/v2/memberships"
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

    // List memberships in a room
    roomID := os.Getenv("WEBEX_ROOM_ID")
    if roomID == "" {
        log.Fatal("WEBEX_ROOM_ID environment variable is required")
    }

    options := &memberships.ListOptions{
        RoomID: roomID,
        Max:    10,
    }

    page, err := client.Memberships().List(options)
    if err != nil {
        log.Fatalf("Error listing memberships: %v", err)
    }

    fmt.Printf("Found %d memberships:\n", len(page.Items))
    for i, membership := range page.Items {
        fmt.Printf("%d. %s (%s) - Moderator: %t\n",
            i+1,
            membership.PersonEmail,
            membership.PersonDisplayName,
            membership.IsModerator)
    }

    // Add a person to the room
    personEmail := "new.person@example.com"
    membership := &memberships.Membership{
        RoomID:      roomID,
        PersonEmail: personEmail,
        IsModerator: false,
    }

    createdMembership, err := client.Memberships().Create(membership)
    if err != nil {
        log.Fatalf("Error creating membership: %v", err)
    }

    fmt.Printf("Added %s to the room with membership ID: %s\n",
        createdMembership.PersonEmail,
        createdMembership.ID)

    // Update membership (make person a moderator)
    updatedMembership := &memberships.Membership{
        IsModerator: true,
    }

    result, err := client.Memberships().Update(createdMembership.ID, updatedMembership)
    if err != nil {
        log.Fatalf("Error updating membership: %v", err)
    }

    fmt.Printf("Updated %s to moderator status: %t\n",
        result.PersonEmail,
        result.IsModerator)

    // Get membership details
    membershipDetails, err := client.Memberships().Get(createdMembership.ID)
    if err != nil {
        log.Fatalf("Error getting membership: %v", err)
    }

    fmt.Printf("Membership details:\n")
    fmt.Printf("  ID: %s\n", membershipDetails.ID)
    fmt.Printf("  Room ID: %s\n", membershipDetails.RoomID)
    fmt.Printf("  Person ID: %s\n", membershipDetails.PersonID)
    fmt.Printf("  Person Email: %s\n", membershipDetails.PersonEmail)
    fmt.Printf("  Is Moderator: %t\n", membershipDetails.IsModerator)
    fmt.Printf("  Created: %v\n", membershipDetails.Created)

    // Remove person from the room
    err = client.Memberships().Delete(createdMembership.ID)
    if err != nil {
        log.Fatalf("Error deleting membership: %v", err)
    }

    fmt.Printf("Successfully removed %s from the room\n", createdMembership.PersonEmail)
}
```

## Error Handling

All methods return structured errors from the `webexsdk` package. Use the convenience functions to check error types:

```go
membership, err := client.Memberships().Get("MEMBERSHIP_ID")
if err != nil {
    switch {
    case webexsdk.IsNotFound(err):
        log.Println("Membership not found")
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

- [Webex Memberships API Documentation](https://developer.webex.com/docs/api/v1/memberships)

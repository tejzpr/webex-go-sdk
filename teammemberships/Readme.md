# TeamMemberships

The TeamMemberships module provides functionality for interacting with the Webex Team Memberships API. This module allows you to manage team members in Webex Teams, including adding members to teams, retrieving membership details, updating moderator status, and removing members from teams.

## Overview

Team memberships represent a person's membership in a Webex team. This module allows you to:

1. Add people to teams
2. Retrieve membership details
3. List memberships in a team or for a person
4. Update membership properties (moderator status)
5. Remove people from teams

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the teammemberships module:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/teammemberships"
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

// Access the TeamMemberships API
teamMembershipsClient := client.TeamMemberships()
```

### Creating a Team Membership

To add a user to a team:

```go
// Add a user to a team by email
newMembership := &teammemberships.TeamMembership{
    TeamID:      "team-id",
    PersonEmail: "user@example.com",
    IsModerator: false,  // Set to true to make the user a team moderator
}

createdMembership, err := client.TeamMemberships().Create(newMembership)
if err != nil {
    log.Printf("Failed to create team membership: %v", err)
} else {
    fmt.Printf("Added %s to the team with membership ID: %s\n", 
        createdMembership.PersonDisplayName, 
        createdMembership.ID)
}
```

You can add a person to a team using either their email address (`PersonEmail`) or their person ID (`PersonID`). At least one of these fields must be provided.

### Getting a Team Membership

Retrieve details of a specific team membership:

```go
membership, err := client.TeamMemberships().Get("membership-id")
if err != nil {
    log.Printf("Failed to get team membership details: %v", err)
} else {
    fmt.Printf("Membership details for: %s\n", membership.PersonDisplayName)
    fmt.Printf("Team ID: %s\n", membership.TeamID)
    fmt.Printf("Moderator status: %t\n", membership.IsModerator)
}
```

### Listing Team Memberships

List all members in a specific team:

```go
memberships, err := client.TeamMemberships().List(&teammemberships.ListOptions{
    TeamID: "team-id",
    Max:    100,  // Optional: maximum number of results to return
})
if err != nil {
    log.Printf("Failed to list team memberships: %v", err)
} else {
    fmt.Printf("Found %d team members\n", len(memberships.Items))
    for i, membership := range memberships.Items {
        fmt.Printf("%d. %s (%s) - Moderator: %t\n", 
            i+1, 
            membership.PersonDisplayName, 
            membership.PersonEmail, 
            membership.IsModerator)
    }
}
```

### Updating a Team Membership

Update a team member's moderator status:

```go
// Update a user to be a moderator
updatedMembership, err := client.TeamMemberships().Update("membership-id", true)
if err != nil {
    log.Printf("Failed to update team membership: %v", err)
} else {
    fmt.Printf("Updated %s's moderator status to: %t\n", 
        updatedMembership.PersonDisplayName, 
        updatedMembership.IsModerator)
}
```

Note: The only property that can be updated is the `IsModerator` status.

### Deleting a Team Membership

Remove a user from a team:

```go
err = client.TeamMemberships().Delete("membership-id")
if err != nil {
    log.Printf("Failed to delete team membership: %v", err)
} else {
    fmt.Println("Successfully removed user from the team")
}
```

## Data Structures

### TeamMembership Structure

The TeamMembership structure represents a member in a Webex team with the following fields:

```go
type TeamMembership struct {
    ID                string     // Unique identifier of the membership
    TeamID            string     // ID of the team the membership belongs to
    PersonID          string     // ID of the person who is a member
    PersonEmail       string     // Email of the person who is a member
    PersonDisplayName string     // Display name of the person who is a member
    IsModerator       bool       // Whether the person is a moderator of the team
    Created           *time.Time // Time when the membership was created
}
```

### ListOptions

When listing team memberships, you must specify the team ID:

```go
type ListOptions struct {
    TeamID string // Required: ID of the team to list memberships from
    Max    int    // Optional: Maximum number of results to return
}
```

## Complete Example

Here's a complete example demonstrating the major operations with team memberships:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/teammemberships"
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

    // Get a team ID to work with
    teamID := os.Getenv("WEBEX_TEAM_ID")
    if teamID == "" {
        log.Fatalf("WEBEX_TEAM_ID environment variable is required")
    }

    // List team memberships
    fmt.Println("Listing team memberships...")
    memberships, err := client.TeamMemberships().List(&teammemberships.ListOptions{
        TeamID: teamID,
        Max:    100,
    })
    if err != nil {
        log.Fatalf("Failed to list team memberships: %v", err)
    }
    fmt.Printf("Found %d team memberships\n", len(memberships.Items))
    for i, membership := range memberships.Items {
        fmt.Printf("%d. %s (%s) - Moderator: %t\n", 
            i+1, 
            membership.PersonDisplayName, 
            membership.PersonEmail, 
            membership.IsModerator)
    }

    // Get user email to add to the team
    userEmail := os.Getenv("WEBEX_TEST_USER_EMAIL")
    if userEmail == "" {
        log.Printf("WEBEX_TEST_USER_EMAIL environment variable is not set, skipping create example")
        return
    }

    // Create a team membership
    fmt.Println("\nCreating a new team membership...")
    newMembership := &teammemberships.TeamMembership{
        TeamID:      teamID,
        PersonEmail: userEmail,
        IsModerator: false,
    }
    createdMembership, err := client.TeamMemberships().Create(newMembership)
    if err != nil {
        log.Printf("Failed to create team membership: %v\n", err)
        return
    }
    fmt.Printf("Created membership with ID: %s\n", createdMembership.ID)
    fmt.Printf("Person: %s (%s)\n", createdMembership.PersonDisplayName, createdMembership.PersonEmail)
    
    // Get membership details
    fmt.Println("\nFetching team membership details...")
    membershipDetails, err := client.TeamMemberships().Get(createdMembership.ID)
    if err != nil {
        log.Printf("Failed to get team membership details: %v\n", err)
        return
    }
    fmt.Printf("Membership Details: %s is member of team %s\n", 
        membershipDetails.PersonDisplayName, 
        membershipDetails.TeamID)
    
    // Update membership to make person a moderator
    fmt.Println("\nUpdating team membership...")
    updatedMembership, err := client.TeamMemberships().Update(membershipDetails.ID, true)
    if err != nil {
        log.Printf("Failed to update team membership: %v\n", err)
        return
    }
    fmt.Printf("Updated membership moderator status: %t\n", updatedMembership.IsModerator)
    
    // Delete the membership
    fmt.Println("\nDeleting team membership...")
    err = client.TeamMemberships().Delete(updatedMembership.ID)
    if err != nil {
        log.Printf("Failed to delete team membership: %v\n", err)
        return
    }
    fmt.Printf("Successfully deleted team membership\n")
}
```

## Related Resources

- [Webex Team Memberships API Documentation](https://developer.webex.com/docs/api/v1/team-memberships)
- [Webex Teams API Documentation](https://developer.webex.com/docs/api/v1/teams)

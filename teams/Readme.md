# Teams

The Teams module provides functionality for interacting with the Webex Teams API. This module allows you to manage Webex Teams, including creating, retrieving, updating, and deleting teams.

## Overview

Webex Teams are virtual spaces that help organize your collaboration. This module allows you to:

1. Create new teams
2. Retrieve team details
3. List all teams you belong to
4. Update existing teams
5. Delete teams

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the teams module:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/teams"
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

// Access the Teams API
teamsClient := client.Teams()
```

### Creating a Team

To create a new Webex team:

```go
newTeam := &teams.Team{
    Name:        "My New Team",
    Description: "This is a team created using the Go SDK",
}

createdTeam, err := client.Teams().Create(newTeam)
if err != nil {
    log.Printf("Failed to create team: %v", err)
} else {
    fmt.Printf("Created team with ID: %s\n", createdTeam.ID)
    fmt.Printf("Team Name: %s\n", createdTeam.Name)
}
```

The `Name` field is required for creating a team. The `Description` field is optional.

### Getting a Team

Retrieve details of a specific team:

```go
team, err := client.Teams().Get("team-id")
if err != nil {
    log.Printf("Failed to get team details: %v", err)
} else {
    fmt.Printf("Team Name: %s\n", team.Name)
    fmt.Printf("Team Description: %s\n", team.Description)
    fmt.Printf("Creator ID: %s\n", team.CreatorID)
}
```

### Listing Teams

List all teams that you belong to:

```go
teams, err := client.Teams().List(&teams.ListOptions{
    Max: 100,  // Optional: maximum number of results to return
})
if err != nil {
    log.Printf("Failed to list teams: %v", err)
} else {
    fmt.Printf("Found %d teams\n", len(teams.Items))
    for i, team := range teams.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, team.Name, team.ID)
    }
}
```

### Updating a Team

Update an existing team's properties:

```go
updatedTeam := &teams.Team{
    Name:        "Updated Team Name",
    Description: "This is an updated description",
}

result, err := client.Teams().Update("team-id", updatedTeam)
if err != nil {
    log.Printf("Failed to update team: %v", err)
} else {
    fmt.Printf("Updated team name to: %s\n", result.Name)
    fmt.Printf("Updated team description to: %s\n", result.Description)
}
```

The `Name` field is required when updating a team. The `Description` field is optional.

### Deleting a Team

Delete a team:

```go
err = client.Teams().Delete("team-id")
if err != nil {
    log.Printf("Failed to delete team: %v", err)
} else {
    fmt.Println("Successfully deleted team")
}
```

## Data Structures

### Team Structure

The Team structure represents a Webex team with the following fields:

```go
type Team struct {
    ID          string     // Unique identifier of the team
    Name        string     // Name of the team
    Description string     // Description of the team
    CreatorID   string     // ID of the person who created the team
    Created     *time.Time // Time when the team was created
}
```

### ListOptions

When listing teams, you can specify the maximum number of results to return:

```go
type ListOptions struct {
    Max int // Optional: Maximum number of results to return
}
```

## Complete Example

Here's a complete example demonstrating the major operations with teams:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/teams"
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

    // List teams
    fmt.Println("Listing teams...")
    teamsPage, err := client.Teams().List(&teams.ListOptions{
        Max: 100,
    })
    if err != nil {
        log.Fatalf("Failed to list teams: %v", err)
    }
    fmt.Printf("Found %d teams\n", len(teamsPage.Items))
    for i, team := range teamsPage.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, team.Name, team.ID)
        if team.Description != "" {
            fmt.Printf("   Description: %s\n", team.Description)
        }
    }

    // Create a team
    fmt.Println("\nCreating a new team...")
    newTeam := &teams.Team{
        Name:        "API Test Team",
        Description: "Team created via Go SDK example",
    }
    createdTeam, err := client.Teams().Create(newTeam)
    if err != nil {
        log.Printf("Failed to create team: %v\n", err)
        return
    }
    fmt.Printf("Created team with ID: %s\n", createdTeam.ID)
    fmt.Printf("Team Name: %s\n", createdTeam.Name)
    
    // Get team details
    fmt.Println("\nFetching team details...")
    teamDetails, err := client.Teams().Get(createdTeam.ID)
    if err != nil {
        log.Printf("Failed to get team details: %v\n", err)
        return
    }
    fmt.Printf("Team Details: %s\n", teamDetails.Name)
    
    // Update the team
    fmt.Println("\nUpdating team...")
    updatedTeamData := &teams.Team{
        Name:        "Updated API Test Team",
        Description: "Updated team description via Go SDK example",
    }
    updatedTeam, err := client.Teams().Update(teamDetails.ID, updatedTeamData)
    if err != nil {
        log.Printf("Failed to update team: %v\n", err)
        return
    }
    fmt.Printf("Updated team name to: %s\n", updatedTeam.Name)
    
    // Delete the team
    fmt.Println("\nDeleting team...")
    err = client.Teams().Delete(updatedTeam.ID)
    if err != nil {
        log.Printf("Failed to delete team: %v\n", err)
        return
    }
    fmt.Printf("Successfully deleted team\n")
}
```

## Related Resources

- [Webex Teams API Documentation](https://developer.webex.com/docs/api/v1/teams)
- [Webex Team Memberships API Documentation](https://developer.webex.com/docs/api/v1/team-memberships)

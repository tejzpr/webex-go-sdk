# People

The People module provides functionality to interact with the Webex People API, allowing you to retrieve information about users in your Webex organization.

## Overview

The People API in Webex provides access to user details and profiles. This module allows you to:

1. Retrieve your own user information
2. Get details about other users by ID
3. Search for users by email or display name
4. Efficiently retrieve multiple users at once with batch requests

The People module includes a sophisticated batching system that optimizes multiple user lookups by grouping them into fewer API calls, which can help with rate limits and overall performance.

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the people package:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/people"
)
```

## Usage

### Initializing the Client

```go
client, err := webex.NewClient(accessToken, nil)
if err != nil {
    log.Fatalf("Error creating client: %v", err)
}

// Access the people client
peopleClient := client.People()
```

### Getting Your Own User Information

To retrieve information about the currently authenticated user:

```go
me, err := client.People().Get("me")
// Or use the specialized method
me, err := client.People().GetMe()

if err != nil {
    log.Fatalf("Error getting current user: %v", err)
}

fmt.Printf("Current user: %s (%s)\n", me.DisplayName, me.Emails[0])
```

### Getting a Person by ID

To retrieve information about a specific user by their ID:

```go
person, err := client.People().Get("PERSON_ID")
if err != nil {
    log.Fatalf("Error getting person: %v", err)
}

fmt.Printf("Person: %s (%s)\n", person.DisplayName, person.Emails[0])
```

### Searching for People

To search for people by email:

```go
options := &people.ListOptions{
    Email: "user@example.com", // Can be a partial match
}
page, err := client.People().List(options)
if err != nil {
    log.Fatalf("Error listing people: %v", err)
}

for _, person := range page.Items {
    fmt.Printf("Found: %s (%s)\n", person.DisplayName, person.Emails[0])
}
```

To search for people by display name:

```go
options := &people.ListOptions{
    DisplayName: "John", // Can be a partial match
}
page, err := client.People().List(options)
```

### Batch Retrieval of Multiple People

To efficiently retrieve multiple people at once:

```go
options := &people.ListOptions{
    IDs: []string{"PERSON_ID_1", "PERSON_ID_2", "PERSON_ID_3"},
}
page, err := client.People().List(options)
if err != nil {
    log.Fatalf("Error listing people: %v", err)
}

for _, person := range page.Items {
    fmt.Printf("Person: %s (%s)\n", person.DisplayName, person.Emails[0])
}
```

The module automatically batches these requests for optimal performance.

## Data Structures

### Person Structure

The `Person` structure contains the following fields:

| Field       | Type       | Description                                           |
|-------------|------------|-------------------------------------------------------|
| ID          | string     | Unique identifier for the person                      |
| Emails      | []string   | Array of email addresses associated with the person   |
| DisplayName | string     | Full name of the person                               |
| NickName    | string     | Nickname or preferred name of the person              |
| FirstName   | string     | First name of the person                              |
| LastName    | string     | Last name of the person                               |
| Avatar      | string     | URL of the person's avatar image                      |
| OrgID       | string     | Organization ID the person belongs to                 |
| Roles       | []string   | Array of roles assigned to the person                 |
| Licenses    | []string   | Array of licenses assigned to the person              |
| Created     | time.Time  | Timestamp when the person was created                 |
| Status      | string     | Current status of the person (active, inactive, etc.) |
| Type        | string     | Type of account (usually "person")                    |

### ListOptions

When listing or searching for people, you can use the following filter options:

| Option       | Type      | Description                                           |
|--------------|-----------|-------------------------------------------------------|
| Email        | string    | Filter by email address (partial match supported)     |
| DisplayName  | string    | Filter by display name (partial match supported)      |
| IDs          | []string  | Get multiple people by their IDs (batch operation)    |
| Max          | int       | Maximum number of results to return                   |
| ShowAllTypes | bool      | Include non-person types in results                   |

### Configuration Options

The People module supports the following configuration options:

| Option        | Description                                                         | Default               |
|---------------|---------------------------------------------------------------------|----------------------|
| BatcherWait   | Time to wait before processing a batch request                      | 100 milliseconds     |
| MaxBatchCalls | Maximum number of batch calls to make at once                       | 10                   |
| MaxBatchWait  | Maximum time to wait before processing a batch request              | 1500 milliseconds    |
| ShowAllTypes  | Include non-person types (e.g., SX10, webhook_integration, etc.)    | false                |

You can customize these settings when creating the client:

```go
config := &people.Config{
    BatcherWait:   200 * time.Millisecond,
    MaxBatchCalls: 20,
    ShowAllTypes:  true,
}

client, err := webex.NewClient(accessToken, nil)
peopleClient := people.New(client, config)
```

## Complete Example

Here's a complete example that demonstrates the major features of the People module:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/people"
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

    // Get current user information
    me, err := client.People().GetMe()
    if err != nil {
        log.Fatalf("Error getting current user: %v", err)
    }
    
    fmt.Printf("Current user: %s (%s)\n", me.DisplayName, me.Emails[0])
    fmt.Printf("  ID: %s\n", me.ID)
    fmt.Printf("  Name: %s %s\n", me.FirstName, me.LastName)
    fmt.Printf("  Organization: %s\n", me.OrgID)
    
    if len(me.Roles) > 0 {
        fmt.Printf("  Roles: %v\n", me.Roles)
    }
    
    // Search for people with similar email domain
    emailDomain := me.Emails[0][me.Emails[0].IndexByte('@'):]
    fmt.Printf("\nSearching for people with email domain %s...\n", emailDomain)
    
    options := &people.ListOptions{
        Email: emailDomain,
        Max:   10,
    }
    
    page, err := client.People().List(options)
    if err != nil {
        log.Fatalf("Error listing people: %v", err)
    }
    
    fmt.Printf("Found %d people:\n", len(page.Items))
    
    // Collect IDs for batch retrieval demonstration
    var ids []string
    for i, person := range page.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, person.DisplayName, person.Emails[0])
        ids = append(ids, person.ID)
    }
    
    // Demonstrate batch retrieval
    if len(ids) > 1 {
        fmt.Printf("\nRetrieving %d people at once using batch API...\n", len(ids))
        
        batchOptions := &people.ListOptions{
            IDs: ids,
        }
        
        batchPage, err := client.People().List(batchOptions)
        if err != nil {
            log.Fatalf("Error retrieving batch of people: %v", err)
        }
        
        fmt.Printf("Successfully retrieved %d people in a single batch request\n", 
            len(batchPage.Items))
    }
}
```

## Related Resources

- [Webex People API Documentation](https://developer.webex.com/docs/api/v1/people)

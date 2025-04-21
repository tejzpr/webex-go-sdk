# Webhooks

The Webhooks module provides functionality for interacting with the Webex Webhooks API. Webhooks allow your application to be notified in real-time about events happening in Webex, such as messages being created or received, room memberships changing, or meetings being scheduled.

## Overview

Webex Webhooks provide a way to receive notifications about events in Webex. This module allows you to:

1. Create webhooks to listen for specific events
2. Retrieve webhook details
3. List all webhooks
4. Update existing webhooks
5. Delete webhooks
6. Process incoming webhook notifications

## Installation

This module is part of the Webex Go SDK. To use it, import the SDK and the webhooks module:

```go
import (
    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/webhooks"
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

// Access the Webhooks API
webhooksClient := client.Webhooks()
```

### Creating a Webhook

To create a new webhook that will notify your application when events occur:

```go
newWebhook := &webhooks.Webhook{
    Name:      "My Message Webhook",
    TargetURL: "https://example.com/webhook-receiver",
    Resource:  "messages",
    Event:     "created",
    Filter:    "roomId=YOUR_ROOM_ID", // Optional filter
    Secret:    "mySecretToValidateRequests", // Optional secret
}

createdWebhook, err := client.Webhooks().Create(newWebhook)
if err != nil {
    log.Printf("Failed to create webhook: %v", err)
} else {
    fmt.Printf("Created webhook with ID: %s\n", createdWebhook.ID)
}
```

Required fields:
- `Name`: A user-friendly name for the webhook
- `TargetURL`: The URL that will receive HTTP POST requests for events
- `Resource`: The resource to monitor (e.g., "messages", "memberships", "rooms", "meetings")
- `Event`: The event to monitor (e.g., "created", "updated", "deleted")

Optional fields:
- `Filter`: A filter to apply (e.g., "roomId=123")
- `Secret`: A secret used to compute the signature in the `X-Webex-Signature` header

### Getting a Webhook

Retrieve details of a specific webhook:

```go
webhook, err := client.Webhooks().Get("webhook-id")
if err != nil {
    log.Printf("Failed to get webhook details: %v", err)
} else {
    fmt.Printf("Webhook Name: %s\n", webhook.Name)
    fmt.Printf("Target URL: %s\n", webhook.TargetURL)
    fmt.Printf("Resource: %s\n", webhook.Resource)
    fmt.Printf("Event: %s\n", webhook.Event)
    fmt.Printf("Status: %s\n", webhook.Status)
}
```

### Listing Webhooks

List all webhooks registered for your application:

```go
webhooks, err := client.Webhooks().List(&webhooks.ListOptions{
    Max: 100,  // Optional: maximum number of results to return
})
if err != nil {
    log.Printf("Failed to list webhooks: %v", err)
} else {
    fmt.Printf("Found %d webhooks\n", len(webhooks.Items))
    for i, webhook := range webhooks.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, webhook.Name, webhook.ID)
        fmt.Printf("   Status: %s\n", webhook.Status)
    }
}
```

### Updating a Webhook

Update an existing webhook's properties:

```go
// Create a webhook with only the fields that can be updated
updatedWebhook := webhooks.NewUpdateWebhook(
    "Updated Webhook Name",
    "https://example.com/new-webhook-url",
    "newSecret", // Leave empty if not changing
    "active",    // Can be "active" or "inactive"
)

result, err := client.Webhooks().Update("webhook-id", updatedWebhook)
if err != nil {
    log.Printf("Failed to update webhook: %v", err)
} else {
    fmt.Printf("Updated webhook name to: %s\n", result.Name)
    fmt.Printf("Status: %s\n", result.Status)
}
```

Only the following fields can be updated:
- `Name`: The webhook name
- `TargetURL`: The URL that will receive HTTP POST requests
- `Secret`: The secret used for computing the signature
- `Status`: Either "active" or "inactive"

### Deleting a Webhook

Delete a webhook:

```go
err = client.Webhooks().Delete("webhook-id")
if err != nil {
    log.Printf("Failed to delete webhook: %v", err)
} else {
    fmt.Println("Successfully deleted webhook")
}
```

## Data Structures

### Webhook Structure

The Webhook structure represents a Webex webhook with the following fields:

```go
type Webhook struct {
    ID        string     // Unique identifier of the webhook
    Name      string     // User-friendly name for the webhook
    TargetURL string     // URL that receives HTTP POST requests
    Resource  string     // Resource being monitored (messages, memberships, etc.)
    Event     string     // Event being monitored (created, updated, deleted)
    Filter    string     // Optional filter (e.g., roomId=123)
    Secret    string     // Secret used to compute the X-Webex-Signature header
    Status    string     // Status of the webhook (active or inactive)
    Created   *time.Time // Time when the webhook was created
}
```

### ListOptions

When listing webhooks, you can specify the maximum number of results to return:

```go
type ListOptions struct {
    Max int // Optional: Maximum number of results to return
}
```

## Handling Webhook Notifications

When an event occurs that matches a webhook's criteria, Webex will send an HTTP POST request to the webhook's target URL. Here's an example of how to handle these notifications in a web server:

```go
package main

import (
    "crypto/hmac"
    "crypto/sha1"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
)

// WebhookData represents the data sent in a webhook notification
type WebhookData struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Resource  string `json:"resource"`
    Event     string `json:"event"`
    Filter    string `json:"filter"`
    OrgID     string `json:"orgId"`
    CreatedBy string `json:"createdBy"`
    AppID     string `json:"appId"`
    OwnerID   string `json:"ownerId"`
    Status    string `json:"status"`
    ActorID   string `json:"actorId"`
    Data      struct {
        ID              string   `json:"id"`
        RoomID          string   `json:"roomId"`
        PersonID        string   `json:"personId"`
        PersonEmail     string   `json:"personEmail"`
        Created         string   `json:"created"`
        // Additional fields vary based on the resource type
    } `json:"data"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    // Read the request body
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }
    
    // Verify the signature if a secret was set
    secret := "mySecretToValidateRequests"
    signature := r.Header.Get("X-Webex-Signature")
    if secret != "" && signature != "" {
        mac := hmac.New(sha1.New, []byte(secret))
        mac.Write(body)
        expectedSignature := hex.EncodeToString(mac.Sum(nil))
        
        if signature != expectedSignature {
            http.Error(w, "Invalid signature", http.StatusUnauthorized)
            return
        }
    }
    
    // Parse the webhook data
    var webhookData WebhookData
    if err := json.Unmarshal(body, &webhookData); err != nil {
        http.Error(w, "Error parsing webhook data", http.StatusBadRequest)
        return
    }
    
    // Process the webhook based on resource and event
    fmt.Printf("Received webhook: %s - %s\n", webhookData.Resource, webhookData.Event)
    
    // Handle specific resource types
    switch webhookData.Resource {
    case "messages":
        if webhookData.Event == "created" {
            fmt.Printf("New message from %s in room %s\n", 
                webhookData.Data.PersonEmail, 
                webhookData.Data.RoomID)
        }
    case "memberships":
        fmt.Printf("Membership event: %s\n", webhookData.Event)
    // Handle other resource types
    }
    
    // Return a 200 OK to acknowledge receipt
    w.WriteHeader(http.StatusOK)
}

func main() {
    http.HandleFunc("/webhook-receiver", handleWebhook)
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Complete Example

Here's a complete example demonstrating the major operations with webhooks:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tejzpr/webex-go-sdk/v2"
    "github.com/tejzpr/webex-go-sdk/v2/webhooks"
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

    // List existing webhooks
    fmt.Println("Listing webhooks...")
    webhooksPage, err := client.Webhooks().List(&webhooks.ListOptions{
        Max: 100,
    })
    if err != nil {
        log.Fatalf("Failed to list webhooks: %v", err)
    }
    fmt.Printf("Found %d webhooks\n", len(webhooksPage.Items))
    for i, webhook := range webhooksPage.Items {
        fmt.Printf("%d. %s (%s)\n", i+1, webhook.Name, webhook.ID)
        fmt.Printf("   Resource: %s, Event: %s\n", webhook.Resource, webhook.Event)
    }

    // Create a new webhook
    fmt.Println("\nCreating a new webhook...")
    newWebhook := &webhooks.Webhook{
        Name:      "Messages Webhook",
        TargetURL: "https://example.com/webhook",
        Resource:  "messages",
        Event:     "created",
        Secret:    "myWebhookSecret",
    }

    createdWebhook, err := client.Webhooks().Create(newWebhook)
    if err != nil {
        log.Printf("Failed to create webhook: %v\n", err)
        return
    }
    fmt.Printf("Created webhook with ID: %s\n", createdWebhook.ID)
    
    // Get webhook details
    fmt.Println("\nFetching webhook details...")
    webhookDetails, err := client.Webhooks().Get(createdWebhook.ID)
    if err != nil {
        log.Printf("Failed to get webhook details: %v\n", err)
        return
    }
    fmt.Printf("Webhook Details: %s is monitoring %s %s events\n", 
        webhookDetails.Name, 
        webhookDetails.Resource,
        webhookDetails.Event)
    
    // Update the webhook
    fmt.Println("\nUpdating webhook...")
    updatedWebhook := webhooks.NewUpdateWebhook(
        "Updated Messages Webhook",
        webhookDetails.TargetURL,
        webhookDetails.Secret,
        "active",
    )
    updatedResult, err := client.Webhooks().Update(webhookDetails.ID, updatedWebhook)
    if err != nil {
        log.Printf("Failed to update webhook: %v\n", err)
        return
    }
    fmt.Printf("Updated webhook name to: %s\n", updatedResult.Name)
    
    // Delete the webhook
    fmt.Println("\nDeleting webhook...")
    err = client.Webhooks().Delete(updatedResult.ID)
    if err != nil {
        log.Printf("Failed to delete webhook: %v\n", err)
        return
    }
    fmt.Printf("Successfully deleted webhook\n")
}
```

## Related Resources

- [Webex Webhooks API Documentation](https://developer.webex.com/docs/api/v1/webhooks)

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

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

	// Example 1: List Events with filters
	fmt.Println("Listing events...")

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

	// Example 2: Get Event Details
	if len(eventsPage.Items) > 0 {
		fmt.Println("\nGetting details for the first event...")
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

			// Print data fields based on resource type
			fmt.Printf("  Data:\n")
			if eventDetails.Resource == "messages" {
				fmt.Printf("    Room ID: %s\n", eventDetails.Data.RoomID)
				fmt.Printf("    Room Type: %s\n", eventDetails.Data.RoomType)
				fmt.Printf("    Person ID: %s\n", eventDetails.Data.PersonID)
				fmt.Printf("    Person Email: %s\n", eventDetails.Data.PersonEmail)
			} else if eventDetails.Resource == "meetings" {
				fmt.Printf("    Meeting ID: %s\n", eventDetails.Data.MeetingID)
				fmt.Printf("    Creator ID: %s\n", eventDetails.Data.CreatorID)
				fmt.Printf("    Recording Enabled: %s\n", eventDetails.Data.RecordingEnabled)
			}
		}
	}

	fmt.Println("\nNote: The Events API is only available to Compliance Officers with the appropriate permissions.")
}

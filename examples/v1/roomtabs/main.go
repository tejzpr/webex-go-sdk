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

	"github.com/tejzpr/webex-go-sdk/v1"
	"github.com/tejzpr/webex-go-sdk/v1/roomtabs"
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

	// Example 1: List Room Tabs
	fmt.Println("Listing room tabs...")
	tabs, err := client.RoomTabs().List(&roomtabs.ListOptions{
		RoomID: roomID,
	})
	if err != nil {
		log.Fatalf("Failed to list room tabs: %v", err)
	}
	fmt.Printf("Found %d room tabs\n", len(tabs.Items))
	for i, tab := range tabs.Items {
		fmt.Printf("%d. %s (%s)\n", i+1, tab.DisplayName, tab.ID)
	}

	// Example 2: Create a Room Tab
	fmt.Println("\nCreating a new room tab...")
	newTab := &roomtabs.RoomTab{
		RoomID:      roomID,
		DisplayName: "API Demo Tab",
		ContentURL:  "https://example.com/demo",
	}
	createdTab, err := client.RoomTabs().Create(newTab)
	if err != nil {
		log.Printf("Failed to create room tab: %v\n", err)
	} else {
		fmt.Printf("Created tab with ID: %s\n", createdTab.ID)

		// Example 3: Get Room Tab Details
		fmt.Println("\nFetching room tab details...")
		tabDetails, err := client.RoomTabs().Get(createdTab.ID)
		if err != nil {
			log.Printf("Failed to get room tab details: %v\n", err)
		} else {
			fmt.Printf("Tab Details:\n")
			fmt.Printf("  ID: %s\n", tabDetails.ID)
			fmt.Printf("  Display Name: %s\n", tabDetails.DisplayName)
			fmt.Printf("  Content URL: %s\n", tabDetails.ContentURL)
			fmt.Printf("  Room Type: %s\n", tabDetails.RoomType)
			if tabDetails.Created != nil {
				fmt.Printf("  Created: %s\n", tabDetails.Created.Format("2006-01-02 15:04:05"))
			}

			// Example 4: Update Room Tab
			fmt.Println("\nUpdating room tab...")
			tabDetails.DisplayName = "Updated API Demo Tab"
			tabDetails.ContentURL = "https://example.com/updated-demo"
			updatedTab, err := client.RoomTabs().Update(tabDetails.ID, tabDetails)
			if err != nil {
				log.Printf("Failed to update room tab: %v\n", err)
			} else {
				fmt.Printf("Updated tab display name to: %s\n", updatedTab.DisplayName)
				fmt.Printf("Updated tab content URL to: %s\n", updatedTab.ContentURL)

				// Example 5: Delete Room Tab
				fmt.Println("\nDeleting room tab...")
				err = client.RoomTabs().Delete(updatedTab.ID)
				if err != nil {
					log.Printf("Failed to delete room tab: %v\n", err)
				} else {
					fmt.Printf("Successfully deleted room tab with ID: %s\n", updatedTab.ID)
				}
			}
		}
	}
}

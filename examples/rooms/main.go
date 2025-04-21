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

	"github.com/tejzpr/webex-go-sdk/v2"
	"github.com/tejzpr/webex-go-sdk/v2/rooms"
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

	// Example 1: Create a room
	fmt.Println("Creating a new room...")
	room := &rooms.Room{
		Title: "Go SDK Room Example",
	}

	createdRoom, err := client.Rooms().Create(room)
	if err != nil {
		log.Fatalf("Error creating room: %v", err)
	}
	fmt.Printf("Room created: ID=%s, Title=%s\n", createdRoom.ID, createdRoom.Title)

	// Example 2: Get room details
	fmt.Printf("\nGetting details for room %s...\n", createdRoom.ID)
	roomDetails, err := client.Rooms().Get(createdRoom.ID)
	if err != nil {
		log.Fatalf("Error getting room: %v", err)
	}
	fmt.Printf("Room details:\n")
	fmt.Printf("  ID: %s\n", roomDetails.ID)
	fmt.Printf("  Title: %s\n", roomDetails.Title)
	fmt.Printf("  Type: %s\n", roomDetails.Type)
	fmt.Printf("  IsLocked: %t\n", roomDetails.IsLocked)
	fmt.Printf("  CreatorID: %s\n", roomDetails.CreatorID)
	fmt.Printf("  Created: %v\n", roomDetails.Created)

	// Example 3: List rooms
	fmt.Println("\nListing rooms...")
	options := &rooms.ListOptions{
		Max: 10,
	}

	page, err := client.Rooms().List(options)
	if err != nil {
		log.Fatalf("Error listing rooms: %v", err)
	}

	fmt.Printf("Found %d rooms:\n", len(page.Items))
	for i, r := range page.Items {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, r.Title, r.ID)
	}

	// Example 4: Update a room
	fmt.Println("\nUpdating the room title...")
	updateRoom := &rooms.Room{
		Title: "Go SDK Room Example (Updated)",
	}

	updatedRoom, err := client.Rooms().Update(createdRoom.ID, updateRoom)
	if err != nil {
		log.Fatalf("Error updating room: %v", err)
	}
	fmt.Printf("Room updated: ID=%s, New Title=%s\n", updatedRoom.ID, updatedRoom.Title)

	// Example 5: Delete a room
	fmt.Printf("\nDeleting room %s...\n", createdRoom.ID)
	err = client.Rooms().Delete(createdRoom.ID)
	if err != nil {
		log.Fatalf("Error deleting room: %v", err)
	}
	fmt.Println("Room deleted successfully")

	// Verify deletion
	_, err = client.Rooms().Get(createdRoom.ID)
	if err != nil {
		fmt.Printf("Confirmed: Room no longer exists (Error: %v)\n", err)
	} else {
		fmt.Println("Warning: Room still exists after deletion attempt")
	}
}

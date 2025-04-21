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
	"github.com/tejzpr/webex-go-sdk/v1/messages"
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

	// Example 1: Send a message to a room
	roomID := os.Getenv("WEBEX_ROOM_ID")
	if roomID == "" {
		log.Fatal("WEBEX_ROOM_ID environment variable is required")
	}

	fmt.Println("Sending message to room...")
	message := &messages.Message{
		RoomID: roomID,
		Text:   "Hello, World!",
	}

	createdMessage, err := client.Messages().Create(message)
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}
	fmt.Printf("Message sent: ID=%s\n", createdMessage.ID)

	// Example 2: Get a message by ID
	fmt.Printf("\nGetting message with ID %s...\n", createdMessage.ID)
	fetchedMessage, err := client.Messages().Get(createdMessage.ID)
	if err != nil {
		log.Fatalf("Error getting message: %v", err)
	}
	fmt.Printf("Message: ID=%s, Text=%s, Created=%v\n",
		fetchedMessage.ID,
		fetchedMessage.Text,
		fetchedMessage.Created)

	// Example 3: List messages in a room
	fmt.Printf("\nListing messages in room %s...\n", roomID)
	options := &messages.ListOptions{
		RoomID: roomID,
		Max:    10,
	}
	page, err := client.Messages().List(options)
	if err != nil {
		log.Fatalf("Error listing messages: %v", err)
	}

	fmt.Printf("Found %d messages:\n", len(page.Items))
	for i, msg := range page.Items {
		fmt.Printf("%d. %s (%s): %s\n", i+1, msg.PersonEmail, msg.Created.Format("2006-01-02 15:04:05"), msg.Text)
	}

	// Example 4: Update a message
	fmt.Printf("\nUpdating message with ID %s...\n", createdMessage.ID)
	updateMessage := &messages.Message{
		RoomID: roomID,
		Text:   "Updated: Hello, Go SDK!",
	}
	updatedMessage, err := client.Messages().Update(createdMessage.ID, updateMessage)
	if err != nil {
		log.Fatalf("Error updating message: %v", err)
	}
	fmt.Printf("Message updated: ID=%s, Text=%s\n", updatedMessage.ID, updatedMessage.Text)

	// Example 5: Delete a message
	fmt.Printf("\nDeleting message with ID %s...\n", createdMessage.ID)
	err = client.Messages().Delete(createdMessage.ID)
	if err != nil {
		log.Fatalf("Error deleting message: %v", err)
	}
	fmt.Println("Message deleted successfully")
}

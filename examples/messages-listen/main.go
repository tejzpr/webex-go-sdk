/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v1"
	"github.com/WebexCommunity/webex-go-sdk/v1/messages"
)

func main() {
	// Get access token from environment
	accessToken := os.Getenv("WEBEX_ACCESS_TOKEN")
	if accessToken == "" {
		fmt.Println("WEBEX_ACCESS_TOKEN environment variable is required")
		os.Exit(1)
	}

	// Create client
	client, err := webex.NewClient(accessToken, nil)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Get the Messages client
	messagesClient := client.Messages()

	// Define our message handler function
	messageHandler := func(message *messages.Message) {
		// Print information about the received message
		fmt.Printf("\n=== New Message Event ===\n")
		fmt.Printf("Message ID: %s\n", message.ID)
		fmt.Printf("Room ID: %s\n", message.RoomID)
		fmt.Printf("Person ID: %s\n", message.PersonID)
		fmt.Printf("Person Email: %s\n", message.PersonEmail)
		if message.Created != nil {
			fmt.Printf("Created: %s\n", message.Created.Format(time.RFC3339))
		}
		fmt.Printf("Text: %s\n", message.Text)
		fmt.Printf("========================\n\n")
	}

	// Start listening for messages using the simpler API
	fmt.Println("=== STARTING MESSAGE LISTENER ===")
	err = messagesClient.Listen(messageHandler)
	if err != nil {
		fmt.Printf("Error starting message listener: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Listening for messages. Send a message in a space where the bot is a member.")
	fmt.Println("Press Ctrl+C to exit.")

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigCh

	// Clean up
	fmt.Println("Stopping message listener...")
	messagesClient.StopListening()

	fmt.Println("Exiting.")
}

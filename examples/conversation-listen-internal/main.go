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

	"github.com/tejzpr/webex-go-sdk/v2"
	"github.com/tejzpr/webex-go-sdk/v2/conversation"
)

func main() {
	// Create a new Webex client
	token := os.Getenv("WEBEX_ACCESS_TOKEN")
	if token == "" {
		fmt.Println("WEBEX_ACCESS_TOKEN environment variable is required")
		os.Exit(1)
	}

	// Create Webex client with token
	client, err := webex.NewClient(token, nil)
	if err != nil {
		fmt.Printf("Error creating Webex client: %v\n", err)
		os.Exit(1)
	}

	// Get a fully-wired conversation client (handles device registration,
	// Mercury WebSocket setup, and encryption/KMS authentication automatically)
	fmt.Println("Initializing conversation client...")
	conversationClient, err := client.Conversation()
	if err != nil {
		fmt.Printf("Error initializing conversation client: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Conversation client ready!")

	// Register handlers for different activity types
	conversationClient.On("post", func(activity *conversation.Activity) {
		fmt.Printf("\nReceived POST message from %s:\n", activity.Actor.DisplayName)

		// Display encryption information
		if activity.EncryptionKeyURL != "" {
			fmt.Printf("  Encrypted with key: %s\n", activity.EncryptionKeyURL)
		}

		// Get decrypted message content
		content, err := conversationClient.GetMessageContent(activity)
		if err != nil {
			fmt.Printf("  Error decrypting: %v\n", err)
			// Show raw content as fallback
			if activity.DecryptedObject != nil && activity.DecryptedObject.DisplayName != "" {
				fmt.Printf("  Raw (encrypted): %s\n", truncateString(activity.DecryptedObject.DisplayName, 80))
			}
		} else {
			fmt.Printf("  Content: %s\n", content)
		}

		fmt.Printf("  Room ID: %s\n", activity.Target.ID)
		fmt.Println("----------------------------")
	})

	conversationClient.On("share", func(activity *conversation.Activity) {
		fmt.Printf("\nReceived SHARE message from %s:\n", activity.Actor.DisplayName)

		if activity.EncryptionKeyURL != "" {
			fmt.Printf("  Encrypted with key: %s\n", activity.EncryptionKeyURL)
		}

		content, err := conversationClient.GetMessageContent(activity)
		if err != nil {
			fmt.Printf("  Error decrypting: %v\n", err)
		} else {
			fmt.Printf("  Content: %s\n", content)
		}

		fmt.Printf("  Room ID: %s\n", activity.Target.ID)
		fmt.Println("----------------------------")
	})

	conversationClient.On("acknowledge", func(activity *conversation.Activity) {
		fmt.Printf("Received ACKNOWLEDGE from %s\n", activity.Actor.DisplayName)
		fmt.Printf("Acknowledged message ID: %s\n", activity.Object["id"])
		fmt.Println("----------------------------")
	})

	// Connect to the Mercury service (WebSocket)
	fmt.Println("Connecting to WebSocket...")
	err = conversationClient.Connect()
	if err != nil {
		fmt.Printf("Error connecting to WebSocket: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to WebSocket!")
	fmt.Println("Listening for conversation events. Press Ctrl+C to exit.")
	fmt.Println("Messages will be automatically decrypted using the KMS service.")

	// Wait for Ctrl+C to exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Disconnect
	fmt.Println("\nDisconnecting from WebSocket...")
	conversationClient.Disconnect()
	fmt.Println("Disconnected!")
}

// Helper function to truncate long strings
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

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

	"github.com/WebexCommunity/webex-go-sdk/v1"
	"github.com/WebexCommunity/webex-go-sdk/v1/conversation"
	"github.com/WebexCommunity/webex-go-sdk/v1/device"
	"github.com/WebexCommunity/webex-go-sdk/v1/mercury"
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

	// Create device client for WebSocket URL
	deviceClient := device.New(client.Core(), nil)

	// Create Mercury client for WebSocket connection
	mercuryClient := mercury.New(client.Core(), nil)
	mercuryClient.SetDeviceProvider(deviceClient)

	// Create conversation client
	conversationClient := conversation.New(client.Core(), nil)
	conversationClient.SetMercuryClient(mercuryClient)

	// Register handlers for different activity types
	conversationClient.On("post", func(activity *conversation.Activity) {
		fmt.Printf("Received POST message from %s: \n", activity.Actor.DisplayName)

		// Display encryption information
		if activity.EncryptionKeyURL != "" {
			fmt.Printf("Message is encrypted with key: %s\n", activity.EncryptionKeyURL)
		} else {
			fmt.Printf("Message is not encrypted\n")
		}

		// Get raw content before decryption
		var rawContent string
		if activity.DecryptedObject != nil {
			rawContent = activity.DecryptedObject.DisplayName
		} else if activity.Object != nil {
			if c, ok := activity.Object["displayName"].(string); ok {
				rawContent = c
			}
		}

		if rawContent != "" {
			fmt.Printf("Raw encrypted content: %s...\n", truncateString(rawContent, 30))
		}

		// Get decrypted message content
		// TODO: Implement message decryption
		// content, err := conversationClient.GetMessageContent(activity)
		// if err != nil {
		// 	fmt.Printf("Error getting message content: %v\n", err)
		// 	return
		// }

		// Show the (possibly) decrypted content
		// fmt.Printf("Decrypted content: %s\n", content)
		fmt.Printf("Room ID: %s\n", activity.Target.ID)
		fmt.Println("----------------------------")
	})

	conversationClient.On("share", func(activity *conversation.Activity) {
		fmt.Printf("Received SHARE message from %s: \n", activity.Actor.DisplayName)

		// Display encryption information
		if activity.EncryptionKeyURL != "" {
			fmt.Printf("Message is encrypted with key: %s\n", activity.EncryptionKeyURL)
		} else {
			fmt.Printf("Message is not encrypted\n")
		}

		// Get raw content before decryption
		var rawContent string
		if activity.DecryptedObject != nil {
			rawContent = activity.DecryptedObject.DisplayName
		} else if activity.Object != nil {
			if c, ok := activity.Object["displayName"].(string); ok {
				rawContent = c
			}
		}

		if rawContent != "" {
			fmt.Printf("Raw encrypted content: %s...\n", truncateString(rawContent, 30))
		}

		// Get decrypted message content
		content, err := conversationClient.GetMessageContent(activity)
		if err != nil {
			fmt.Printf("Error getting message content: %v\n", err)
			return
		}

		fmt.Printf("Decrypted content: %s\n", content)
		fmt.Printf("Room ID: %s\n", activity.Target.ID)
		fmt.Println("----------------------------")
	})

	conversationClient.On("acknowledge", func(activity *conversation.Activity) {
		fmt.Printf("Received ACKNOWLEDGE from %s\n", activity.Actor.DisplayName)
		fmt.Printf("Acknowledged message ID: %s\n", activity.Object["id"])
		fmt.Println("----------------------------")
	})

	// Connect to the Mercury service (WebSocket)
	fmt.Println("Connecting to WebSocket...")
	err = mercuryClient.Connect()
	if err != nil {
		fmt.Printf("Error connecting to WebSocket: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to WebSocket!")
	fmt.Println("Listening for conversation events. Press Ctrl+C to exit.")
	fmt.Println("Messages in encrypted spaces will include encryption key information.")

	// Wait for Ctrl+C to exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Disconnect
	fmt.Println("\nDisconnecting from WebSocket...")
	mercuryClient.Disconnect()
	fmt.Println("Disconnected!")
}

// Helper function to truncate long strings
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

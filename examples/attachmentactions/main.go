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

	"github.com/WebexCommunity/webex-go-sdk/v2"
	"github.com/WebexCommunity/webex-go-sdk/v2/attachmentactions"
	"github.com/WebexCommunity/webex-go-sdk/v2/messages"
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

	// Example 1: Send a message with an adaptive card
	roomID := os.Getenv("WEBEX_ROOM_ID")
	if roomID == "" {
		log.Fatal("WEBEX_ROOM_ID environment variable is required")
	}

	fmt.Println("Sending message with adaptive card to room...")
	message := &messages.Message{
		RoomID: roomID,
		Text:   "This is a fallback text for clients that don't support cards",
		Attachments: []messages.Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: map[string]interface{}{
					"type":    "AdaptiveCard",
					"version": "1.0",
					"body": []map[string]interface{}{
						{
							"type": "TextBlock",
							"text": "Adaptive Card Example",
							"size": "large",
						},
						{
							"type":        "Input.Text",
							"id":          "userInput",
							"placeholder": "Enter some text",
						},
						{
							"type":     "Input.Toggle",
							"id":       "acceptTerms",
							"title":    "I accept the terms and conditions",
							"valueOn":  "true",
							"valueOff": "false",
						},
					},
					"actions": []map[string]interface{}{
						{
							"type":  "Action.Submit",
							"title": "Submit",
						},
					},
				},
			},
		},
	}

	createdMessage, err := client.Messages().Create(message)
	if err != nil {
		log.Fatalf("Error sending message: %v", err)
	}
	fmt.Printf("Message with adaptive card sent: ID=%s\n", createdMessage.ID)
	fmt.Println("Use the Webex Teams client to interact with the card...")

	// Example 2: If you have an action ID, you can retrieve it
	if len(os.Args) > 1 {
		actionID := os.Args[1]
		fmt.Printf("\nGetting attachment action with ID %s...\n", actionID)
		action, err := client.AttachmentActions().Get(actionID)
		if err != nil {
			log.Fatalf("Error getting attachment action: %v", err)
		}
		fmt.Printf("Action details:\n")
		fmt.Printf("  ID: %s\n", action.ID)
		fmt.Printf("  Type: %s\n", action.Type)
		fmt.Printf("  Message ID: %s\n", action.MessageID)
		fmt.Printf("  Room ID: %s\n", action.RoomID)
		fmt.Printf("  Person ID: %s\n", action.PersonID)
		fmt.Printf("  Created: %v\n", action.Created)
		fmt.Printf("  Inputs: %v\n", action.Inputs)
	}

	// Example 3: Create an attachment action programmatically
	fmt.Println("\nSimulating an attachment action submission...")
	action := &attachmentactions.AttachmentAction{
		Type:      "submit",
		MessageID: createdMessage.ID,
		Inputs: map[string]interface{}{
			"userInput":   "This is a test input",
			"acceptTerms": "true",
		},
	}

	createdAction, err := client.AttachmentActions().Create(action)
	if err != nil {
		log.Fatalf("Error creating attachment action: %v", err)
	}
	fmt.Printf("Attachment action created: ID=%s\n", createdAction.ID)
	fmt.Printf("Inputs submitted: %v\n", createdAction.Inputs)
}

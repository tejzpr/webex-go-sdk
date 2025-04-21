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
	"github.com/tejzpr/webex-go-sdk/v2/memberships"
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

	// Example 1: List memberships in a room
	roomID := os.Getenv("WEBEX_ROOM_ID")
	if roomID == "" {
		log.Fatal("WEBEX_ROOM_ID environment variable is required")
	}

	fmt.Println("Listing memberships in the room...")
	options := &memberships.ListOptions{
		RoomID: roomID,
		Max:    10,
	}

	page, err := client.Memberships().List(options)
	if err != nil {
		log.Fatalf("Error listing memberships: %v", err)
	}

	fmt.Printf("Found %d memberships:\n", len(page.Items))
	for i, membership := range page.Items {
		fmt.Printf("%d. %s (%s) - Moderator: %t\n",
			i+1,
			membership.PersonEmail,
			membership.PersonDisplayName,
			membership.IsModerator)
	}

	// Example 2: Create a membership (add someone to a room)
	fmt.Println("\nAdding a person to the room...")
	personEmail := os.Getenv("WEBEX_PERSON_EMAIL")
	if personEmail == "" {
		fmt.Println("WEBEX_PERSON_EMAIL not provided, skipping create membership example")
	} else {
		membership := &memberships.Membership{
			RoomID:      roomID,
			PersonEmail: personEmail,
			IsModerator: false,
		}

		createdMembership, err := client.Memberships().Create(membership)
		if err != nil {
			log.Printf("Error creating membership: %v", err)
		} else {
			fmt.Printf("Added %s to the room with membership ID: %s\n",
				createdMembership.PersonEmail,
				createdMembership.ID)

			// Example 3: Update a membership (make someone a moderator)
			fmt.Println("\nUpdating membership to make the person a moderator...")
			updatedMembership := &memberships.Membership{
				IsModerator: true,
			}

			result, err := client.Memberships().Update(createdMembership.ID, updatedMembership)
			if err != nil {
				log.Printf("Error updating membership: %v", err)
			} else {
				fmt.Printf("Updated %s to moderator status: %t\n",
					result.PersonEmail,
					result.IsModerator)
			}

			// Example 4: Get a specific membership
			fmt.Printf("\nGetting details for membership %s...\n", createdMembership.ID)
			membership, err := client.Memberships().Get(createdMembership.ID)
			if err != nil {
				log.Printf("Error getting membership: %v", err)
			} else {
				fmt.Printf("Membership details:\n")
				fmt.Printf("  ID: %s\n", membership.ID)
				fmt.Printf("  Room ID: %s\n", membership.RoomID)
				fmt.Printf("  Person ID: %s\n", membership.PersonID)
				fmt.Printf("  Person Email: %s\n", membership.PersonEmail)
				fmt.Printf("  Is Moderator: %t\n", membership.IsModerator)
				fmt.Printf("  Created: %v\n", membership.Created)
			}

			// Example 5: Delete a membership (remove someone from a room)
			fmt.Printf("\nRemoving %s from the room...\n", createdMembership.PersonEmail)
			err = client.Memberships().Delete(createdMembership.ID)
			if err != nil {
				log.Printf("Error deleting membership: %v", err)
			} else {
				fmt.Printf("Successfully removed %s from the room\n", createdMembership.PersonEmail)
			}
		}
	}
}

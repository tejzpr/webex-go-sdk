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

	"github.com/WebexCommunity/webex-go-sdk/v1"
	"github.com/WebexCommunity/webex-go-sdk/v1/teammemberships"
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

	// Get a team ID to work with
	teamID := os.Getenv("WEBEX_TEAM_ID")
	if teamID == "" {
		log.Fatalf("WEBEX_TEAM_ID environment variable is required")
	}

	// Example 1: List Team Memberships
	fmt.Println("Listing team memberships...")
	memberships, err := client.TeamMemberships().List(&teammemberships.ListOptions{
		TeamID: teamID,
		Max:    100,
	})
	if err != nil {
		log.Fatalf("Failed to list team memberships: %v", err)
	}
	fmt.Printf("Found %d team memberships\n", len(memberships.Items))
	for i, membership := range memberships.Items {
		fmt.Printf("%d. %s (%s) - Moderator: %t\n", i+1, membership.PersonDisplayName, membership.PersonEmail, membership.IsModerator)
	}

	// Get an email to add to the team
	userEmail := os.Getenv("WEBEX_TEST_USER_EMAIL")
	if userEmail == "" {
		log.Printf("WEBEX_TEST_USER_EMAIL environment variable is not set, skipping create/update/delete examples")
		return
	}

	// Example 2: Create a Team Membership
	fmt.Println("\nCreating a new team membership...")
	newMembership := &teammemberships.TeamMembership{
		TeamID:      teamID,
		PersonEmail: userEmail,
		IsModerator: false,
	}
	createdMembership, err := client.TeamMemberships().Create(newMembership)
	if err != nil {
		log.Printf("Failed to create team membership: %v\n", err)
	} else {
		fmt.Printf("Created membership with ID: %s\n", createdMembership.ID)
		fmt.Printf("Person: %s (%s)\n", createdMembership.PersonDisplayName, createdMembership.PersonEmail)
		fmt.Printf("Moderator: %t\n", createdMembership.IsModerator)

		// Example 3: Get Team Membership Details
		fmt.Println("\nFetching team membership details...")
		membershipDetails, err := client.TeamMemberships().Get(createdMembership.ID)
		if err != nil {
			log.Printf("Failed to get team membership details: %v\n", err)
		} else {
			fmt.Printf("Membership Details:\n")
			fmt.Printf("  ID: %s\n", membershipDetails.ID)
			fmt.Printf("  Team ID: %s\n", membershipDetails.TeamID)
			fmt.Printf("  Person ID: %s\n", membershipDetails.PersonID)
			fmt.Printf("  Person Email: %s\n", membershipDetails.PersonEmail)
			fmt.Printf("  Person Display Name: %s\n", membershipDetails.PersonDisplayName)
			fmt.Printf("  Moderator: %t\n", membershipDetails.IsModerator)
			if membershipDetails.Created != nil {
				fmt.Printf("  Created: %s\n", membershipDetails.Created.Format("2006-01-02 15:04:05"))
			}

			// Example 4: Update Team Membership
			fmt.Println("\nUpdating team membership...")
			updatedMembership, err := client.TeamMemberships().Update(membershipDetails.ID, true)
			if err != nil {
				log.Printf("Failed to update team membership: %v\n", err)
			} else {
				fmt.Printf("Updated membership moderator status: %t\n", updatedMembership.IsModerator)

				// Example 5: Delete Team Membership
				fmt.Println("\nDeleting team membership...")
				err = client.TeamMemberships().Delete(updatedMembership.ID)
				if err != nil {
					log.Printf("Failed to delete team membership: %v\n", err)
				} else {
					fmt.Printf("Successfully deleted team membership with ID: %s\n", updatedMembership.ID)
				}
			}
		}
	}
}

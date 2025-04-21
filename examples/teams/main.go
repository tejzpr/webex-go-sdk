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
	"github.com/tejzpr/webex-go-sdk/v1/teams"
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

	// Example 1: List Teams
	fmt.Println("Listing teams...")
	teamsPage, err := client.Teams().List(&teams.ListOptions{
		Max: 100,
	})
	if err != nil {
		log.Fatalf("Failed to list teams: %v", err)
	}
	fmt.Printf("Found %d teams\n", len(teamsPage.Items))
	for i, team := range teamsPage.Items {
		fmt.Printf("%d. %s (%s)\n", i+1, team.Name, team.ID)
		if team.Description != "" {
			fmt.Printf("   Description: %s\n", team.Description)
		}
	}

	// Example 2: Create a Team
	fmt.Println("\nCreating a new team...")
	newTeam := &teams.Team{
		Name:        "API Test Team",
		Description: "Team created via Go SDK example",
	}
	createdTeam, err := client.Teams().Create(newTeam)
	if err != nil {
		log.Printf("Failed to create team: %v\n", err)
	} else {
		fmt.Printf("Created team with ID: %s\n", createdTeam.ID)
		fmt.Printf("Team Name: %s\n", createdTeam.Name)
		fmt.Printf("Team Description: %s\n", createdTeam.Description)
		fmt.Printf("Team Creator ID: %s\n", createdTeam.CreatorID)
		if createdTeam.Created != nil {
			fmt.Printf("Created: %s\n", createdTeam.Created.Format("2006-01-02 15:04:05"))
		}

		// Example 3: Get Team Details
		fmt.Println("\nFetching team details...")
		teamDetails, err := client.Teams().Get(createdTeam.ID)
		if err != nil {
			log.Printf("Failed to get team details: %v\n", err)
		} else {
			fmt.Printf("Team Details:\n")
			fmt.Printf("  ID: %s\n", teamDetails.ID)
			fmt.Printf("  Name: %s\n", teamDetails.Name)
			fmt.Printf("  Description: %s\n", teamDetails.Description)
			fmt.Printf("  Creator ID: %s\n", teamDetails.CreatorID)
			if teamDetails.Created != nil {
				fmt.Printf("  Created: %s\n", teamDetails.Created.Format("2006-01-02 15:04:05"))
			}

			// Example 4: Update Team
			fmt.Println("\nUpdating team...")
			updatedTeamData := &teams.Team{
				Name:        "Updated API Test Team",
				Description: "Updated team description via Go SDK example",
			}
			updatedTeam, err := client.Teams().Update(teamDetails.ID, updatedTeamData)
			if err != nil {
				log.Printf("Failed to update team: %v\n", err)
			} else {
				fmt.Printf("Updated team name to: %s\n", updatedTeam.Name)
				fmt.Printf("Updated team description to: %s\n", updatedTeam.Description)

				// Example 5: Delete Team
				fmt.Println("\nDeleting team...")
				err = client.Teams().Delete(updatedTeam.ID)
				if err != nil {
					log.Printf("Failed to delete team: %v\n", err)
				} else {
					fmt.Printf("Successfully deleted team with ID: %s\n", updatedTeam.ID)
				}
			}
		}
	}
}

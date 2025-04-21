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

	webex "github.com/tejzpr/webex-go-sdk/v1"
	"github.com/tejzpr/webex-go-sdk/v1/people"
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

	// Example 1: Get current user
	fmt.Println("Getting current user...")
	me, err := client.People().Get("me")
	if err != nil {
		log.Fatalf("Error getting current user: %v", err)
	}
	fmt.Printf("Current user: %s (%s)\n", me.DisplayName, me.Emails[0])

	// Example 2: List people by email
	email := me.Emails[0] // Use the current user's email
	fmt.Printf("\nListing people with email containing %s...\n", email)
	options := &people.ListOptions{
		Email: email,
	}
	page, err := client.People().List(options)
	if err != nil {
		log.Fatalf("Error listing people: %v", err)
	}

	fmt.Printf("Found %d people:\n", len(page.Items))
	for i, person := range page.Items {
		fmt.Printf("%d. %s (%s)\n", i+1, person.DisplayName, person.Emails[0])
	}

	// Example 3: Get person by ID
	if len(page.Items) > 0 {
		personID := page.Items[0].ID
		fmt.Printf("\nGetting person with ID %s...\n", personID)
		person, err := client.People().Get(personID)
		if err != nil {
			log.Fatalf("Error getting person: %v", err)
		}
		fmt.Printf("Person: %s (%s)\n", person.DisplayName, person.Emails[0])
	}
}

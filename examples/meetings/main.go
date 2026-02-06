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
	"github.com/tejzpr/webex-go-sdk/v2/meetings"
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

	// Example 1: List Meeting Series (active/scheduled meetings)
	fmt.Println("Listing meeting series...")
	meetingsPage, err := client.Meetings().List(&meetings.ListOptions{
		Max: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list meetings: %v", err)
	}
	fmt.Printf("Found %d meeting series\n", len(meetingsPage.Items))
	for i, m := range meetingsPage.Items {
		fmt.Printf("%d. %s (State: %s, Type: %s)\n", i+1, m.Title, m.State, m.MeetingType)
		fmt.Printf("   Start: %s | End: %s\n", m.Start, m.End)
		if m.WebLink != "" {
			fmt.Printf("   Link: %s\n", m.WebLink)
		}
	}

	// Example 1b: List Past Meeting Instances
	// Note: The Webex API requires meetingType when using the state filter.
	// Use meetingType="meeting" to get actual meeting instances (not series).
	fmt.Println("\nListing past meeting instances...")
	pastMeetingsPage, err := client.Meetings().List(&meetings.ListOptions{
		MeetingType: "meeting",
		State:       "ended",
		Max:         5,
	})
	if err != nil {
		log.Printf("Failed to list past meetings: %v\n", err)
	} else {
		fmt.Printf("Found %d past meetings\n", len(pastMeetingsPage.Items))
		for i, m := range pastMeetingsPage.Items {
			fmt.Printf("%d. %s (State: %s)\n", i+1, m.Title, m.State)
			fmt.Printf("   Start: %s | End: %s\n", m.Start, m.End)
			fmt.Printf("   Has Recording: %v | Has Transcription: %v\n", m.HasRecording, m.HasTranscription)
		}
	}

	// Example 2: Create a Meeting
	fmt.Println("\nCreating a new meeting...")
	newMeeting := &meetings.Meeting{
		Title:    "API Test Meeting",
		Start:    "2026-03-01T10:00:00Z",
		End:      "2026-03-01T10:30:00Z",
		Timezone: "UTC",
		Agenda:   "Meeting created via Go SDK example",
	}
	createdMeeting, err := client.Meetings().Create(newMeeting)
	if err != nil {
		log.Printf("Failed to create meeting: %v\n", err)
	} else {
		fmt.Printf("Created meeting with ID: %s\n", createdMeeting.ID)
		fmt.Printf("Meeting Title: %s\n", createdMeeting.Title)
		fmt.Printf("Meeting Number: %s\n", createdMeeting.MeetingNumber)
		fmt.Printf("Web Link: %s\n", createdMeeting.WebLink)
		fmt.Printf("SIP Address: %s\n", createdMeeting.SipAddress)

		// Example 3: Get Meeting Details
		fmt.Println("\nFetching meeting details...")
		meetingDetails, err := client.Meetings().Get(createdMeeting.ID)
		if err != nil {
			log.Printf("Failed to get meeting details: %v\n", err)
		} else {
			fmt.Printf("Meeting Details:\n")
			fmt.Printf("  ID: %s\n", meetingDetails.ID)
			fmt.Printf("  Title: %s\n", meetingDetails.Title)
			fmt.Printf("  Start: %s\n", meetingDetails.Start)
			fmt.Printf("  End: %s\n", meetingDetails.End)
			fmt.Printf("  State: %s\n", meetingDetails.State)
			fmt.Printf("  Host: %s\n", meetingDetails.HostEmail)

			// Example 4: Update Meeting
			fmt.Println("\nUpdating meeting...")
			updatedMeetingData := &meetings.Meeting{
				Title:  "Updated API Test Meeting",
				Start:  meetingDetails.Start,
				End:    meetingDetails.End,
				Agenda: "Updated agenda via Go SDK example",
			}
			updatedMeeting, err := client.Meetings().Update(meetingDetails.ID, updatedMeetingData)
			if err != nil {
				log.Printf("Failed to update meeting: %v\n", err)
			} else {
				fmt.Printf("Updated meeting title to: %s\n", updatedMeeting.Title)

				// Example 5: Delete Meeting
				fmt.Println("\nDeleting meeting...")
				err = client.Meetings().Delete(updatedMeeting.ID)
				if err != nil {
					log.Printf("Failed to delete meeting: %v\n", err)
				} else {
					fmt.Printf("Successfully deleted meeting with ID: %s\n", updatedMeeting.ID)
				}
			}
		}
	}
}

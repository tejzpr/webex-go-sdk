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
	"github.com/tejzpr/webex-go-sdk/v2/transcripts"
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

	// Example 1: List Transcripts
	fmt.Println("Listing transcripts...")
	transcriptsPage, err := client.Transcripts().List(&transcripts.ListOptions{
		Max: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list transcripts: %v", err)
	}
	fmt.Printf("Found %d transcripts\n", len(transcriptsPage.Items))
	for i, t := range transcriptsPage.Items {
		fmt.Printf("%d. %s (Status: %s)\n", i+1, t.Title, t.Status)
		fmt.Printf("   Meeting ID: %s\n", t.MeetingID)
		fmt.Printf("   Start Time: %s\n", t.StartTime)
	}

	if len(transcriptsPage.Items) == 0 {
		fmt.Println("\nNo transcripts found. Transcripts are generated when meeting recording is enabled with Webex Assistant or Closed Captions.")
		return
	}

	// Use the first transcript for subsequent examples
	transcriptID := transcriptsPage.Items[0].ID
	fmt.Printf("\nUsing transcript ID: %s\n", transcriptID)

	// Example 2: Download Transcript as TXT
	fmt.Println("\nDownloading transcript as TXT...")
	txtContent, err := client.Transcripts().Download(transcriptID, "txt")
	if err != nil {
		log.Printf("Failed to download transcript as TXT: %v\n", err)
	} else {
		// Print first 500 characters
		preview := txtContent
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("Transcript content (preview):\n%s\n", preview)
	}

	// Example 3: Download Transcript as VTT
	fmt.Println("\nDownloading transcript as VTT...")
	vttContent, err := client.Transcripts().Download(transcriptID, "vtt")
	if err != nil {
		log.Printf("Failed to download transcript as VTT: %v\n", err)
	} else {
		// Print first 500 characters
		preview := vttContent
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("VTT content (preview):\n%s\n", preview)
	}

	// Example 4: List Transcript Snippets
	fmt.Println("\nListing transcript snippets...")
	snippetsPage, err := client.Transcripts().ListSnippets(transcriptID, &transcripts.SnippetListOptions{
		Max: 10,
	})
	if err != nil {
		log.Printf("Failed to list snippets: %v\n", err)
	} else {
		fmt.Printf("Found %d snippets\n", len(snippetsPage.Items))
		for i, s := range snippetsPage.Items {
			fmt.Printf("%d. [%s] %s: %s\n", i+1, s.StartTime, s.PersonName, s.Text)
			if s.Duration > 0 {
				fmt.Printf("   Duration: %.1f seconds\n", s.Duration)
			}
		}

		// Example 5: Get a Specific Snippet
		if len(snippetsPage.Items) > 0 {
			snippetID := snippetsPage.Items[0].ID
			fmt.Printf("\nGetting snippet details for ID: %s\n", snippetID)
			snippet, err := client.Transcripts().GetSnippet(transcriptID, snippetID)
			if err != nil {
				log.Printf("Failed to get snippet: %v\n", err)
			} else {
				fmt.Printf("Snippet Details:\n")
				fmt.Printf("  Speaker: %s (%s)\n", snippet.PersonName, snippet.PersonEmail)
				fmt.Printf("  Text: %s\n", snippet.Text)
				fmt.Printf("  Start: %s\n", snippet.StartTime)
				fmt.Printf("  End: %s\n", snippet.EndTime)
				fmt.Printf("  Duration: %.1f seconds\n", snippet.Duration)
				fmt.Printf("  Language: %s\n", snippet.Language)
			}
		}
	}
}

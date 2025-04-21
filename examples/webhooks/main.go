/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/tejzpr/webex-go-sdk/v1"
	"github.com/tejzpr/webex-go-sdk/v1/webhooks"
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

	// Example 1: List Webhooks
	fmt.Println("Listing webhooks...")
	webhooksPage, err := client.Webhooks().List(&webhooks.ListOptions{
		Max: 100,
	})
	if err != nil {
		log.Fatalf("Failed to list webhooks: %v", err)
	}
	fmt.Printf("Found %d webhooks\n", len(webhooksPage.Items))
	for i, webhook := range webhooksPage.Items {
		fmt.Printf("%d. %s (%s)\n", i+1, webhook.Name, webhook.ID)
		fmt.Printf("   Resource: %s, Event: %s\n", webhook.Resource, webhook.Event)
		fmt.Printf("   Target URL: %s\n", webhook.TargetURL)
		fmt.Printf("   Status: %s\n", webhook.Status)
	}

	// Delete all webhooks
	fmt.Println("\nDeleting all webhooks...")
	for _, webhook := range webhooksPage.Items {
		err = client.Webhooks().Delete(webhook.ID)
		if err != nil {
			log.Printf("Failed to delete webhook: %v\n", err)
		}
		fmt.Printf("Deleted webhook with ID: %s\n", webhook.ID)
	}

	// Example 2: Create a new webhook...
	fmt.Println("\nCreating a new webhook...")
	newWebhook := &webhooks.Webhook{
		Name:      "API Test Webhook",
		TargetURL: "https://example.com/webhook",
		Resource:  "messages",
		Event:     "created",
		Secret:    "webhooksecret",
	}

	createdWebhook, err := client.Webhooks().Create(newWebhook)
	if err != nil {
		log.Printf("Failed to create webhook: %v\n", err)
	} else {
		fmt.Printf("Created webhook with ID: %s\n", createdWebhook.ID)
		fmt.Printf("Name: %s\n", createdWebhook.Name)
		fmt.Printf("Target URL: %s\n", createdWebhook.TargetURL)
		fmt.Printf("Resource: %s\n", createdWebhook.Resource)
		fmt.Printf("Event: %s\n", createdWebhook.Event)
		fmt.Printf("Status: %s\n", createdWebhook.Status)
		if createdWebhook.Created != nil {
			fmt.Printf("Created: %s\n", createdWebhook.Created.Format("2006-01-02 15:04:05"))
		}

		// Example 3: Get Webhook Details
		fmt.Println("\nFetching webhook details...")
		webhookDetails, err := client.Webhooks().Get(createdWebhook.ID)
		if err != nil {
			log.Printf("Failed to get webhook details: %v\n", err)
		} else {
			fmt.Printf("Webhook Details:\n")
			fmt.Printf("  ID: %s\n", webhookDetails.ID)
			fmt.Printf("  Name: %s\n", webhookDetails.Name)
			fmt.Printf("  Target URL: %s\n", webhookDetails.TargetURL)
			fmt.Printf("  Resource: %s\n", webhookDetails.Resource)
			fmt.Printf("  Event: %s\n", webhookDetails.Event)
			fmt.Printf("  Status: %s\n", webhookDetails.Status)
			if webhookDetails.Created != nil {
				fmt.Printf("  Created: %s\n", webhookDetails.Created.Format("2006-01-02 15:04:05"))
			}

			// Example 4: Update Webhook
			fmt.Println("\nUpdating webhook...")
			// Create a minimal update with only the required fields
			updatedWebhookData := &struct {
				Name      string `json:"name"`
				TargetURL string `json:"targetUrl"`
			}{
				Name:      webhookDetails.Name,        // Keep original name
				TargetURL: "https://httpbin.org/post", // Use a known valid URL that accepts POST requests
			}

			path := fmt.Sprintf("webhooks/%s", webhookDetails.ID)
			resp, err := client.Core().Request(http.MethodPut, path, nil, updatedWebhookData)
			if err != nil {
				log.Printf("Failed to update webhook via direct request: %v\n", err)
			} else {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Update response: %s\n", string(body))

				var result webhooks.Webhook
				if err := json.Unmarshal(body, &result); err != nil {
					log.Printf("Failed to parse webhook update response: %v\n", err)
				} else {
					fmt.Printf("Updated webhook name to: %s\n", result.Name)
					fmt.Printf("Updated webhook target URL to: %s\n", result.TargetURL)
					if result.Status != "" {
						fmt.Printf("Updated webhook status to: %s\n", result.Status)
					}
				}
			}

			// Try using the SDK method as well for comparison
			updatedWebhook, err := client.Webhooks().Update(webhookDetails.ID, webhooks.NewUpdateWebhook(
				webhookDetails.Name,
				"https://httpbin.org/post",
				"",
				"",
			))
			if err != nil {
				log.Printf("Failed to update webhook via SDK method: %v\n", err)
			} else {
				fmt.Printf("Updated webhook name to: %s\n", updatedWebhook.Name)
				fmt.Printf("Updated webhook target URL to: %s\n", updatedWebhook.TargetURL)
				fmt.Printf("Updated webhook status to: %s\n", updatedWebhook.Status)

				// Example 5: Delete Webhook
				fmt.Println("\nDeleting webhook...")
				err = client.Webhooks().Delete(updatedWebhook.ID)
				if err != nil {
					log.Printf("Failed to delete webhook: %v\n", err)
				} else {
					fmt.Printf("Successfully deleted webhook with ID: %s\n", updatedWebhook.ID)
				}
			}
		}
	}
}

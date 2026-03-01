//go:build functional

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webhooks

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func functionalClient(t *testing.T) *webexsdk.Client {
	t.Helper()
	token := os.Getenv("WEBEX_ACCESS_TOKEN")
	if token == "" {
		t.Fatal("WEBEX_ACCESS_TOKEN environment variable is required")
	}
	client, err := webexsdk.NewClient(token, &webexsdk.Config{
		BaseURL: "https://webexapis.com/v1",
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Webex client: %v", err)
	}
	return client
}

// TestFunctionalWebhooksCRUD tests the full Create → Get → Update → Delete lifecycle
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalWebhooksCRUD -v ./webhooks/
func TestFunctionalWebhooksCRUD(t *testing.T) {
	client := functionalClient(t)
	webhooksClient := New(client, nil)

	// Create
	webhook, err := webhooksClient.Create(&Webhook{
		Name:      "SDK Func Test Webhook",
		TargetURL: "https://example.com/webhook-receiver",
		Resource:  "messages",
		Event:     "created",
		Secret:    "sdk-test-secret",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer func() {
		if err := webhooksClient.Delete(webhook.ID); err != nil {
			t.Logf("Warning: cleanup delete failed: %v", err)
		}
	}()

	if webhook.ID == "" {
		t.Fatal("Created webhook has empty ID")
	}
	t.Logf("Created webhook: ID=%s Name=%q Resource=%s Event=%s Status=%s",
		webhook.ID, webhook.Name, webhook.Resource, webhook.Event, webhook.Status)

	// Get
	got, err := webhooksClient.Get(webhook.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != webhook.ID {
		t.Errorf("Get ID mismatch: got %s, want %s", got.ID, webhook.ID)
	}
	if got.Name != "SDK Func Test Webhook" {
		t.Errorf("Get name mismatch: got %q", got.Name)
	}
	t.Logf("Get confirmed: Name=%q Status=%s", got.Name, got.Status)

	// Update
	updated, err := webhooksClient.Update(webhook.ID, &Webhook{
		Name:      "SDK Func Test Webhook Updated",
		TargetURL: "https://example.com/webhook-updated",
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != "SDK Func Test Webhook Updated" {
		t.Errorf("Update name mismatch: got %q", updated.Name)
	}
	t.Logf("Updated webhook: Name=%q TargetURL=%s", updated.Name, updated.TargetURL)
}

// TestFunctionalWebhooksList tests listing webhooks
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalWebhooksList -v ./webhooks/
func TestFunctionalWebhooksList(t *testing.T) {
	client := functionalClient(t)
	webhooksClient := New(client, nil)

	page, err := webhooksClient.List(&ListOptions{Max: 20})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	t.Logf("Found %d webhooks", len(page.Items))
	for i, w := range page.Items {
		_, _ = fmt.Fprintf(os.Stdout, "[%d] ID=%s Name=%q Resource=%s Event=%s Status=%s TargetURL=%s\n",
			i+1, w.ID, w.Name, w.Resource, w.Event, w.Status, w.TargetURL)
	}
}

// TestFunctionalWebhooksListPagination tests pagination through webhooks
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalWebhooksListPagination -v ./webhooks/
func TestFunctionalWebhooksListPagination(t *testing.T) {
	client := functionalClient(t)
	webhooksClient := New(client, nil)

	// Create 4 webhooks for pagination
	createdIDs := make([]string, 0, 4)
	resources := []string{"messages", "rooms", "memberships", "messages"}
	for i := 0; i < 4; i++ {
		wh, err := webhooksClient.Create(&Webhook{
			Name:      fmt.Sprintf("SDK Pagination Webhook %d", i+1),
			TargetURL: fmt.Sprintf("https://example.com/webhook-%d", i+1),
			Resource:  resources[i],
			Event:     "created",
		})
		if err != nil {
			t.Fatalf("Failed to create webhook %d: %v", i+1, err)
		}
		createdIDs = append(createdIDs, wh.ID)
	}
	defer func() {
		for _, id := range createdIDs {
			if err := webhooksClient.Delete(id); err != nil {
				t.Logf("Warning: cleanup delete webhook %s failed: %v", id, err)
			}
		}
	}()

	t.Logf("Created %d webhooks for pagination test", len(createdIDs))

	page, err := webhooksClient.List(&ListOptions{Max: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	totalItems := len(page.Items)
	pageCount := 1
	t.Logf("Page %d: %d items, hasNext=%v", pageCount, len(page.Items), page.HasNext)

	for page.HasNext && pageCount < 10 {
		nextPage, err := page.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		page.Page = nextPage
		pageCount++
		totalItems += len(nextPage.Items)
		t.Logf("Page %d: %d raw items, hasNext=%v", pageCount, len(nextPage.Items), nextPage.HasNext)
	}

	t.Logf("Pagination complete: %d total items across %d pages", totalItems, pageCount)
}

// TestFunctionalWebhooksNotFound tests structured error on invalid webhook ID
// Run with:
//
//	WEBEX_ACCESS_TOKEN=<your-token> go test -tags functional -run TestFunctionalWebhooksNotFound -v ./webhooks/
func TestFunctionalWebhooksNotFound(t *testing.T) {
	client := functionalClient(t)
	webhooksClient := New(client, nil)

	_, err := webhooksClient.Get("invalid-webhook-id")
	if err == nil {
		t.Fatal("Expected error for invalid webhook ID, got nil")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	t.Logf("Got expected API error: status=%d message=%q trackingId=%s",
		apiErr.StatusCode, apiErr.Message, apiErr.TrackingID)
}

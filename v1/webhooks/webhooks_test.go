/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */


package webhooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-sdk/v1/webexsdk"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/webhooks" {
			t.Errorf("Expected path '/webhooks', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var webhook Webhook
		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify webhook
		if webhook.Name != "Test Webhook" {
			t.Errorf("Expected name 'Test Webhook', got '%s'", webhook.Name)
		}
		if webhook.TargetURL != "https://example.com/webhook" {
			t.Errorf("Expected targetUrl 'https://example.com/webhook', got '%s'", webhook.TargetURL)
		}
		if webhook.Resource != "messages" {
			t.Errorf("Expected resource 'messages', got '%s'", webhook.Resource)
		}
		if webhook.Event != "created" {
			t.Errorf("Expected event 'created', got '%s'", webhook.Event)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseWebhook := Webhook{
			ID:        "test-webhook-id",
			Name:      webhook.Name,
			TargetURL: webhook.TargetURL,
			Resource:  webhook.Resource,
			Event:     webhook.Event,
			Filter:    webhook.Filter,
			Status:    "active",
			Secret:    webhook.Secret,
			Created:   &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(responseWebhook)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create webhooks plugin
	webhooksPlugin := New(client, nil)

	// Create webhook
	webhook := &Webhook{
		Name:      "Test Webhook",
		TargetURL: "https://example.com/webhook",
		Resource:  "messages",
		Event:     "created",
		Filter:    "roomId=test-room-id",
		Secret:    "testsecret",
	}

	result, err := webhooksPlugin.Create(webhook)
	if err != nil {
		t.Fatalf("Failed to create webhook: %v", err)
	}

	// Check webhook
	if result.ID != "test-webhook-id" {
		t.Errorf("Expected ID 'test-webhook-id', got '%s'", result.ID)
	}
	if result.Name != "Test Webhook" {
		t.Errorf("Expected name 'Test Webhook', got '%s'", result.Name)
	}
	if result.TargetURL != "https://example.com/webhook" {
		t.Errorf("Expected targetUrl 'https://example.com/webhook', got '%s'", result.TargetURL)
	}
	if result.Resource != "messages" {
		t.Errorf("Expected resource 'messages', got '%s'", result.Resource)
	}
	if result.Event != "created" {
		t.Errorf("Expected event 'created', got '%s'", result.Event)
	}
	if result.Filter != "roomId=test-room-id" {
		t.Errorf("Expected filter 'roomId=test-room-id', got '%s'", result.Filter)
	}
	if result.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", result.Status)
	}
	if result.Secret != "testsecret" {
		t.Errorf("Expected secret 'testsecret', got '%s'", result.Secret)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/webhooks/test-webhook-id" {
			t.Errorf("Expected path '/webhooks/test-webhook-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		webhook := Webhook{
			ID:        "test-webhook-id",
			Name:      "Test Webhook",
			TargetURL: "https://example.com/webhook",
			Resource:  "messages",
			Event:     "created",
			Filter:    "roomId=test-room-id",
			Status:    "active",
			Secret:    "testsecret",
			Created:   &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(webhook)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create webhooks plugin
	webhooksPlugin := New(client, nil)

	// Get webhook
	webhook, err := webhooksPlugin.Get("test-webhook-id")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}

	// Check webhook
	if webhook.ID != "test-webhook-id" {
		t.Errorf("Expected ID 'test-webhook-id', got '%s'", webhook.ID)
	}
	if webhook.Name != "Test Webhook" {
		t.Errorf("Expected name 'Test Webhook', got '%s'", webhook.Name)
	}
	if webhook.TargetURL != "https://example.com/webhook" {
		t.Errorf("Expected targetUrl 'https://example.com/webhook', got '%s'", webhook.TargetURL)
	}
	if webhook.Resource != "messages" {
		t.Errorf("Expected resource 'messages', got '%s'", webhook.Resource)
	}
	if webhook.Event != "created" {
		t.Errorf("Expected event 'created', got '%s'", webhook.Event)
	}
	if webhook.Filter != "roomId=test-room-id" {
		t.Errorf("Expected filter 'roomId=test-room-id', got '%s'", webhook.Filter)
	}
	if webhook.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", webhook.Status)
	}
	if webhook.Secret != "testsecret" {
		t.Errorf("Expected secret 'testsecret', got '%s'", webhook.Secret)
	}
	if webhook.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/webhooks" {
			t.Errorf("Expected path '/webhooks', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("max") != "50" {
			t.Errorf("Expected max '50', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		webhooks := []Webhook{
			{
				ID:        "test-webhook-id-1",
				Name:      "Test Webhook 1",
				TargetURL: "https://example.com/webhook1",
				Resource:  "messages",
				Event:     "created",
				Filter:    "roomId=test-room-id-1",
				Status:    "active",
				Secret:    "testsecret1",
				Created:   &createdAt,
			},
			{
				ID:        "test-webhook-id-2",
				Name:      "Test Webhook 2",
				TargetURL: "https://example.com/webhook2",
				Resource:  "messages",
				Event:     "created",
				Filter:    "roomId=test-room-id-2",
				Status:    "active",
				Secret:    "testsecret2",
				Created:   &createdAt,
			},
		}

		// Prepare response with items
		response := struct {
			Items []Webhook `json:"items"`
		}{
			Items: webhooks,
		}

		// Write response
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create webhooks plugin
	webhooksPlugin := New(client, nil)

	// List webhooks
	options := &ListOptions{
		Max: 50,
	}
	page, err := webhooksPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list webhooks: %v", err)
	}

	// Check page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}

	// Check first webhook
	webhook := page.Items[0]
	if webhook.ID != "test-webhook-id-1" {
		t.Errorf("Expected ID 'test-webhook-id-1', got '%s'", webhook.ID)
	}
	if webhook.Name != "Test Webhook 1" {
		t.Errorf("Expected name 'Test Webhook 1', got '%s'", webhook.Name)
	}
	if webhook.TargetURL != "https://example.com/webhook1" {
		t.Errorf("Expected targetUrl 'https://example.com/webhook1', got '%s'", webhook.TargetURL)
	}
	if webhook.Resource != "messages" {
		t.Errorf("Expected resource 'messages', got '%s'", webhook.Resource)
	}
	if webhook.Event != "created" {
		t.Errorf("Expected event 'created', got '%s'", webhook.Event)
	}
}

func TestUpdate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/webhooks/test-webhook-id" {
			t.Errorf("Expected path '/webhooks/test-webhook-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}

		// Parse request body
		var webhook Webhook
		if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify webhook
		if webhook.Name != "Updated Webhook" {
			t.Errorf("Expected name 'Updated Webhook', got '%s'", webhook.Name)
		}
		if webhook.TargetURL != "https://example.com/updated-webhook" {
			t.Errorf("Expected targetUrl 'https://example.com/updated-webhook', got '%s'", webhook.TargetURL)
		}
		if webhook.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", webhook.Status)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseWebhook := Webhook{
			ID:        "test-webhook-id",
			Name:      webhook.Name,
			TargetURL: webhook.TargetURL,
			Resource:  "messages",
			Event:     "created",
			Filter:    "roomId=test-room-id",
			Status:    webhook.Status,
			Secret:    "testsecret",
			Created:   &createdAt,
		}

		// Write response
		json.NewEncoder(w).Encode(responseWebhook)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create webhooks plugin
	webhooksPlugin := New(client, nil)

	// Update webhook
	webhook := &Webhook{
		Name:      "Updated Webhook",
		TargetURL: "https://example.com/updated-webhook",
		Status:    "inactive",
	}

	result, err := webhooksPlugin.Update("test-webhook-id", webhook)
	if err != nil {
		t.Fatalf("Failed to update webhook: %v", err)
	}

	// Check webhook
	if result.ID != "test-webhook-id" {
		t.Errorf("Expected ID 'test-webhook-id', got '%s'", result.ID)
	}
	if result.Name != "Updated Webhook" {
		t.Errorf("Expected name 'Updated Webhook', got '%s'", result.Name)
	}
	if result.TargetURL != "https://example.com/updated-webhook" {
		t.Errorf("Expected targetUrl 'https://example.com/updated-webhook', got '%s'", result.TargetURL)
	}
	if result.Status != "inactive" {
		t.Errorf("Expected status 'inactive', got '%s'", result.Status)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/webhooks/test-webhook-id" {
			t.Errorf("Expected path '/webhooks/test-webhook-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}

		// Write response
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Create a proper client with the test server
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Override the base URL
	client.BaseURL = baseURL

	// Create webhooks plugin
	webhooksPlugin := New(client, nil)

	// Delete webhook
	err = webhooksPlugin.Delete("test-webhook-id")
	if err != nil {
		t.Fatalf("Failed to delete webhook: %v", err)
	}
}

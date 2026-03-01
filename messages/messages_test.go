/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package messages

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path '/messages', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Parse request body
		var message Message
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify message
		if message.RoomID != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", message.RoomID)
		}
		if message.Text != "Hello, World!" {
			t.Errorf("Expected text 'Hello, World!', got '%s'", message.Text)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		responseMsg := Message{
			ID:          "test-message-id",
			RoomID:      message.RoomID,
			Text:        message.Text,
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(responseMsg)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Create message
	message := &Message{
		RoomID: "test-room-id",
		Text:   "Hello, World!",
	}

	result, err := messagesPlugin.Create(message)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Check message
	if result.ID != "test-message-id" {
		t.Errorf("Expected ID 'test-message-id', got '%s'", result.ID)
	}
	if result.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", result.RoomID)
	}
	if result.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", result.Text)
	}
	if result.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", result.PersonID)
	}
	if result.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", result.PersonEmail)
	}
	if result.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestGet(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages/test-message-id" {
			t.Errorf("Expected path '/messages/test-message-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		message := Message{
			ID:          "test-message-id",
			RoomID:      "test-room-id",
			Text:        "Hello, World!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Write response
		_ = json.NewEncoder(w).Encode(message)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Get message
	message, err := messagesPlugin.Get("test-message-id")
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	// Check message
	if message.ID != "test-message-id" {
		t.Errorf("Expected ID 'test-message-id', got '%s'", message.ID)
	}
	if message.RoomID != "test-room-id" {
		t.Errorf("Expected roomId 'test-room-id', got '%s'", message.RoomID)
	}
	if message.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", message.Text)
	}
	if message.PersonID != "test-person-id" {
		t.Errorf("Expected personId 'test-person-id', got '%s'", message.PersonID)
	}
	if message.PersonEmail != "test@example.com" {
		t.Errorf("Expected personEmail 'test@example.com', got '%s'", message.PersonEmail)
	}
	if message.Created == nil {
		t.Error("Expected created timestamp, got nil")
	}
}

func TestList(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path '/messages', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}

		// Check query parameters
		if r.URL.Query().Get("roomId") != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", r.URL.Query().Get("roomId"))
		}
		if r.URL.Query().Get("max") != "50" {
			t.Errorf("Expected max '50', got '%s'", r.URL.Query().Get("max"))
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Create a sample response
		createdAt := time.Now()
		message1 := Message{
			ID:          "test-message-id-1",
			RoomID:      "test-room-id",
			Text:        "Hello, World!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}
		message2 := Message{
			ID:          "test-message-id-2",
			RoomID:      "test-room-id",
			Text:        "Hello again!",
			PersonID:    "test-person-id",
			PersonEmail: "test@example.com",
			Created:     &createdAt,
		}

		// Create response with items array
		response := struct {
			Items []Message `json:"items"`
		}{
			Items: []Message{message1, message2},
		}

		// Write response
		_ = json.NewEncoder(w).Encode(response)
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// List messages
	options := &ListOptions{
		RoomID: "test-room-id",
		Max:    50,
	}

	page, err := messagesPlugin.List(options)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}

	// Check messages
	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(page.Items))
	}

	message1 := page.Items[0]
	if message1.ID != "test-message-id-1" {
		t.Errorf("Expected ID 'test-message-id-1', got '%s'", message1.ID)
	}
	if message1.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got '%s'", message1.Text)
	}

	message2 := page.Items[1]
	if message2.ID != "test-message-id-2" {
		t.Errorf("Expected ID 'test-message-id-2', got '%s'", message2.ID)
	}
	if message2.Text != "Hello again!" {
		t.Errorf("Expected text 'Hello again!', got '%s'", message2.Text)
	}
}

func TestDelete(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/messages/test-message-id" {
			t.Errorf("Expected path '/messages/test-message-id', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}

		// Write response with no content
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

	// Create messages plugin
	messagesPlugin := New(client, nil)

	// Delete message
	err = messagesPlugin.Delete("test-message-id")
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}
}

// newTestClient creates a messages client backed by the given test server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		DefaultHeaders: make(map[string]string),
	}
	wc, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	wc.BaseURL = baseURL
	return New(wc, nil)
}

// --- CreateWithAttachment tests ---

func TestCreateWithAttachment_RawBytes(t *testing.T) {
	fileContent := []byte("hello world file content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("Expected path '/messages', got '%s'", r.URL.Path)
		}

		// Must be multipart
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("Expected multipart/form-data Content-Type, got %s", ct)
		}

		// Check auth
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer test-token auth header")
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// Check text fields
		if r.FormValue("roomId") != "test-room-id" {
			t.Errorf("Expected roomId 'test-room-id', got '%s'", r.FormValue("roomId"))
		}
		if r.FormValue("text") != "Check this file" {
			t.Errorf("Expected text 'Check this file', got '%s'", r.FormValue("text"))
		}

		// Check file
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("Failed to get uploaded file: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got '%s'", header.Filename)
		}
		body, _ := io.ReadAll(file)
		if string(body) != string(fileContent) {
			t.Errorf("Expected file content '%s', got '%s'", fileContent, body)
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		createdAt := time.Now()
		resp := Message{
			ID:      "msg-with-file",
			RoomID:  "test-room-id",
			Text:    "Check this file",
			Files:   []string{"https://webexapis.com/v1/contents/file123"},
			Created: &createdAt,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mc := newTestClient(t, server)
	msg := &Message{RoomID: "test-room-id", Text: "Check this file"}
	result, err := mc.CreateWithAttachment(msg, &FileUpload{
		FileName:  "test.txt",
		FileBytes: fileContent,
	})
	if err != nil {
		t.Fatalf("CreateWithAttachment failed: %v", err)
	}
	if result.ID != "msg-with-file" {
		t.Errorf("Expected ID 'msg-with-file', got '%s'", result.ID)
	}
	if len(result.Files) != 1 {
		t.Errorf("Expected 1 file URL, got %d", len(result.Files))
	}
}

func TestCreateWithBase64File(t *testing.T) {
	originalContent := "PDF file content for testing"
	b64Data := base64.StdEncoding.EncodeToString([]byte(originalContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart: %v", err)
		}

		// Verify the file was base64 decoded correctly
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("Failed to get uploaded file: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "report.pdf" {
			t.Errorf("Expected filename 'report.pdf', got '%s'", header.Filename)
		}

		body, _ := io.ReadAll(file)
		if string(body) != originalContent {
			t.Errorf("Base64 decode mismatch: expected '%s', got '%s'", originalContent, body)
		}

		if r.FormValue("toPersonEmail") != "user@example.com" {
			t.Errorf("Expected toPersonEmail 'user@example.com', got '%s'", r.FormValue("toPersonEmail"))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := Message{ID: "msg-b64", Text: "Here is the report"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	mc := newTestClient(t, server)
	msg := &Message{ToPersonEmail: "user@example.com", Text: "Here is the report"}
	result, err := mc.CreateWithBase64File(msg, "report.pdf", b64Data)
	if err != nil {
		t.Fatalf("CreateWithBase64File failed: %v", err)
	}
	if result.ID != "msg-b64" {
		t.Errorf("Expected ID 'msg-b64', got '%s'", result.ID)
	}
}

func TestCreateWithAttachment_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Server should not be called for validation errors")
	}))
	defer server.Close()
	mc := newTestClient(t, server)

	// Missing destination
	_, err := mc.CreateWithAttachment(&Message{Text: "no dest"}, &FileUpload{FileBytes: []byte("data")})
	if err == nil {
		t.Error("Expected error for missing destination")
	}

	// Nil file
	_, err = mc.CreateWithAttachment(&Message{RoomID: "room"}, nil)
	if err == nil {
		t.Error("Expected error for nil file")
	}

	// No file data
	_, err = mc.CreateWithAttachment(&Message{RoomID: "room"}, &FileUpload{FileName: "empty.txt"})
	if err == nil {
		t.Error("Expected error for empty file data")
	}

	// Invalid base64
	_, err = mc.CreateWithBase64File(&Message{RoomID: "room"}, "bad.txt", "!!!not-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64 data")
	}
}

func TestCreateWithAttachment_MultipartFields(t *testing.T) {
	// Verify all optional fields are sent when present
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart: %v", err)
		}

		if r.FormValue("toPersonId") != "person-123" {
			t.Errorf("Expected toPersonId 'person-123', got '%s'", r.FormValue("toPersonId"))
		}
		if r.FormValue("markdown") != "**bold text**" {
			t.Errorf("Expected markdown '**bold text**', got '%s'", r.FormValue("markdown"))
		}
		if r.FormValue("parentId") != "parent-msg-id" {
			t.Errorf("Expected parentId 'parent-msg-id', got '%s'", r.FormValue("parentId"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Message{ID: "msg-fields"})
	}))
	defer server.Close()

	mc := newTestClient(t, server)
	msg := &Message{
		ToPersonID: "person-123",
		Markdown:   "**bold text**",
		ParentID:   "parent-msg-id",
	}
	result, err := mc.CreateWithAttachment(msg, &FileUpload{
		FileName:  "img.png",
		FileBytes: []byte{0x89, 0x50, 0x4E, 0x47}, // PNG header bytes
	})
	if err != nil {
		t.Fatalf("CreateWithAttachment failed: %v", err)
	}
	if result.ID != "msg-fields" {
		t.Errorf("Expected ID 'msg-fields', got '%s'", result.ID)
	}
}

// --- CreateWithAdaptiveCard tests ---

func TestCreateWithAdaptiveCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Should be regular JSON (not multipart)
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Expected application/json, got %s", ct)
		}

		var msg struct {
			RoomID      string       `json:"roomId"`
			Text        string       `json:"text"`
			Attachments []Attachment `json:"attachments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		if msg.RoomID != "room-123" {
			t.Errorf("Expected roomId 'room-123', got '%s'", msg.RoomID)
		}
		if msg.Text != "Card fallback" {
			t.Errorf("Expected text 'Card fallback', got '%s'", msg.Text)
		}
		if len(msg.Attachments) != 1 {
			t.Fatalf("Expected 1 attachment, got %d", len(msg.Attachments))
		}
		if msg.Attachments[0].ContentType != "application/vnd.microsoft.card.adaptive" {
			t.Errorf("Expected adaptive card content type, got '%s'", msg.Attachments[0].ContentType)
		}

		// Verify the card body was serialized correctly
		cardContent, ok := msg.Attachments[0].Content.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected card content to be a map")
		}
		if cardContent["type"] != "AdaptiveCard" {
			t.Errorf("Expected card type 'AdaptiveCard', got '%v'", cardContent["type"])
		}
		if cardContent["version"] != "1.3" {
			t.Errorf("Expected card version '1.3', got '%v'", cardContent["version"])
		}

		w.Header().Set("Content-Type", "application/json")
		createdAt := time.Now()
		_ = json.NewEncoder(w).Encode(Message{
			ID:      "msg-card",
			RoomID:  "room-123",
			Text:    "Card fallback",
			Created: &createdAt,
		})
	}))
	defer server.Close()

	mc := newTestClient(t, server)

	cardBody := map[string]interface{}{
		"type":    "AdaptiveCard",
		"version": "1.3",
		"body": []map[string]interface{}{
			{
				"type": "TextBlock",
				"text": "Hello from Adaptive Card!",
				"size": "Large",
			},
		},
		"actions": []map[string]interface{}{
			{
				"type":  "Action.Submit",
				"title": "Click me",
			},
		},
	}
	card := NewAdaptiveCard(cardBody)

	result, err := mc.CreateWithAdaptiveCard(
		&Message{RoomID: "room-123"},
		card,
		"Card fallback",
	)
	if err != nil {
		t.Fatalf("CreateWithAdaptiveCard failed: %v", err)
	}
	if result.ID != "msg-card" {
		t.Errorf("Expected ID 'msg-card', got '%s'", result.ID)
	}
}

func TestCreateWithAdaptiveCard_DefaultFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		// When no fallback text is provided, default should be "Adaptive Card"
		if msg.Text != "Adaptive Card" {
			t.Errorf("Expected default fallback text 'Adaptive Card', got '%s'", msg.Text)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Message{ID: "msg-default"})
	}))
	defer server.Close()

	mc := newTestClient(t, server)
	card := NewAdaptiveCard(map[string]interface{}{"type": "AdaptiveCard", "version": "1.3"})
	result, err := mc.CreateWithAdaptiveCard(&Message{RoomID: "room"}, card, "")
	if err != nil {
		t.Fatalf("CreateWithAdaptiveCard failed: %v", err)
	}
	if result.ID != "msg-default" {
		t.Errorf("Expected ID 'msg-default', got '%s'", result.ID)
	}
}

func TestCreateWithAdaptiveCard_ValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Server should not be called for validation errors")
	}))
	defer server.Close()

	mc := newTestClient(t, server)
	card := NewAdaptiveCard(map[string]interface{}{"type": "AdaptiveCard"})

	// Missing destination
	_, err := mc.CreateWithAdaptiveCard(&Message{}, card, "fallback")
	if err == nil {
		t.Error("Expected error for missing destination")
	}
}

// --- resolveFileBytes tests ---

func TestResolveFileBytes_RawBytes(t *testing.T) {
	data := []byte("raw content")
	result, err := resolveFileBytes(&FileUpload{FileBytes: data})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(result) != "raw content" {
		t.Errorf("Expected 'raw content', got '%s'", result)
	}
}

func TestResolveFileBytes_StdBase64(t *testing.T) {
	original := "standard base64 content"
	b64 := base64.StdEncoding.EncodeToString([]byte(original))
	result, err := resolveFileBytes(&FileUpload{Base64Data: b64})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(result) != original {
		t.Errorf("Expected '%s', got '%s'", original, result)
	}
}

func TestResolveFileBytes_URLBase64(t *testing.T) {
	// Use content that produces + and / in standard base64
	original := []byte{0xfb, 0xff, 0xfe}
	b64 := base64.URLEncoding.EncodeToString(original)
	result, err := resolveFileBytes(&FileUpload{Base64Data: b64})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if string(result) != string(original) {
		t.Errorf("URL-safe base64 decode mismatch")
	}
}

func TestResolveFileBytes_NoData(t *testing.T) {
	_, err := resolveFileBytes(&FileUpload{FileName: "empty.txt"})
	if err == nil {
		t.Error("Expected error for empty file data")
	}
}

// --- NewAdaptiveCard tests ---

func TestNewAdaptiveCard(t *testing.T) {
	cardBody := map[string]interface{}{
		"type":    "AdaptiveCard",
		"version": "1.3",
	}
	card := NewAdaptiveCard(cardBody)

	if card.ContentType != "application/vnd.microsoft.card.adaptive" {
		t.Errorf("Expected content type 'application/vnd.microsoft.card.adaptive', got '%s'", card.ContentType)
	}
	if card.Content == nil {
		t.Error("Expected non-nil card content")
	}
}

func TestCreate_RemoteFileAttachment(t *testing.T) {
	// Per eval.md: remote file attachments use the files JSON parameter with a URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}

		// Verify files contains the remote URL
		if len(msg.Files) != 1 {
			t.Fatalf("Expected 1 file URL, got %d", len(msg.Files))
		}
		if msg.Files[0] != "http://www.example.com/images/media.png" {
			t.Errorf("Expected remote URL, got %q", msg.Files[0])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Message{
			ID:     "msg-1",
			RoomID: msg.RoomID,
			Text:   msg.Text,
			Files:  msg.Files,
		})
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:    server.URL,
		Timeout:    5 * time.Second,
		HttpClient: server.Client(),
	}
	client, _ := webexsdk.NewClient("test-token", config)
	client.BaseURL = baseURL

	messagesPlugin := New(client, nil)
	result, err := messagesPlugin.Create(&Message{
		RoomID: "room-123",
		Text:   "Check out this file",
		Files:  []string{"http://www.example.com/images/media.png"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("Expected 1 file in response, got %d", len(result.Files))
	}
	if result.Files[0] != "http://www.example.com/images/media.png" {
		t.Errorf("Expected file URL in response, got %q", result.Files[0])
	}
}

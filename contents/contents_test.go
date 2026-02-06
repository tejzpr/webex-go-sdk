/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package contents

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

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

func TestDownload(t *testing.T) {
	fileData := []byte("fake PDF content here")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/contents/content-id-123" {
			t.Errorf("Expected path '/contents/content-id-123', got '%s'", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected auth header")
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		w.WriteHeader(http.StatusOK)
		w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	info, err := cc.Download("content-id-123")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if info.ContentType != "application/pdf" {
		t.Errorf("Expected content type 'application/pdf', got '%s'", info.ContentType)
	}
	if info.ContentDisposition != `attachment; filename="report.pdf"` {
		t.Errorf("Unexpected content disposition: %s", info.ContentDisposition)
	}
	if string(info.Data) != string(fileData) {
		t.Errorf("Data mismatch")
	}
}

func TestDownload_EmptyID(t *testing.T) {
	cc := New(nil, nil)
	_, err := cc.Download("")
	if err == nil {
		t.Error("Expected error for empty contentID")
	}
}

func TestDownload_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	_, err := cc.Download("nonexistent-id")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestDownloadFromURL(t *testing.T) {
	imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A} // PNG header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected auth header")
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", `attachment; filename="photo.png"`)
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	info, err := cc.DownloadFromURL(server.URL + "/v1/contents/some-content-id")
	if err != nil {
		t.Fatalf("DownloadFromURL failed: %v", err)
	}

	if info.ContentType != "image/png" {
		t.Errorf("Expected 'image/png', got '%s'", info.ContentType)
	}
	if string(info.Data) != string(imageData) {
		t.Errorf("Data mismatch")
	}
}

func TestDownloadFromURL_EmptyURL(t *testing.T) {
	cc := New(nil, nil)
	_, err := cc.DownloadFromURL("")
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}

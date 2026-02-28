/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package contents

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
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
		if _, err := w.Write(fileData); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
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
		if _, err := w.Write([]byte(`{"message":"not found"}`)); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
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
		_, _ = w.Write(imageData)
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

// --- Anti-malware tests ---

func TestDownload_StructuredErrors(t *testing.T) {
	// Download should return structured APIError types, not plain fmt.Errorf
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"content not found","trackingId":"test-123"}`))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	_, err := cc.Download("missing-id")
	if err == nil {
		t.Fatal("Expected error")
	}

	var apiErr *webexsdk.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected *webexsdk.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", apiErr.StatusCode)
	}
	if !webexsdk.IsNotFound(err) {
		t.Error("Expected IsNotFound to be true")
	}
}

func TestDownload_410Gone_InfectedFile(t *testing.T) {
	// 410 Gone means the file was scanned and found to be infected
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"message":"file is infected and unavailable"}`))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	_, err := cc.Download("infected-id")
	if err == nil {
		t.Fatal("Expected error for infected file")
	}

	if !webexsdk.IsGone(err) {
		t.Errorf("Expected IsGone, got %T: %v", err, err)
	}
}

func TestDownload_428PreconditionRequired_UnscannableFile(t *testing.T) {
	// 428 means the file cannot be scanned (e.g., encrypted)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(428)
		_, _ = w.Write([]byte(`{"message":"file cannot be scanned"}`))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	_, err := cc.Download("encrypted-id")
	if err == nil {
		t.Fatal("Expected error for unscannable file")
	}

	if !webexsdk.IsPreconditionRequired(err) {
		t.Errorf("Expected IsPreconditionRequired, got %T: %v", err, err)
	}
}

func TestDownloadWithOptions_AllowUnscannable(t *testing.T) {
	// When AllowUnscannable is true, ?allow=unscannable is appended
	fileData := []byte("encrypted file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify allow=unscannable query parameter
		if r.URL.Query().Get("allow") != "unscannable" {
			w.WriteHeader(428)
			_, _ = w.Write([]byte(`{"message":"file cannot be scanned"}`))
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="encrypted.zip"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	opts := &DownloadOptions{AllowUnscannable: true}
	info, err := cc.DownloadWithOptions("encrypted-id", opts)
	if err != nil {
		t.Fatalf("DownloadWithOptions failed: %v", err)
	}

	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
	if info.ContentType != "application/octet-stream" {
		t.Errorf("Expected application/octet-stream, got %s", info.ContentType)
	}
}

func TestDownloadWithOptions_NilOptions(t *testing.T) {
	// Nil options should work like regular Download
	fileData := []byte("safe content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("allow") != "" {
			t.Error("Expected no allow query param with nil options")
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	info, err := cc.DownloadWithOptions("safe-id", nil)
	if err != nil {
		t.Fatalf("DownloadWithOptions failed: %v", err)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
}

func TestDownloadWithOptions_FalseUnscannable(t *testing.T) {
	// AllowUnscannable=false should not add the query param
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("allow") != "" {
			t.Error("Expected no allow query param when AllowUnscannable=false")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	opts := &DownloadOptions{AllowUnscannable: false}
	_, err := cc.DownloadWithOptions("safe-id", opts)
	if err != nil {
		t.Fatalf("DownloadWithOptions failed: %v", err)
	}
}

func TestDownloadFromURL_StructuredErrors(t *testing.T) {
	// DownloadFromURL should also return structured errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"access denied","trackingId":"track-456"}`))
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	_, err := cc.DownloadFromURL(server.URL + "/v1/contents/some-id")
	if err == nil {
		t.Fatal("Expected error")
	}

	if !webexsdk.IsForbidden(err) {
		t.Errorf("Expected IsForbidden, got %T: %v", err, err)
	}
}

func TestDownloadFromURLWithOptions_AllowUnscannable(t *testing.T) {
	fileData := []byte("encrypted content via URL")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("allow") != "unscannable" {
			w.WriteHeader(428)
			_, _ = w.Write([]byte(`{"message":"unscannable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClient(t, server)
	opts := &DownloadOptions{AllowUnscannable: true}
	info, err := cc.DownloadFromURLWithOptions(server.URL+"/v1/contents/enc-id", opts)
	if err != nil {
		t.Fatalf("DownloadFromURLWithOptions failed: %v", err)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
}

func TestDownload_423Locked_ReturnsLockedError(t *testing.T) {
	// 423 means file is being scanned — returns LockedError
	// With MaxRetries=0 (no retries), we should get the error immediately.
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(423)
		_, _ = w.Write([]byte(`{"message":"file is being scanned"}`))
	}))
	defer server.Close()

	// Use a client with no retries to test the raw error
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		MaxRetries:     0,
		DefaultHeaders: make(map[string]string),
	}
	wc, _ := webexsdk.NewClient("test-token", config)
	wc.BaseURL = baseURL
	cc := New(wc, nil)

	_, err := cc.Download("scanning-id")
	if err == nil {
		t.Fatal("Expected error for locked file")
	}
	if !webexsdk.IsLocked(err) {
		t.Errorf("Expected IsLocked, got %T: %v", err, err)
	}
	if attempts != 1 {
		t.Errorf("Expected exactly 1 attempt with MaxRetries=0, got %d", attempts)
	}
}

func newTestClientWithRetries(t *testing.T, server *httptest.Server, maxRetries int) *Client {
	t.Helper()
	baseURL, _ := url.Parse(server.URL)
	config := &webexsdk.Config{
		BaseURL:        server.URL,
		Timeout:        5 * time.Second,
		HttpClient:     server.Client(),
		MaxRetries:     maxRetries,
		RetryBaseDelay: 10 * time.Millisecond, // fast retries for tests
		DefaultHeaders: make(map[string]string),
	}
	wc, err := webexsdk.NewClient("test-token", config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	wc.BaseURL = baseURL
	return New(wc, nil)
}

func TestDownload_423AutoRetrySuccess(t *testing.T) {
	// Server returns 423 twice, then succeeds on 3rd attempt.
	// With MaxRetries=3, Download should succeed automatically.
	fileData := []byte("scanned file content")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(423)
			_, _ = w.Write([]byte(`{"message":"file is being scanned"}`))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", `attachment; filename="scanned.txt"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 3)
	info, err := cc.Download("scan-pending-id")
	if err != nil {
		t.Fatalf("Download should have succeeded after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (2x 423 + 1x 200), got %d", attempts)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch after retry")
	}
	if info.ContentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", info.ContentType)
	}
}

func TestDownload_423RetriesExhausted(t *testing.T) {
	// Server always returns 423 — after MaxRetries the caller gets a LockedError.
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(423)
		_, _ = w.Write([]byte(`{"message":"file is being scanned"}`))
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 2)
	_, err := cc.Download("forever-scanning-id")
	if err == nil {
		t.Fatal("Expected error after retries exhausted")
	}
	if !webexsdk.IsLocked(err) {
		t.Errorf("Expected IsLocked error, got %T: %v", err, err)
	}
	// 1 initial + 2 retries = 3 total
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 + MaxRetries=2), got %d", attempts)
	}
}

func TestDownload_423RetryWithRetryAfterHeader(t *testing.T) {
	// Verify that Retry-After header is respected (indirectly via timing).
	fileData := []byte("finally scanned")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(423)
			_, _ = w.Write([]byte(`{"message":"scanning"}`))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 3)
	start := time.Now()
	info, err := cc.Download("slow-scan-id")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Download should have succeeded: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	// Retry-After: 1 means at least 1 second delay
	if elapsed < 900*time.Millisecond {
		t.Errorf("Expected at least ~1s delay for Retry-After, got %v", elapsed)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
}

func TestDownloadFromURL_423AutoRetrySuccess(t *testing.T) {
	// DownloadFromURL should also auto-retry 423 via RequestURL → RequestURLWithRetry.
	fileData := []byte("url-scanned content")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(423)
			_, _ = w.Write([]byte(`{"message":"scanning"}`))
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", `attachment; filename="photo.png"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 3)
	info, err := cc.DownloadFromURL(server.URL + "/v1/contents/scan-url-id")
	if err != nil {
		t.Fatalf("DownloadFromURL should have succeeded after retry: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
	if info.ContentType != "image/png" {
		t.Errorf("Expected image/png, got %s", info.ContentType)
	}
}

func TestDownloadFromURLWithOptions_423AutoRetry(t *testing.T) {
	// DownloadFromURLWithOptions with AllowUnscannable also retries on 423.
	fileData := []byte("encrypted but scanned")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(423)
			_, _ = w.Write([]byte(`{"message":"scanning"}`))
			return
		}
		// Verify allow=unscannable is still present on retry
		if r.URL.Query().Get("allow") != "unscannable" {
			t.Error("Expected allow=unscannable query param on retried request")
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 3)
	opts := &DownloadOptions{AllowUnscannable: true}
	info, err := cc.DownloadFromURLWithOptions(server.URL+"/v1/contents/enc-id", opts)
	if err != nil {
		t.Fatalf("DownloadFromURLWithOptions should have succeeded: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if string(info.Data) != string(fileData) {
		t.Error("Data mismatch")
	}
}

func TestDownloadWithOptions_423AutoRetry(t *testing.T) {
	// DownloadWithOptions (path-based) also retries on 423.
	fileData := []byte("safe after scan")
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(423)
			_, _ = w.Write([]byte(`{"message":"scanning"}`))
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fileData)
	}))
	defer server.Close()

	cc := newTestClientWithRetries(t, server, 2)
	info, err := cc.DownloadWithOptions("scan-id", &DownloadOptions{AllowUnscannable: true})
	if err != nil {
		t.Fatalf("DownloadWithOptions should have succeeded: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
	if info.ContentType != "application/pdf" {
		t.Errorf("Expected application/pdf, got %s", info.ContentType)
	}
}

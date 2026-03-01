/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package webexsdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// --- parseLinkHeader tests ---

func TestParseLinkHeader_SingleNext(t *testing.T) {
	header := `<https://webexapis.com/v1/people?displayName=Harold&max=10&after=Y2lzY29>; rel="next"`
	links := parseLinkHeader(header)

	if links["next"] != "https://webexapis.com/v1/people?displayName=Harold&max=10&after=Y2lzY29" {
		t.Errorf("Expected next URL, got %q", links["next"])
	}
}

func TestParseLinkHeader_MultipleLinks(t *testing.T) {
	header := `<https://example.com/v1/items?page=2>; rel="next", <https://example.com/v1/items?page=0>; rel="prev"`
	links := parseLinkHeader(header)

	if links["next"] != "https://example.com/v1/items?page=2" {
		t.Errorf("Expected next URL, got %q", links["next"])
	}
	if links["prev"] != "https://example.com/v1/items?page=0" {
		t.Errorf("Expected prev URL, got %q", links["prev"])
	}
}

func TestParseLinkHeader_AllTypes(t *testing.T) {
	header := `<https://example.com/next>; rel="next", <https://example.com/prev>; rel="prev", <https://example.com/first>; rel="first"`
	links := parseLinkHeader(header)

	if links["next"] != "https://example.com/next" {
		t.Errorf("Expected next URL, got %q", links["next"])
	}
	if links["prev"] != "https://example.com/prev" {
		t.Errorf("Expected prev URL, got %q", links["prev"])
	}
	if links["first"] != "https://example.com/first" {
		t.Errorf("Expected first URL, got %q", links["first"])
	}
}

func TestParseLinkHeader_Empty(t *testing.T) {
	links := parseLinkHeader("")
	if len(links) != 0 {
		t.Errorf("Expected empty map, got %v", links)
	}
}

func TestParseLinkHeader_Malformed(t *testing.T) {
	// Should gracefully handle garbage
	links := parseLinkHeader("not a valid link header")
	if len(links) != 0 {
		t.Errorf("Expected empty map for malformed header, got %v", links)
	}
}

func TestParseLinkHeader_MissingRel(t *testing.T) {
	// Link without rel attribute — should be skipped
	header := `<https://example.com/something>`
	links := parseLinkHeader(header)
	if len(links) != 0 {
		t.Errorf("Expected empty map for link without rel, got %v", links)
	}
}

// --- NewPage with Link header tests ---

func TestNewPage_LinkHeader(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/items":
			// First page with Link header
			w.Header().Set("Link", fmt.Sprintf(`<%s/page2>; rel="next"`, serverURL))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item1"}, {"id": "item2"}]}`)
		case "/page2":
			// Second page with prev link
			w.Header().Set("Link", fmt.Sprintf(`<%s/items>; rel="prev"`, serverURL))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item3"}, {"id": "item4"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	// Get first page
	resp, err := client.Request(http.MethodGet, "items", nil, nil)
	if err != nil {
		t.Fatalf("Failed to get first page: %v", err)
	}

	page, err := NewPage(resp, client, "items")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Verify first page
	if len(page.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(page.Items))
	}
	if !page.HasNext {
		t.Error("Expected HasNext to be true")
	}
	if page.HasPrev {
		t.Error("Expected HasPrev to be false")
	}

	// Navigate to next page
	nextPage, err := page.Next()
	if err != nil {
		t.Fatalf("Failed to get next page: %v", err)
	}

	if len(nextPage.Items) != 2 {
		t.Errorf("Expected 2 items on second page, got %d", len(nextPage.Items))
	}
	if nextPage.HasNext {
		t.Error("Expected HasNext to be false on last page")
	}
	if !nextPage.HasPrev {
		t.Error("Expected HasPrev to be true on second page")
	}

	// Navigate back
	prevPage, err := nextPage.Prev()
	if err != nil {
		t.Fatalf("Failed to get previous page: %v", err)
	}
	if len(prevPage.Items) != 2 {
		t.Errorf("Expected 2 items on first page, got %d", len(prevPage.Items))
	}
}

func TestNewPage_NoLinkHeader_EmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// No Link header — final page
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"items": [{"id": "only-item"}]}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, err := client.Request(http.MethodGet, "items", nil, nil)
	if err != nil {
		t.Fatalf("Failed to get page: %v", err)
	}

	page, err := NewPage(resp, client, "items")
	if err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	if page.HasNext {
		t.Error("Expected HasNext to be false when no Link header")
	}
	if page.HasPrev {
		t.Error("Expected HasPrev to be false when no Link header")
	}
}

func TestNewPage_EmptyPageWithLinkHeader(t *testing.T) {
	// Per eval.md: "a Link header may sometimes point to an empty page if all data
	// was already returned on the previous page"
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/items":
			w.Header().Set("Link", fmt.Sprintf(`<%s/page2>; rel="next"`, serverURL))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item1"}]}`)
		case "/page2":
			// Empty page, no Link header — end of pagination
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"items": []}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, _ := client.Request(http.MethodGet, "items", nil, nil)
	page, err := NewPage(resp, client, "items")
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if !page.HasNext {
		t.Fatal("Expected HasNext on first page")
	}

	nextPage, err := page.Next()
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if len(nextPage.Items) != 0 {
		t.Errorf("Expected 0 items on empty page, got %d", len(nextPage.Items))
	}
	if nextPage.HasNext {
		t.Error("Expected HasNext false on final empty page")
	}
}

func TestNewPage_NoNextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"items": []}`)
	}))
	defer server.Close()

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	resp, _ := client.Request(http.MethodGet, "items", nil, nil)
	page, _ := NewPage(resp, client, "items")

	_, err := page.Next()
	if err == nil {
		t.Error("Expected error when calling Next() with no next page")
	}
	_, err = page.Prev()
	if err == nil {
		t.Error("Expected error when calling Prev() with no prev page")
	}
}

// --- Module pagination with Link headers ---

func TestModulePagination_WithLinkHeaders(t *testing.T) {
	// Simulate a module (rooms) using Link-based pagination
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/rooms":
			// Check if this is a paginated request
			if r.URL.Query().Get("after") == "cursor1" {
				// Second page - no Link header (last page)
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, `{"items": [{"id": "room3", "title": "Room 3"}]}`)
				return
			}
			// First page
			nextURL := fmt.Sprintf("%s/rooms?max=2&after=cursor1", serverURL)
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "room1", "title": "Room 1"}, {"id": "room2", "title": "Room 2"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	baseURL, _ := url.Parse(server.URL)
	config := &Config{
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	// First page
	resp, err := client.Request(http.MethodGet, "rooms", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	page, err := NewPage(resp, client, "rooms")
	if err != nil {
		t.Fatalf("NewPage failed: %v", err)
	}

	// Verify items
	if len(page.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(page.Items))
	}

	// Unmarshal first item
	var item struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(page.Items[0], &item); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if item.ID != "room1" {
		t.Errorf("Expected room1, got %s", item.ID)
	}

	if !page.HasNext {
		t.Fatal("Expected HasNext")
	}

	// Second page
	page2, err := page.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("Expected 1 item on page2, got %d", len(page2.Items))
	}
	if page2.HasNext {
		t.Error("Expected no next on final page")
	}
}

// --- RequestURL tests ---

func TestRequestURL_FullURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header is sent
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Bearer auth, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	config := &Config{
		BaseURL:        server.URL,
		HttpClient:     server.Client(),
		Timeout:        5 * time.Second,
		DefaultHeaders: make(map[string]string),
	}
	client, _ := NewClient("test-token", config)

	resp, err := client.RequestURL(http.MethodGet, server.URL+"/some/path?query=1", nil)
	if err != nil {
		t.Fatalf("RequestURL failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// --- PageFromCursor tests ---

func TestPageFromCursor_DirectNavigation(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/resources" && r.URL.Query().Get("cursor") == "":
			// Page 1
			w.Header().Set("Link", fmt.Sprintf(`<%s/v1/resources?cursor=page2>; rel="next"`, serverURL))
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item1"}, {"id": "item2"}]}`)
		case r.URL.Path == "/v1/resources" && r.URL.Query().Get("cursor") == "page2":
			// Page 2
			w.Header().Set("Link", fmt.Sprintf(`<%s/v1/resources?cursor=page3>; rel="next", <%s/v1/resources>; rel="prev"`, serverURL, serverURL))
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item3"}, {"id": "item4"}]}`)
		case r.URL.Path == "/v1/resources" && r.URL.Query().Get("cursor") == "page3":
			// Page 3 (last)
			w.Header().Set("Link", fmt.Sprintf(`<%s/v1/resources?cursor=page2>; rel="prev"`, serverURL))
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "item5"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	config := &Config{
		BaseURL:    server.URL + "/v1",
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)

	// Jump directly to page 2 — skip page 1 entirely
	page2, err := client.PageFromCursor(serverURL + "/v1/resources?cursor=page2")
	if err != nil {
		t.Fatalf("PageFromCursor failed: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Errorf("Expected 2 items on page 2, got %d", len(page2.Items))
	}
	if !page2.HasNext {
		t.Error("Expected HasNext=true on page 2")
	}
	if !page2.HasPrev {
		t.Error("Expected HasPrev=true on page 2")
	}

	// Chain: navigate from page 2 to page 3 using Next()
	page3, err := page2.Next()
	if err != nil {
		t.Fatalf("Next() from page 2 failed: %v", err)
	}
	if len(page3.Items) != 1 {
		t.Errorf("Expected 1 item on page 3, got %d", len(page3.Items))
	}
	if page3.HasNext {
		t.Error("Expected HasNext=false on page 3 (last page)")
	}

	// Chain: navigate back from page 3 to page 2 using Prev()
	backToPage2, err := page3.Prev()
	if err != nil {
		t.Fatalf("Prev() from page 3 failed: %v", err)
	}
	if len(backToPage2.Items) != 2 {
		t.Errorf("Expected 2 items going back to page 2, got %d", len(backToPage2.Items))
	}
}

func TestPageFromCursor_EmptyCursorError(t *testing.T) {
	config := &Config{BaseURL: "https://example.com"}
	client, _ := NewClient("test-token", config)

	_, err := client.PageFromCursor("")
	if err == nil {
		t.Fatal("Expected error for empty cursor URL")
	}
	if err.Error() != "cursor URL is empty" {
		t.Errorf("Expected 'cursor URL is empty' error, got: %v", err)
	}
}

func TestPageFromCursor_SaveAndResume(t *testing.T) {
	// Simulates saving a cursor from one pagination session and resuming later
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/items" && r.URL.Query().Get("cursor") == "":
			w.Header().Set("Link", fmt.Sprintf(`<%s/v1/items?cursor=abc123>; rel="next"`, serverURL))
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "a"}]}`)
		case r.URL.Path == "/v1/items" && r.URL.Query().Get("cursor") == "abc123":
			_, _ = fmt.Fprintln(w, `{"items": [{"id": "b"}, {"id": "c"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	baseURL, _ := url.Parse(server.URL + "/v1")
	config := &Config{
		BaseURL:    server.URL + "/v1",
		HttpClient: server.Client(),
	}
	client, _ := NewClient("test-token", config)
	client.BaseURL = baseURL

	// Session 1: get first page, save cursor
	resp, _ := client.Request(http.MethodGet, "items", nil, nil)
	page1, err := NewPage(resp, client, "items")
	if err != nil {
		t.Fatalf("NewPage failed: %v", err)
	}
	savedCursor := page1.NextPage
	if savedCursor == "" {
		t.Fatal("Expected a non-empty cursor to save")
	}
	t.Logf("Saved cursor: %s", savedCursor)

	// Session 2: create a new client, resume from cursor
	client2, _ := NewClient("test-token", config)
	client2.BaseURL = baseURL

	resumedPage, err := client2.PageFromCursor(savedCursor)
	if err != nil {
		t.Fatalf("PageFromCursor failed: %v", err)
	}
	if len(resumedPage.Items) != 2 {
		t.Errorf("Expected 2 items on resumed page, got %d", len(resumedPage.Items))
	}
	if resumedPage.HasNext {
		t.Error("Expected HasNext=false on last page")
	}
}

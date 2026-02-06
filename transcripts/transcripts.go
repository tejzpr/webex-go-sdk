/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package transcripts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// Transcript represents a Webex meeting transcript
type Transcript struct {
	ID              string `json:"id,omitempty"`
	MeetingID       string `json:"meetingId,omitempty"`
	HostEmail       string `json:"hostEmail,omitempty"`
	Title           string `json:"title,omitempty"`
	StartTime       string `json:"startTime,omitempty"`
	Status          string `json:"status,omitempty"`
	VttDownloadLink string `json:"vttDownloadLink,omitempty"`
	TxtDownloadLink string `json:"txtDownloadLink,omitempty"`
}

// Snippet represents a short segment of a transcript spoken by a specific participant
type Snippet struct {
	ID                string  `json:"id,omitempty"`
	TranscriptID      string  `json:"transcriptId,omitempty"`
	Text              string  `json:"text,omitempty"`
	PersonName        string  `json:"personName,omitempty"`
	PersonEmail       string  `json:"personEmail,omitempty"`
	PeopleID          string  `json:"peopleId,omitempty"`
	StartTime         string  `json:"startTime,omitempty"`
	EndTime           string  `json:"endTime,omitempty"`
	Duration          float64 `json:"duration,omitempty"`
	OffsetMillisecond int     `json:"offsetMillisecond,omitempty"`
	Language          string  `json:"language,omitempty"`
}

// ListOptions contains the options for listing transcripts
type ListOptions struct {
	MeetingID string `url:"meetingId,omitempty"`
	HostEmail string `url:"hostEmail,omitempty"`
	SiteURL   string `url:"siteUrl,omitempty"`
	From      string `url:"from,omitempty"`
	To        string `url:"to,omitempty"`
	Max       int    `url:"max,omitempty"`
}

// SnippetListOptions contains the options for listing transcript snippets
type SnippetListOptions struct {
	Max int `url:"max,omitempty"`
}

// TranscriptsPage represents a paginated list of transcripts
type TranscriptsPage struct {
	Items []Transcript `json:"items"`
	*webexsdk.Page
}

// SnippetsPage represents a paginated list of transcript snippets
type SnippetsPage struct {
	Items []Snippet `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Transcripts plugin
type Config struct {
	// Any configuration settings for the transcripts plugin can go here
}

// DefaultConfig returns the default configuration for the Transcripts plugin
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the transcripts API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Transcripts plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of meeting transcripts
func (c *Client) List(options *ListOptions) (*TranscriptsPage, error) {
	params := url.Values{}

	if options != nil {
		if options.MeetingID != "" {
			params.Set("meetingId", options.MeetingID)
		}
		if options.HostEmail != "" {
			params.Set("hostEmail", options.HostEmail)
		}
		if options.SiteURL != "" {
			params.Set("siteUrl", options.SiteURL)
		}
		if options.From != "" {
			params.Set("from", options.From)
		}
		if options.To != "" {
			params.Set("to", options.To)
		}
		if options.Max > 0 {
			params.Set("max", fmt.Sprintf("%d", options.Max))
		}
	}

	resp, err := c.webexClient.Request(http.MethodGet, "meetingTranscripts", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "meetingTranscripts")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Transcripts
	transcriptsPage := &TranscriptsPage{
		Page:  page,
		Items: make([]Transcript, len(page.Items)),
	}

	for i, item := range page.Items {
		var transcript Transcript
		if err := json.Unmarshal(item, &transcript); err != nil {
			return nil, err
		}
		transcriptsPage.Items[i] = transcript
	}

	return transcriptsPage, nil
}

// Download downloads the transcript content in the specified format.
// Format should be "vtt" or "txt". Defaults to "txt" if empty.
// Returns the raw transcript content as a string.
func (c *Client) Download(transcriptID string, format string) (string, error) {
	if transcriptID == "" {
		return "", fmt.Errorf("transcriptID is required")
	}

	if format == "" {
		format = "txt"
	}

	if format != "vtt" && format != "txt" {
		return "", fmt.Errorf("format must be 'vtt' or 'txt'")
	}

	params := url.Values{}
	params.Set("format", format)

	path := fmt.Sprintf("meetingTranscripts/%s/download", transcriptID)
	resp, err := c.webexClient.Request(http.MethodGet, path, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return string(body), nil
}

// ListSnippets returns a list of snippets for a transcript
func (c *Client) ListSnippets(transcriptID string, options *SnippetListOptions) (*SnippetsPage, error) {
	if transcriptID == "" {
		return nil, fmt.Errorf("transcriptID is required")
	}

	params := url.Values{}
	if options != nil {
		if options.Max > 0 {
			params.Set("max", fmt.Sprintf("%d", options.Max))
		}
	}

	path := fmt.Sprintf("meetingTranscripts/%s/snippets", transcriptID)
	resp, err := c.webexClient.Request(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, path)
	if err != nil {
		return nil, err
	}

	// Unmarshal items into Snippets
	snippetsPage := &SnippetsPage{
		Page:  page,
		Items: make([]Snippet, len(page.Items)),
	}

	for i, item := range page.Items {
		var snippet Snippet
		if err := json.Unmarshal(item, &snippet); err != nil {
			return nil, err
		}
		snippetsPage.Items[i] = snippet
	}

	return snippetsPage, nil
}

// GetSnippet returns a single transcript snippet
func (c *Client) GetSnippet(transcriptID, snippetID string) (*Snippet, error) {
	if transcriptID == "" {
		return nil, fmt.Errorf("transcriptID is required")
	}
	if snippetID == "" {
		return nil, fmt.Errorf("snippetID is required")
	}

	path := fmt.Sprintf("meetingTranscripts/%s/snippets/%s", transcriptID, snippetID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var snippet Snippet
	if err := webexsdk.ParseResponse(resp, &snippet); err != nil {
		return nil, err
	}

	return &snippet, nil
}

// UpdateSnippet updates a transcript snippet's text
func (c *Client) UpdateSnippet(transcriptID, snippetID string, snippet *Snippet) (*Snippet, error) {
	if transcriptID == "" {
		return nil, fmt.Errorf("transcriptID is required")
	}
	if snippetID == "" {
		return nil, fmt.Errorf("snippetID is required")
	}
	if snippet.Text == "" {
		return nil, fmt.Errorf("snippet text is required")
	}

	// Only send the updatable field
	updateData := &struct {
		Text string `json:"text"`
	}{
		Text: snippet.Text,
	}

	path := fmt.Sprintf("meetingTranscripts/%s/snippets/%s", transcriptID, snippetID)
	resp, err := c.webexClient.Request(http.MethodPut, path, nil, updateData)
	if err != nil {
		return nil, err
	}

	var result Snippet
	if err := webexsdk.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

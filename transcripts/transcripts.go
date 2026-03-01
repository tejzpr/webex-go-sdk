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
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// Transcript represents a Webex meeting transcript
type Transcript struct {
	ID                 string                  `json:"id,omitempty"`
	MeetingID          string                  `json:"meetingId,omitempty"`
	MeetingTopic       string                  `json:"meetingTopic,omitempty"`
	SiteURL            string                  `json:"siteUrl,omitempty"`
	ScheduledMeetingID string                  `json:"scheduledMeetingId,omitempty"`
	MeetingSeriesID    string                  `json:"meetingSeriesId,omitempty"`
	HostUserID         string                  `json:"hostUserId,omitempty"`
	HostEmail          string                  `json:"hostEmail,omitempty"`
	StartTime          string                  `json:"startTime,omitempty"`
	EndTime            string                  `json:"endTime,omitempty"`
	Duration           int                     `json:"duration,omitempty"`
	Status             string                  `json:"status,omitempty"`
	VttDownloadLink    string                  `json:"vttDownloadLink,omitempty"`
	TxtDownloadLink    string                  `json:"txtDownloadLink,omitempty"`
	Created            string                  `json:"created,omitempty"`
	Updated            string                  `json:"updated,omitempty"`
	Errors             webexsdk.ResourceErrors `json:"errors,omitempty"`
}

// Snippet represents a short segment of a transcript spoken by a specific participant
type Snippet struct {
	ID                  string                  `json:"id,omitempty"`
	TranscriptID        string                  `json:"transcriptId,omitempty"`
	Text                string                  `json:"text,omitempty"`
	PersonName          string                  `json:"personName,omitempty"`
	PersonEmail         string                  `json:"personEmail,omitempty"`
	PeopleID            string                  `json:"peopleId,omitempty"`
	StartTime           string                  `json:"startTime,omitempty"`
	EndTime             string                  `json:"endTime,omitempty"`
	Duration            float64                 `json:"duration,omitempty"`
	DurationMillisecond int                     `json:"durationMillisecond,omitempty"`
	OffsetMillisecond   int                     `json:"offsetMillisecond,omitempty"`
	Language            string                  `json:"language,omitempty"`
	Confidence          float64                 `json:"confidence,omitempty"`
	Errors              webexsdk.ResourceErrors `json:"errors,omitempty"`
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
	Max         int    `url:"max,omitempty"`
	PersonEmail string `url:"personEmail,omitempty"`
	PeopleID    string `url:"peopleId,omitempty"`
	From        string `url:"from,omitempty"`
	To          string `url:"to,omitempty"`
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

// List returns a list of meeting transcripts.
// The Webex API requires the time range between 'from' and 'to' to be within 30 days.
// If 'from' and 'to' are not specified and no meetingId is provided, the SDK defaults
// to the last 30 days to ensure results are returned.
func (c *Client) List(options *ListOptions) (*TranscriptsPage, error) {
	params := url.Values{}

	if options == nil {
		options = &ListOptions{}
	}

	if options.MeetingID != "" {
		params.Set("meetingId", options.MeetingID)
	}
	if options.HostEmail != "" {
		params.Set("hostEmail", options.HostEmail)
	}
	if options.SiteURL != "" {
		params.Set("siteUrl", options.SiteURL)
	}

	// Default to last 30 days if no date range and no meetingId specified,
	// since the Webex API may return empty results without a date range.
	if options.From == "" && options.To == "" && options.MeetingID == "" {
		now := time.Now().UTC()
		options.From = now.AddDate(0, 0, -30).Format(time.RFC3339)
		options.To = now.Format(time.RFC3339)
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

	resp, err := c.webexClient.Request(http.MethodGet, "meetingTranscripts", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, webexsdk.ResourceTranscripts)
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

// DownloadOptions contains optional parameters for downloading a transcript
type DownloadOptions struct {
	// MeetingID is the unique identifier of the meeting instance.
	// The Webex API download links include this parameter.
	MeetingID string
}

// Download downloads the transcript content in the specified format.
// Format should be "vtt" or "txt". Defaults to "txt" if empty.
// Returns the raw transcript content as a string.
// An optional DownloadOptions can be provided to include the meetingId parameter
// as returned by the Webex API in vttDownloadLink/txtDownloadLink.
func (c *Client) Download(transcriptID string, format string, opts ...*DownloadOptions) (string, error) {
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

	if len(opts) > 0 && opts[0] != nil && opts[0].MeetingID != "" {
		params.Set("meetingId", opts[0].MeetingID)
	}

	path := fmt.Sprintf("meetingTranscripts/%s/download", transcriptID)
	resp, err := c.webexClient.Request(http.MethodGet, path, params, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", webexsdk.NewAPIError(resp, body)
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
		if options.PersonEmail != "" {
			params.Set("personEmail", options.PersonEmail)
		}
		if options.PeopleID != "" {
			params.Set("peopleId", options.PeopleID)
		}
		if options.From != "" {
			params.Set("from", options.From)
		}
		if options.To != "" {
			params.Set("to", options.To)
		}
	}

	path := fmt.Sprintf("meetingTranscripts/%s/snippets", transcriptID)
	resp, err := c.webexClient.Request(http.MethodGet, path, params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, webexsdk.Resource(path))
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

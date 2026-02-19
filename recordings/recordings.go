/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package recordings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// Recording represents a Webex meeting recording
type Recording struct {
	ID                           string                  `json:"id,omitempty"`
	MeetingID                    string                  `json:"meetingId,omitempty"`
	ScheduledMeetingID           string                  `json:"scheduledMeetingId,omitempty"`
	MeetingSeriesID              string                  `json:"meetingSeriesId,omitempty"`
	Topic                        string                  `json:"topic,omitempty"`
	CreateTime                   string                  `json:"createTime,omitempty"`
	TimeRecorded                 string                  `json:"timeRecorded,omitempty"`
	HostEmail                    string                  `json:"hostEmail,omitempty"`
	SiteURL                      string                  `json:"siteUrl,omitempty"`
	DownloadURL                  string                  `json:"downloadUrl,omitempty"`
	PlaybackURL                  string                  `json:"playbackUrl,omitempty"`
	Password                     string                  `json:"password,omitempty"`
	TemporaryDirectDownloadLinks *TemporaryDownloadLinks `json:"temporaryDirectDownloadLinks,omitempty"`
	Format                       string                  `json:"format,omitempty"`
	DurationSeconds              int                     `json:"durationSeconds,omitempty"`
	SizeBytes                    int64                   `json:"sizeBytes,omitempty"`
	ShareToMe                    bool                    `json:"shareToMe,omitempty"`
	ServiceType                  string                  `json:"serviceType,omitempty"`
	Status                       string                  `json:"status,omitempty"`
}

// TemporaryDownloadLinks contains time-limited direct download URLs for
// the recording video, audio, and transcript. These links expire after
// a short period (typically 3 hours) as indicated by the Expiration field.
type TemporaryDownloadLinks struct {
	RecordingDownloadLink  string `json:"recordingDownloadLink,omitempty"`
	AudioDownloadLink      string `json:"audioDownloadLink,omitempty"`
	TranscriptDownloadLink string `json:"transcriptDownloadLink,omitempty"`
	Expiration             string `json:"expiration,omitempty"`
}

// DownloadedContent holds the raw bytes and metadata of a downloaded recording file.
type DownloadedContent struct {
	// ContentType is the MIME type (e.g., "video/mp4", "audio/mpeg").
	ContentType string
	// ContentDisposition is the Content-Disposition header value if present.
	ContentDisposition string
	// ContentLength is the size in bytes (-1 if unknown).
	ContentLength int64
	// Data is the raw file content.
	Data []byte
}

// ListOptions contains the options for listing recordings
type ListOptions struct {
	MeetingID       string `url:"meetingId,omitempty"`
	MeetingSeriesID string `url:"meetingSeriesId,omitempty"`
	HostEmail       string `url:"hostEmail,omitempty"`
	SiteURL         string `url:"siteUrl,omitempty"`
	ServiceType     string `url:"serviceType,omitempty"`
	From            string `url:"from,omitempty"`
	To              string `url:"to,omitempty"`
	Max             int    `url:"max,omitempty"`
	Status          string `url:"status,omitempty"`
	Topic           string `url:"topic,omitempty"`
	Format          string `url:"format,omitempty"`
}

// RecordingsPage represents a paginated list of recordings
type RecordingsPage struct {
	Items []Recording `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the Recordings plugin
type Config struct{}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the recordings API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Recordings plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// List returns a list of recordings.
// Use ListOptions to filter by meetingId, date range, host, etc.
func (c *Client) List(options *ListOptions) (*RecordingsPage, error) {
	params := url.Values{}

	if options != nil {
		if options.MeetingID != "" {
			params.Set("meetingId", options.MeetingID)
		}
		if options.MeetingSeriesID != "" {
			params.Set("meetingSeriesId", options.MeetingSeriesID)
		}
		if options.HostEmail != "" {
			params.Set("hostEmail", options.HostEmail)
		}
		if options.SiteURL != "" {
			params.Set("siteUrl", options.SiteURL)
		}
		if options.ServiceType != "" {
			params.Set("serviceType", options.ServiceType)
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
		if options.Status != "" {
			params.Set("status", options.Status)
		}
		if options.Topic != "" {
			params.Set("topic", options.Topic)
		}
		if options.Format != "" {
			params.Set("format", options.Format)
		}
	}

	resp, err := c.webexClient.Request(http.MethodGet, "recordings", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "recordings")
	if err != nil {
		return nil, err
	}

	recordingsPage := &RecordingsPage{
		Page:  page,
		Items: make([]Recording, len(page.Items)),
	}

	for i, item := range page.Items {
		var recording Recording
		if err := json.Unmarshal(item, &recording); err != nil {
			return nil, err
		}
		recordingsPage.Items[i] = recording
	}

	return recordingsPage, nil
}

// Get returns details for a single recording, including temporary direct download links
// for the video, audio, and transcript files.
func (c *Client) Get(recordingID string) (*Recording, error) {
	if recordingID == "" {
		return nil, fmt.Errorf("recordingID is required")
	}

	path := fmt.Sprintf("recordings/%s", recordingID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var recording Recording
	if err := webexsdk.ParseResponse(resp, &recording); err != nil {
		return nil, err
	}

	return &recording, nil
}

// Delete deletes a recording
func (c *Client) Delete(recordingID string) error {
	if recordingID == "" {
		return fmt.Errorf("recordingID is required")
	}

	path := fmt.Sprintf("recordings/%s", recordingID)
	resp, err := c.webexClient.Request(http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetAudioDownloadLink retrieves the temporary direct download link for the audio (MP3) of a recording.
// The link expires after a short period (check TemporaryDownloadLinks.Expiration).
// Returns the audio download URL and the full recording object.
func (c *Client) GetAudioDownloadLink(recordingID string) (string, *Recording, error) {
	recording, err := c.Get(recordingID)
	if err != nil {
		return "", nil, err
	}

	if recording.TemporaryDirectDownloadLinks == nil {
		return "", recording, fmt.Errorf("no temporary download links available for recording %s", recordingID)
	}

	if recording.TemporaryDirectDownloadLinks.AudioDownloadLink == "" {
		return "", recording, fmt.Errorf("no audio download link available for recording %s", recordingID)
	}

	return recording.TemporaryDirectDownloadLinks.AudioDownloadLink, recording, nil
}

// DownloadAudio downloads the audio (MP3) content of a recording.
// This first fetches the temporary download link, then downloads the audio file.
func (c *Client) DownloadAudio(recordingID string) (*DownloadedContent, error) {
	audioURL, _, err := c.GetAudioDownloadLink(recordingID)
	if err != nil {
		return nil, err
	}

	return c.downloadFromURL(audioURL)
}

// getDownloadLink retrieves a specific download link from a recording.
func (c *Client) getDownloadLink(recordingID, linkType string) (string, error) {
	recording, err := c.Get(recordingID)
	if err != nil {
		return "", err
	}

	if recording.TemporaryDirectDownloadLinks == nil {
		return "", fmt.Errorf("no temporary download links available for recording %s", recordingID)
	}

	var link string
	switch linkType {
	case "recording":
		link = recording.TemporaryDirectDownloadLinks.RecordingDownloadLink
	case "transcript":
		link = recording.TemporaryDirectDownloadLinks.TranscriptDownloadLink
	default:
		return "", fmt.Errorf("unknown link type: %s", linkType)
	}

	if link == "" {
		return "", fmt.Errorf("no %s download link available for recording %s", linkType, recordingID)
	}

	return link, nil
}

// DownloadRecording downloads the video recording (MP4) content.
// This first fetches the temporary download link, then downloads the recording file.
func (c *Client) DownloadRecording(recordingID string) (*DownloadedContent, error) {
	link, err := c.getDownloadLink(recordingID, "recording")
	if err != nil {
		return nil, err
	}
	return c.downloadFromURL(link)
}

// DownloadTranscript downloads the transcript file for a recording.
// This first fetches the temporary download link, then downloads the transcript.
func (c *Client) DownloadTranscript(recordingID string) (*DownloadedContent, error) {
	link, err := c.getDownloadLink(recordingID, "transcript")
	if err != nil {
		return nil, err
	}
	return c.downloadFromURL(link)
}

// downloadFromURL fetches content from a direct download URL.
func (c *Client) downloadFromURL(downloadURL string) (*DownloadedContent, error) {
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating download request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.webexClient.GetAccessToken())

	resp, err := c.webexClient.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download error: %d - %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading download body: %w", err)
	}

	return &DownloadedContent{
		ContentType:        resp.Header.Get("Content-Type"),
		ContentDisposition: resp.Header.Get("Content-Disposition"),
		ContentLength:      resp.ContentLength,
		Data:               data,
	}, nil
}

/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package contents

import (
	"fmt"
	"io"
	"net/http"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// FileInfo holds metadata about a downloaded file attachment.
type FileInfo struct {
	// ContentType is the MIME type of the file (e.g., "image/png", "application/pdf").
	ContentType string
	// ContentDisposition contains the original filename from the server (e.g., "attachment; filename=\"report.pdf\"").
	ContentDisposition string
	// ContentLength is the size in bytes (-1 if unknown).
	ContentLength int64
	// Data is the raw file content.
	Data []byte
}

// Config holds the configuration for the Contents plugin.
type Config struct{}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{}
}

// Client is the Contents API client for downloading file attachments.
// Webex file attachment URLs (returned in Message.Files) point to
// GET /v1/contents/{contentId} which requires Bearer token auth.
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
}

// New creates a new Contents plugin.
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	return &Client{
		webexClient: webexClient,
		config:      config,
	}
}

// Download fetches a file attachment by its content ID.
// The contentID is the Webex content identifier (the base64-encoded ID
// found at the end of URLs like https://webexapis.com/v1/contents/{contentId}).
func (c *Client) Download(contentID string) (*FileInfo, error) {
	if contentID == "" {
		return nil, fmt.Errorf("contentID is required")
	}

	path := fmt.Sprintf("contents/%s", contentID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading content body: %w", err)
	}

	return &FileInfo{
		ContentType:        resp.Header.Get("Content-Type"),
		ContentDisposition: resp.Header.Get("Content-Disposition"),
		ContentLength:      resp.ContentLength,
		Data:               data,
	}, nil
}

// DownloadFromURL fetches a file attachment from a full Webex content URL.
// This accepts the URLs directly from Message.Files (e.g.,
// "https://webexapis.com/v1/contents/Y2lzY29...").
func (c *Client) DownloadFromURL(contentURL string) (*FileInfo, error) {
	if contentURL == "" {
		return nil, fmt.Errorf("contentURL is required")
	}

	// Make a direct HTTP request to the full URL with auth
	req, err := http.NewRequest(http.MethodGet, contentURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.webexClient.GetAccessToken())

	resp, err := c.webexClient.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading content body: %w", err)
	}

	return &FileInfo{
		ContentType:        resp.Header.Get("Content-Type"),
		ContentDisposition: resp.Header.Get("Content-Disposition"),
		ContentLength:      resp.ContentLength,
		Data:               data,
	}, nil
}

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

// DownloadOptions configures file download behavior.
type DownloadOptions struct {
	// AllowUnscannable when true adds ?allow=unscannable to the request,
	// enabling download of files that cannot be scanned for malware
	// (e.g., encrypted files). The user assumes all risks.
	AllowUnscannable bool
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
//
// Automatic retry for file scanning (423):
//
//	The SDK automatically retries HTTP 423 (Locked) responses, which Webex
//	returns while a file is being scanned for malware. Retries honour the
//	Retry-After header and are governed by Config.MaxRetries (default 3)
//	and Config.RetryBaseDelay (default 1 s) on the underlying webexsdk.Client.
//	If scanning still hasn't completed after all retries, a structured
//	*webexsdk.APIError with StatusCode 423 is returned.
//
// Other anti-malware responses:
//   - Returns *webexsdk.APIError with 410 Gone if the file is infected.
//   - Returns *webexsdk.APIError with 428 Precondition Required if the file is unscannable.
//     Use DownloadWithOptions with AllowUnscannable=true to download such files.
func (c *Client) Download(contentID string) (*FileInfo, error) {
	return c.DownloadWithOptions(contentID, nil)
}

// DownloadWithOptions fetches a file attachment with configurable options.
// When opts.AllowUnscannable is true, ?allow=unscannable is appended to bypass
// the 428 Precondition Required response for unscannable (e.g., encrypted) files.
func (c *Client) DownloadWithOptions(contentID string, opts *DownloadOptions) (*FileInfo, error) {
	if contentID == "" {
		return nil, fmt.Errorf("contentID is required")
	}

	path := fmt.Sprintf("contents/%s", contentID)
	if opts != nil && opts.AllowUnscannable {
		path += "?allow=unscannable"
	}

	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, webexsdk.NewAPIError(resp, body)
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
//
// Automatic retry behaviour and anti-malware semantics are identical to
// [Download]; see its documentation for details.
func (c *Client) DownloadFromURL(contentURL string) (*FileInfo, error) {
	return c.DownloadFromURLWithOptions(contentURL, nil)
}

// DownloadFromURLWithOptions fetches a file from a full URL with configurable options.
// When opts.AllowUnscannable is true, ?allow=unscannable is appended to bypass
// the 428 Precondition Required response for unscannable files.
func (c *Client) DownloadFromURLWithOptions(contentURL string, opts *DownloadOptions) (*FileInfo, error) {
	if contentURL == "" {
		return nil, fmt.Errorf("contentURL is required")
	}

	requestURL := contentURL
	if opts != nil && opts.AllowUnscannable {
		// Append query parameter, handling existing query strings
		if hasQueryString(requestURL) {
			requestURL += "&allow=unscannable"
		} else {
			requestURL += "?allow=unscannable"
		}
	}

	resp, err := c.webexClient.RequestURL(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, webexsdk.NewAPIError(resp, body)
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

// hasQueryString returns true if the URL already has query parameters.
func hasQueryString(u string) bool {
	for i := 0; i < len(u); i++ {
		if u[i] == '?' {
			return true
		}
	}
	return false
}

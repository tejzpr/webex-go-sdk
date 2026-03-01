/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

// VoicemailClient provides methods for retrieving and managing voicemail messages.
type VoicemailClient struct {
	core    *webexsdk.Client
	config  *Config
	baseURL string
}

func newVoicemailClient(core *webexsdk.Client, config *Config) *VoicemailClient {
	return &VoicemailClient{
		core:    core,
		config:  config,
		baseURL: config.BaseURL,
	}
}

// doRequest is a helper that performs an HTTP request and returns the response body and status code.
func (c *VoicemailClient) doRequest(method, url string) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response: %w", err)
	}

	return body, resp.StatusCode, nil
}

// GetVoicemailList retrieves a list of voicemails with pagination and sorting.
//
// Parameters:
//   - offset: Number of records to skip.
//   - offsetLimit: Maximum number of voicemails to retrieve.
//   - sort: Sort order (ASC or DESC).
func (c *VoicemailClient) GetVoicemailList(offset, offsetLimit int, sort Sort) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages?offset=%s&limit=%s&sort=%s",
		c.baseURL,
		strconv.Itoa(offset),
		strconv.Itoa(offsetLimit),
		string(sort),
	)

	body, statusCode, err := c.doRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}

	if statusCode >= 200 && statusCode < 300 {
		if len(body) > 0 {
			var listResp struct {
				Items []VoicemailMessage `json:"items"`
			}
			if err := json.Unmarshal(body, &listResp); err != nil {
				return nil, fmt.Errorf("error parsing response: %w", err)
			}
			result.Data.VoicemailList = listResp.Items
		}
		result.Message = "SUCCESS"
	} else {
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			result.Data.Error = errResp.Message
		}
		result.Message = "FAILURE"
	}

	return result, nil
}

// GetVoicemailContent retrieves the content of a voicemail message by its messageId.
func (c *VoicemailClient) GetVoicemailContent(messageID string) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/%s/content", c.baseURL, messageID)

	body, statusCode, err := c.doRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}

	if statusCode >= 200 && statusCode < 300 {
		content := &VoicemailContent{}
		if err := json.Unmarshal(body, content); err != nil {
			// If not JSON, treat as raw audio content
			content.Type = "audio"
			content.Content = string(body)
		}
		result.Data.VoicemailContent = content
		result.Message = "SUCCESS"
	} else {
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			result.Data.Error = errResp.Message
		}
		result.Message = "FAILURE"
	}

	return result, nil
}

// GetVoicemailSummary retrieves a quantitative summary of voicemails for the user.
func (c *VoicemailClient) GetVoicemailSummary() (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/summary", c.baseURL)

	body, statusCode, err := c.doRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}

	if statusCode >= 200 && statusCode < 300 {
		if len(body) > 0 {
			summary := &VoicemailSummary{}
			if err := json.Unmarshal(body, summary); err != nil {
				return nil, fmt.Errorf("error parsing response: %w", err)
			}
			result.Data.VoicemailSummary = summary
		}
		result.Message = "SUCCESS"
	} else {
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			result.Data.Error = errResp.Message
		}
		result.Message = "FAILURE"
	}

	return result, nil
}

// MarkAsRead marks a voicemail message as read.
func (c *VoicemailClient) MarkAsRead(messageID string) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/%s/markAsRead", c.baseURL, messageID)

	_, statusCode, err := c.doRequest(http.MethodPost, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}
	if statusCode >= 200 && statusCode < 300 {
		result.Message = "SUCCESS"
	} else {
		result.Message = "FAILURE"
	}

	return result, nil
}

// MarkAsUnread marks a voicemail message as unread.
func (c *VoicemailClient) MarkAsUnread(messageID string) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/%s/markAsUnread", c.baseURL, messageID)

	_, statusCode, err := c.doRequest(http.MethodPost, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}
	if statusCode >= 200 && statusCode < 300 {
		result.Message = "SUCCESS"
	} else {
		result.Message = "FAILURE"
	}

	return result, nil
}

// Delete deletes a voicemail message by its messageId.
func (c *VoicemailClient) Delete(messageID string) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/%s", c.baseURL, messageID)

	_, statusCode, err := c.doRequest(http.MethodDelete, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}
	if statusCode >= 200 && statusCode < 300 {
		result.Message = "SUCCESS"
	} else {
		result.Message = "FAILURE"
	}

	return result, nil
}

// GetTranscript retrieves the transcript of a voicemail message.
func (c *VoicemailClient) GetTranscript(messageID string) (*VoicemailResponse, error) {
	url := fmt.Sprintf("%s/telephony/voiceMessages/%s/transcript", c.baseURL, messageID)

	body, statusCode, err := c.doRequest(http.MethodGet, url)
	if err != nil {
		return nil, err
	}

	result := &VoicemailResponse{StatusCode: statusCode}

	if statusCode >= 200 && statusCode < 300 {
		var transcriptResp struct {
			Transcript string `json:"transcript"`
		}
		if err := json.Unmarshal(body, &transcriptResp); err != nil {
			// If not JSON, treat the body as the transcript text
			transcript := string(body)
			result.Data.VoicemailTranscript = &transcript
		} else {
			result.Data.VoicemailTranscript = &transcriptResp.Transcript
		}
		result.Message = "SUCCESS"
	} else {
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil {
			result.Data.Error = errResp.Message
		}
		result.Message = "FAILURE"
	}

	return result, nil
}

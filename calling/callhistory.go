/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// CallHistoryClient provides methods for retrieving and managing call history records.
type CallHistoryClient struct {
	core    *webexsdk.Client
	config  *Config
	baseURL string
}

func newCallHistoryClient(core *webexsdk.Client, config *Config) *CallHistoryClient {
	return &CallHistoryClient{
		core:    core,
		config:  config,
		baseURL: config.BaseURL,
	}
}

// GetCallHistoryData retrieves call history records based on specified parameters.
//
// Parameters:
//   - days: Number of days to fetch call history data for.
//   - limit: Maximum number of records to fetch.
//   - sort: Sort order (ASC or DESC).
//   - sortBy: Field to sort by (endTime or startTime).
func (c *CallHistoryClient) GetCallHistoryData(days, limit int, sort Sort, sortBy SortBy) (*CallHistoryResponse, error) {
	url := fmt.Sprintf("%s/telephony/callHistory?days=%s&limit=%s&sort=%s&sortBy=%s",
		c.baseURL,
		strconv.Itoa(days),
		strconv.Itoa(limit),
		string(sort),
		string(sortBy),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	result := &CallHistoryResponse{
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var sessions struct {
			UserSessions []UserSession `json:"userSessions"`
		}
		if err := json.Unmarshal(body, &sessions); err != nil {
			return nil, fmt.Errorf("error parsing response: %w", err)
		}
		result.Data.UserSessions = sessions.UserSessions
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

// UpdateMissedCalls updates the read state of missed calls.
//
// Parameters:
//   - endTimeSessionIDs: An array of EndTimeSessionID identifying the missed call records to mark as read.
func (c *CallHistoryClient) UpdateMissedCalls(endTimeSessionIDs []EndTimeSessionID) (*UpdateMissedCallsResponse, error) {
	url := fmt.Sprintf("%s/telephony/callHistory/missedCalls", c.baseURL)

	payload := struct {
		EndTimeSessionIDs []EndTimeSessionID `json:"endTimeSessionIds"`
	}{
		EndTimeSessionIDs: endTimeSessionIDs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	result := &UpdateMissedCallsResponse{
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Data.ReadStatusMessage = "Missed calls updated successfully"
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

// DeleteCallHistoryRecords deletes call history records.
//
// Parameters:
//   - deleteSessionIDs: An array of EndTimeSessionID identifying the records to delete.
func (c *CallHistoryClient) DeleteCallHistoryRecords(deleteSessionIDs []EndTimeSessionID) (*DeleteCallHistoryResponse, error) {
	url := fmt.Sprintf("%s/telephony/callHistory/delete", c.baseURL)

	payload := struct {
		EndTimeSessionIDs []EndTimeSessionID `json:"endTimeSessionIds"`
	}{
		EndTimeSessionIDs: deleteSessionIDs,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.core.GetAccessToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.core.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	result := &DeleteCallHistoryResponse{
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Data.DeleteStatusMessage = "Call history records deleted successfully"
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

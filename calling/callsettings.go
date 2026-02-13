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

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// CallSettingsClient provides methods for retrieving and updating call settings
// such as Call Waiting, Do Not Disturb, Call Forwarding, and Voicemail configuration.
type CallSettingsClient struct {
	core    *webexsdk.Client
	config  *Config
	baseURL string
}

func newCallSettingsClient(core *webexsdk.Client, config *Config) *CallSettingsClient {
	return &CallSettingsClient{
		core:    core,
		config:  config,
		baseURL: config.BaseURL,
	}
}

// doSettingsRequest is a helper that performs an HTTP request and returns a CallSettingResponse.
func (c *CallSettingsClient) doSettingsRequest(method, url string, body interface{}) (*CallSettingResponse, error) {
	var reqBody io.Reader
	if body != nil {
		payloadBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling payload: %w", err)
		}
		reqBody = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	result := &CallSettingResponse{
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(respBody) > 0 {
			var raw json.RawMessage
			if err := json.Unmarshal(respBody, &raw); err == nil {
				result.Data.CallSetting = raw
			}
		}
		result.Message = "SUCCESS"
	} else {
		var errResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			result.Data.Error = errResp.Message
		}
		result.Message = "FAILURE"
	}

	return result, nil
}

// GetCallWaitingSetting fetches the call waiting setting for the authenticated user.
func (c *CallSettingsClient) GetCallWaitingSetting() (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/callWaiting", c.baseURL)
	return c.doSettingsRequest(http.MethodGet, url, nil)
}

// GetDoNotDisturbSetting fetches the Do Not Disturb (DND) setting.
func (c *CallSettingsClient) GetDoNotDisturbSetting() (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/doNotDisturb", c.baseURL)
	return c.doSettingsRequest(http.MethodGet, url, nil)
}

// SetDoNotDisturbSetting enables or disables Do Not Disturb.
func (c *CallSettingsClient) SetDoNotDisturbSetting(enabled bool) (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/doNotDisturb", c.baseURL)
	payload := ToggleSetting{Enabled: enabled}
	return c.doSettingsRequest(http.MethodPut, url, payload)
}

// GetCallForwardSetting fetches the call forwarding settings.
func (c *CallSettingsClient) GetCallForwardSetting() (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/callForwarding", c.baseURL)
	return c.doSettingsRequest(http.MethodGet, url, nil)
}

// SetCallForwardSetting updates the call forwarding settings.
func (c *CallSettingsClient) SetCallForwardSetting(setting CallForwardSetting) (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/callForwarding", c.baseURL)
	return c.doSettingsRequest(http.MethodPut, url, setting)
}

// GetVoicemailSetting fetches the voicemail settings.
func (c *CallSettingsClient) GetVoicemailSetting() (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/voicemail", c.baseURL)
	return c.doSettingsRequest(http.MethodGet, url, nil)
}

// SetVoicemailSetting updates the voicemail settings.
func (c *CallSettingsClient) SetVoicemailSetting(setting VoicemailSettingConfig) (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/voicemail", c.baseURL)
	return c.doSettingsRequest(http.MethodPut, url, setting)
}

// GetCallForwardAlwaysSetting fetches the call forward always setting.
// The optional directoryNumber parameter is only required for CCUC backends.
func (c *CallSettingsClient) GetCallForwardAlwaysSetting(directoryNumber string) (*CallSettingResponse, error) {
	url := fmt.Sprintf("%s/people/me/features/callForwarding/always", c.baseURL)
	if directoryNumber != "" {
		url += "?directoryNumber=" + directoryNumber
	}
	return c.doSettingsRequest(http.MethodGet, url, nil)
}

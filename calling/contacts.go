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

// ContactsClient provides methods for managing contacts and contact groups.
type ContactsClient struct {
	core    *webexsdk.Client
	config  *Config
	baseURL string
}

func newContactsClient(core *webexsdk.Client, config *Config) *ContactsClient {
	return &ContactsClient{
		core:    core,
		config:  config,
		baseURL: config.BaseURL,
	}
}

// doContactsRequest is a helper that performs an HTTP request and returns a ContactResponse.
func (c *ContactsClient) doContactsRequest(method, url string, body interface{}) (*ContactResponse, error) {
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

	result := &ContactResponse{
		StatusCode: resp.StatusCode,
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &result.Data); err != nil {
				return nil, fmt.Errorf("error parsing response: %w", err)
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

// GetContacts fetches the list of contacts and contact groups for the authenticated user.
func (c *ContactsClient) GetContacts() (*ContactResponse, error) {
	url := fmt.Sprintf("%s/people/me/contacts", c.baseURL)
	return c.doContactsRequest(http.MethodGet, url, nil)
}

// CreateContactGroup creates a new contact group with the given display name.
//
// Parameters:
//   - displayName: The name of the contact group.
//   - encryptionKeyURL: Optional encryption key URL.
//   - groupType: The type of group (NORMAL or EXTERNAL).
func (c *ContactsClient) CreateContactGroup(displayName, encryptionKeyURL string, groupType GroupType) (*ContactResponse, error) {
	url := fmt.Sprintf("%s/people/me/contactGroups", c.baseURL)

	payload := struct {
		DisplayName      string    `json:"displayName"`
		EncryptionKeyURL string    `json:"encryptionKeyUrl,omitempty"`
		GroupType        GroupType `json:"groupType"`
	}{
		DisplayName:      displayName,
		EncryptionKeyURL: encryptionKeyURL,
		GroupType:        groupType,
	}

	return c.doContactsRequest(http.MethodPost, url, payload)
}

// DeleteContactGroup deletes a contact group by its groupId.
func (c *ContactsClient) DeleteContactGroup(groupID string) (*ContactResponse, error) {
	url := fmt.Sprintf("%s/people/me/contactGroups/%s", c.baseURL, groupID)
	return c.doContactsRequest(http.MethodDelete, url, nil)
}

// CreateContact creates a new contact.
func (c *ContactsClient) CreateContact(contact Contact) (*ContactResponse, error) {
	url := fmt.Sprintf("%s/people/me/contacts", c.baseURL)
	return c.doContactsRequest(http.MethodPost, url, contact)
}

// DeleteContact deletes a contact by its contactId.
func (c *ContactsClient) DeleteContact(contactID string) (*ContactResponse, error) {
	url := fmt.Sprintf("%s/people/me/contacts/%s", c.baseURL, contactID)
	return c.doContactsRequest(http.MethodDelete, url, nil)
}

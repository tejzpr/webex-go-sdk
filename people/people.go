/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package people

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/tejzpr/webex-go-sdk/v2/webexsdk"
)

// BatchResponse represents the response from a batch people request
type BatchResponse struct {
	Items       []Person `json:"items"`
	NotFoundIDs []string `json:"notFoundIds,omitempty"`
}

// Person represents a Webex person
type Person struct {
	ID          string    `json:"id"`
	Emails      []string  `json:"emails"`
	DisplayName string    `json:"displayName"`
	NickName    string    `json:"nickName,omitempty"`
	FirstName   string    `json:"firstName,omitempty"`
	LastName    string    `json:"lastName,omitempty"`
	Avatar      string    `json:"avatar,omitempty"`
	OrgID       string    `json:"orgId,omitempty"`
	Roles       []string  `json:"roles,omitempty"`
	Licenses    []string  `json:"licenses,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Status      string    `json:"status,omitempty"`
	Type        string    `json:"type,omitempty"`
}

// ListOptions contains the options for listing people
type ListOptions struct {
	Email        string   `url:"email,omitempty"`
	DisplayName  string   `url:"displayName,omitempty"`
	IDs          []string `url:"-"` // Handled separately in the List method
	Max          int      `url:"max,omitempty"`
	ShowAllTypes bool     `url:"showAllTypes,omitempty"`
}

// PeoplePage represents a paginated list of people
type PeoplePage struct {
	Items []Person `json:"items"`
	*webexsdk.Page
}

// Config holds the configuration for the People plugin
type Config struct {
	// BatcherWait is the time to wait before processing a batch request
	BatcherWait time.Duration

	// MaxBatchCalls is the maximum number of batch calls to make at once
	MaxBatchCalls int

	// MaxBatchWait is the maximum time to wait before processing a batch request
	MaxBatchWait time.Duration

	// ShowAllTypes is a flag that requires the API to send every type field,
	// even if the type is not "person" (e.g.: SX10, webhook_integration, etc.)
	ShowAllTypes bool
}

// DefaultConfig returns the default configuration for the People plugin
func DefaultConfig() *Config {
	return &Config{
		BatcherWait:   100 * time.Millisecond,
		MaxBatchCalls: 10,
		MaxBatchWait:  1500 * time.Millisecond,
		ShowAllTypes:  false,
	}
}

// Batcher manages batched requests for people
type Batcher struct {
	webexClient *webexsdk.Client
	config      *Config

	// Map of request IDs to channels for handling responses
	requestChans map[string]chan *Person

	// Queue of pending requests
	requestQueue []string

	// Lock for concurrent access to requestChans and requestQueue
	mu sync.Mutex

	// Timer for batch processing
	timer *time.Timer

	// Flag to indicate if the batcher is running
	isRunning bool
}

// NewBatcher creates a new people batcher
func NewBatcher(client *webexsdk.Client, config *Config) *Batcher {
	batcher := &Batcher{
		webexClient:  client,
		config:       config,
		requestChans: make(map[string]chan *Person),
		requestQueue: make([]string, 0),
		timer:        time.NewTimer(config.BatcherWait),
	}

	// Stop the timer initially since we don't have any requests yet
	if !batcher.timer.Stop() {
		<-batcher.timer.C
	}

	return batcher
}

// Request adds a request to the batch and returns the result
func (b *Batcher) Request(id string) (*Person, error) {
	hydraID := InferPersonIDFromUUID(id)
	fmt.Println("hydraID: ", hydraID)
	b.mu.Lock()

	// Create a channel for this request
	resultChan := make(chan *Person, 1)
	b.requestChans[hydraID] = resultChan

	// Add to queue
	b.requestQueue = append(b.requestQueue, hydraID)

	// Start timer if not already running
	if !b.isRunning {
		b.timer.Reset(b.config.BatcherWait)
		b.isRunning = true

		// Start processing goroutine
		go b.processBatch()
	}

	b.mu.Unlock()

	// Wait for result
	result := <-resultChan
	if result == nil {
		return nil, fmt.Errorf("person not found: %s", id)
	}

	return result, nil
}

// BatchRequest processes a batch of requests immediately
func (b *Batcher) BatchRequest(ids []string) ([]Person, error) {
	// For empty list, return empty result
	if len(ids) == 0 {
		return []Person{}, nil
	}

	// Convert all IDs to Hydra IDs
	hydraIDs := make([]string, len(ids))
	for i, id := range ids {
		hydraIDs[i] = InferPersonIDFromUUID(id)
	}

	// Join IDs for the query - using proper format for filtering by ID
	// The /people endpoint accepts multiple id parameters for filtering
	idParam := url.Values{}
	for _, id := range hydraIDs {
		idParam.Add("id", id)
	}

	if b.config.ShowAllTypes {
		idParam.Add("showAllTypes", "true")
	}

	resp, err := b.webexClient.Request(http.MethodGet, "people", idParam, nil)
	if err != nil {
		return nil, err
	}

	var batchResp BatchResponse
	if err := webexsdk.ParseResponse(resp, &batchResp); err != nil {
		return nil, err
	}

	return batchResp.Items, nil
}

// processBatch processes the batch of requests
func (b *Batcher) processBatch() {
	// Wait for timer to expire
	<-b.timer.C

	b.mu.Lock()
	b.isRunning = false

	// Get the current batch of requests
	currentBatch := make([]string, len(b.requestQueue))
	copy(currentBatch, b.requestQueue)
	b.requestQueue = make([]string, 0)

	// Get response channels for this batch
	respChans := make(map[string]chan *Person)
	for _, id := range currentBatch {
		respChans[id] = b.requestChans[id]
		delete(b.requestChans, id)
	}

	b.mu.Unlock()

	// If there are no requests, just return
	if len(currentBatch) == 0 {
		return
	}

	// Process the batch - using the /people endpoint with id filter parameters
	// This follows the Webex API guidelines for filtering resources
	idParam := url.Values{}
	for _, id := range currentBatch {
		idParam.Add("id", id)
	}

	if b.config.ShowAllTypes {
		idParam.Add("showAllTypes", "true")
	}

	resp, err := b.webexClient.Request(http.MethodGet, "people", idParam, nil)

	if err != nil {
		// Failed to get response, send nil to all channels
		for _, ch := range respChans {
			ch <- nil
			close(ch)
		}
		return
	}

	var batchResp BatchResponse
	if err := webexsdk.ParseResponse(resp, &batchResp); err != nil {
		// Failed to parse response, send nil to all channels
		for _, ch := range respChans {
			ch <- nil
			close(ch)
		}
		return
	}

	// Create a map of IDs to persons
	personMap := make(map[string]*Person)
	for i := range batchResp.Items {
		person := &batchResp.Items[i]
		personMap[person.ID] = person
	}

	// Send results to channels
	for id, ch := range respChans {
		if person, ok := personMap[id]; ok {
			ch <- person
		} else {
			ch <- nil
		}
		close(ch)
	}

	// Check if we have new requests that came in while processing
	b.mu.Lock()
	hasNewRequests := len(b.requestQueue) > 0

	if hasNewRequests {
		b.timer.Reset(b.config.BatcherWait)
		b.isRunning = true
		go b.processBatch()
	}

	b.mu.Unlock()
}

// Client is the people API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
	batcher     *Batcher
}

// New creates a new People plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	client := &Client{
		webexClient: webexClient,
		config:      config,
	}

	client.batcher = NewBatcher(webexClient, config)

	return client
}

// Get returns a single person by ID
func (c *Client) Get(personID string) (*Person, error) {
	if personID == "" {
		return nil, fmt.Errorf("person ID is required")
	}

	if personID == "me" {
		return c.getMe()
	}

	// For single person retrieval, use direct API call with path parameter
	// instead of using the batcher
	path := fmt.Sprintf("people/%s", personID)
	resp, err := c.webexClient.Request(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var person Person
	if err := webexsdk.ParseResponse(resp, &person); err != nil {
		return nil, err
	}

	return &person, nil
}

// getMe fetches the current user from the /people/me endpoint
func (c *Client) getMe() (*Person, error) {
	resp, err := c.webexClient.Request(http.MethodGet, "people/me", nil, nil)
	if err != nil {
		return nil, err
	}

	var person Person
	if err := webexsdk.ParseResponse(resp, &person); err != nil {
		return nil, err
	}

	return &person, nil
}

// GetMe returns the current authenticated user
func (c *Client) GetMe() (*Person, error) {
	return c.getMe()
}

// List returns a list of people
func (c *Client) List(options *ListOptions) (*PeoplePage, error) {
	// Handle batch request if IDs are provided
	if options != nil && len(options.IDs) > 0 {
		persons, err := c.batcher.BatchRequest(options.IDs)
		if err != nil {
			return nil, err
		}

		return &PeoplePage{
			Items: persons,
		}, nil
	}

	// Standard list request
	params := url.Values{}
	if options != nil {
		if options.Email != "" {
			params.Set("email", options.Email)
		}
		if options.DisplayName != "" {
			params.Set("displayName", options.DisplayName)
		}
		if options.Max > 0 {
			params.Set("max", fmt.Sprintf("%d", options.Max))
		}
		if options.ShowAllTypes {
			params.Set("showAllTypes", "true")
		}
	}

	resp, err := c.webexClient.Request(http.MethodGet, "people", params, nil)
	if err != nil {
		return nil, err
	}

	page, err := webexsdk.NewPage(resp, c.webexClient, "people")
	if err != nil {
		return nil, err
	}

	// Unmarshal items into People
	peoplePage := &PeoplePage{
		Page:  page,
		Items: make([]Person, len(page.Items)),
	}

	for i, item := range page.Items {
		var person Person
		if err := json.Unmarshal(item, &person); err != nil {
			return nil, err
		}
		peoplePage.Items[i] = person
	}

	return peoplePage, nil
}

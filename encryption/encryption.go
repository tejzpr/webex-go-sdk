/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v1/webexsdk"
)

const (
	kmsURIPrefix     = "kms://"
	kmsRequestMethod = "retrieve"
	kmsRequestID     = "go-sdk-request"
	defaultCluster   = "a"
	defaultTimeout   = 10 * time.Second
)

// Config holds the configuration for the Encryption plugin
type Config struct {
	// HTTPTimeout is the timeout for HTTP requests to the KMS service
	HTTPTimeout time.Duration
	// DefaultCluster is the default KMS cluster to use when not specified
	DefaultCluster string
	// DisableCache disables the key caching mechanism
	DisableCache bool
}

// DefaultConfig returns the default configuration for the Encryption plugin
func DefaultConfig() *Config {
	return &Config{
		HTTPTimeout:    defaultTimeout,
		DefaultCluster: defaultCluster,
		DisableCache:   false,
	}
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`           // Key type
	Crv string `json:"crv,omitempty"` // Curve (for EC keys)
	X   string `json:"x,omitempty"`   // X coordinate (for EC keys)
	Y   string `json:"y,omitempty"`   // Y coordinate (for EC keys)
	D   string `json:"d,omitempty"`   // Private key (for EC keys)
	N   string `json:"n,omitempty"`   // Modulus (for RSA keys)
	E   string `json:"e,omitempty"`   // Exponent (for RSA keys)
	Kid string `json:"kid,omitempty"` // Key ID
	Alg string `json:"alg,omitempty"` // Algorithm
}

// Key represents a KMS key
type Key struct {
	URI string `json:"uri"` // URI of the key
	JWK JWK    `json:"jwk"` // JSON Web Key
}

// KMSMessage represents a message to/from the KMS service
type KMSMessage struct {
	Method      string                 `json:"method,omitempty"`      // Method to perform (retrieve, create, etc.)
	URI         string                 `json:"uri,omitempty"`         // URI of the key
	ResourceURI string                 `json:"resourceUri,omitempty"` // URI of the resource
	RequestID   string                 `json:"requestId,omitempty"`   // ID of the request
	Status      string                 `json:"status,omitempty"`      // Status of the response
	Key         *Key                   `json:"key,omitempty"`         // Key in the response
	Keys        []*Key                 `json:"keys,omitempty"`        // Multiple keys in the response
	UserIDs     []string               `json:"userIds,omitempty"`     // User IDs for key creation
	KeyURIs     []string               `json:"keyUris,omitempty"`     // Key URIs for batch operations
	Resource    map[string]interface{} `json:"resource,omitempty"`    // Resource data
}

// Client is the Encryption API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
	httpClient  *http.Client
	mu          sync.RWMutex // Use RWMutex for better concurrency
	keyCache    map[string]*Key
}

// New creates a new Encryption plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	return &Client{
		webexClient: webexClient,
		config:      config,
		httpClient:  httpClient,
		keyCache:    make(map[string]*Key),
	}
}

// GetKey retrieves a key from KMS
func (c *Client) GetKey(keyURI string) (*Key, error) {
	// Check cache first (using read lock)
	if !c.config.DisableCache {
		c.mu.RLock()
		if key, ok := c.keyCache[keyURI]; ok {
			c.mu.RUnlock()
			return key, nil
		}
		c.mu.RUnlock()
	}

	// Parse and validate the KMS URI
	domain, path, err := parseKMSURI(keyURI)
	if err != nil {
		return nil, err
	}

	// Create and execute KMS request
	key, err := c.retrieveKeyFromKMS(domain, path, keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key from KMS: %w", err)
	}

	// Cache the key (if caching is enabled)
	if !c.config.DisableCache {
		c.mu.Lock()
		c.keyCache[keyURI] = key
		c.mu.Unlock()
	}

	return key, nil
}

// parseKMSURI parses a KMS URI and returns the domain and path
func parseKMSURI(keyURI string) (domain string, path string, err error) {
	// Validate KMS URI format
	if !strings.HasPrefix(keyURI, kmsURIPrefix) {
		return "", "", fmt.Errorf("invalid KMS URI format (missing prefix): %s", keyURI)
	}

	// Extract the domain and path
	uriWithoutPrefix := keyURI[len(kmsURIPrefix):]
	parts := strings.SplitN(uriWithoutPrefix, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid KMS URI format (invalid structure): %s", keyURI)
	}

	return parts[0], parts[1], nil
}

// retrieveKeyFromKMS sends a request to the KMS service to retrieve a key
func (c *Client) retrieveKeyFromKMS(domain, path, keyURI string) (*Key, error) {
	// Create KMS request
	kmsMessage := &KMSMessage{
		Method:    kmsRequestMethod,
		URI:       keyURI,
		RequestID: kmsRequestID,
	}

	// Convert to JSON
	kmsRequestData, err := json.Marshal(kmsMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS request: %w", err)
	}

	// Determine KMS endpoint
	cluster := getClusterFromDomain(domain, c.config.DefaultCluster)
	kmsEndpoint := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/%s", cluster, path)

	// Create request with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kmsEndpoint, bytes.NewBuffer(kmsRequestData))
	if err != nil {
		return nil, fmt.Errorf("error creating KMS request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)

	// Send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making KMS request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading KMS response: %w", err)
	}

	// Check for HTTP error
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KMS request failed with status %d: %s", resp.StatusCode, respBody)
	}

	// Parse response
	var kmsResponse KMSMessage
	if err := json.Unmarshal(respBody, &kmsResponse); err != nil {
		return nil, fmt.Errorf("error parsing KMS response: %w", err)
	}

	// Check for KMS error
	if kmsResponse.Status != "success" {
		return nil, fmt.Errorf("KMS request failed with status: %s", kmsResponse.Status)
	}

	// Validate response
	if kmsResponse.Key == nil {
		return nil, fmt.Errorf("no key found in KMS response")
	}

	return kmsResponse.Key, nil
}

// getClusterFromDomain determines the appropriate cluster from the domain
func getClusterFromDomain(domain, defaultCluster string) string {
	// Default to "a" for cisco.com domain
	if domain == "cisco.com" {
		return "a"
	}
	return defaultCluster
}

// DecryptText decrypts JWE encrypted text using a KMS key
func (c *Client) DecryptText(keyURI string, ciphertext string) (string, error) {
	// Parameter validation
	if keyURI == "" {
		return "", fmt.Errorf("key URI is required")
	}
	if ciphertext == "" {
		return "", fmt.Errorf("ciphertext is required")
	}

	// Get the key from KMS
	key, err := c.GetKey(keyURI)
	if err != nil {
		return "", fmt.Errorf("error getting key: %w", err)
	}

	// Validate JWE format (should have 5 parts separated by periods)
	parts := strings.Split(ciphertext, ".")
	if len(parts) != 5 {
		return "", fmt.Errorf("invalid JWE format: expected 5 parts, got %d", len(parts))
	}

	// Try to decode the header to confirm it's a valid JWE
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("error decoding JWE header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", fmt.Errorf("error parsing JWE header: %w", err)
	}

	// Extract algorithm information
	alg, _ := header["alg"].(string)
	enc, _ := header["enc"].(string)

	// For now, return a message that explains what we would do with proper decryption
	// Note: This is a placeholder for actual decryption logic
	return fmt.Sprintf("[ENCRYPTED CONTENT] - This would be decrypted with key %s, algorithm %s, encryption %s",
		key.JWK.Kid, alg, enc), nil
}

// DecryptMessageContent attempts to decrypt message content using encryption key URL
func (c *Client) DecryptMessageContent(encryptionKeyURL string, encryptedContent string) (string, error) {
	// Parameter validation
	if encryptionKeyURL == "" {
		return "", fmt.Errorf("encryption key URL is required")
	}
	if encryptedContent == "" {
		return "", fmt.Errorf("encrypted content is required")
	}

	return c.DecryptText(encryptionKeyURL, encryptedContent)
}

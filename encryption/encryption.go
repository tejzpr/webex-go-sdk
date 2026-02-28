/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package encryption

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	jose "github.com/go-jose/go-jose/v4"
)

const (
	kmsURIPrefix   = "kms://"
	defaultCluster = "a"
	defaultTimeout = 10 * time.Second

	// kmsResponseTimeout is how long to wait for an async KMS response via Mercury.
	kmsResponseTimeout = 30 * time.Second
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
	Kty string `json:"kty"`           // Key type (oct, EC, RSA)
	K   string `json:"k,omitempty"`   // Key value (for symmetric/oct keys, base64url-encoded)
	Crv string `json:"crv,omitempty"` // Curve (for EC keys)
	X   string `json:"x,omitempty"`   // X coordinate (for EC keys)
	Y   string `json:"y,omitempty"`   // Y coordinate (for EC keys)
	D   string `json:"d,omitempty"`   // Private key (for EC keys)
	N   string `json:"n,omitempty"`   // Modulus (for RSA keys)
	E   string `json:"e,omitempty"`   // Exponent (for RSA keys)
	Kid string `json:"kid,omitempty"` // Key ID
	Alg string `json:"alg,omitempty"` // Algorithm
}

// SymmetricKey extracts the raw symmetric key bytes from an oct-type JWK.
// Returns an error if the key type is not "oct" or the key value is empty.
func (j *JWK) SymmetricKey() ([]byte, error) {
	if j.Kty != "oct" {
		return nil, fmt.Errorf("key type is %q, expected \"oct\" for symmetric key", j.Kty)
	}
	if j.K == "" {
		return nil, fmt.Errorf("symmetric key value (k) is empty")
	}
	return base64.RawURLEncoding.DecodeString(j.K)
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
	Status      interface{}            `json:"status,omitempty"`      // Status of the response (string or int)
	Key         *Key                   `json:"key,omitempty"`         // Key in the response
	Keys        []*Key                 `json:"keys,omitempty"`        // Multiple keys in the response
	UserIDs     []string               `json:"userIds,omitempty"`     // User IDs for key creation
	KeyURIs     []string               `json:"keyUris,omitempty"`     // Key URIs for batch operations
	Resource    map[string]interface{} `json:"resource,omitempty"`    // Resource data
	JWK         *JWK                   `json:"jwk,omitempty"`         // JWK for ECDH key exchange
}

// IsSuccess checks if the KMS response indicates success.
// Handles both string ("success") and numeric (200/201) status values.
func (m *KMSMessage) IsSuccess() bool {
	switch s := m.Status.(type) {
	case string:
		return s == "success" || s == "200" || s == "201"
	case float64:
		return int(s) == 200 || int(s) == 201
	case int:
		return s == 200 || s == 201
	default:
		return false
	}
}

// pendingKMSRequest represents an async KMS request awaiting a Mercury response.
type pendingKMSRequest struct {
	responseCh chan []byte       // Receives the decrypted KMS response payload
	ecdsaKey   *ecdsa.PrivateKey // For ECDH exchange responses (nil for shared-secret responses)
}

// inflightKeyRequest tracks an in-progress key retrieval so that concurrent
// callers requesting the same key URI share a single KMS round-trip.
type inflightKeyRequest struct {
	done chan struct{} // Closed when the retrieval completes
	key  *Key
	err  error
}

// Client is the Encryption API client
type Client struct {
	webexClient *webexsdk.Client
	config      *Config
	httpClient  *http.Client

	// Key cache
	mu       sync.RWMutex
	keyCache map[string]*Key

	// In-flight key retrieval deduplication (singleflight pattern)
	inflightMu   sync.Mutex
	inflightKeys map[string]*inflightKeyRequest

	// ECDH context for KMS communication
	ecdhMu       sync.Mutex
	ecdhCond     *sync.Cond
	ecdhCreating bool
	ecdhCtx      *ECDHContext
	deviceURL    string // Device URL used as KMS client ID
	userID       string // User ID for KMS requests

	// Pending async KMS requests (correlated via requestId)
	pendingMu       sync.Mutex
	pendingRequests map[string]*pendingKMSRequest
}

// New creates a new Encryption plugin
func New(webexClient *webexsdk.Client, config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	c := &Client{
		webexClient:     webexClient,
		config:          config,
		httpClient:      httpClient,
		keyCache:        make(map[string]*Key),
		inflightKeys:    make(map[string]*inflightKeyRequest),
		pendingRequests: make(map[string]*pendingKMSRequest),
	}
	c.ecdhCond = sync.NewCond(&c.ecdhMu)
	return c
}

// SetDeviceInfo sets the device URL and user ID for KMS communication.
// The device URL is used as the client ID in KMS requests.
// The user ID is used to look up the KMS cluster info.
func (c *Client) SetDeviceInfo(deviceURL, userID string) {
	c.ecdhMu.Lock()
	defer c.ecdhMu.Unlock()
	c.deviceURL = deviceURL
	c.userID = userID
}

// GetKey retrieves a key from KMS.
// It uses a singleflight pattern: if multiple goroutines request the same
// key URI concurrently, only one KMS round-trip is made and the result is
// shared with all waiters. This prevents thundering-herd flooding of KMS
// when many encrypted messages arrive at once referencing the same key.
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
	if _, _, err := parseKMSURI(keyURI); err != nil {
		return nil, err
	}

	// Singleflight: check if a retrieval is already in-flight for this URI
	c.inflightMu.Lock()
	if inflight, ok := c.inflightKeys[keyURI]; ok {
		// Another goroutine is already fetching this key — wait for it
		c.inflightMu.Unlock()
		<-inflight.done
		if inflight.err != nil {
			return nil, fmt.Errorf("failed to retrieve key from KMS (shared): %w", inflight.err)
		}
		return inflight.key, nil
	}

	// First caller: register and proceed with the retrieval
	inflight := &inflightKeyRequest{done: make(chan struct{})}
	c.inflightKeys[keyURI] = inflight
	c.inflightMu.Unlock()

	// Ensure we always clean up and wake waiters
	defer func() {
		close(inflight.done)
		c.inflightMu.Lock()
		delete(c.inflightKeys, keyURI)
		c.inflightMu.Unlock()
	}()

	// Retrieve key using KMS protocol
	key, err := c.retrieveKeyFromKMS(keyURI)
	if err != nil {
		inflight.err = err
		return nil, fmt.Errorf("failed to retrieve key from KMS: %w", err)
	}

	inflight.key = key

	// Cache the key (if caching is enabled)
	if !c.config.DisableCache {
		c.mu.Lock()
		c.keyCache[keyURI] = key
		c.mu.Unlock()
	}

	return key, nil
}

// CacheKey manually caches a key. This is used when keys arrive via
// Mercury WebSocket events (e.g., encryption.kmsMessages).
func (c *Client) CacheKey(key *Key) {
	if key == nil || key.URI == "" {
		return
	}
	c.mu.Lock()
	c.keyCache[key.URI] = key
	c.mu.Unlock()
}

// registerPendingRequest registers an async KMS request that is waiting for a
// response via Mercury WebSocket. Returns a channel that will receive the
// decrypted response payload. For ECDH exchange requests, pass the ECDSA
// private key so ProcessKMSMessages can decrypt the ECDH-ES response.
func (c *Client) registerPendingRequest(requestID string, ecdsaKey *ecdsa.PrivateKey) chan []byte {
	ch := make(chan []byte, 1)
	c.pendingMu.Lock()
	c.pendingRequests[requestID] = &pendingKMSRequest{
		responseCh: ch,
		ecdsaKey:   ecdsaKey,
	}
	c.pendingMu.Unlock()
	return ch
}

// unregisterPendingRequest removes a pending request (cleanup after timeout or delivery).
func (c *Client) unregisterPendingRequest(requestID string) {
	c.pendingMu.Lock()
	delete(c.pendingRequests, requestID)
	c.pendingMu.Unlock()
}

// ProcessKMSMessages processes KMS messages received via Mercury WebSocket events.
// These messages are JWE-encrypted and may contain:
// - ECDH exchange responses (encrypted with client's ECDH public key)
// - Key retrieval responses (encrypted with the ECDH shared secret)
// - Key rotation notifications
//
// Messages matching a pending request (via requestId) are delivered to the
// waiting goroutine. Others are processed for key caching.
func (c *Client) ProcessKMSMessages(jweStrings []string) {
	// Read the current ECDH context (shared secret) for decryption
	c.ecdhMu.Lock()
	ecdhCtx := c.ecdhCtx
	c.ecdhMu.Unlock()

	// Collect pending ECDH private keys for ECDH exchange response decryption
	c.pendingMu.Lock()
	pendingECDHKeys := make(map[string]*ecdsa.PrivateKey)
	for reqID, req := range c.pendingRequests {
		if req.ecdsaKey != nil {
			pendingECDHKeys[reqID] = req.ecdsaKey
		}
	}
	c.pendingMu.Unlock()

	for _, jweStr := range jweStrings {
		if jweStr == "" {
			continue
		}

		var plaintext []byte

		// KMS responses from Mercury may be:
		// - JWS-signed (3 parts, 2 dots): ECDH exchange responses with plaintext JSON payload
		// - JWE-encrypted (5 parts, 4 dots): Key retrieval responses encrypted with shared secret
		dotCount := strings.Count(jweStr, ".")
		if dotCount == 2 {
			// JWS: extract the payload (2nd part) which is the plaintext response
			parts := strings.SplitN(jweStr, ".", 3)
			if len(parts) == 3 {
				payload, err := base64.RawURLEncoding.DecodeString(parts[1])
				if err == nil && len(payload) > 0 {
					if payload[0] == '{' {
						// Payload is plaintext JSON (ECDH exchange response)
						plaintext = payload
					} else {
						// Payload might be a JWE - try decrypting it below
						jweStr = string(payload)
					}
				}
			}
		}

		// If not yet decoded (JWE compact format), try decryption
		if plaintext == nil && strings.Count(jweStr, ".") == 4 {
			// Try 1: Decrypt with ECDH shared secret (for key retrieval responses)
			if ecdhCtx != nil && ecdhCtx.sharedSecret != nil {
				decrypted, err := unwrapWithSharedSecret(jweStr, ecdhCtx.sharedSecret)
				if err == nil {
					plaintext = decrypted
				}
			}

			// Try 2: Decrypt with pending ECDH private keys (for ECDH exchange responses)
			if plaintext == nil {
				for _, ecKey := range pendingECDHKeys {
					jweObj, parseErr := jose.ParseEncrypted(jweStr,
						[]jose.KeyAlgorithm{jose.ECDH_ES, jose.ECDH_ES_A256KW, jose.DIRECT, jose.RSA_OAEP},
						[]jose.ContentEncryption{jose.A256GCM, jose.A128GCM})
					if parseErr != nil {
						continue
					}
					decrypted, err := jweObj.Decrypt(ecKey)
					if err == nil {
						plaintext = decrypted
						break
					}
				}
			}
		}

		// For strings that start with '{' directly (JSON without JWS/JWE wrapper)
		if plaintext == nil && len(jweStr) > 0 && jweStr[0] == '{' {
			plaintext = []byte(jweStr)
		}

		if plaintext == nil {
			continue
		}

		// Parse the decrypted/extracted message
		var msg KMSMessage
		if err := json.Unmarshal(plaintext, &msg); err != nil {
			continue
		}

		// Check if this matches a pending request (deliver via channel)
		if msg.RequestID != "" {
			c.pendingMu.Lock()
			if req, ok := c.pendingRequests[msg.RequestID]; ok {
				select {
				case req.responseCh <- plaintext:
				default:
				}
				delete(c.pendingRequests, msg.RequestID)
				c.pendingMu.Unlock()
				continue
			}
			c.pendingMu.Unlock()
		}

		// Not a pending request match - cache any keys from the message
		if msg.Key != nil {
			c.CacheKey(msg.Key)
		}
		for _, key := range msg.Keys {
			c.CacheKey(key)
		}
	}
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

// DecryptText decrypts JWE-encrypted text using a KMS key.
// The ciphertext must be in JWE compact serialization format (5 dot-separated parts).
// Supports alg:dir + enc:A256GCM as used by Webex end-to-end encryption.
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

	// Extract raw symmetric key bytes from the JWK
	rawKey, err := key.JWK.SymmetricKey()
	if err != nil {
		return "", fmt.Errorf("error extracting symmetric key: %w", err)
	}

	// Parse the JWE compact serialization, restricting to the algorithms
	// used by Webex: direct key agreement with AES-256-GCM content encryption
	jweObj, err := jose.ParseEncrypted(ciphertext,
		[]jose.KeyAlgorithm{jose.DIRECT},
		[]jose.ContentEncryption{jose.A256GCM})
	if err != nil {
		return "", fmt.Errorf("error parsing JWE: %w", err)
	}

	// Decrypt with the symmetric key
	plaintext, err := jweObj.Decrypt(rawKey)
	if err != nil {
		return "", fmt.Errorf("error decrypting JWE: %w", err)
	}

	return string(plaintext), nil
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

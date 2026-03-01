/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package encryption

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// KMSEnvelope is the HTTP request/response envelope for
// POST /encryption/api/v1/kms/messages
type KMSEnvelope struct {
	KMSMessages []string `json:"kmsMessages"`
	Destination string   `json:"destination,omitempty"`
}

// KMSRequestPayload is the full KMS request payload (inside the JWE).
// This includes the client credentials needed for KMS authentication.
type KMSRequestPayload struct {
	Client    *KMSClient `json:"client"`
	RequestID string     `json:"requestId"`
	Method    string     `json:"method"`
	URI       string     `json:"uri"`
	JWK       *JWK       `json:"jwk,omitempty"` // For ECDH exchange
}

// KMSClient identifies the client making a KMS request.
type KMSClient struct {
	ClientID   string         `json:"clientId"`
	Credential *KMSCredential `json:"credential"`
}

// KMSCredential holds authentication info for KMS requests.
type KMSCredential struct {
	UserID string `json:"userId"`
	Bearer string `json:"bearer"`
}

// retrieveKeyFromKMS retrieves a key from KMS using the ECDH-based protocol.
// The protocol is asynchronous: HTTP requests return 202, and responses
// are delivered via Mercury WebSocket events.
func (c *Client) retrieveKeyFromKMS(keyURI string) (*Key, error) {
	key, err := c.retrieveKeyViaECDH(keyURI)
	if err != nil {
		return nil, fmt.Errorf("ECDH retrieval: %w", err)
	}
	return key, nil
}

// isECDHSessionError determines whether a KMS retrieval error indicates
// that the ECDH session itself is invalid (shared secret mismatch, rejected
// credentials, etc.) vs. a transient error (network timeout, server 500, rate
// limit) where retrying with the same ECDH context is appropriate.
func isECDHSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// HTTP 400/403 from KMS typically mean the shared secret or kid is wrong
	if strings.Contains(msg, "status 400") || strings.Contains(msg, "status 403") {
		return true
	}
	// Decryption failures mean the shared secret doesn't match
	if strings.Contains(msg, "error decrypting") || strings.Contains(msg, "failed with status") {
		return true
	}
	// Everything else (timeouts, 500s, 429s, network errors) is transient
	return false
}

// retrieveKeyViaECDH retrieves a key using the ECDH-encrypted KMS protocol.
// On failure it classifies the error: ECDH-session errors (400, 403, decrypt
// failures) trigger ECDH invalidation + re-exchange, while transient errors
// (timeouts, 500, rate limits) are retried with the existing ECDH context.
func (c *Client) retrieveKeyViaECDH(keyURI string) (*Key, error) {
	// Ensure ECDH context exists
	ecdhCtx, err := c.getOrCreateECDH()
	if err != nil {
		return nil, fmt.Errorf("failed to establish ECDH context: %w", err)
	}

	// Build and send the KMS retrieve request
	key, err := c.doKMSRetrieve(keyURI, ecdhCtx)
	if err != nil {
		if isECDHSessionError(err) {
			// ECDH session is invalid — invalidate and re-exchange
			c.invalidateECDH()
			ecdhCtx, retryErr := c.getOrCreateECDH()
			if retryErr != nil {
				return nil, fmt.Errorf("retry ECDH failed: %w (original: %v)", retryErr, err)
			}
			key, retryErr = c.doKMSRetrieve(keyURI, ecdhCtx)
			if retryErr != nil {
				return nil, fmt.Errorf("retry KMS retrieve failed: %w (original: %v)", retryErr, err)
			}
			return key, nil
		}

		// Transient error — retry once with the same ECDH context
		key, retryErr := c.doKMSRetrieve(keyURI, ecdhCtx)
		if retryErr != nil {
			return nil, fmt.Errorf("retry KMS retrieve failed: %w (original: %v)", retryErr, err)
		}
		return key, nil
	}

	return key, nil
}

// doKMSRetrieve performs a single KMS key retrieval attempt.
// It sends the request via HTTP, and if the response is async (202),
// waits for the response to arrive via Mercury WebSocket.
func (c *Client) doKMSRetrieve(keyURI string, ecdhCtx *ECDHContext) (*Key, error) {
	userID, _ := c.getUserID()
	requestID := generateRequestID()

	// Register pending request BEFORE sending so we don't miss the response
	responseCh := c.registerPendingRequest(requestID, nil)
	defer c.unregisterPendingRequest(requestID)

	// Determine the destination cluster from the key URI
	domain, _, _ := parseKMSURI(keyURI)
	destination := kmsClusterFromDomain(domain, ecdhCtx.kmsCluster)

	// Create KMS retrieve request payload.
	// KMS expects the UUID format for userId.
	kmsRequest := &KMSRequestPayload{
		Client: &KMSClient{
			ClientID: c.getClientID(),
			Credential: &KMSCredential{
				UserID: decodeWebexID(userID),
				Bearer: c.webexClient.GetAccessToken(),
			},
		},
		RequestID: requestID,
		Method:    "retrieve",
		URI:       keyURI,
	}

	requestPayload, err := json.Marshal(kmsRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS request: %w", err)
	}

	// Wrap with ECDH shared secret (dir + A256GCM), using the ECDHE key URI as kid
	wrappedRequest, err := wrapWithSharedSecret(requestPayload, ecdhCtx.sharedSecret, ecdhCtx.ecdhKeyURI)
	if err != nil {
		return nil, fmt.Errorf("error wrapping KMS request: %w", err)
	}

	// Send to KMS
	responseJWEs, err := c.sendKMSMessage(wrappedRequest, destination)
	if err != nil {
		return nil, fmt.Errorf("KMS request failed: %w", err)
	}

	// If we got a synchronous response (200 OK), process it directly
	if len(responseJWEs) > 0 {
		return c.processKeyResponseJWEs(responseJWEs, ecdhCtx)
	}

	// Otherwise, wait for async response via Mercury (202 Accepted)
	select {
	case payload := <-responseCh:
		return c.parseKeyFromPayload(payload)
	case <-time.After(kmsResponseTimeout):
		return nil, fmt.Errorf("timeout waiting for KMS key response via Mercury")
	}
}

// processKeyResponseJWEs processes synchronous KMS response JWEs.
func (c *Client) processKeyResponseJWEs(responseJWEs []string, ecdhCtx *ECDHContext) (*Key, error) {
	for _, respJWE := range responseJWEs {
		responsePayload, err := unwrapWithSharedSecret(respJWE, ecdhCtx.sharedSecret)
		if err != nil {
			continue
		}
		return c.parseKeyFromPayload(responsePayload)
	}
	return nil, fmt.Errorf("no key found in KMS response JWEs")
}

// parseKeyFromPayload parses a key from a decrypted KMS response payload.
func (c *Client) parseKeyFromPayload(payload []byte) (*Key, error) {
	var kmsResponse KMSMessage
	if err := json.Unmarshal(payload, &kmsResponse); err != nil {
		return nil, fmt.Errorf("error parsing KMS response: %w", err)
	}

	if !kmsResponse.IsSuccess() {
		return nil, fmt.Errorf("KMS request failed with status: %v", kmsResponse.Status)
	}

	if kmsResponse.Key != nil {
		return kmsResponse.Key, nil
	}
	if len(kmsResponse.Keys) > 0 {
		return kmsResponse.Keys[0], nil
	}
	return nil, fmt.Errorf("no key found in KMS response")
}

// sendKMSMessage sends a wrapped KMS message to the encryption service.
// Returns response JWEs for synchronous (200) responses, or nil for async (202) responses.
func (c *Client) sendKMSMessage(wrappedMessage string, destination string) ([]string, error) {
	envelope := &KMSEnvelope{
		KMSMessages: []string{wrappedMessage},
		Destination: destination,
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS envelope: %w", err)
	}

	// Always send to the default cluster's encryption endpoint.
	// The 'destination' field in the envelope handles routing to the correct KMS cluster.
	kmsEndpoint := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/kms/messages",
		c.config.DefaultCluster)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kmsEndpoint, bytes.NewBuffer(envelopeJSON))
	if err != nil {
		return nil, fmt.Errorf("error creating KMS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.webexClient.GetAccessToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending KMS request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading KMS response: %w", err)
	}

	// 202 Accepted: response will come asynchronously via Mercury WebSocket
	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KMS request failed with status %d: %s", resp.StatusCode, respBody)
	}

	// 200 OK: parse synchronous response
	var responseEnvelope KMSEnvelope
	if err := json.Unmarshal(respBody, &responseEnvelope); err != nil {
		return nil, fmt.Errorf("error parsing KMS response envelope: %w", err)
	}

	return responseEnvelope.KMSMessages, nil
}

// --- JWE Wrapping/Unwrapping Helpers ---

// wrapWithSharedSecret encrypts payload using dir + A256GCM with the ECDH shared secret.
// The kid (Key ID) is the ECDHE key URI from the KMS ECDH exchange response, which
// tells the KMS server which shared secret to use for decryption.
func wrapWithSharedSecret(payload []byte, sharedSecret []byte, kid string) (string, error) {
	recipient := jose.Recipient{
		Algorithm: jose.DIRECT,
		Key:       sharedSecret,
		KeyID:     kid,
	}

	encrypter, err := jose.NewEncrypter(jose.A256GCM, recipient, nil)
	if err != nil {
		return "", fmt.Errorf("error creating encrypter: %w", err)
	}

	jweObj, err := encrypter.Encrypt(payload)
	if err != nil {
		return "", fmt.Errorf("error encrypting payload: %w", err)
	}

	return jweObj.CompactSerialize()
}

// unwrapWithSharedSecret decrypts a JWE using the ECDH shared secret.
func unwrapWithSharedSecret(jweString string, sharedSecret []byte) ([]byte, error) {
	jweObj, err := jose.ParseEncrypted(jweString,
		[]jose.KeyAlgorithm{jose.DIRECT},
		[]jose.ContentEncryption{jose.A256GCM})
	if err != nil {
		return nil, fmt.Errorf("error parsing JWE: %w", err)
	}

	plaintext, err := jweObj.Decrypt(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("error decrypting JWE: %w", err)
	}

	return plaintext, nil
}

// --- Utility Helpers ---

// kmsClusterFromDomain determines the KMS cluster destination from a key URI domain.
// The destination is a plain hostname like "kms-cisco.wbx2.com" (without kms:// prefix).
// For domains like "kms-cisco.wbx2.com", returns the domain as-is (it's already a cluster).
// For domains like "cisco.com", returns the user's default KMS cluster.
func kmsClusterFromDomain(domain, defaultKMSCluster string) string {
	// Strip kms:// prefix if present (KMS cluster URLs from the API may have it)
	clean := func(s string) string {
		return strings.TrimPrefix(s, "kms://")
	}

	if domain == "" {
		return clean(defaultKMSCluster)
	}

	domain = clean(domain)

	// Full KMS cluster URLs (e.g., "kms-a.wbx2.com", "kms-cisco.wbx2.com")
	if strings.HasPrefix(domain, "kms-") && strings.Contains(domain, ".wbx2.com") {
		return domain
	}

	// Simple cluster names
	if len(domain) <= 5 && !strings.Contains(domain, ".") {
		return domain
	}

	// Known Webex domains -> use default cluster
	return clean(defaultKMSCluster)
}

// getClusterFromDomain determines the cluster identifier from a KMS domain or cluster URL.
// Used for constructing encryption endpoint URLs.
func getClusterFromDomain(domain, defaultCluster string) string {
	if domain == "" {
		return defaultCluster
	}

	// Handle full KMS cluster URLs like "kms-a.wbx2.com"
	if strings.HasPrefix(domain, "kms-") && strings.Contains(domain, ".wbx2.com") {
		parts := strings.SplitN(strings.TrimPrefix(domain, "kms-"), ".", 2)
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Handle simple cluster IDs (e.g., "a", "b")
	if len(domain) <= 5 && !strings.Contains(domain, ".") {
		return domain
	}

	// Handle known domains
	if domain == "cisco.com" || domain == "ciscospark.com" {
		return "a"
	}

	return defaultCluster
}

// getClientID returns the client ID for KMS requests.
// Uses the device URL if set, otherwise generates a placeholder.
func (c *Client) getClientID() string {
	if c.deviceURL != "" {
		return c.deviceURL
	}
	return "webex-go-sdk-client"
}

// generateRequestID generates a unique request ID for KMS messages.
func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("go-sdk-%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

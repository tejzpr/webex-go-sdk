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

// retrieveKeyFromKMS retrieves a key from KMS.
// It first tries the ECDH-based protocol (used by the Webex JS SDK), and
// falls back to a direct HTTP approach if ECDH fails.
func (c *Client) retrieveKeyFromKMS(keyURI string) (*Key, error) {
	// Try ECDH-based retrieval first (the proper Webex protocol)
	key, err := c.retrieveKeyViaECDH(keyURI)
	if err == nil {
		return key, nil
	}

	// Fall back to direct HTTP retrieval
	directKey, directErr := c.retrieveKeyDirect(keyURI)
	if directErr == nil {
		return directKey, nil
	}

	// Both approaches failed, return the ECDH error as primary
	return nil, fmt.Errorf("ECDH retrieval: %w; direct retrieval: %v", err, directErr)
}

// retrieveKeyViaECDH retrieves a key using the ECDH-encrypted KMS protocol.
func (c *Client) retrieveKeyViaECDH(keyURI string) (*Key, error) {
	// Ensure ECDH context exists
	ecdhCtx, err := c.getOrCreateECDH()
	if err != nil {
		return nil, fmt.Errorf("failed to establish ECDH context: %w", err)
	}

	// Build the KMS retrieve request
	key, err := c.doKMSRetrieve(keyURI, ecdhCtx)
	if err != nil {
		// If the request fails, invalidate ECDH and retry once
		// (the shared secret may have expired on the server side)
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

	return key, nil
}

// doKMSRetrieve performs a single KMS key retrieval attempt using the given ECDH context.
func (c *Client) doKMSRetrieve(keyURI string, ecdhCtx *ECDHContext) (*Key, error) {
	// Get user ID for the KMS request credentials
	userID, _ := c.getUserID()

	// Create KMS retrieve request payload
	kmsRequest := &KMSRequestPayload{
		Client: &KMSClient{
			ClientID: c.getClientID(),
			Credential: &KMSCredential{
				UserID: userID,
				Bearer: c.webexClient.AccessToken,
			},
		},
		RequestID: generateRequestID(),
		Method:    "retrieve",
		URI:       keyURI,
	}

	requestPayload, err := json.Marshal(kmsRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS request: %w", err)
	}

	// Wrap with ECDH shared secret (dir + A256GCM)
	wrappedRequest, err := wrapWithSharedSecret(requestPayload, ecdhCtx.sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("error wrapping KMS request: %w", err)
	}

	// Send to KMS
	responseJWEs, err := c.sendKMSMessage(wrappedRequest, ecdhCtx.kmsCluster)
	if err != nil {
		return nil, fmt.Errorf("KMS request failed: %w", err)
	}

	if len(responseJWEs) == 0 {
		return nil, fmt.Errorf("empty response from KMS")
	}

	// Unwrap and parse response
	for _, respJWE := range responseJWEs {
		responsePayload, err := unwrapWithSharedSecret(respJWE, ecdhCtx.sharedSecret)
		if err != nil {
			continue
		}

		var kmsResponse KMSMessage
		if err := json.Unmarshal(responsePayload, &kmsResponse); err != nil {
			continue
		}

		if !kmsResponse.IsSuccess() {
			return nil, fmt.Errorf("KMS request failed with status: %v", kmsResponse.Status)
		}

		if kmsResponse.Key != nil {
			return kmsResponse.Key, nil
		}

		// Check keys array (batch operations)
		if len(kmsResponse.Keys) > 0 {
			return kmsResponse.Keys[0], nil
		}
	}

	return nil, fmt.Errorf("no key found in KMS response")
}

// retrieveKeyDirect retrieves a key using the direct HTTP approach.
// This is a fallback for when the ECDH protocol is not available.
func (c *Client) retrieveKeyDirect(keyURI string) (*Key, error) {
	domain, path, err := parseKMSURI(keyURI)
	if err != nil {
		return nil, err
	}

	// Create KMS request
	kmsMessage := &KMSMessage{
		Method:    "retrieve",
		URI:       keyURI,
		RequestID: generateRequestID(),
	}

	kmsRequestData, err := json.Marshal(kmsMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS request: %w", err)
	}

	// Determine KMS endpoint
	cluster := getClusterFromDomain(domain, c.config.DefaultCluster)
	kmsEndpoint := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/%s", cluster, path)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kmsEndpoint, bytes.NewBuffer(kmsRequestData))
	if err != nil {
		return nil, fmt.Errorf("error creating KMS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making KMS request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading KMS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KMS request failed with status %d: %s", resp.StatusCode, respBody)
	}

	var kmsResponse KMSMessage
	if err := json.Unmarshal(respBody, &kmsResponse); err != nil {
		return nil, fmt.Errorf("error parsing KMS response: %w", err)
	}

	if !kmsResponse.IsSuccess() {
		return nil, fmt.Errorf("KMS request failed with status: %v", kmsResponse.Status)
	}

	if kmsResponse.Key == nil {
		return nil, fmt.Errorf("no key found in KMS response")
	}

	return kmsResponse.Key, nil
}

// sendKMSMessage sends a wrapped KMS message to the encryption service and returns response JWEs.
func (c *Client) sendKMSMessage(wrappedMessage string, cluster string) ([]string, error) {
	envelope := &KMSEnvelope{
		KMSMessages: []string{wrappedMessage},
		Destination: cluster,
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("error marshaling KMS envelope: %w", err)
	}

	clusterID := getClusterFromDomain(cluster, c.config.DefaultCluster)
	kmsEndpoint := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/kms/messages", clusterID)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kmsEndpoint, bytes.NewBuffer(envelopeJSON))
	if err != nil {
		return nil, fmt.Errorf("error creating KMS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending KMS request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading KMS response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KMS request failed with status %d: %s", resp.StatusCode, respBody)
	}

	var responseEnvelope KMSEnvelope
	if err := json.Unmarshal(respBody, &responseEnvelope); err != nil {
		return nil, fmt.Errorf("error parsing KMS response envelope: %w", err)
	}

	return responseEnvelope.KMSMessages, nil
}

// --- JWE Wrapping/Unwrapping Helpers ---

// wrapWithSharedSecret encrypts payload using dir + A256GCM with the ECDH shared secret.
func wrapWithSharedSecret(payload []byte, sharedSecret []byte) (string, error) {
	recipient := jose.Recipient{
		Algorithm: jose.DIRECT,
		Key:       sharedSecret,
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

// getClusterFromDomain determines the cluster identifier from a KMS domain or cluster URL.
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

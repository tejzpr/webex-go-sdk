/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package encryption

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

const (
	// ecdhTTL is the maximum lifetime of an ECDH context before it is
	// considered expired and must be renegotiated. This matches the typical
	// server-side expiry of ECDHE keys on the Webex KMS.
	ecdhTTL = 1 * time.Hour
)

// ECDHContext holds the ECDH key exchange state for KMS communication.
// After a successful key exchange, the shared secret is used to wrap/unwrap
// all subsequent KMS requests and responses using dir+A256GCM.
type ECDHContext struct {
	localPrivateKey  *ecdsa.PrivateKey // Client's ephemeral ECDH private key (for go-jose)
	localECDHPrivate *ecdh.PrivateKey  // Same key in crypto/ecdh format (for raw ECDH)
	sharedSecret     []byte            // 32-byte AES-256 key derived from ECDH
	ecdhKeyURI       string            // ECDHE key URI from KMS (used as kid in JWE headers)
	kmsCluster       string            // KMS cluster URL (e.g., "kms-a.wbx2.com")
	createdAt        time.Time
}

// KMSInfo represents the response from GET /encryption/api/v1/kms/{userId}
type KMSInfo struct {
	KMSCluster   string          `json:"kmsCluster"`
	RSAPublicKey json.RawMessage `json:"rsaPublicKey"`
}

// getOrCreateECDH returns the current ECDH context, creating one if needed.
// Uses sync.Cond to avoid deadlock: the lock is released while the ECDH
// exchange is in progress, allowing ProcessKMSMessages to run concurrently.
// A TTL check ensures stale contexts are proactively refreshed before the
// server rejects them.
func (c *Client) getOrCreateECDH() (*ECDHContext, error) {
	c.ecdhMu.Lock()

	// Wait if another goroutine is already creating the context
	for c.ecdhCreating {
		c.ecdhCond.Wait()
	}

	// Check if context was created by the other goroutine and is still fresh
	if c.ecdhCtx != nil {
		if time.Since(c.ecdhCtx.createdAt) < ecdhTTL {
			ctx := c.ecdhCtx
			c.ecdhMu.Unlock()
			return ctx, nil
		}
		// Context expired — clear it so we create a fresh one
		c.ecdhCtx = nil
	}

	// Mark as creating and release lock (avoids deadlock with ProcessKMSMessages)
	c.ecdhCreating = true
	c.ecdhMu.Unlock()

	// Perform the exchange without holding the lock
	ctx, err := c.performECDHExchange()

	// Re-acquire lock, update state, wake waiting goroutines
	c.ecdhMu.Lock()
	c.ecdhCreating = false
	if err == nil {
		c.ecdhCtx = ctx
	}
	c.ecdhCond.Broadcast()
	c.ecdhMu.Unlock()

	return ctx, err
}

// invalidateECDH clears the ECDH context, forcing a new exchange on next use.
func (c *Client) invalidateECDH() {
	c.ecdhMu.Lock()
	defer c.ecdhMu.Unlock()
	c.ecdhCtx = nil
}

// performECDHExchange performs the full ECDH key exchange with KMS.
//
// The protocol follows the Webex JS SDK's approach:
//  1. Fetch KMS cluster info + RSA public key via GET /encryption/api/v1/kms/{userId}
//  2. Generate an ephemeral ECDH P-256 key pair
//  3. Send client's ECDH public key to KMS, wrapped with RSA-OAEP + A256GCM
//  4. Wait for KMS's response (sync via HTTP 200, or async via Mercury for HTTP 202)
//  5. Derive the shared secret using raw ECDH (P-256)
//  6. Use the 32-byte shared secret as the AES-256-GCM key for all future KMS communication
func (c *Client) performECDHExchange() (*ECDHContext, error) {
	// Step 1: Get user ID (fetch from /people/me if not set)
	userID, err := c.getUserID()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Step 2: Get KMS info (RSA public key + cluster)
	kmsInfo, err := c.getKMSInfo(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS info: %w", err)
	}

	// Step 3: Parse KMS RSA public key
	rsaPubKey, rsaKid, err := parseRSAPublicKeyFromJSON(kmsInfo.RSAPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KMS RSA public key: %w", err)
	}

	// Step 4: Generate ephemeral ECDH P-256 key pair
	ecdsaPrivKey, ecdhPrivKey, err := generateECDHKeyPair()
	if err != nil {
		return nil, err
	}

	// Step 5: Send ECDH request and wait for response (handles both sync and async)
	ecdhResponse, err := c.sendECDHRequest(ecdsaPrivKey, rsaPubKey, rsaKid, kmsInfo.KMSCluster, userID)
	if err != nil {
		return nil, err
	}

	// Step 6: Extract server's ECDH public key and derive shared secret
	sharedSecret, err := deriveSharedSecret(ecdhResponse, ecdhPrivKey)
	if err != nil {
		return nil, err
	}

	// Step 7: Extract ECDHE key URI (used as kid in all subsequent JWE headers)
	ecdhKeyURI := ""
	if ecdhResponse.Key != nil {
		ecdhKeyURI = ecdhResponse.Key.URI
	}

	return &ECDHContext{
		localPrivateKey:  ecdsaPrivKey,
		localECDHPrivate: ecdhPrivKey,
		sharedSecret:     sharedSecret,
		ecdhKeyURI:       ecdhKeyURI,
		kmsCluster:       kmsInfo.KMSCluster,
		createdAt:        time.Now(),
	}, nil
}

// generateECDHKeyPair generates an ephemeral ECDH P-256 key pair.
func generateECDHKeyPair() (*ecdsa.PrivateKey, *ecdh.PrivateKey, error) {
	ecdsaPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDH key pair: %w", err)
	}

	ecdhPrivKey, err := ecdsaPrivKey.ECDH()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert key to ECDH format: %w", err)
	}

	return ecdsaPrivKey, ecdhPrivKey, nil
}

// sendECDHRequest creates, sends the ECDH exchange request to KMS, and waits
// for the response. The response may arrive synchronously (HTTP 200) or
// asynchronously via Mercury WebSocket (HTTP 202).
func (c *Client) sendECDHRequest(ecdsaPrivKey *ecdsa.PrivateKey, rsaPubKey *rsa.PublicKey, rsaKid string, cluster string, userID string) (*KMSMessage, error) {
	requestID := generateRequestID()

	// Register pending request with ECDH private key for decryption.
	// The Mercury-delivered response is encrypted with ECDH-ES using our public key.
	responseCh := c.registerPendingRequest(requestID, ecdsaPrivKey)
	defer c.unregisterPendingRequest(requestID)

	clientPubJWK := ecdsaPublicKeyToJWK(&ecdsaPrivKey.PublicKey)

	// Include client credentials and requestId for KMS authentication and correlation.
	// KMS expects the UUID format for userId, not the base64-encoded Webex API ID.
	ecdhRequest := &KMSRequestPayload{
		Client: &KMSClient{
			ClientID: c.getClientID(),
			Credential: &KMSCredential{
				UserID: decodeWebexID(userID),
				Bearer: c.webexClient.GetAccessToken(),
			},
		},
		RequestID: requestID,
		Method:    "create",
		URI:       "/ecdhe",
		JWK:       clientPubJWK,
	}

	ecdhPayload, err := json.Marshal(ecdhRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ECDH request: %w", err)
	}

	wrappedRequest, err := wrapWithRSA(ecdhPayload, rsaPubKey, rsaKid)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap ECDH request: %w", err)
	}

	responseJWEs, err := c.sendKMSMessage(wrappedRequest, strings.TrimPrefix(cluster, "kms://"))
	if err != nil {
		return nil, fmt.Errorf("failed to send ECDH request to KMS: %w", err)
	}

	// Try synchronous response first (200 OK with JWEs in body)
	if len(responseJWEs) > 0 {
		return decryptECDHResponse(responseJWEs, ecdsaPrivKey)
	}

	// Wait for async response via Mercury (202 Accepted)
	select {
	case payload := <-responseCh:
		var ecdhResponse KMSMessage
		if err := json.Unmarshal(payload, &ecdhResponse); err != nil {
			return nil, fmt.Errorf("error parsing ECDH response: %w", err)
		}
		return &ecdhResponse, nil
	case <-time.After(kmsResponseTimeout):
		return nil, fmt.Errorf("timeout waiting for KMS ECDH response via Mercury")
	}
}

// decryptECDHResponse decrypts the KMS ECDH exchange response.
// The response is encrypted using ECDH-ES with our public key as recipient.
func decryptECDHResponse(responseJWEs []string, ecdsaPrivKey *ecdsa.PrivateKey) (*KMSMessage, error) {
	for _, respJWE := range responseJWEs {
		jweObj, err := jose.ParseEncrypted(respJWE,
			[]jose.KeyAlgorithm{jose.ECDH_ES, jose.ECDH_ES_A256KW, jose.DIRECT, jose.RSA_OAEP},
			[]jose.ContentEncryption{jose.A256GCM, jose.A128GCM})
		if err != nil {
			continue
		}

		plaintext, err := jweObj.Decrypt(ecdsaPrivKey)
		if err != nil {
			continue
		}

		var ecdhResponse KMSMessage
		if err := json.Unmarshal(plaintext, &ecdhResponse); err != nil {
			continue
		}

		return &ecdhResponse, nil
	}

	return nil, fmt.Errorf("failed to decrypt ECDH response from KMS")
}

// deriveSharedSecret extracts the server's ECDH public key from the response
// and derives the shared secret using ECDH + HKDF-SHA-256.
//
// The derivation follows the Webex JS SDK (node-kms) protocol:
//  1. Perform raw ECDH to get the shared point x-coordinate (32 bytes for P-256)
//  2. Apply HKDF-SHA-256 (extract-then-expand) with empty salt and empty info
//  3. The resulting 32 bytes are used as the AES-256-GCM key for KMS communication
func deriveSharedSecret(ecdhResponse *KMSMessage, ecdhPrivKey *ecdh.PrivateKey) ([]byte, error) {
	serverJWK := extractServerECKey(ecdhResponse)
	if serverJWK == nil {
		return nil, fmt.Errorf("KMS ECDH response missing EC public key")
	}

	serverPubKey, err := jwkToECDHPublicKey(serverJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server ECDH public key: %w", err)
	}

	rawSecret, err := ecdhPrivKey.ECDH(serverPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ECDH shared secret: %w", err)
	}

	// Apply HKDF-SHA-256 (matching node-kms ECDH-HKDF derivation)
	sharedSecret := hkdfSHA256(rawSecret, nil, nil, 32)
	return sharedSecret, nil
}

// hkdfSHA256 implements HKDF (RFC 5869) with SHA-256.
// If salt is nil, uses HashLen (32) bytes of zeros.
// If info is nil, uses empty info.
func hkdfSHA256(ikm, salt, info []byte, length int) []byte {
	hashLen := sha256.Size // 32

	// Extract: PRK = HMAC-SHA-256(salt, IKM)
	if salt == nil {
		salt = make([]byte, hashLen)
	}
	mac := hmac.New(sha256.New, salt)
	mac.Write(ikm)
	prk := mac.Sum(nil)

	// Expand: OKM = T(1) || T(2) || ...
	if info == nil {
		info = []byte{}
	}
	n := (length + hashLen - 1) / hashLen
	okm := make([]byte, 0, n*hashLen)
	var prev []byte
	for i := 1; i <= n; i++ {
		mac = hmac.New(sha256.New, prk)
		mac.Write(prev)
		mac.Write(info)
		mac.Write([]byte{byte(i)})
		prev = mac.Sum(nil)
		okm = append(okm, prev...)
	}

	return okm[:length]
}

// extractServerECKey extracts the server's EC public key from a KMS response.
// The key may be in the Key.JWK field or directly in the JWK field.
func extractServerECKey(response *KMSMessage) *JWK {
	if response.Key != nil && response.Key.JWK.Kty == "EC" {
		return &response.Key.JWK
	}
	if response.JWK != nil && response.JWK.Kty == "EC" {
		return response.JWK
	}
	return nil
}

// getUserID returns the user ID, fetching from /v1/people/me if not already set.
func (c *Client) getUserID() (string, error) {
	if c.userID != "" {
		return c.userID, nil
	}

	// Fetch from /v1/people/me
	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://webexapis.com/v1/people/me", nil)
	if err != nil {
		return "", fmt.Errorf("error creating people/me request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.webexClient.GetAccessToken())
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading user info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("people/me request failed with status %d: %s", resp.StatusCode, body)
	}

	var userInfo struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", fmt.Errorf("error parsing user info: %w", err)
	}

	if userInfo.ID == "" {
		return "", fmt.Errorf("user ID is empty in people/me response")
	}

	c.userID = userInfo.ID
	return c.userID, nil
}

// getKMSInfo fetches KMS cluster info and RSA public key for a user.
func (c *Client) getKMSInfo(userID string) (*KMSInfo, error) {
	cluster := c.config.DefaultCluster

	// The KMS endpoint expects a UUID, not the base64-encoded Webex API ID.
	// Decode if necessary (e.g., "Y2lzY29zcGFy..." -> "c488502d-...")
	kmsUserID := decodeWebexID(userID)

	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	kmsURL := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/kms/%s", cluster, kmsUserID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kmsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating KMS info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.webexClient.GetAccessToken())
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching KMS info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading KMS info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KMS info request failed with status %d: %s", resp.StatusCode, body)
	}

	var kmsInfo KMSInfo
	if err := json.Unmarshal(body, &kmsInfo); err != nil {
		return nil, fmt.Errorf("error parsing KMS info: %w", err)
	}

	return &kmsInfo, nil
}

// --- Key Conversion Helpers ---

// ecdsaPublicKeyToJWK converts an ECDSA P-256 public key to a JWK.
func ecdsaPublicKeyToJWK(pub *ecdsa.PublicKey) *JWK {
	return &JWK{
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(padTo32Bytes(pub.X.Bytes())),
		Y:   base64.RawURLEncoding.EncodeToString(padTo32Bytes(pub.Y.Bytes())),
	}
}

// jwkToECDHPublicKey converts a JWK (EC P-256) to an ecdh.PublicKey.
func jwkToECDHPublicKey(jwk *JWK) (*ecdh.PublicKey, error) {
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported key type/curve: %s/%s (expected EC/P-256)", jwk.Kty, jwk.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("error decoding X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("error decoding Y coordinate: %w", err)
	}

	// Construct uncompressed point: 0x04 || x (32 bytes) || y (32 bytes)
	xPadded := padTo32Bytes(xBytes)
	yPadded := padTo32Bytes(yBytes)

	pubBytes := make([]byte, 1+32+32)
	pubBytes[0] = 0x04
	copy(pubBytes[1:], xPadded)
	copy(pubBytes[33:], yPadded)

	return ecdh.P256().NewPublicKey(pubBytes)
}

// parseRSAPublicKeyFromJSON parses an RSA public key from JSON.
// Handles multiple formats: a single JWK object, a JWKS key set, or a serialized string.
func parseRSAPublicKeyFromJSON(raw json.RawMessage) (*rsa.PublicKey, string, error) {
	// Try parsing as a single JWK object
	var jwk JWK
	if err := json.Unmarshal(raw, &jwk); err == nil && jwk.Kty == "RSA" {
		pub, err := jwkToRSAPublicKey(&jwk)
		return pub, jwk.Kid, err
	}

	// Try parsing as a JWKS (key set)
	var jwks struct {
		Keys []JWK `json:"keys"`
	}
	if err := json.Unmarshal(raw, &jwks); err == nil && len(jwks.Keys) > 0 {
		for _, k := range jwks.Keys {
			if k.Kty == "RSA" {
				pub, err := jwkToRSAPublicKey(&k)
				return pub, k.Kid, err
			}
		}
	}

	// Try parsing as a JSON string containing a JWK
	var jwkStr string
	if err := json.Unmarshal(raw, &jwkStr); err == nil {
		var innerJWK JWK
		if err := json.Unmarshal([]byte(jwkStr), &innerJWK); err == nil && innerJWK.Kty == "RSA" {
			pub, err := jwkToRSAPublicKey(&innerJWK)
			return pub, innerJWK.Kid, err
		}
	}

	return nil, "", fmt.Errorf("unable to parse RSA public key from KMS info response")
}

// jwkToRSAPublicKey converts an RSA JWK to *rsa.PublicKey.
func jwkToRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("key type is %q, expected RSA", jwk.Kty)
	}
	if jwk.N == "" || jwk.E == "" {
		return nil, fmt.Errorf("RSA key missing modulus (n) or exponent (e)")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("error decoding RSA modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("error decoding RSA exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	if n.BitLen() < 2048 {
		return nil, fmt.Errorf("RSA key size %d bits is too small (minimum 2048)", n.BitLen())
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// wrapWithRSA encrypts payload using RSA-OAEP + A256GCM (SHA-256).
// This is used for the initial ECDH exchange request to KMS.
func wrapWithRSA(payload []byte, rsaPubKey *rsa.PublicKey, kid string) (string, error) {
	recipient := jose.Recipient{
		Algorithm: jose.RSA_OAEP,
		Key:       rsaPubKey,
		KeyID:     kid,
	}

	encrypter, err := jose.NewEncrypter(jose.A256GCM, recipient, nil)
	if err != nil {
		return "", fmt.Errorf("error creating RSA encrypter: %w", err)
	}

	jweObj, err := encrypter.Encrypt(payload)
	if err != nil {
		return "", fmt.Errorf("error encrypting with RSA-OAEP: %w", err)
	}

	return jweObj.CompactSerialize()
}

// decodeWebexID converts a Webex API ID to the internal UUID format used by KMS.
// Webex API IDs are base64url-encoded URIs like "ciscospark://us/PEOPLE/{uuid}".
// If the input is already a UUID or unrecognizable, it is returned unchanged.
func decodeWebexID(id string) string {
	// If it looks like a UUID already (contains hyphens, ~36 chars), return as-is
	if len(id) == 36 && strings.Count(id, "-") == 4 {
		return id
	}

	// Try to decode as base64url (with or without padding)
	decoded, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		// Try with standard base64
		decoded, err = base64.URLEncoding.DecodeString(id)
		if err != nil {
			return id
		}
	}

	// Extract UUID from the decoded URI
	// Format: "ciscospark://us/PEOPLE/{uuid}" or "ciscospark://us/ORGANIZATION/{uuid}"
	decodedStr := string(decoded)
	lastSlash := strings.LastIndex(decodedStr, "/")
	if lastSlash >= 0 && lastSlash < len(decodedStr)-1 {
		candidate := decodedStr[lastSlash+1:]
		// Verify it looks like a UUID
		if len(candidate) == 36 && strings.Count(candidate, "-") == 4 {
			return candidate
		}
	}

	return id
}

// padTo32Bytes pads or truncates a byte slice to exactly 32 bytes.
// Used for P-256 coordinate encoding where each coordinate must be 32 bytes.
func padTo32Bytes(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}

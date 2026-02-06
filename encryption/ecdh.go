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
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// ECDHContext holds the ECDH key exchange state for KMS communication.
// After a successful key exchange, the shared secret is used to wrap/unwrap
// all subsequent KMS requests and responses using dir+A256GCM.
type ECDHContext struct {
	localPrivateKey  *ecdsa.PrivateKey // Client's ephemeral ECDH private key (for go-jose)
	localECDHPrivate *ecdh.PrivateKey  // Same key in crypto/ecdh format (for raw ECDH)
	sharedSecret     []byte            // 32-byte AES-256 key derived from ECDH
	kmsCluster       string            // KMS cluster URL (e.g., "kms-a.wbx2.com")
	createdAt        time.Time
}

// KMSInfo represents the response from GET /encryption/api/v1/kms/{userId}
type KMSInfo struct {
	KMSCluster   string          `json:"kmsCluster"`
	RSAPublicKey json.RawMessage `json:"rsaPublicKey"`
}

// getOrCreateECDH returns the current ECDH context, creating one if needed.
// This method is safe for concurrent use.
func (c *Client) getOrCreateECDH() (*ECDHContext, error) {
	c.ecdhMu.Lock()
	defer c.ecdhMu.Unlock()

	if c.ecdhCtx != nil {
		return c.ecdhCtx, nil
	}

	ctx, err := c.performECDHExchange()
	if err != nil {
		return nil, fmt.Errorf("ECDH key exchange failed: %w", err)
	}

	c.ecdhCtx = ctx
	return ctx, nil
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
//  4. Receive KMS's ECDH public key in the response
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

	// Step 5: Create and send ECDH request
	responseJWEs, err := c.sendECDHRequest(ecdsaPrivKey, rsaPubKey, rsaKid, kmsInfo.KMSCluster)
	if err != nil {
		return nil, err
	}

	// Step 6: Decrypt the response and extract server's public key
	ecdhResponse, err := decryptECDHResponse(responseJWEs, ecdsaPrivKey)
	if err != nil {
		return nil, err
	}

	// Step 7: Extract server's ECDH public key and derive shared secret
	sharedSecret, err := deriveSharedSecret(ecdhResponse, ecdhPrivKey)
	if err != nil {
		return nil, err
	}

	return &ECDHContext{
		localPrivateKey:  ecdsaPrivKey,
		localECDHPrivate: ecdhPrivKey,
		sharedSecret:     sharedSecret,
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

// sendECDHRequest creates and sends the ECDH exchange request to KMS.
func (c *Client) sendECDHRequest(ecdsaPrivKey *ecdsa.PrivateKey, rsaPubKey *rsa.PublicKey, rsaKid string, cluster string) ([]string, error) {
	clientPubJWK := ecdsaPublicKeyToJWK(&ecdsaPrivKey.PublicKey)
	ecdhRequest := &KMSMessage{
		Method: "create",
		URI:    "/ecdhe",
		JWK:    clientPubJWK,
	}

	ecdhPayload, err := json.Marshal(ecdhRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ECDH request: %w", err)
	}

	wrappedRequest, err := wrapWithRSA(ecdhPayload, rsaPubKey, rsaKid)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap ECDH request: %w", err)
	}

	responseJWEs, err := c.sendKMSMessage(wrappedRequest, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to send ECDH request to KMS: %w", err)
	}

	if len(responseJWEs) == 0 {
		return nil, fmt.Errorf("empty response from KMS ECDH exchange")
	}

	return responseJWEs, nil
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
// and derives the shared secret. For P-256, this produces a 32-byte secret
// (the x-coordinate of the shared point), used directly as the AES-256-GCM key.
func deriveSharedSecret(ecdhResponse *KMSMessage, ecdhPrivKey *ecdh.PrivateKey) ([]byte, error) {
	serverJWK := extractServerECKey(ecdhResponse)
	if serverJWK == nil {
		return nil, fmt.Errorf("KMS ECDH response missing EC public key")
	}

	serverPubKey, err := jwkToECDHPublicKey(serverJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server ECDH public key: %w", err)
	}

	sharedSecret, err := ecdhPrivKey.ECDH(serverPubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive ECDH shared secret: %w", err)
	}

	return sharedSecret, nil
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

	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching user info: %w", err)
	}
	defer resp.Body.Close()

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

	ctx, cancel := context.WithTimeout(context.Background(), c.config.HTTPTimeout)
	defer cancel()

	kmsURL := fmt.Sprintf("https://encryption-%s.wbx2.com/encryption/api/v1/kms/%s", cluster, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kmsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating KMS info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.webexClient.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching KMS info: %w", err)
	}
	defer resp.Body.Close()

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

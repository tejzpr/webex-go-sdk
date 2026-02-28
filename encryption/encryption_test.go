/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package encryption

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
	jose "github.com/go-jose/go-jose/v4"
)

// --- Helper Functions ---

// newTestClient creates an encryption Client with a mock webex client.
func newTestClient(t *testing.T) *Client {
	t.Helper()
	webexClient, err := webexsdk.NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("failed to create webex client: %v", err)
	}
	return New(webexClient, &Config{
		HTTPTimeout:    defaultTimeout,
		DefaultCluster: "a",
		DisableCache:   false,
	})
}

// generateTestSymmetricKey generates a random 32-byte AES-256 key and returns
// it along with the base64url-encoded form.
func generateTestSymmetricKey(t *testing.T) ([]byte, string) {
	t.Helper()
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		t.Fatalf("failed to generate random key: %v", err)
	}
	return rawKey, base64.RawURLEncoding.EncodeToString(rawKey)
}

// encryptJWE creates a JWE compact serialization using dir+A256GCM.
func encryptJWE(t *testing.T, plaintext []byte, key []byte) string {
	t.Helper()
	encrypter, err := jose.NewEncrypter(jose.A256GCM,
		jose.Recipient{Algorithm: jose.DIRECT, Key: key}, nil)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}
	jweObj, err := encrypter.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}
	serialized, err := jweObj.CompactSerialize()
	if err != nil {
		t.Fatalf("failed to serialize JWE: %v", err)
	}
	return serialized
}

// --- JWK Tests ---

func TestJWKSymmetricKey(t *testing.T) {
	rawKey, encodedKey := generateTestSymmetricKey(t)
	jwk := JWK{
		Kty: "oct",
		K:   encodedKey,
		Kid: "test-key-1",
	}

	result, err := jwk.SymmetricKey()
	if err != nil {
		t.Fatalf("SymmetricKey() returned error: %v", err)
	}
	if len(result) != 32 {
		t.Errorf("expected key length 32, got %d", len(result))
	}
	for i := range rawKey {
		if result[i] != rawKey[i] {
			t.Fatalf("key mismatch at byte %d: expected %d, got %d", i, rawKey[i], result[i])
		}
	}
}

func TestJWKSymmetricKeyWrongType(t *testing.T) {
	jwk := JWK{Kty: "EC", Crv: "P-256", X: "abc", Y: "def"}
	_, err := jwk.SymmetricKey()
	if err == nil {
		t.Fatal("expected error for non-oct key type")
	}
}

func TestJWKSymmetricKeyEmptyK(t *testing.T) {
	jwk := JWK{Kty: "oct", K: ""}
	_, err := jwk.SymmetricKey()
	if err == nil {
		t.Fatal("expected error for empty K field")
	}
}

// --- KMSMessage Tests ---

func TestKMSMessageIsSuccess(t *testing.T) {
	tests := []struct {
		name   string
		status interface{}
		want   bool
	}{
		{"string success", "success", true},
		{"string 200", "200", true},
		{"float64 200", float64(200), true},
		{"int 200", 200, true},
		{"string failure", "failure", false},
		{"float64 400", float64(400), false},
		{"nil status", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &KMSMessage{Status: tt.status}
			if got := msg.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- parseKMSURI Tests ---

func TestParseKMSURI(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		wantDomain string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "valid URI",
			uri:        "kms://ciscospark.com/keys/abc123",
			wantDomain: "ciscospark.com",
			wantPath:   "keys/abc123",
		},
		{
			name:       "valid URI with long path",
			uri:        "kms://kms-a.wbx2.com/keys/some-uuid-value",
			wantDomain: "kms-a.wbx2.com",
			wantPath:   "keys/some-uuid-value",
		},
		{
			name:    "missing prefix",
			uri:     "https://example.com/key",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "prefix only",
			uri:     "kms://",
			wantErr: true,
		},
		{
			name:    "domain only, no path",
			uri:     "kms://ciscospark.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, path, err := parseKMSURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

// --- JWE Decryption Tests ---

func TestDecryptText(t *testing.T) {
	rawKey, encodedKey := generateTestSymmetricKey(t)
	keyURI := "kms://test-domain.com/keys/test-key-1"
	plaintext := "Hello, Webex! This is an encrypted message."

	// Create JWE ciphertext
	ciphertext := encryptJWE(t, []byte(plaintext), rawKey)

	// Create client and pre-cache the key
	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{
			Kty: "oct",
			K:   encodedKey,
			Kid: "test-key-1",
		},
	})

	// Decrypt
	result, err := client.DecryptText(keyURI, ciphertext)
	if err != nil {
		t.Fatalf("DecryptText() returned error: %v", err)
	}
	if result != plaintext {
		t.Errorf("DecryptText() = %q, want %q", result, plaintext)
	}
}

func TestDecryptTextUnicode(t *testing.T) {
	rawKey, encodedKey := generateTestSymmetricKey(t)
	keyURI := "kms://test-domain.com/keys/unicode-key"
	plaintext := "Hello 🌍 — 日本語テスト — مرحبا"

	ciphertext := encryptJWE(t, []byte(plaintext), rawKey)

	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{Kty: "oct", K: encodedKey, Kid: "unicode-key"},
	})

	result, err := client.DecryptText(keyURI, ciphertext)
	if err != nil {
		t.Fatalf("DecryptText() returned error: %v", err)
	}
	if result != plaintext {
		t.Errorf("DecryptText() = %q, want %q", result, plaintext)
	}
}

func TestDecryptTextEmptyKeyURI(t *testing.T) {
	client := newTestClient(t)
	_, err := client.DecryptText("", "some.jwe.content.here.!")
	if err == nil {
		t.Fatal("expected error for empty key URI")
	}
}

func TestDecryptTextEmptyCiphertext(t *testing.T) {
	client := newTestClient(t)
	_, err := client.DecryptText("kms://test/keys/1", "")
	if err == nil {
		t.Fatal("expected error for empty ciphertext")
	}
}

func TestDecryptTextInvalidJWE(t *testing.T) {
	_, encodedKey := generateTestSymmetricKey(t)
	keyURI := "kms://test-domain.com/keys/test-key"

	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{Kty: "oct", K: encodedKey, Kid: "test-key"},
	})

	_, err := client.DecryptText(keyURI, "not.a.valid.jwe.string")
	if err == nil {
		t.Fatal("expected error for invalid JWE")
	}
}

func TestDecryptTextWrongKey(t *testing.T) {
	rawKey1, _ := generateTestSymmetricKey(t)
	_, encodedKey2 := generateTestSymmetricKey(t) // Different key
	keyURI := "kms://test-domain.com/keys/wrong-key"

	ciphertext := encryptJWE(t, []byte("secret message"), rawKey1)

	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{Kty: "oct", K: encodedKey2, Kid: "wrong-key"},
	})

	_, err := client.DecryptText(keyURI, ciphertext)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestDecryptMessageContent(t *testing.T) {
	rawKey, encodedKey := generateTestSymmetricKey(t)
	keyURI := "kms://test-domain.com/keys/msg-key"
	plaintext := "Decrypted message content"

	ciphertext := encryptJWE(t, []byte(plaintext), rawKey)

	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{Kty: "oct", K: encodedKey, Kid: "msg-key"},
	})

	result, err := client.DecryptMessageContent(keyURI, ciphertext)
	if err != nil {
		t.Fatalf("DecryptMessageContent() returned error: %v", err)
	}
	if result != plaintext {
		t.Errorf("DecryptMessageContent() = %q, want %q", result, plaintext)
	}
}

// --- Key Cache Tests ---

func TestCacheKey(t *testing.T) {
	client := newTestClient(t)
	key := &Key{
		URI: "kms://test/keys/cached",
		JWK: JWK{Kty: "oct", K: "dGVzdA", Kid: "cached"},
	}
	client.CacheKey(key)

	client.mu.RLock()
	cached, ok := client.keyCache["kms://test/keys/cached"]
	client.mu.RUnlock()

	if !ok {
		t.Fatal("key was not cached")
	}
	if cached.URI != key.URI {
		t.Errorf("cached key URI = %q, want %q", cached.URI, key.URI)
	}
}

func TestCacheKeyNil(t *testing.T) {
	client := newTestClient(t)
	client.CacheKey(nil)                       // Should not panic
	client.CacheKey(&Key{URI: "", JWK: JWK{}}) // Empty URI, should not cache

	client.mu.RLock()
	count := len(client.keyCache)
	client.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected empty cache, got %d entries", count)
	}
}

// --- JWE Wrap/Unwrap Tests ---

func TestWrapUnwrapWithSharedSecret(t *testing.T) {
	// Generate a 32-byte shared secret
	sharedSecret := make([]byte, 32)
	if _, err := rand.Read(sharedSecret); err != nil {
		t.Fatalf("failed to generate shared secret: %v", err)
	}

	plaintext := []byte(`{"method":"retrieve","uri":"kms://test/keys/123"}`)

	// Wrap
	wrapped, err := wrapWithSharedSecret(plaintext, sharedSecret, "kms://test/ecdhe/test-kid")
	if err != nil {
		t.Fatalf("wrapWithSharedSecret() error: %v", err)
	}

	// Verify it's a valid JWE (5 dot-separated parts)
	parts := countDots(wrapped)
	if parts != 4 {
		t.Errorf("expected 4 dots in JWE compact serialization, got %d", parts)
	}

	// Unwrap
	result, err := unwrapWithSharedSecret(wrapped, sharedSecret)
	if err != nil {
		t.Fatalf("unwrapWithSharedSecret() error: %v", err)
	}

	if string(result) != string(plaintext) {
		t.Errorf("unwrapped = %q, want %q", string(result), string(plaintext))
	}
}

func TestUnwrapWithWrongSecret(t *testing.T) {
	secret1 := make([]byte, 32)
	secret2 := make([]byte, 32)
	rand.Read(secret1)
	rand.Read(secret2)

	wrapped, err := wrapWithSharedSecret([]byte("test data"), secret1, "")
	if err != nil {
		t.Fatalf("wrapWithSharedSecret() error: %v", err)
	}

	_, err = unwrapWithSharedSecret(wrapped, secret2)
	if err == nil {
		t.Fatal("expected error when unwrapping with wrong secret")
	}
}

// --- RSA Wrapping Tests ---

func TestWrapWithRSA(t *testing.T) {
	// Generate RSA key pair for testing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	payload := []byte(`{"method":"create","uri":"/ecdhe"}`)

	// Wrap with RSA public key
	wrapped, err := wrapWithRSA(payload, &rsaKey.PublicKey, "test-rsa-kid")
	if err != nil {
		t.Fatalf("wrapWithRSA() error: %v", err)
	}

	// Verify it's a valid JWE
	parts := countDots(wrapped)
	if parts != 4 {
		t.Errorf("expected 4 dots in JWE compact serialization, got %d", parts)
	}

	// Decrypt with private key to verify
	jweObj, err := jose.ParseEncrypted(wrapped,
		[]jose.KeyAlgorithm{jose.RSA_OAEP},
		[]jose.ContentEncryption{jose.A256GCM})
	if err != nil {
		t.Fatalf("failed to parse JWE: %v", err)
	}

	plaintext, err := jweObj.Decrypt(rsaKey)
	if err != nil {
		t.Fatalf("failed to decrypt JWE: %v", err)
	}

	if string(plaintext) != string(payload) {
		t.Errorf("decrypted = %q, want %q", string(plaintext), string(payload))
	}
}

// --- ECDH Key Conversion Tests ---

func TestECDSAPublicKeyToJWK(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}

	jwk := ecdsaPublicKeyToJWK(&privKey.PublicKey)

	if jwk.Kty != "EC" {
		t.Errorf("Kty = %q, want EC", jwk.Kty)
	}
	if jwk.Crv != "P-256" {
		t.Errorf("Crv = %q, want P-256", jwk.Crv)
	}
	if jwk.X == "" || jwk.Y == "" {
		t.Error("X or Y coordinate is empty")
	}

	// Decode and verify length
	xBytes, _ := base64.RawURLEncoding.DecodeString(jwk.X)
	yBytes, _ := base64.RawURLEncoding.DecodeString(jwk.Y)
	if len(xBytes) != 32 {
		t.Errorf("X length = %d, want 32", len(xBytes))
	}
	if len(yBytes) != 32 {
		t.Errorf("Y length = %d, want 32", len(yBytes))
	}
}

func TestJWKToECDHPublicKeyRoundTrip(t *testing.T) {
	// Generate an ECDSA key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Convert to JWK
	jwk := ecdsaPublicKeyToJWK(&privKey.PublicKey)

	// Convert back to ecdh.PublicKey
	ecdhPub, err := jwkToECDHPublicKey(jwk)
	if err != nil {
		t.Fatalf("jwkToECDHPublicKey() error: %v", err)
	}

	// Compare with original
	origECDH, err := privKey.PublicKey.ECDH()
	if err != nil {
		t.Fatalf("ECDH() error: %v", err)
	}

	if !ecdhPub.Equal(origECDH) {
		t.Error("round-tripped public key does not match original")
	}
}

func TestJWKToECDHPublicKeyInvalidCurve(t *testing.T) {
	jwk := &JWK{Kty: "EC", Crv: "P-384", X: "abc", Y: "def"}
	_, err := jwkToECDHPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for unsupported curve")
	}
}

func TestJWKToECDHPublicKeyInvalidType(t *testing.T) {
	jwk := &JWK{Kty: "RSA", N: "abc", E: "def"}
	_, err := jwkToECDHPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for RSA key type")
	}
}

// --- ECDH Shared Secret Derivation Tests ---

func TestECDHSharedSecretDerivation(t *testing.T) {
	// Simulate a full ECDH key exchange

	// Client generates key pair
	clientPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}
	clientECDH, _ := clientPriv.ECDH()

	// Server generates key pair
	serverPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate server key: %v", err)
	}
	serverECDH, _ := serverPriv.ECDH()

	// Convert keys through JWK format (simulating network transfer)
	clientPubJWK := ecdsaPublicKeyToJWK(&clientPriv.PublicKey)
	serverPubJWK := ecdsaPublicKeyToJWK(&serverPriv.PublicKey)

	// Client computes shared secret with server's public key
	serverPubFromJWK, err := jwkToECDHPublicKey(serverPubJWK)
	if err != nil {
		t.Fatalf("failed to convert server pub JWK: %v", err)
	}
	clientSharedSecret, err := clientECDH.ECDH(serverPubFromJWK)
	if err != nil {
		t.Fatalf("client ECDH failed: %v", err)
	}

	// Server computes shared secret with client's public key
	clientPubFromJWK, err := jwkToECDHPublicKey(clientPubJWK)
	if err != nil {
		t.Fatalf("failed to convert client pub JWK: %v", err)
	}
	serverSharedSecret, err := serverECDH.ECDH(clientPubFromJWK)
	if err != nil {
		t.Fatalf("server ECDH failed: %v", err)
	}

	// Both should derive the same shared secret
	if len(clientSharedSecret) != 32 {
		t.Errorf("client shared secret length = %d, want 32", len(clientSharedSecret))
	}
	if len(serverSharedSecret) != 32 {
		t.Errorf("server shared secret length = %d, want 32", len(serverSharedSecret))
	}
	for i := range clientSharedSecret {
		if clientSharedSecret[i] != serverSharedSecret[i] {
			t.Fatalf("shared secrets differ at byte %d", i)
		}
	}

	// Verify the shared secret can be used for JWE wrap/unwrap
	testPayload := []byte(`{"method":"retrieve","uri":"kms://test/keys/123"}`)
	wrapped, err := wrapWithSharedSecret(testPayload, clientSharedSecret, "")
	if err != nil {
		t.Fatalf("wrap error: %v", err)
	}
	unwrapped, err := unwrapWithSharedSecret(wrapped, serverSharedSecret)
	if err != nil {
		t.Fatalf("unwrap error: %v", err)
	}
	if string(unwrapped) != string(testPayload) {
		t.Errorf("unwrapped = %q, want %q", string(unwrapped), string(testPayload))
	}
}

// --- RSA Key Parsing Tests ---

func TestParseRSAPublicKeyFromJWK(t *testing.T) {
	// Generate RSA key for testing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	jwk := JWK{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(rsaKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaKey.PublicKey.E)).Bytes()),
		Kid: "test-rsa-kid",
	}

	jwkJSON, _ := json.Marshal(jwk)

	pub, kid, err := parseRSAPublicKeyFromJSON(json.RawMessage(jwkJSON))
	if err != nil {
		t.Fatalf("parseRSAPublicKeyFromJSON() error: %v", err)
	}

	if kid != "test-rsa-kid" {
		t.Errorf("kid = %q, want %q", kid, "test-rsa-kid")
	}
	if pub.N.Cmp(rsaKey.PublicKey.N) != 0 {
		t.Error("RSA modulus does not match")
	}
	if pub.E != rsaKey.PublicKey.E {
		t.Errorf("RSA exponent = %d, want %d", pub.E, rsaKey.PublicKey.E)
	}
}

func TestParseRSAPublicKeyFromString(t *testing.T) {
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwk := JWK{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(rsaKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(rsaKey.PublicKey.E)).Bytes()),
		Kid: "string-rsa-kid",
	}

	// Wrap as a JSON string (how some KMS endpoints return it)
	jwkJSON, _ := json.Marshal(jwk)
	jwkStr, _ := json.Marshal(string(jwkJSON))

	pub, kid, err := parseRSAPublicKeyFromJSON(json.RawMessage(jwkStr))
	if err != nil {
		t.Fatalf("parseRSAPublicKeyFromJSON() error: %v", err)
	}

	if kid != "string-rsa-kid" {
		t.Errorf("kid = %q, want %q", kid, "string-rsa-kid")
	}
	if pub.N.Cmp(rsaKey.PublicKey.N) != 0 {
		t.Error("RSA modulus does not match")
	}
}

func TestJWKToRSAPublicKeyTooSmall(t *testing.T) {
	// Generate a small RSA key (1024-bit, below minimum)
	smallKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	jwk := &JWK{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(smallKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(smallKey.PublicKey.E)).Bytes()),
	}

	_, err := jwkToRSAPublicKey(jwk)
	if err == nil {
		t.Fatal("expected error for RSA key < 2048 bits")
	}
}

// --- getClusterFromDomain Tests ---

func TestGetClusterFromDomain(t *testing.T) {
	tests := []struct {
		domain         string
		defaultCluster string
		want           string
	}{
		{"kms-a.wbx2.com", "a", "a"},
		{"kms-b.wbx2.com", "a", "b"},
		{"cisco.com", "b", "a"},
		{"ciscospark.com", "b", "a"},
		{"", "c", "c"},
		{"a", "z", "a"},
		{"unknown.example.com", "d", "d"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := getClusterFromDomain(tt.domain, tt.defaultCluster)
			if got != tt.want {
				t.Errorf("getClusterFromDomain(%q, %q) = %q, want %q",
					tt.domain, tt.defaultCluster, got, tt.want)
			}
		})
	}
}

// --- padTo32Bytes Tests ---

func TestPadTo32Bytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"shorter", make([]byte, 20), 32},
		{"exact", make([]byte, 32), 32},
		{"longer", make([]byte, 40), 32},
		{"empty", []byte{}, 32},
		{"one byte", []byte{0x42}, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padTo32Bytes(tt.input)
			if len(result) != tt.want {
				t.Errorf("padTo32Bytes() length = %d, want %d", len(result), tt.want)
			}
		})
	}

	// Verify padding preserves value
	input := []byte{0x01, 0x02, 0x03}
	result := padTo32Bytes(input)
	if result[29] != 0x01 || result[30] != 0x02 || result[31] != 0x03 {
		t.Error("padding did not preserve value at end")
	}
	for i := 0; i < 29; i++ {
		if result[i] != 0 {
			t.Errorf("expected zero padding at position %d, got %d", i, result[i])
		}
	}
}

// --- SetDeviceInfo Tests ---

func TestSetDeviceInfo(t *testing.T) {
	client := newTestClient(t)
	client.SetDeviceInfo("https://wdm-a.wbx2.com/wdm/api/v1/devices/abc", "user-123")

	if client.deviceURL != "https://wdm-a.wbx2.com/wdm/api/v1/devices/abc" {
		t.Errorf("deviceURL = %q, want wdm URL", client.deviceURL)
	}
	if client.userID != "user-123" {
		t.Errorf("userID = %q, want user-123", client.userID)
	}
}

// --- generateRequestID Tests ---

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("generated empty request ID")
	}
	if id1 == id2 {
		t.Error("generated duplicate request IDs")
	}
	if len(id1) < 20 {
		t.Errorf("request ID too short: %q", id1)
	}
}

// --- ProcessKMSMessages Tests ---

func TestProcessKMSMessages(t *testing.T) {
	client := newTestClient(t)

	// Create a shared secret and ECDH context
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)

	clientPriv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecdhPriv, _ := clientPriv.ECDH()

	client.ecdhCtx = &ECDHContext{
		localPrivateKey:  clientPriv,
		localECDHPrivate: ecdhPriv,
		sharedSecret:     sharedSecret,
		kmsCluster:       "kms-a.wbx2.com",
	}

	// Create a KMS message with a key
	kmsMsg := KMSMessage{
		Status: float64(200),
		Key: &Key{
			URI: "kms://test/keys/from-mercury",
			JWK: JWK{
				Kty: "oct",
				K:   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
				Kid: "from-mercury",
			},
		},
	}

	msgJSON, _ := json.Marshal(kmsMsg)
	wrapped, err := wrapWithSharedSecret(msgJSON, sharedSecret, "")
	if err != nil {
		t.Fatalf("failed to wrap KMS message: %v", err)
	}

	// Process the KMS message
	client.ProcessKMSMessages([]string{wrapped})

	// Verify the key was cached
	client.mu.RLock()
	cached, ok := client.keyCache["kms://test/keys/from-mercury"]
	client.mu.RUnlock()

	if !ok {
		t.Fatal("key from KMS message was not cached")
	}
	if cached.JWK.Kid != "from-mercury" {
		t.Errorf("cached key Kid = %q, want from-mercury", cached.JWK.Kid)
	}
}

func TestProcessKMSMessagesNoECDHContext(t *testing.T) {
	client := newTestClient(t)
	// No ECDH context set - should not panic
	client.ProcessKMSMessages([]string{"invalid.jwe.data.here.now"})
}

// --- Integration Test: Full KMS Key Retrieval with Mock Server ---

func TestGetKeyCached(t *testing.T) {
	// Test that GetKey returns a cached key without making HTTP calls
	key := &Key{
		URI: "kms://mock-kms.example.com/keys/test-123",
		JWK: JWK{
			Kty: "oct",
			K:   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
			Kid: "test-123",
		},
	}

	client := newTestClient(t)
	client.CacheKey(key)

	result, err := client.GetKey("kms://mock-kms.example.com/keys/test-123")
	if err != nil {
		t.Fatalf("GetKey() error: %v", err)
	}
	if result.URI != key.URI {
		t.Errorf("GetKey() URI = %q, want %q", result.URI, key.URI)
	}
	if result.JWK.Kid != "test-123" {
		t.Errorf("GetKey() Kid = %q, want test-123", result.JWK.Kid)
	}
}

func TestPendingRequestMechanism(t *testing.T) {
	// Test the async pending request registration and delivery
	webexClient, _ := webexsdk.NewClient("test-token", nil)
	client := New(webexClient, nil)

	requestID := "test-req-123"

	// Register a pending request
	ch := client.registerPendingRequest(requestID, nil)

	// Simulate delivering a response (as ProcessKMSMessages would)
	go func() {
		time.Sleep(50 * time.Millisecond)
		client.pendingMu.Lock()
		if req, ok := client.pendingRequests[requestID]; ok {
			payload := []byte(`{"status":200,"key":{"uri":"kms://test/keys/1","jwk":{"kty":"oct","k":"test"}}}`)
			req.responseCh <- payload
			delete(client.pendingRequests, requestID)
		}
		client.pendingMu.Unlock()
	}()

	// Wait for the response
	select {
	case payload := <-ch:
		var msg KMSMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if !msg.IsSuccess() {
			t.Errorf("Expected success status, got %v", msg.Status)
		}
		if msg.Key == nil || msg.Key.URI != "kms://test/keys/1" {
			t.Errorf("Expected key URI kms://test/keys/1, got %v", msg.Key)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for pending request response")
	}

	// Verify cleanup
	client.unregisterPendingRequest(requestID)
	client.pendingMu.Lock()
	_, exists := client.pendingRequests[requestID]
	client.pendingMu.Unlock()
	if exists {
		t.Error("Pending request not cleaned up after unregister")
	}
}

func TestKmsClusterFromDomain(t *testing.T) {
	tests := []struct {
		domain string
		def    string
		want   string
	}{
		{"kms-a.wbx2.com", "kms-a.wbx2.com", "kms-a.wbx2.com"},
		{"kms-cisco.wbx2.com", "kms-a.wbx2.com", "kms-cisco.wbx2.com"},
		{"cisco.com", "kms-a.wbx2.com", "kms-a.wbx2.com"},
		{"", "kms-a.wbx2.com", "kms-a.wbx2.com"},
	}
	for _, tt := range tests {
		got := kmsClusterFromDomain(tt.domain, tt.def)
		if got != tt.want {
			t.Errorf("kmsClusterFromDomain(%q, %q) = %q, want %q", tt.domain, tt.def, got, tt.want)
		}
	}
}

// --- End-to-End Decrypt Test ---

func TestEndToEndDecrypt(t *testing.T) {
	// Simulate the full decryption pipeline:
	// 1. Create a symmetric key
	// 2. Encrypt a message with JWE (dir + A256GCM)
	// 3. Cache the key
	// 4. Decrypt the message

	rawKey := make([]byte, 32)
	rand.Read(rawKey)
	keyURI := "kms://ciscospark.com/keys/e2e-test-key"

	// Create JWE-encrypted message (simulating what Webex would produce)
	messages := []string{
		"Hello from Webex!",
		"<p>HTML <b>formatted</b> message</p>",
		`{"json":"payload","count":42}`,
		"Multi\nline\nmessage",
		"A short one",
	}

	client := newTestClient(t)
	client.CacheKey(&Key{
		URI: keyURI,
		JWK: JWK{
			Kty: "oct",
			K:   base64.RawURLEncoding.EncodeToString(rawKey),
			Kid: "e2e-test-key",
		},
	})

	for _, msg := range messages {
		t.Run(fmt.Sprintf("msg_%d_bytes", len(msg)), func(t *testing.T) {
			ciphertext := encryptJWE(t, []byte(msg), rawKey)
			result, err := client.DecryptText(keyURI, ciphertext)
			if err != nil {
				t.Fatalf("DecryptText() error: %v", err)
			}
			if result != msg {
				t.Errorf("DecryptText() = %q, want %q", result, msg)
			}
		})
	}
}

// --- Helpers ---

func countDots(s string) int {
	count := 0
	for _, c := range s {
		if c == '.' {
			count++
		}
	}
	return count
}

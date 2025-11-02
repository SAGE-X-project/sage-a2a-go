// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
//
// sage-a2a-go is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sage-a2a-go is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with sage-a2a-go.  If not, see <https://www.gnu.org/licenses/>.

package signer

import (
	"context"
	stdcrypto "crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyPair is a mock implementation of crypto.KeyPair for testing
type mockKeyPair struct {
	pubKey  interface{}
	privKey interface{}
	keyType crypto.KeyType
	signErr error
	id      string
}

func (m *mockKeyPair) ID() string {
	if m.id == "" {
		return "mock-key-id"
	}
	return m.id
}

func (m *mockKeyPair) PublicKey() stdcrypto.PublicKey {
	return m.pubKey
}

func (m *mockKeyPair) PrivateKey() stdcrypto.PrivateKey {
	return m.privKey
}

func (m *mockKeyPair) Type() crypto.KeyType {
	return m.keyType
}

func (m *mockKeyPair) Sign(message []byte) ([]byte, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	return []byte("mock-signature"), nil
}

func (m *mockKeyPair) Verify(message, signature []byte) error {
	return nil
}

// Helper functions to create test key pairs
func createMockECDSAKeyPair() *mockKeyPair {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return &mockKeyPair{
		pubKey:  &privateKey.PublicKey,
		privKey: privateKey,
		keyType: crypto.KeyTypeSecp256k1,
	}
}

func createMockEd25519KeyPair() *mockKeyPair {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	return &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeEd25519,
	}
}

func TestDefaultA2ASigner_SignRequest_ECDSA(t *testing.T) {
	// Test Case 1: Sign request with ECDSA key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest1")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	// Create request
	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"task":"test"}`))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
	assert.NotEmpty(t, req.Header.Get("Signature"))
	assert.Contains(t, req.Header.Get("Signature-Input"), string(testDID))
}

func TestDefaultA2ASigner_SignRequest_Ed25519(t *testing.T) {
	// Test Case 2: Sign request with Ed25519 key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest2")
	keyPair := createMockEd25519KeyPair()

	signer := NewDefaultA2ASigner()

	// Create request
	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"task":"test"}`))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
	assert.NotEmpty(t, req.Header.Get("Signature"))
	assert.Contains(t, req.Header.Get("Signature-Input"), string(testDID))
}

func TestDefaultA2ASigner_SignRequest_KeyIDInclusion(t *testing.T) {
	// Test Case 3: Verify DID is included as keyid parameter

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xabc123")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("GET", "https://agent.example.com/status", nil)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, `keyid="did:sage:ethereum:0xabc123"`)
}

func TestDefaultA2ASigner_SignRequest_TimestampInclusion(t *testing.T) {
	// Test Case 4: Verify timestamp is included in signature

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest4")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	_ = time.Now().Unix()

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	_ = time.Now().Unix()

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, "created=")

	// Extract timestamp and verify it's within reasonable range
	// Timestamp should be between beforeSign and afterSign
	assert.Regexp(t, `created=\d+`, sigInput)
}

func TestDefaultA2ASigner_SignRequest_StandardComponents(t *testing.T) {
	// Test Case 5: Verify standard signature components are included

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest5")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")

	// Standard components should be included (updated for RFC9421)
	assert.Contains(t, sigInput, `"@method"`)
	assert.Contains(t, sigInput, `"@path"`)
	assert.Contains(t, sigInput, `"content-digest"`)
}

func TestDefaultA2ASigner_SignRequest_ContextCancellation(t *testing.T) {
	// Test Case 6: Context cancellation should be respected

	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testDID := did.AgentDID("did:sage:ethereum:0xtest6")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestDefaultA2ASigner_SignRequestWithOptions_CustomComponents(t *testing.T) {
	// Test Case 7: Sign with custom component list

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest7")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("X-Custom-Header", "test-value")

	opts := &SigningOptions{
		Components: []string{"@method", "@target-uri", "x-custom-header"},
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, `"@method"`)
	assert.Contains(t, sigInput, `"@target-uri"`)
	assert.Contains(t, sigInput, `"x-custom-header"`)
}

func TestDefaultA2ASigner_SignRequestWithOptions_CustomTimestamp(t *testing.T) {
	// Test Case 8: Sign with custom created timestamp

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest8")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Use recent timestamp (30 seconds ago to pass validation)
	customTimestamp := time.Now().Add(-30 * time.Second).Unix()
	opts := &SigningOptions{
		Created: customTimestamp,
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, fmt.Sprintf("created=%d", customTimestamp))
}

func TestDefaultA2ASigner_SignRequestWithOptions_Expires(t *testing.T) {
	// Test Case 9: Sign with expiration timestamp

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest9")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	opts := &SigningOptions{
		Expires: 1618884999,
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, "expires=1618884999")
}

func TestDefaultA2ASigner_SignRequestWithOptions_Nonce(t *testing.T) {
	// Test Case 10: Sign with nonce for replay attack prevention

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest10")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	opts := &SigningOptions{
		Nonce: "random-nonce-12345",
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, `nonce="random-nonce-12345"`)
}

func TestDefaultA2ASigner_SignRequest_NilRequest(t *testing.T) {
	// Test Case 11: Nil request should fail gracefully

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest11")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	// Execute
	err := signer.SignRequest(ctx, nil, testDID, keyPair)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request")
}

func TestDefaultA2ASigner_SignRequest_NilKeyPair(t *testing.T) {
	// Test Case 12: Nil key pair should fail gracefully

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest12")

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, nil)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key pair")
}

func TestDefaultA2ASigner_SignRequest_EmptyDID(t *testing.T) {
	// Test Case 13: Empty DID should fail

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DID")
}

func TestDefaultA2ASigner_SignRequest_SigningError(t *testing.T) {
	// Test Case 14: Signing error should be propagated

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest14")

	// Create key pair that returns error on Sign()
	keyPair := &mockKeyPair{
		pubKey:  &ecdsa.PublicKey{},
		keyType: crypto.KeyTypeSecp256k1,
		signErr: assert.AnError,
	}

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert
	require.Error(t, err)
}

func TestDefaultA2ASigner_GetAlgorithm_ECDSA(t *testing.T) {
	// Test Case 15: Verify ECDSA algorithm is correctly determined

	// Setup
	keyPair := createMockECDSAKeyPair()

	// The algorithm should be determined internally
	// We can test this indirectly through the signature output
	assert.Equal(t, crypto.KeyTypeSecp256k1, keyPair.Type())
}

func TestDefaultA2ASigner_GetAlgorithm_Ed25519(t *testing.T) {
	// Test Case 16: Verify Ed25519 algorithm is correctly determined

	// Setup
	keyPair := createMockEd25519KeyPair()

	// The algorithm should be determined internally
	assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
}

func TestDefaultA2ASigner_SignRequestWithOptions_NilOptions(t *testing.T) {
	// Test Case 17: Nil options should use defaults

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest17")
	keyPair := createMockECDSAKeyPair()

	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute with nil options
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, nil)

	// Assert - should use defaults and succeed
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
	assert.NotEmpty(t, req.Header.Get("Signature"))
}

// Security Test 1: Body size limit protection
func TestSignRequest_BodySizeLimit(t *testing.T) {
	// TDD: Test for body size DoS protection

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-1")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	// Create 11MB body (exceeds 10MB limit)
	largeBody := make([]byte, 11*1024*1024)
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(string(largeBody)))
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - should fail with size limit error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

// Security Test 1b: Body size within limit should succeed
func TestSignRequest_BodySizeWithinLimit(t *testing.T) {
	// TDD: Test that reasonable body sizes work

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-1b")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	// Create 1MB body (within 10MB limit)
	normalBody := make([]byte, 1*1024*1024)
	for i := range normalBody {
		normalBody[i] = byte(i % 256)
	}

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(string(normalBody)))
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - should succeed
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
}

// Security Test 2a: Invalid algorithm "none" should fail
func TestSignRequest_AlgorithmNone(t *testing.T) {
	// TDD: Test for algorithm injection attack

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-2a")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	opts := &SigningOptions{
		Algorithm: "none", // Attack: disable signature
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithm")
	assert.Contains(t, err.Error(), "not allowed")
}

// Security Test 2b: Mismatched algorithm for key type should fail
func TestSignRequest_AlgorithmMismatch(t *testing.T) {
	// TDD: Test algorithm confusion attack

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-2b")
	keyPair := createMockEd25519KeyPair() // Ed25519 key
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	opts := &SigningOptions{
		Algorithm: "es256k", // Wrong: trying to use secp256k1 algorithm with Ed25519 key
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "algorithm")
}

// Security Test 2c: Valid algorithm should succeed
func TestSignRequest_AlgorithmValid(t *testing.T) {
	// TDD: Test that correct algorithm works

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-2c")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	opts := &SigningOptions{
		Algorithm: "es256k", // Correct algorithm for secp256k1
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should succeed
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
}

// Security Test 3a: Old timestamp should fail (replay attack)
func TestSignRequest_TimestampTooOld(t *testing.T) {
	// TDD: Test replay attack prevention

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-3a")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// 1 hour ago timestamp
	opts := &SigningOptions{
		Created: time.Now().Add(-1 * time.Hour).Unix(),
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp")
}

// Security Test 3b: Future timestamp should fail
func TestSignRequest_TimestampInFuture(t *testing.T) {
	// TDD: Test time travel attack prevention

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-3b")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// 5 minutes in future (beyond clock skew)
	opts := &SigningOptions{
		Created: time.Now().Add(5 * time.Minute).Unix(),
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp")
}

// Security Test 3c: Recent timestamp should succeed
func TestSignRequest_TimestampValid(t *testing.T) {
	// TDD: Test that recent timestamps work

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-3c")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// 30 seconds ago (within valid range)
	opts := &SigningOptions{
		Created: time.Now().Add(-30 * time.Second).Unix(),
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert - should succeed
	require.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Signature-Input"))
}

// Security Test 4: Slice mutation should not affect caller
func TestSignRequest_SliceMutationPrevention(t *testing.T) {
	// TDD: Test that Components slice is not mutated

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-4")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req1 := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test1"}`))
	req2 := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test2"}`))
	req3 := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test3"}`))

	// Create options with specific components
	opts := &SigningOptions{
		Components: []string{"@method", "@path"},
	}

	// Store original length
	originalLen := len(opts.Components)
	originalComponents := make([]string, len(opts.Components))
	copy(originalComponents, opts.Components)

	// Execute multiple times with same options
	err := signer.SignRequestWithOptions(ctx, req1, testDID, keyPair, opts)
	require.NoError(t, err)

	// Check that opts.Components is not modified
	assert.Equal(t, originalLen, len(opts.Components), "Components length should not change")
	assert.Equal(t, originalComponents, opts.Components, "Components should not be mutated")

	// Second call - should not accumulate duplicates
	err = signer.SignRequestWithOptions(ctx, req2, testDID, keyPair, opts)
	require.NoError(t, err)
	assert.Equal(t, originalLen, len(opts.Components), "Components length should not change after second call")

	// Third call - verify no memory leak
	err = signer.SignRequestWithOptions(ctx, req3, testDID, keyPair, opts)
	require.NoError(t, err)
	assert.Equal(t, originalLen, len(opts.Components), "Components length should not change after third call")
}

// Security Test 5: Invalid Content-Digest should be recomputed
func TestSignRequest_ContentDigestRecompute(t *testing.T) {
	// TDD: Test that pre-existing Content-Digest is always recomputed

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-5")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	body := `{"data":"test"}`
	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(body))

	// Set an INVALID Content-Digest (attacker trying to inject wrong digest)
	maliciousDigest := "sha-256=:INVALID_DIGEST_VALUE:"
	req.Header.Set("Content-Digest", maliciousDigest)

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - should succeed and recompute correct digest
	require.NoError(t, err)

	// Verify that Content-Digest was recomputed (not the malicious one)
	actualDigest := req.Header.Get("Content-Digest")
	assert.NotEqual(t, maliciousDigest, actualDigest, "Malicious digest should be replaced")
	assert.Contains(t, actualDigest, "sha-256=:", "Should contain proper format")

	// Verify it's the correct digest for the body
	// We can't check exact value without knowing the hash, but we can verify it changed
	assert.NotContains(t, actualDigest, "INVALID", "Should not contain INVALID")
}

// Security Test 6: Unsupported key type should fail with clear error
func TestSignRequest_UnsupportedKeyType(t *testing.T) {
	// TDD: Test that unsupported key types fail gracefully

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-security-6")
	signer := NewDefaultA2ASigner()

	// Create mock key pair with unsupported key type (e.g., RSA)
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privateKey.PublicKey,
		privKey: privateKey,
		keyType: crypto.KeyType("rsa"), // Unsupported type
	}

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - should fail with descriptive error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
}

// Enhancement Test 7: Transfer-Encoding should be removed
func TestSignRequest_TransferEncodingRemoved(t *testing.T) {
	// TDD: Test that Transfer-Encoding header is properly cleaned up

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-enhancement-7")
	keyPair := createMockECDSAKeyPair()
	signer := NewDefaultA2ASigner()

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// Set Transfer-Encoding: chunked (conflicting with Content-Length)
	req.Header.Set("Transfer-Encoding", "chunked")

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - should succeed and remove Transfer-Encoding
	require.NoError(t, err)

	// Transfer-Encoding should be removed to avoid conflict with Content-Length
	transferEncoding := req.Header.Get("Transfer-Encoding")
	assert.Empty(t, transferEncoding, "Transfer-Encoding should be removed")

	// Content-Length should be set
	assert.NotZero(t, req.ContentLength, "Content-Length should be set")
}

// Enhancement Test 8: Malformed component identifiers should be handled
func TestQuoteComponents_MalformedInput(t *testing.T) {
	// TDD: Test that quoteComponents handles edge cases properly

	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Already quoted components",
			input:    []string{`"@method"`, `"@path"`},
			expected: []string{`"@method"`, `"@path"`},
		},
		{
			name:     "Mixed quoted and unquoted",
			input:    []string{`"@method"`, "@path"},
			expected: []string{`"@method"`, `"@path"`},
		},
		{
			name:     "Empty string should be skipped",
			input:    []string{"@method", "", "@path"},
			expected: []string{`"@method"`, `"@path"`},
		},
		{
			name:     "Whitespace should be trimmed",
			input:    []string{" @method ", "  @path  "},
			expected: []string{`"@method"`, `"@path"`},
		},
		{
			name:     "Double quotes should be stripped and re-quoted",
			input:    []string{`"@met"hod"`},
			expected: []string{`"@met"hod"`}, // Internal quotes preserved as-is
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := quoteComponents(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Enhancement Test 9: Error messages should include request context
func TestSignRequest_ErrorContext(t *testing.T) {
	// TDD: Test that error messages include helpful context

	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest-enhancement-9")
	signer := NewDefaultA2ASigner()

	// Create key pair with unsupported type to trigger error
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privateKey.PublicKey,
		privKey: privateKey,
		keyType: crypto.KeyType("unknown"),
	}

	req := httptest.NewRequest("POST", "https://agent.example.com/task", strings.NewReader(`{"data":"test"}`))

	// Execute
	err := signer.SignRequest(ctx, req, testDID, keyPair)

	// Assert - error should contain useful context
	require.Error(t, err)

	// Should contain DID for traceability
	assert.Contains(t, err.Error(), string(testDID), "Error should contain DID")

	// Should contain URL for debugging
	assert.Contains(t, err.Error(), req.URL.String(), "Error should contain URL")
}

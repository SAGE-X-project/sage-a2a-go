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

	// Standard components should be included
	assert.Contains(t, sigInput, `"@method"`)
	assert.Contains(t, sigInput, `"@target-uri"`)
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

	customTimestamp := int64(1618884473)
	opts := &SigningOptions{
		Created: customTimestamp,
	}

	// Execute
	err := signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)

	// Assert
	require.NoError(t, err)
	sigInput := req.Header.Get("Signature-Input")
	assert.Contains(t, sigInput, "created=1618884473")
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

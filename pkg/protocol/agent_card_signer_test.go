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

package protocol

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyPair implements crypto.KeyPair for testing
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

func (m *mockKeyPair) Sign(data []byte) ([]byte, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	// Simple mock signature
	return []byte("mock-signature-" + base64.StdEncoding.EncodeToString(data[:min(10, len(data))])), nil
}

func (m *mockKeyPair) Verify(data, signature []byte) error {
	// Simple mock verification - accepts any mock signature
	if len(signature) > 15 && string(signature[:15]) == "mock-signature-" {
		return nil
	}
	return errors.New("invalid signature")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// mockEthereumClient for DID resolution in verification tests
type mockEthereumClient struct {
	publicKeys map[did.AgentDID]map[did.KeyType]interface{}
	err        error
}

func (m *mockEthereumClient) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.publicKeys != nil {
		if keyMap, found := m.publicKeys[agentDID]; found {
			if pubKey, found := keyMap[keyType]; found {
				return pubKey, nil
			}
		}
	}

	return nil, errors.New("key not found")
}

// Helper functions to create test keys
func createTestECDSAKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return privKey, &privKey.PublicKey
}

func createTestEd25519KeyPair() (ed25519.PrivateKey, ed25519.PublicKey) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return privKey, pubKey
}

func TestAgentCardBuilder_Build(t *testing.T) {
	// Test Case 1: Build basic Agent Card with required fields
	testDID := did.AgentDID("did:sage:ethereum:0xtest1")

	builder := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com")
	card := builder.Build()

	require.NotNil(t, card)
	assert.Equal(t, string(testDID), card.DID)
	assert.Equal(t, "Test Agent", card.Name)
	assert.Equal(t, "https://agent.example.com", card.Endpoint)
	assert.Equal(t, "1.0", card.Version)
	assert.NotZero(t, card.CreatedAt)
}

func TestAgentCardBuilder_WithDescription(t *testing.T) {
	// Test Case 2: Build Agent Card with description
	testDID := did.AgentDID("did:sage:ethereum:0xtest2")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("A test agent for unit testing").
		Build()

	assert.Equal(t, "A test agent for unit testing", card.Description)
}

func TestAgentCardBuilder_WithCapabilities(t *testing.T) {
	// Test Case 3: Build Agent Card with capabilities
	testDID := did.AgentDID("did:sage:ethereum:0xtest3")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithCapabilities("task.execute", "data.process").
		WithCapabilities("messaging.send").
		Build()

	assert.Len(t, card.Capabilities, 3)
	assert.Contains(t, card.Capabilities, "task.execute")
	assert.Contains(t, card.Capabilities, "data.process")
	assert.Contains(t, card.Capabilities, "messaging.send")
}

func TestAgentCardBuilder_WithPublicKey(t *testing.T) {
	// Test Case 4: Build Agent Card with public keys
	testDID := did.AgentDID("did:sage:ethereum:0xtest4")

	keyInfo := PublicKeyInfo{
		ID:      "key-1",
		Type:    "EcdsaSecp256k1VerificationKey2019",
		KeyData: "base64-encoded-key-data",
		Purpose: []string{"authentication", "signing"},
	}

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithPublicKey(keyInfo).
		Build()

	assert.Len(t, card.PublicKeys, 1)
	assert.Equal(t, "key-1", card.PublicKeys[0].ID)
	assert.Equal(t, "EcdsaSecp256k1VerificationKey2019", card.PublicKeys[0].Type)
}

func TestAgentCardBuilder_WithExpiresAt(t *testing.T) {
	// Test Case 5: Build Agent Card with expiration
	testDID := did.AgentDID("did:sage:ethereum:0xtest5")
	expiresAt := time.Now().Add(24 * time.Hour)

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithExpiresAt(expiresAt).
		Build()

	assert.Equal(t, expiresAt.Unix(), card.ExpiresAt)
}

func TestAgentCardBuilder_WithMetadata(t *testing.T) {
	// Test Case 6: Build Agent Card with metadata
	testDID := did.AgentDID("did:sage:ethereum:0xtest6")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithMetadata("region", "us-west-2").
		WithMetadata("tier", "premium").
		Build()

	assert.Len(t, card.Metadata, 2)
	assert.Equal(t, "us-west-2", card.Metadata["region"])
	assert.Equal(t, "premium", card.Metadata["tier"])
}

func TestAgentCardBuilder_FluentAPI(t *testing.T) {
	// Test Case 7: Test fluent API chaining
	testDID := did.AgentDID("did:sage:ethereum:0xtest7")
	expiresAt := time.Now().Add(24 * time.Hour)

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("Full featured agent").
		WithCapabilities("task.execute", "messaging.send").
		WithExpiresAt(expiresAt).
		WithMetadata("region", "us-west-2").
		Build()

	assert.Equal(t, "Full featured agent", card.Description)
	assert.Len(t, card.Capabilities, 2)
	assert.NotZero(t, card.ExpiresAt)
	assert.NotEmpty(t, card.Metadata)
}

func TestAgentCard_Validate_Success(t *testing.T) {
	// Test Case 8: Validate valid Agent Card
	card := &AgentCard{
		DID:       "did:sage:ethereum:0xtest8",
		Name:      "Test Agent",
		Endpoint:  "https://agent.example.com",
		CreatedAt: time.Now().Unix(),
	}

	err := card.Validate()
	assert.NoError(t, err)
}

func TestAgentCard_Validate_MissingDID(t *testing.T) {
	// Test Case 9: Validate Agent Card with missing DID
	card := &AgentCard{
		Name:      "Test Agent",
		Endpoint:  "https://agent.example.com",
		CreatedAt: time.Now().Unix(),
	}

	err := card.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DID is required")
}

func TestAgentCard_Validate_MissingName(t *testing.T) {
	// Test Case 10: Validate Agent Card with missing name
	card := &AgentCard{
		DID:       "did:sage:ethereum:0xtest10",
		Endpoint:  "https://agent.example.com",
		CreatedAt: time.Now().Unix(),
	}

	err := card.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestAgentCard_Validate_MissingEndpoint(t *testing.T) {
	// Test Case 11: Validate Agent Card with missing endpoint
	card := &AgentCard{
		DID:       "did:sage:ethereum:0xtest11",
		Name:      "Test Agent",
		CreatedAt: time.Now().Unix(),
	}

	err := card.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

func TestAgentCard_Validate_MissingCreatedAt(t *testing.T) {
	// Test Case 12: Validate Agent Card with missing createdAt
	card := &AgentCard{
		DID:      "did:sage:ethereum:0xtest12",
		Name:     "Test Agent",
		Endpoint: "https://agent.example.com",
	}

	err := card.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "createdAt is required")
}

func TestAgentCard_IsExpired_NotExpired(t *testing.T) {
	// Test Case 13: Check non-expired Agent Card
	card := &AgentCard{
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	assert.False(t, card.IsExpired())
}

func TestAgentCard_IsExpired_Expired(t *testing.T) {
	// Test Case 14: Check expired Agent Card
	card := &AgentCard{
		ExpiresAt: time.Now().Add(-24 * time.Hour).Unix(),
	}

	assert.True(t, card.IsExpired())
}

func TestAgentCard_IsExpired_NoExpiration(t *testing.T) {
	// Test Case 15: Check Agent Card with no expiration
	card := &AgentCard{
		ExpiresAt: 0,
	}

	assert.False(t, card.IsExpired())
}

func TestAgentCard_HasCapability_Found(t *testing.T) {
	// Test Case 16: Check existing capability
	card := &AgentCard{
		Capabilities: []string{"task.execute", "messaging.send", "data.process"},
	}

	assert.True(t, card.HasCapability("task.execute"))
	assert.True(t, card.HasCapability("messaging.send"))
}

func TestAgentCard_HasCapability_NotFound(t *testing.T) {
	// Test Case 17: Check non-existing capability
	card := &AgentCard{
		Capabilities: []string{"task.execute", "messaging.send"},
	}

	assert.False(t, card.HasCapability("data.process"))
}

func TestDefaultAgentCardSigner_SignAgentCard_ECDSA(t *testing.T) {
	// Test Case 18: Sign Agent Card with ECDSA key
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest18")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("Test signing with ECDSA").
		Build()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	require.NoError(t, err)
	require.NotNil(t, signedCard)
	assert.NotEmpty(t, signedCard.Signature)
	assert.NotZero(t, signedCard.SignedAt)
	assert.Equal(t, card, signedCard.Card)
}

func TestDefaultAgentCardSigner_SignAgentCard_Ed25519(t *testing.T) {
	// Test Case 19: Sign Agent Card with Ed25519 key
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest19")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("Test signing with Ed25519").
		Build()

	privKey, pubKey := createTestEd25519KeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeEd25519,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	require.NoError(t, err)
	require.NotNil(t, signedCard)
	assert.NotEmpty(t, signedCard.Signature)
	assert.NotZero(t, signedCard.SignedAt)
}

func TestDefaultAgentCardSigner_SignAgentCard_NilCard(t *testing.T) {
	// Test Case 20: Sign with nil card
	ctx := context.Background()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, nil, keyPair)

	require.Error(t, err)
	assert.Nil(t, signedCard)
	assert.Contains(t, err.Error(), "card cannot be nil")
}

func TestDefaultAgentCardSigner_SignAgentCard_NilKeyPair(t *testing.T) {
	// Test Case 21: Sign with nil key pair
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest21")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, nil)

	require.Error(t, err)
	assert.Nil(t, signedCard)
	assert.Contains(t, err.Error(), "keyPair cannot be nil")
}

func TestDefaultAgentCardSigner_SignAgentCard_InvalidCard(t *testing.T) {
	// Test Case 22: Sign invalid card
	ctx := context.Background()

	card := &AgentCard{
		Name: "Test Agent",
		// Missing required fields
	}

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	require.Error(t, err)
	assert.Nil(t, signedCard)
	assert.Contains(t, err.Error(), "invalid agent card")
}

func TestDefaultAgentCardSigner_SignAgentCard_SigningError(t *testing.T) {
	// Test Case 23: Handle signing error
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest23")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
		signErr: errors.New("signing failed"),
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	require.Error(t, err)
	assert.Nil(t, signedCard)
	assert.Contains(t, err.Error(), "signing failed")
}

func TestDefaultAgentCardSigner_SignAgentCard_ContextCancellation(t *testing.T) {
	// Test Case 24: Handle context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testDID := did.AgentDID("did:sage:ethereum:0xtest24")
	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	require.Error(t, err)
	assert.Nil(t, signedCard)
	assert.Contains(t, err.Error(), "context")
}

func TestDefaultAgentCardSigner_VerifyAgentCard_ValidSignature(t *testing.T) {
	// Test Case 25: Verify valid Agent Card signature
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest25")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	// Create mock client that returns the public key
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: pubKey,
			},
		},
	}

	signer := NewDefaultAgentCardSigner(client)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)
	require.NoError(t, err)

	// Verify the signature
	err = signer.VerifyAgentCard(ctx, signedCard)
	assert.NoError(t, err)
}

func TestDefaultAgentCardSigner_VerifyAgentCard_InvalidSignature(t *testing.T) {
	// Test Case 26: Verify Agent Card with invalid signature (wrong format)
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest26")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	_, pubKey := createTestECDSAKeyPair()

	// Create mock client
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: pubKey,
			},
		},
	}

	signer := NewDefaultAgentCardSigner(client)

	// Create signed card with invalid JWS format
	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: "invalid-signature-data",
		SignedAt:  time.Now().Unix(),
	}

	err := signer.VerifyAgentCard(ctx, signedCard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JWS format")
}

func TestDefaultAgentCardSigner_VerifyAgentCard_NilSignedCard(t *testing.T) {
	// Test Case 27: Verify with nil signed card
	ctx := context.Background()

	signer := NewDefaultAgentCardSigner(nil)
	err := signer.VerifyAgentCard(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "signedCard cannot be nil")
}

func TestDefaultAgentCardSigner_VerifyAgentCard_NilCard(t *testing.T) {
	// Test Case 28: Verify with nil card in signed card
	ctx := context.Background()

	signedCard := &SignedAgentCard{
		Card:      nil,
		Signature: "some-signature",
		SignedAt:  time.Now().Unix(),
	}

	signer := NewDefaultAgentCardSigner(nil)
	err := signer.VerifyAgentCard(ctx, signedCard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "card cannot be nil")
}

func TestDefaultAgentCardSigner_VerifyAgentCard_KeyResolutionError(t *testing.T) {
	// Test Case 29: Handle key resolution error
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest29")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create a properly formatted JWS with valid header
	header := map[string]string{"alg": "ES256K", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(card)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	// Create properly formatted JWS (header.payload.signature)
	jwsSignature := headerB64 + "." + payloadB64 + ".fake-signature-data"

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	client := &mockEthereumClient{
		err: errors.New("DID resolution failed"),
	}

	signer := NewDefaultAgentCardSigner(client)
	err := signer.VerifyAgentCard(ctx, signedCard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DID resolution failed")
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_ValidSignature(t *testing.T) {
	// Test Case 30: Verify with provided public key
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest30")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeSecp256k1,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)
	require.NoError(t, err)

	// Verify with the public key directly
	err = signer.VerifyAgentCardWithKey(ctx, signedCard, pubKey)
	assert.NoError(t, err)
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_InvalidSignature(t *testing.T) {
	// Test Case 31: Verify with wrong signature format
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest31")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create a properly formatted JWS with non-mock signature that will fail verification
	header := map[string]string{"alg": "ES256K", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(card)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	// Use a non-mock signature that will fail verification
	invalidSig := base64.RawURLEncoding.EncodeToString([]byte("not-a-valid-signature"))
	jwsSignature := headerB64 + "." + payloadB64 + "." + invalidSig

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	_, pubKey := createTestECDSAKeyPair()

	signer := NewDefaultAgentCardSigner(nil)
	err := signer.VerifyAgentCardWithKey(ctx, signedCard, pubKey)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "verify")
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_NilPublicKey(t *testing.T) {
	// Test Case 32: Verify with nil public key
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest32")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: "some-signature",
		SignedAt:  time.Now().Unix(),
	}

	signer := NewDefaultAgentCardSigner(nil)
	err := signer.VerifyAgentCardWithKey(ctx, signedCard, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "publicKey cannot be nil")
}

func TestSignedAgentCard_JSON_Serialization(t *testing.T) {
	// Test Case 33: JSON serialization/deserialization
	testDID := did.AgentDID("did:sage:ethereum:0xtest33")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("Test JSON serialization").
		WithCapabilities("task.execute").
		Build()

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: "test-signature-data",
		SignedAt:  time.Now().Unix(),
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(signedCard)
	require.NoError(t, err)

	// Deserialize from JSON
	var decoded SignedAgentCard
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	assert.Equal(t, signedCard.Card.DID, decoded.Card.DID)
	assert.Equal(t, signedCard.Card.Name, decoded.Card.Name)
	assert.Equal(t, signedCard.Signature, decoded.Signature)
	assert.Equal(t, signedCard.SignedAt, decoded.SignedAt)
}

func TestAgentCard_JSON_Serialization(t *testing.T) {
	// Test Case 34: Agent Card JSON serialization
	testDID := did.AgentDID("did:sage:ethereum:0xtest34")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").
		WithDescription("Full featured card").
		WithCapabilities("task.execute", "messaging.send").
		WithMetadata("region", "us-west-2").
		Build()

	// Serialize to JSON
	jsonData, err := json.Marshal(card)
	require.NoError(t, err)

	// Deserialize from JSON
	var decoded AgentCard
	err = json.Unmarshal(jsonData, &decoded)
	require.NoError(t, err)

	assert.Equal(t, card.DID, decoded.DID)
	assert.Equal(t, card.Name, decoded.Name)
	assert.Equal(t, card.Description, decoded.Description)
	assert.Equal(t, card.Endpoint, decoded.Endpoint)
	assert.ElementsMatch(t, card.Capabilities, decoded.Capabilities)
	assert.Equal(t, card.Metadata["region"], decoded.Metadata["region"])
}

func TestDefaultAgentCardSigner_VerifyAgentCard_Ed25519(t *testing.T) {
	// Test Case 35: Verify Ed25519 signature
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest35")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	privKey, pubKey := createTestEd25519KeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: crypto.KeyTypeEd25519,
	}

	// Create mock client that returns the Ed25519 public key
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeEd25519: pubKey,
			},
		},
	}

	signer := NewDefaultAgentCardSigner(client)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)
	require.NoError(t, err)

	// Verify the signature
	err = signer.VerifyAgentCard(ctx, signedCard)
	assert.NoError(t, err)
}

func TestDefaultAgentCardSigner_ExtractKeyType_InvalidJSON(t *testing.T) {
	// Test Case 36: Test extractKeyTypeFromSignature with invalid base64
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest36")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create JWS with invalid base64 in header
	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: "!!!invalid-base64!!!.payload.signature",
		SignedAt:  time.Now().Unix(),
	}

	client := &mockEthereumClient{}
	signer := NewDefaultAgentCardSigner(client)

	err := signer.VerifyAgentCard(ctx, signedCard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract key type")
}

func TestDefaultAgentCardSigner_ExtractKeyType_MissingAlgorithm(t *testing.T) {
	// Test Case 37: Test extractKeyTypeFromSignature with missing algorithm
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest37")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create header without "alg" field
	header := map[string]string{"typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(card)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	jwsSignature := headerB64 + "." + payloadB64 + ".signature"

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	client := &mockEthereumClient{}
	signer := NewDefaultAgentCardSigner(client)

	err := signer.VerifyAgentCard(ctx, signedCard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing algorithm")
}

func TestDefaultAgentCardSigner_ExtractKeyType_UnsupportedAlgorithm(t *testing.T) {
	// Test Case 38: Test extractKeyTypeFromSignature with unsupported algorithm
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest38")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create header with unsupported algorithm
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(card)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	jwsSignature := headerB64 + "." + payloadB64 + ".signature"

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	client := &mockEthereumClient{}
	signer := NewDefaultAgentCardSigner(client)

	err := signer.VerifyAgentCard(ctx, signedCard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_InvalidPayload(t *testing.T) {
	// Test Case 39: Test VerifyAgentCardWithKey with invalid payload
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest39")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create JWS with invalid payload
	header := map[string]string{"alg": "ES256K", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Invalid base64 payload
	jwsSignature := headerB64 + ".!!!invalid-base64!!!.bW9jay1zaWduYXR1cmUtZGF0YQ"

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	_, pubKey := createTestECDSAKeyPair()
	signer := NewDefaultAgentCardSigner(nil)

	err := signer.VerifyAgentCardWithKey(ctx, signedCard, pubKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_PayloadMismatch(t *testing.T) {
	// Test Case 40: Test VerifyAgentCardWithKey with payload DID mismatch
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest40")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create payload with different DID
	differentCard := NewAgentCardBuilder("did:sage:ethereum:0xdifferent", "Other Agent", "https://other.com").Build()

	header := map[string]string{"alg": "ES256K", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(differentCard)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	// Use mock signature
	mockSig := base64.RawURLEncoding.EncodeToString([]byte("mock-signature-test"))
	jwsSignature := headerB64 + "." + payloadB64 + "." + mockSig

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	_, pubKey := createTestECDSAKeyPair()
	signer := NewDefaultAgentCardSigner(nil)

	err := signer.VerifyAgentCardWithKey(ctx, signedCard, pubKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "payload DID mismatch")
}

func TestDefaultAgentCardSigner_GetAlgorithm_UnknownKeyType(t *testing.T) {
	// Test Case 41: Test getAlgorithmFromKeyType with unknown key type
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest41")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Use an unknown key type
	privKey, pubKey := createTestECDSAKeyPair()
	keyPair := &mockKeyPair{
		pubKey:  pubKey,
		privKey: privKey,
		keyType: "unknown-key-type", // Unknown key type
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)

	// Should still succeed, using default algorithm
	require.NoError(t, err)
	assert.NotNil(t, signedCard)
}

func TestDefaultAgentCardSigner_VerifySignature_UnsupportedKeyType(t *testing.T) {
	// Test Case 42: Test verifySignature with unsupported public key type
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest42")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create JWS with valid structure
	header := map[string]string{"alg": "ES256K", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	payload, _ := json.Marshal(card)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	mockSig := base64.RawURLEncoding.EncodeToString([]byte("mock-signature-test"))
	jwsSignature := headerB64 + "." + payloadB64 + "." + mockSig

	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: jwsSignature,
		SignedAt:  time.Now().Unix(),
	}

	// Use an unsupported public key type (string instead of crypto key)
	unsupportedPubKey := "not-a-real-public-key"

	signer := NewDefaultAgentCardSigner(nil)
	err := signer.VerifyAgentCardWithKey(ctx, signedCard, unsupportedPubKey)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported public key type")
}

func TestDefaultAgentCardSigner_VerifyAgentCardWithKey_InvalidJWSFormat(t *testing.T) {
	// Test Case 43: Test VerifyAgentCardWithKey with invalid JWS parts count
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest43")

	card := NewAgentCardBuilder(testDID, "Test Agent", "https://agent.example.com").Build()

	// Create JWS with wrong number of parts (only 2 instead of 3)
	signedCard := &SignedAgentCard{
		Card:      card,
		Signature: "header.payload", // Missing signature part
		SignedAt:  time.Now().Unix(),
	}

	_, pubKey := createTestECDSAKeyPair()
	signer := NewDefaultAgentCardSigner(nil)

	err := signer.VerifyAgentCardWithKey(ctx, signedCard, pubKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JWS format")
}

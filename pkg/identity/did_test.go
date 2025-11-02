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

package identity

import (
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentDID_Type(t *testing.T) {
	// Test that AgentDID is a proper type alias
	did := AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	assert.NotEmpty(t, did)
	assert.IsType(t, AgentDID(""), did)
}

func TestKeyTypeConstants(t *testing.T) {
	// Verify key type constants are properly exported
	// KeyType is an int type in SAGE with specific values
	assert.Equal(t, KeyType(0), KeyTypeECDSA)   // MUST be 0
	assert.Equal(t, KeyType(1), KeyTypeEd25519) // MUST be 1
	assert.Equal(t, KeyType(2), KeyTypeX25519)  // MUST be 2

	// Verify they are distinct
	types := map[KeyType]bool{
		KeyTypeECDSA:   true,
		KeyTypeEd25519: true,
		KeyTypeX25519:  true,
	}
	assert.Equal(t, 3, len(types), "All key types should be distinct")
}

func TestValidateDID_ValidDIDs(t *testing.T) {
	validDIDs := []string{
		"did:sage:ethereum:0x1234567890123456789012345678901234567890",
		"did:sage:solana:11111111111111111111111111111111",
	}

	for _, didStr := range validDIDs {
		err := ValidateDID(didStr)
		assert.NoError(t, err, "DID should be valid: %s", didStr)
	}
}

func TestValidateDID_InvalidDIDs(t *testing.T) {
	invalidDIDs := []string{
		"",
		"not-a-did",
		"did:other:ethereum:0x1234",
		"did:sage:",
		"did:sage:ethereum",
		"sage:ethereum:0x1234567890123456789012345678901234567890",
	}

	for _, didStr := range invalidDIDs {
		err := ValidateDID(didStr)
		assert.Error(t, err, "DID should be invalid: %s", didStr)
	}
}

func TestParseDID_EthereumDID(t *testing.T) {
	did := AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	chain, address, err := ParseDID(did)
	require.NoError(t, err)

	assert.Equal(t, "ethereum", string(chain))
	assert.Equal(t, "0x1234567890123456789012345678901234567890", address)
}

func TestParseDID_SolanaDID(t *testing.T) {
	did := AgentDID("did:sage:solana:11111111111111111111111111111111")

	chain, address, err := ParseDID(did)
	require.NoError(t, err)

	assert.Equal(t, "solana", string(chain))
	assert.Equal(t, "11111111111111111111111111111111", address)
}

func TestParseDID_InvalidFormat(t *testing.T) {
	invalidDIDs := []AgentDID{
		"not-a-did",
		"did:other:ethereum:0x1234",
		"did:sage:",
		"",
	}

	for _, did := range invalidDIDs {
		chain, address, err := ParseDID(did)
		assert.Error(t, err, "Should fail for invalid DID: %s", did)
		assert.Empty(t, chain)
		assert.Empty(t, address)
	}
}

func TestMarshalPublicKey_ECDSA(t *testing.T) {
	// Generate ECDSA key pair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey()

	// Marshal public key
	keyData, err := MarshalPublicKey(pubKey)
	require.NoError(t, err)
	assert.NotEmpty(t, keyData)

	// Verify marshaled data is not empty
	assert.Greater(t, len(keyData), 0)
}

func TestMarshalPublicKey_Ed25519(t *testing.T) {
	// Generate Ed25519 key pair
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey()

	// Marshal public key
	keyData, err := MarshalPublicKey(pubKey)
	require.NoError(t, err)
	assert.NotEmpty(t, keyData)

	// Ed25519 public key should be 32 bytes
	assert.Equal(t, 32, len(keyData))
}

func TestUnmarshalPublicKey_Secp256k1(t *testing.T) {
	// Generate and marshal ECDSA key
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	originalPubKey := keyPair.PublicKey()
	keyData, err := MarshalPublicKey(originalPubKey)
	require.NoError(t, err)

	// Unmarshal public key
	pubKey, err := UnmarshalPublicKey(keyData, "secp256k1")
	require.NoError(t, err)
	assert.NotNil(t, pubKey)

	// Verify it's the same key
	assert.Equal(t, originalPubKey, pubKey)
}

func TestUnmarshalPublicKey_Ed25519(t *testing.T) {
	// Generate and marshal Ed25519 key
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	originalPubKey := keyPair.PublicKey()
	keyData, err := MarshalPublicKey(originalPubKey)
	require.NoError(t, err)

	// Unmarshal public key
	pubKey, err := UnmarshalPublicKey(keyData, "ed25519")
	require.NoError(t, err)
	assert.NotNil(t, pubKey)

	// Verify it's the same key
	assert.Equal(t, originalPubKey, pubKey)
}

func TestUnmarshalPublicKey_InvalidKeyType(t *testing.T) {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	keyData, err := MarshalPublicKey(keyPair.PublicKey())
	require.NoError(t, err)

	// Try to unmarshal with invalid key type
	pubKey, err := UnmarshalPublicKey(keyData, "invalid-key-type")
	assert.Error(t, err)
	assert.Nil(t, pubKey)
}

func TestUnmarshalPublicKey_InvalidData(t *testing.T) {
	invalidData := []byte("invalid-key-data")

	pubKey, err := UnmarshalPublicKey(invalidData, "secp256k1")
	assert.Error(t, err)
	assert.Nil(t, pubKey)
}

func TestAgentKey_Type(t *testing.T) {
	// Test that AgentKey is properly aliased
	key := AgentKey{
		Type:     KeyTypeECDSA,
		KeyData:  []byte("test"),
		Verified: true,
	}

	assert.Equal(t, KeyTypeECDSA, key.Type)
	assert.True(t, key.Verified)
	assert.NotEmpty(t, key.KeyData)
}

func TestAgentMetadata_Type(t *testing.T) {
	// Test that AgentMetadata is properly aliased
	metadata := AgentMetadata{
		DID:      AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890"),
		IsActive: true,
		Keys:     []AgentKey{},
	}

	assert.NotEmpty(t, metadata.DID)
	assert.True(t, metadata.IsActive)
	assert.NotNil(t, metadata.Keys)
}

func TestMarshalUnmarshalRoundTrip_ECDSA(t *testing.T) {
	// Test complete marshal/unmarshal round trip for ECDSA
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	originalPubKey := keyPair.PublicKey()

	// Marshal
	keyData, err := MarshalPublicKey(originalPubKey)
	require.NoError(t, err)

	// Unmarshal
	recoveredPubKey, err := UnmarshalPublicKey(keyData, "secp256k1")
	require.NoError(t, err)

	// Verify keys match
	assert.Equal(t, originalPubKey, recoveredPubKey)
}

func TestMarshalUnmarshalRoundTrip_Ed25519(t *testing.T) {
	// Test complete marshal/unmarshal round trip for Ed25519
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	originalPubKey := keyPair.PublicKey()

	// Marshal
	keyData, err := MarshalPublicKey(originalPubKey)
	require.NoError(t, err)

	// Unmarshal
	recoveredPubKey, err := UnmarshalPublicKey(keyData, "ed25519")
	require.NoError(t, err)

	// Verify keys match
	assert.Equal(t, originalPubKey, recoveredPubKey)
}

func TestKeyTypeCompatibility(t *testing.T) {
	// Verify that identity.KeyType is compatible with did.KeyType
	assert.Equal(t, did.KeyTypeECDSA, KeyTypeECDSA)
	assert.Equal(t, did.KeyTypeEd25519, KeyTypeEd25519)
	assert.Equal(t, did.KeyTypeX25519, KeyTypeX25519)
}

func TestAgentDIDCompatibility(t *testing.T) {
	// Verify that identity.AgentDID is compatible with did.AgentDID
	identityDID := AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	sageDID := did.AgentDID(identityDID)

	assert.Equal(t, string(identityDID), string(sageDID))
}

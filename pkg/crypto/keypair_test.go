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

package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecp256k1KeyPair(t *testing.T) {
	// Generate key pair
	keyPair, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keyPair)

	// Verify key type
	assert.Equal(t, KeyTypeSecp256k1, keyPair.Type())

	// Verify public key is ECDSA
	pubKey := keyPair.PublicKey()
	assert.NotNil(t, pubKey)
	_, ok := pubKey.(*ecdsa.PublicKey)
	assert.True(t, ok, "Public key should be *ecdsa.PublicKey")

	// Verify private key is not nil
	privKey := keyPair.PrivateKey()
	assert.NotNil(t, privKey)

	// Verify we can generate multiple different keys
	keyPair2, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, keyPair.PublicKey(), keyPair2.PublicKey(), "Each call should generate a unique key")
}

func TestGenerateEd25519KeyPair(t *testing.T) {
	// Generate key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keyPair)

	// Verify key type
	assert.Equal(t, KeyTypeEd25519, keyPair.Type())

	// Verify public key is Ed25519
	pubKey := keyPair.PublicKey()
	assert.NotNil(t, pubKey)
	_, ok := pubKey.(ed25519.PublicKey)
	assert.True(t, ok, "Public key should be ed25519.PublicKey")

	// Verify private key is not nil
	privKey := keyPair.PrivateKey()
	assert.NotNil(t, privKey)

	// Verify public key has correct length (32 bytes)
	ed25519PubKey := pubKey.(ed25519.PublicKey)
	assert.Equal(t, ed25519.PublicKeySize, len(ed25519PubKey))

	// Verify we can generate multiple different keys
	keyPair2, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, keyPair.PublicKey(), keyPair2.PublicKey(), "Each call should generate a unique key")
}

func TestKeyTypeConstants(t *testing.T) {
	// Verify key type constants are properly exported and not empty
	assert.NotEmpty(t, KeyTypeEd25519)
	assert.NotEmpty(t, KeyTypeSecp256k1)
	assert.NotEmpty(t, KeyTypeP256)
	assert.NotEmpty(t, KeyTypeX25519)

	// Verify they are distinct
	types := map[KeyType]bool{
		KeyTypeEd25519:   true,
		KeyTypeSecp256k1: true,
		KeyTypeP256:      true,
		KeyTypeX25519:    true,
	}
	assert.Equal(t, 4, len(types), "All key types should be distinct")
}

func TestKeyPairInterface(t *testing.T) {
	// Test that generated key pairs implement the KeyPair interface
	keyPair1, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	keyPair2, err := GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Verify both implement the interface by calling methods
	_ = keyPair1.PublicKey()
	_ = keyPair1.PrivateKey()
	_ = keyPair1.Type()

	_ = keyPair2.PublicKey()
	_ = keyPair2.PrivateKey()
	_ = keyPair2.Type()
}

func TestSecp256k1KeyPairSigning(t *testing.T) {
	// Generate key pair
	keyPair, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Get private key as crypto.Signer
	privKey := keyPair.PrivateKey()
	assert.NotNil(t, privKey)

	// Verify it's an ECDSA private key
	ecdsaPrivKey, ok := privKey.(*ecdsa.PrivateKey)
	assert.True(t, ok)

	// Verify public keys match
	assert.Equal(t, &ecdsaPrivKey.PublicKey, keyPair.PublicKey())
}

func TestEd25519KeyPairSigning(t *testing.T) {
	// Generate key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Get private key
	privKey := keyPair.PrivateKey()
	assert.NotNil(t, privKey)

	// Verify it's an Ed25519 private key
	ed25519PrivKey, ok := privKey.(ed25519.PrivateKey)
	assert.True(t, ok)

	// Verify private key has correct length (64 bytes)
	assert.Equal(t, ed25519.PrivateKeySize, len(ed25519PrivKey))

	// Verify public key can be derived from private key
	derivedPubKey := ed25519PrivKey.Public()
	assert.Equal(t, derivedPubKey, keyPair.PublicKey())
}

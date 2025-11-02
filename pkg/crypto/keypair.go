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
	"crypto"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// KeyPair represents a cryptographic key pair (wraps SAGE KeyPair)
// This is the main interface for key management in sage-a2a-go
type KeyPair = sagecrypto.KeyPair

// KeyType represents the type of cryptographic key
type KeyType = sagecrypto.KeyType

// Key types
const (
	KeyTypeEd25519   = sagecrypto.KeyTypeEd25519
	KeyTypeSecp256k1 = sagecrypto.KeyTypeSecp256k1
	KeyTypeP256      = sagecrypto.KeyTypeP256
	KeyTypeX25519    = sagecrypto.KeyTypeX25519
)

// GenerateSecp256k1KeyPair generates a new secp256k1 key pair (Ethereum-compatible)
// This is the recommended key type for Ethereum-based agents
func GenerateSecp256k1KeyPair() (KeyPair, error) {
	return keys.GenerateSecp256k1KeyPair()
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair
// This is the recommended key type for Solana-based agents
func GenerateEd25519KeyPair() (KeyPair, error) {
	return keys.GenerateEd25519KeyPair()
}

// GenerateP256KeyPair generates a new P-256 (NIST) key pair
func GenerateP256KeyPair() (KeyPair, error) {
	return keys.GenerateP256KeyPair()
}

// GenerateX25519KeyPair generates a new X25519 key pair (for HPKE/encryption)
func GenerateX25519KeyPair() (KeyPair, error) {
	return keys.GenerateX25519KeyPair()
}

// PublicKey is an alias for crypto.PublicKey
type PublicKey = crypto.PublicKey

// PrivateKey is an alias for crypto.PrivateKey
type PrivateKey = crypto.PrivateKey

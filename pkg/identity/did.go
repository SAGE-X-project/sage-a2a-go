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
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// AgentDID represents a Decentralized Identifier for an agent
// Format: did:sage:<chain>:<address>
// Examples:
//   - did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678
//   - did:sage:solana:11111111111111111111111111111111
type AgentDID = did.AgentDID

// KeyType represents the type of cryptographic key in DID documents
type KeyType = did.KeyType

// Key types for DID documents
const (
	KeyTypeECDSA   = did.KeyTypeECDSA
	KeyTypeEd25519 = did.KeyTypeEd25519
	KeyTypeX25519  = did.KeyTypeX25519
)

// AgentKey represents a public key in an agent's DID document
type AgentKey = did.AgentKey

// AgentMetadata contains metadata about an agent from blockchain registry
// Uses SAGE's AgentMetadata type (unified for v1/v4)
type AgentMetadata = did.AgentMetadata

// MarshalPublicKey marshals a public key to bytes for storage/transmission
func MarshalPublicKey(pubKey interface{}) ([]byte, error) {
	return did.MarshalPublicKey(pubKey)
}

// UnmarshalPublicKey unmarshals a public key from bytes
func UnmarshalPublicKey(data []byte, keyType string) (interface{}, error) {
	return did.UnmarshalPublicKey(data, keyType)
}

// ValidateDID validates a DID string format
func ValidateDID(didStr string) error {
	return did.ValidateDID(didStr)
}

// ParseDID parses a DID string into components
// Returns chain and address from did:sage:<chain>:<address>
func ParseDID(agentDID AgentDID) (chain did.Chain, address string, err error) {
	return did.ParseDID(agentDID)
}

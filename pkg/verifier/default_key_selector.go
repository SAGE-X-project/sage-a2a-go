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

package verifier

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

type DIDResolver interface {
	GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error)
}

type DefaultKeySelector struct {
	resolver DIDResolver
}

func NewDefaultKeySelector(resolver DIDResolver) *DefaultKeySelector {
	return &DefaultKeySelector{resolver: resolver}
}

// SelectKey selects the appropriate public key for a given protocol
//
// Protocol-based key selection:
//   - "ethereum"/"eth": ECDSA (secp256k1)
//   - "solana"/"sol": Ed25519
//   - "hpke"/"kem"/"x25519": X25519 (32 bytes) for HPKE
//   - Others: Fallback order (1) Ed25519, (2) ECDSA, (3) first verified key
func (s *DefaultKeySelector) SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("context error: %w", err)
	}

	meta, err := s.resolver.GetAgentByDID(ctx, string(agentDID))

	if err != nil {
		return nil, 0, fmt.Errorf("resolve agent: %w", err)
	}
	if meta == nil || !meta.IsActive {
		return nil, 0, fmt.Errorf("agent inactive or not found: %s", agentDID)
	}

	// Fast HPKE/KEM handling: check KEM-specific field first, then search key array for X25519
	proto := strings.ToLower(strings.TrimSpace(protocol))
	switch proto {
	case "hpke", "kem", "x25519":
		if len(meta.PublicKEMKey) == 32 {
			return crypto.PublicKey(meta.PublicKEMKey), did.KeyTypeX25519, nil
		}
		if pk, ok := firstByType(meta.Keys, did.KeyTypeX25519); ok {
			// X25519 returns raw 32-byte format
			return crypto.PublicKey(pk.KeyData), did.KeyTypeX25519, nil
		}
		return nil, 0, errors.New("no X25519 (HPKE) key registered")

	case "ethereum", "eth", "eip155":
		if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		}
		// Fallback: Ed25519 → others
		if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		}
		return firstAnyVerified(meta.Keys)

	case "solana", "sol":
		if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		}
		// Fallback: ECDSA → others
		if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		}
		return firstAnyVerified(meta.Keys)
	}

	// Default policy: Ed25519 > ECDSA > first verified key
	if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
		return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
	}
	if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
		return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
	}
	return firstAnyVerified(meta.Keys)
}

func firstByType(keys []did.AgentKey, t did.KeyType) (did.AgentKey, bool) {
	for _, k := range keys {
		if k.Verified && k.Type == t {
			return k, true
		}
	}
	return did.AgentKey{}, false
}

func firstAnyVerified(keys []did.AgentKey) (crypto.PublicKey, did.KeyType, error) {
	for _, k := range keys {
		if !k.Verified {
			continue
		}
		// X25519 returns raw bytes, signature keys need Unmarshal
		switch k.Type {
		case did.KeyTypeX25519:
			if len(k.KeyData) == 32 {
				return crypto.PublicKey(k.KeyData), did.KeyTypeX25519, nil
			}
		case did.KeyTypeECDSA:
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		case did.KeyTypeEd25519:
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		default:
			// Skip unknown types
		}
	}
	return nil, 0, errors.New("no verified keys available")
}

func unmarshalByKeyType(raw []byte, kt did.KeyType) (crypto.PublicKey, did.KeyType, error) {
	switch kt {
	case did.KeyTypeECDSA:
		pk, err := did.UnmarshalPublicKey(raw, "secp256k1")
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshal secp256k1: %w", err)
		}
		return pk.(crypto.PublicKey), did.KeyTypeECDSA, nil
	case did.KeyTypeEd25519:
		pk, err := did.UnmarshalPublicKey(raw, "ed25519")
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshal ed25519: %w", err)
		}
		return pk.(crypto.PublicKey), did.KeyTypeEd25519, nil
	case did.KeyTypeX25519:
		// X25519 returns 32-byte raw format (for HPKE)
		if len(raw) != 32 {
			return nil, 0, fmt.Errorf("x25519: want 32 bytes, got %d", len(raw))
		}
		return crypto.PublicKey(raw), did.KeyTypeX25519, nil
	default:
		return nil, 0, fmt.Errorf("unknown key type: %d", kt)
	}
}

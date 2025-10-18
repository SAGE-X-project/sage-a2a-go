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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// mockEthereumClientBench for benchmarking
type mockEthereumClientBench struct {
	publicKeys map[did.AgentDID]map[did.KeyType]interface{}
}

func (m *mockEthereumClientBench) ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error) {
	keys := []did.AgentKey{}
	if keyMap, found := m.publicKeys[agentDID]; found {
		for keyType, pubKey := range keyMap {
			keyData, _ := did.MarshalPublicKey(pubKey)
			keys = append(keys, did.AgentKey{
				Type:      keyType,
				KeyData:   keyData,
				Verified:  true,
				CreatedAt: time.Now(),
			})
		}
	}
	return keys, nil
}

func (m *mockEthereumClientBench) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
	if keyMap, found := m.publicKeys[agentDID]; found {
		if pubKey, found := keyMap[keyType]; found {
			return pubKey, nil
		}
	}
	return nil, fmt.Errorf("key not found")
}

// Benchmark key selection for Ethereum
func BenchmarkSelectKeyEthereum(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pubKey := &privKey.PublicKey

	client := &mockEthereumClientBench{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: pubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = selector.SelectKey(ctx, testDID, "ethereum")
	}
}

// Benchmark key selection for Solana
func BenchmarkSelectKeySolana(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)

	client := &mockEthereumClientBench{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeEd25519: pubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = selector.SelectKey(ctx, testDID, "solana")
	}
}

// Benchmark key selection with multiple keys
func BenchmarkSelectKeyMultiKey(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	ecdsaPrivKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	ecdsaPubKey := &ecdsaPrivKey.PublicKey
	ed25519PubKey, _, _ := ed25519.GenerateKey(rand.Reader)

	client := &mockEthereumClientBench{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA:   ecdsaPubKey,
				did.KeyTypeEd25519: ed25519PubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = selector.SelectKey(ctx, testDID, "ethereum")
	}
}

// Benchmark public key resolution
func BenchmarkResolvePublicKey(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pubKey := &privKey.PublicKey

	client := &mockEthereumClientBench{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: pubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	sigVerifier := NewRFC9421Verifier()
	verifier := NewDefaultDIDVerifier(client, selector, sigVerifier)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.ResolvePublicKey(ctx, testDID, nil)
	}
}

// Benchmark DID format validation
func BenchmarkIsValidDID(b *testing.B) {
	validDIDs := []string{
		"did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678",
		"did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678:1",
		"did:sage:ethereum:0xABCDEF1234567890abcdef1234567890ABCDEF12",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testDID := validDIDs[i%len(validDIDs)]
		_ = isValidDID(testDID)
	}
}

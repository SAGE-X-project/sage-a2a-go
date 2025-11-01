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
	"errors"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEthereumClient is a mock implementation of ethereum.EthereumClientV4
type mockEthereumClient struct {
    keys       map[did.AgentDID][]did.AgentKey
    publicKeys map[did.AgentDID]map[did.KeyType]interface{} // Direct public key mapping
    err        error
}

func (m *mockEthereumClient) ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error) {
	if m.err != nil {
		return nil, m.err
	}
	keys, found := m.keys[agentDID]
	if !found {
		return []did.AgentKey{}, nil
	}
	return keys, nil
}

func (m *mockEthereumClient) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
    if m.err != nil {
        return nil, m.err
    }

	// Use direct public key mapping to avoid unmarshal issues in tests
	if m.publicKeys != nil {
		if keyMap, found := m.publicKeys[agentDID]; found {
			if pubKey, found := keyMap[keyType]; found {
				return pubKey, nil
			}
		}
	}

    return nil, errors.New("key type not found")
}

// Satisfy DIDResolver used by DefaultKeySelector
func (m *mockEthereumClient) GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error) {
    if m.err != nil {
        return nil, m.err
    }
    d := did.AgentDID(didStr)
    meta := &did.AgentMetadataV4{
        DID:      d,
        IsActive: true,
        Keys:     []did.AgentKey{},
    }
    if ks, ok := m.keys[d]; ok {
        meta.Keys = append(meta.Keys, ks...)
    } else if m.publicKeys != nil {
        if keyMap, ok := m.publicKeys[d]; ok {
            for kt, pk := range keyMap {
                keyData, _ := did.MarshalPublicKey(pk)
                meta.Keys = append(meta.Keys, did.AgentKey{
                    Type:      kt,
                    KeyData:   keyData,
                    Verified:  true,
                    CreatedAt: time.Now(),
                })
            }
        }
    }
    return meta, nil
}

// Satisfy PublicKeyClient used by DefaultDIDVerifier
func (m *mockEthereumClient) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
    if m.err != nil {
        return nil, m.err
    }
    if m.publicKeys != nil {
        if keyMap, ok := m.publicKeys[agentDID]; ok {
            if pk, ok2 := keyMap[did.KeyTypeECDSA]; ok2 {
                return pk, nil
            }
            // return any
            for _, v := range keyMap {
                return v, nil
            }
        }
    }
    return nil, errors.New("DID not found")
}

func (m *mockEthereumClient) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
    if m.err != nil {
        return nil, m.err
    }
    // Return dummy 32-byte KEM key
    return make([]byte, 32), nil
}

// Helper functions to create test keys
func createECDSAKey() *ecdsa.PublicKey {
	// Create a dummy ECDSA public key for testing
	// Using P256 for simplicity in unit tests
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return &privateKey.PublicKey
}

func createEd25519Key() ed25519.PublicKey {
	// Create a dummy Ed25519 public key for testing
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return pubKey
}

func TestDefaultKeySelector_SelectKey_EthereumProtocol(t *testing.T) {
	// Test Case 1: Ethereum protocol should select ECDSA key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest1")

	ecdsaPubKey := createECDSAKey()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, keyType, err := selector.SelectKey(ctx, testDID, "ethereum")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.Equal(t, did.KeyTypeECDSA, keyType)
	assert.IsType(t, &ecdsa.PublicKey{}, pubKey)
}

func TestDefaultKeySelector_SelectKey_SolanaProtocol(t *testing.T) {
	// Test Case 2: Solana protocol should select Ed25519 key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest2")

	ed25519PubKey := createEd25519Key()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeEd25519: ed25519PubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, keyType, err := selector.SelectKey(ctx, testDID, "solana")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.Equal(t, did.KeyTypeEd25519, keyType)
	assert.IsType(t, ed25519.PublicKey{}, pubKey)
}

func TestDefaultKeySelector_SelectKey_UnknownProtocol_FallbackToFirst(t *testing.T) {
	// Test Case 3: Unknown protocol should fallback to first available key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest3")

	ecdsaPubKey := createECDSAKey()
	ecdsaKeyData, _ := did.MarshalPublicKey(ecdsaPubKey)

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   ecdsaKeyData,
					Signature: []byte("sig"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, keyType, err := selector.SelectKey(ctx, testDID, "unknown-protocol")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.Equal(t, did.KeyTypeECDSA, keyType)
}

func TestDefaultKeySelector_SelectKey_NoKeysFound(t *testing.T) {
	// Test Case 4: No keys should return error

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest4")

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {}, // Empty keys
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, _, err := selector.SelectKey(ctx, testDID, "ethereum")

	// Assert
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "no verified keys")
}

func TestDefaultKeySelector_SelectKey_PreferredKeyNotAvailable_Fallback(t *testing.T) {
	// Test Case 5: When preferred key type not available, fallback to first

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest5")

	// Agent only has Ed25519, but we request Ethereum (which prefers ECDSA)
	ed25519PubKey := createEd25519Key()
	ed25519KeyData, _ := did.MarshalPublicKey(ed25519PubKey)

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeEd25519,
					KeyData:   ed25519KeyData,
					Signature: []byte("sig"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeEd25519: ed25519PubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, keyType, err := selector.SelectKey(ctx, testDID, "ethereum")

	// Assert - Should fallback to Ed25519 since ECDSA not available
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.Equal(t, did.KeyTypeEd25519, keyType)
}

func TestDefaultKeySelector_SelectKey_MultipleKeys(t *testing.T) {
	// Test Case 6: Multiple keys scenario - should select correct one for protocol

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest6")

	ecdsaPubKey := createECDSAKey()
	ed25519PubKey := createEd25519Key()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA:   ecdsaPubKey,
				did.KeyTypeEd25519: ed25519PubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Test Ethereum protocol
	pubKeyETH, keyTypeETH, err := selector.SelectKey(ctx, testDID, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, did.KeyTypeECDSA, keyTypeETH)
	assert.IsType(t, &ecdsa.PublicKey{}, pubKeyETH)

	// Test Solana protocol
	pubKeySOL, keyTypeSOL, err := selector.SelectKey(ctx, testDID, "solana")
	require.NoError(t, err)
	assert.Equal(t, did.KeyTypeEd25519, keyTypeSOL)
	assert.IsType(t, ed25519.PublicKey{}, pubKeySOL)
}

func TestDefaultKeySelector_SelectKey_ClientError(t *testing.T) {
	// Test Case 7: Handle client errors gracefully

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest7")

	client := &mockEthereumClient{
		err: errors.New("blockchain connection failed"),
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, _, err := selector.SelectKey(ctx, testDID, "ethereum")

	// Assert
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "blockchain connection failed")
}

func TestDefaultKeySelector_SelectKey_EmptyProtocol(t *testing.T) {
	// Test Case 8: Empty protocol should return first available key

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest8")

	ecdsaPubKey := createECDSAKey()
	ecdsaKeyData, _ := did.MarshalPublicKey(ecdsaPubKey)

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   ecdsaKeyData,
					Signature: []byte("sig"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, keyType, err := selector.SelectKey(ctx, testDID, "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.Equal(t, did.KeyTypeECDSA, keyType)
}

func TestDefaultKeySelector_SelectKey_ContextCancellation(t *testing.T) {
	// Test Case 9: Context cancellation should be respected

	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testDID := did.AgentDID("did:sage:ethereum:0xtest9")

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, _, err := selector.SelectKey(ctx, testDID, "ethereum")

	// Assert - should handle context cancellation
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "context")
}

func TestDefaultKeySelector_SelectKey_UnmarshalFailure(t *testing.T) {
	// Test Case 10: Test unmarshal fallback failure path

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest10")

	// Create invalid key data that will fail unmarshal
	invalidKeyData := []byte("invalid-key-data")

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   invalidKeyData,
					Signature: []byte("sig"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
		// No publicKeys mapping, so ResolvePublicKeyByType will fail
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, _, err := selector.SelectKey(ctx, testDID, "unknown")

	// Assert - should fail on unmarshal
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestDefaultKeySelector_SelectKey_KeyTypeUnknown(t *testing.T) {
	// Test Case 11: Test keyTypeToString with unknown key type

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest11")

	// Use an unknown key type (value 99)
	unknownKeyType := did.KeyType(99)
	invalidKeyData := []byte("invalid-key-data")

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      unknownKeyType,
					KeyData:   invalidKeyData,
					Signature: []byte("sig"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	selector := NewDefaultKeySelector(client)

	// Execute
	pubKey, _, err := selector.SelectKey(ctx, testDID, "")

	// Assert - should fail on unmarshal with unknown key type
	require.Error(t, err)
	assert.Nil(t, pubKey)
}

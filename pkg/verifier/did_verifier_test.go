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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSignatureVerifier is a mock for signature verification
type mockSignatureVerifier struct {
	verifyErr error
	verified  bool
}

func (m *mockSignatureVerifier) VerifyHTTPRequest(req *http.Request, pubKey interface{}) error {
	if m.verifyErr != nil {
		return m.verifyErr
	}
	m.verified = true
	return nil
}

func TestDefaultDIDVerifier_ResolvePublicKey_WithoutKeyType(t *testing.T) {
	// Test Case 1: Resolve public key without specifying key type

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
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	// Execute
	pubKey, err := verifier.ResolvePublicKey(ctx, testDID, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.IsType(t, &ecdsa.PublicKey{}, pubKey)
}

func TestDefaultDIDVerifier_ResolvePublicKey_WithKeyType(t *testing.T) {
	// Test Case 2: Resolve public key with specific key type

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
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	keyType := did.KeyTypeEd25519

	// Execute
	pubKey, err := verifier.ResolvePublicKey(ctx, testDID, &keyType)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, pubKey)
	assert.IsType(t, ed25519.PublicKey{}, pubKey)
}

func TestDefaultDIDVerifier_ResolvePublicKey_DIDNotFound(t *testing.T) {
	// Test Case 3: DID not found should return error

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xnotfound")

	client := &mockEthereumClient{
		keys: map[did.AgentDID][]did.AgentKey{},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	// Execute
	pubKey, err := verifier.ResolvePublicKey(ctx, testDID, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "no verified keys")
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_ValidSignature(t *testing.T) {
	// Test Case 4: Valid HTTP signature should verify successfully

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest4")
	ecdsaPubKey := createECDSAKey()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	mockSigVerifier := &mockSignatureVerifier{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, mockSigVerifier)

	// Create request with signature headers
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest4"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.NoError(t, err)
	assert.True(t, mockSigVerifier.verified)
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_InvalidSignature(t *testing.T) {
	// Test Case 5: Invalid signature should fail verification

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest5")
	ecdsaPubKey := createECDSAKey()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	mockSigVerifier := &mockSignatureVerifier{
		verifyErr: errors.New("invalid signature"),
	}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, mockSigVerifier)

	// Create request with signature headers
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest5"`)
	req.Header.Set("Signature", "sig1=:invalid==:")

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_MissingHeaders(t *testing.T) {
	// Test Case 6: Missing signature headers should fail

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest6")

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: createECDSAKey(),
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	// Create request without signature headers
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature")
}

func TestDefaultDIDVerifier_VerifyHTTPSignatureWithKeyID_Success(t *testing.T) {
	// Test Case 7: Extract DID from keyid and verify

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest7")
	ecdsaPubKey := createECDSAKey()

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: ecdsaPubKey,
			},
		},
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	mockSigVerifier := &mockSignatureVerifier{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, mockSigVerifier)

	// Create request with signature headers including keyid
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest7"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	extractedDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testDID, extractedDID)
	assert.True(t, mockSigVerifier.verified)
}

func TestDefaultDIDVerifier_VerifyHTTPSignatureWithKeyID_InvalidDID(t *testing.T) {
	// Test Case 8: Invalid DID in keyid should fail

	// Setup
	ctx := context.Background()

	client := &mockEthereumClient{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	// Create request with invalid DID in keyid
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="not-a-did"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	extractedDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Empty(t, extractedDID)
	assert.Contains(t, err.Error(), "DID")
}

func TestDefaultDIDVerifier_VerifyHTTPSignatureWithKeyID_MissingKeyID(t *testing.T) {
	// Test Case 9: Missing keyid parameter should fail

	// Setup
	ctx := context.Background()

	client := &mockEthereumClient{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	// Create request without keyid
	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	extractedDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Empty(t, extractedDID)
	assert.Contains(t, err.Error(), "keyid")
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_ContextCancellation(t *testing.T) {
	// Test Case 10: Context cancellation should be respected

	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testDID := did.AgentDID("did:sage:ethereum:0xtest10")

	client := &mockEthereumClient{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestDefaultDIDVerifier_ResolvePublicKey_KeyTypeNotFound(t *testing.T) {
	// Test Case 11: Requested key type not found

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest11")

	// Agent only has ECDSA, but we request Ed25519
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: createECDSAKey(),
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	keyType := did.KeyTypeEd25519

	// Execute
	pubKey, err := verifier.ResolvePublicKey(ctx, testDID, &keyType)

	// Assert
	require.Error(t, err)
	assert.Nil(t, pubKey)
	assert.Contains(t, err.Error(), "key type not found")
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_DIDMismatch(t *testing.T) {
	// Test Case 12: DID in request doesn't match provided DID

	// Setup
	ctx := context.Background()
	requestDID := did.AgentDID("did:sage:ethereum:0xtest12")
	providedDID := did.AgentDID("did:sage:ethereum:0xdifferent")

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			requestDID: {
				did.KeyTypeECDSA: createECDSAKey(),
			},
		},
		keys: map[did.AgentDID][]did.AgentKey{
			requestDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, &mockSignatureVerifier{})

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest12"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute - Using different DID than in keyid
	err := verifier.VerifyHTTPSignature(ctx, req, providedDID)

	// Assert
	require.Error(t, err)
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_NoSignatureVerifier(t *testing.T) {
	// Test Case 13: Missing signature verifier should fail

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest13")

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: createECDSAKey(),
			},
		},
		keys: map[did.AgentDID][]did.AgentKey{
			testDID: {
				{
					Type:      did.KeyTypeECDSA,
					KeyData:   []byte("dummy"),
					Verified:  true,
					CreatedAt: time.Now(),
				},
			},
		},
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil) // No signature verifier

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest13"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature verifier not configured")
}

func TestDefaultDIDVerifier_VerifyHTTPSignature_KeyResolutionError(t *testing.T) {
	// Test Case 14: Key resolution error should fail

	// Setup
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xtest14")

	client := &mockEthereumClient{
		err: errors.New("blockchain connection failed"),
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, &mockSignatureVerifier{})

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest14"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	err := verifier.VerifyHTTPSignature(ctx, req, testDID)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve public key")
}

func TestDefaultDIDVerifier_VerifyHTTPSignatureWithKeyID_MissingSignatureHeader(t *testing.T) {
	// Test Case 15: Missing Signature header (not just Signature-Input)

	// Setup
	ctx := context.Background()

	client := &mockEthereumClient{}
	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, nil)

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	// Only set Signature-Input, not Signature
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest"`)

	// Execute
	extractedDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Empty(t, extractedDID)
}

func TestDefaultDIDVerifier_VerifyHTTPSignatureWithKeyID_KeyResolutionError(t *testing.T) {
	// Test Case 16: Key resolution error in VerifyHTTPSignatureWithKeyID

	// Setup
	ctx := context.Background()

	client := &mockEthereumClient{
		err: errors.New("DID not found"),
	}

	selector := NewDefaultKeySelector(client)
	verifier := NewDefaultDIDVerifier(client, selector, &mockSignatureVerifier{})

	req := httptest.NewRequest("POST", "https://agent.example.com/task", nil)
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");created=1618884473;keyid="did:sage:ethereum:0xtest16"`)
	req.Header.Set("Signature", "sig1=:dGVzdA==:")

	// Execute
	extractedDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Empty(t, extractedDID)
	assert.Contains(t, err.Error(), "signature verification failed")
}

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

package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDIDVerifier for testing
type mockDIDVerifier struct {
	shouldSucceed bool
	extractedDID  did.AgentDID
}

func (m *mockDIDVerifier) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
	if !m.shouldSucceed {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func (m *mockDIDVerifier) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (stdcrypto.PublicKey, error) {
	return nil, nil
}

func (m *mockDIDVerifier) VerifyHTTPSignatureWithKeyID(ctx context.Context, req *http.Request) (did.AgentDID, error) {
	if !m.shouldSucceed {
		return "", fmt.Errorf("signature verification failed")
	}
	return m.extractedDID, nil
}

// mockEthereumClient for testing
type mockEthereumClient struct {
	publicKeys map[did.AgentDID]map[did.KeyType]interface{}
}

func (m *mockEthereumClient) ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error) {
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

func (m *mockEthereumClient) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
	if keyMap, found := m.publicKeys[agentDID]; found {
		if pubKey, found := keyMap[keyType]; found {
			return pubKey, nil
		}
	}
	return nil, nil
}

// Implement DIDResolver for key selection
func (m *mockEthereumClient) GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error) {
	d := did.AgentDID(didStr)
	meta := &did.AgentMetadataV4{
		DID:      d,
		IsActive: true,
		Keys:     []did.AgentKey{},
	}
	if keyMap, ok := m.publicKeys[d]; ok {
		for kt, pk := range keyMap {
			if keyData, err := did.MarshalPublicKey(pk); err == nil {
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

// Test NewDIDAuthMiddleware creates middleware
func TestNewDIDAuthMiddleware(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	testDID := did.AgentDID("did:sage:ethereum:0xtest")

	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			testDID: {
				did.KeyTypeECDSA: &privKey.PublicKey,
			},
		},
	}
	_ = client

	middleware := NewDIDAuthMiddleware(nil, nil)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.verifier)
}

// Test middleware allows valid signed requests
func TestDIDAuthMiddleware_ValidSignature(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")

	// Use mock verifier that always succeeds
	mockVerifier := &mockDIDVerifier{
		shouldSucceed: true,
		extractedDID:  testDID,
	}

	middleware := NewDIDAuthMiddlewareWithVerifier(mockVerifier)

	// Handler that should be called after verification
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// Extract DID from context
		agentDID, ok := GetAgentDIDFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, testDID, agentDID)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Create request with signature headers (mocked)
	body := []byte(`{"method": "test"}`)
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Signature", "mock-signature")
	req.Header.Set("Signature-Input", `sig1=();keyid="did:sage:ethereum:0xtest"`)

	// Execute middleware
	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	// Verify handler was called
	if !handlerCalled {
		t.Logf("Handler not called. Response: %s", rr.Body.String())
	}
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Test middleware rejects unsigned requests
func TestDIDAuthMiddleware_MissingSignature(t *testing.T) {
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{},
	}
	_ = client

	middleware := NewDIDAuthMiddleware(nil, nil)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create unsigned request
	req := httptest.NewRequest("POST", "/test", nil)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	// Verify handler was NOT called
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing signature")
}

// Test middleware rejects invalid signature
func TestDIDAuthMiddleware_InvalidSignature(t *testing.T) {

	// Use a mock verifier that fails verification to avoid resolving dependencies
	middleware := NewDIDAuthMiddlewareWithVerifier(&mockDIDVerifier{shouldSucceed: false})

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create request with invalid signature
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Signature", "invalid-signature")
	req.Header.Set("Signature-Input", `sig1=("@method" "@target-uri");keyid="did:sage:ethereum:0xtest";alg="ES256K";created=1234567890`)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	// Verify handler was NOT called
	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// Test middleware with custom error handler
func TestDIDAuthMiddleware_CustomErrorHandler(t *testing.T) {
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{},
	}
	_ = client

	customErrorCalled := false
	customErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		customErrorCalled = true
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("custom error"))
	}

	middleware := NewDIDAuthMiddleware(nil, nil)
	middleware.SetErrorHandler(customErrorHandler)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	assert.True(t, customErrorCalled)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "custom error", rr.Body.String())
}

// Test middleware with optional verification
func TestDIDAuthMiddleware_OptionalVerification(t *testing.T) {
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{},
	}
	_ = client

	middleware := NewDIDAuthMiddleware(nil, nil)
	middleware.SetOptional(true)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// DID should not be in context for unsigned requests
		_, ok := GetAgentDIDFromContext(r.Context())
		assert.False(t, ok)

		w.WriteHeader(http.StatusOK)
	})

	// Unsigned request
	req := httptest.NewRequest("GET", "/test", nil)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	// Handler should be called even without signature
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Test GetAgentDIDFromContext with missing DID
func TestGetAgentDIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	_, ok := GetAgentDIDFromContext(ctx)
	assert.False(t, ok)
}

// Test GetAgentDIDFromContext with DID
func TestGetAgentDIDFromContext_Present(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	ctx := context.WithValue(context.Background(), agentDIDKey, testDID)

	agentDID, ok := GetAgentDIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, testDID, agentDID)
}

// Test middleware with OPTIONS request (CORS preflight)
func TestDIDAuthMiddleware_OptionsRequest(t *testing.T) {
	client := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{},
	}
	_ = client

	middleware := NewDIDAuthMiddleware(nil, nil)

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// OPTIONS request (CORS preflight)
	req := httptest.NewRequest("OPTIONS", "/test", nil)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	// Handler should be called without signature verification for OPTIONS
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Test middleware preserves request body
func TestDIDAuthMiddleware_PreservesBody(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")

	// Use mock verifier that always succeeds
	mockVerifier := &mockDIDVerifier{
		shouldSucceed: true,
		extractedDID:  testDID,
	}

	middleware := NewDIDAuthMiddlewareWithVerifier(mockVerifier)

	originalBody := []byte(`{"method": "test", "data": "important"}`)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body in handler
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, originalBody, body)

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", bytes.NewReader(originalBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Signature", "mock-signature")
	req.Header.Set("Signature-Input", `sig1=();keyid="did:sage:ethereum:0xtest"`)

	rr := httptest.NewRecorder()
	middleware.Wrap(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

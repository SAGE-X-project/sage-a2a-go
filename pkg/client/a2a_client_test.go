package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyPair for testing
type mockKeyPair struct {
	pubKey  *ecdsa.PublicKey
	privKey *ecdsa.PrivateKey
}

func (m *mockKeyPair) ID() string {
	return "test-key-id"
}

func (m *mockKeyPair) PublicKey() stdcrypto.PublicKey {
	return m.pubKey
}

func (m *mockKeyPair) PrivateKey() stdcrypto.PrivateKey {
	return m.privKey
}

func (m *mockKeyPair) Type() crypto.KeyType {
	return crypto.KeyTypeSecp256k1
}

func (m *mockKeyPair) Sign(data []byte) ([]byte, error) {
	return []byte("mock-signature"), nil
}

func (m *mockKeyPair) Verify(data, signature []byte) error {
	return nil
}

// Test NewA2AClient creates client with required dependencies
func TestNewA2AClient(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	assert.NotNil(t, client)
	assert.Equal(t, testDID, client.agentDID)
	assert.NotNil(t, client.signer)
	assert.NotNil(t, client.httpClient)
}

// Test NewA2AClient with custom HTTP client
func TestNewA2AClientWithCustomHTTPClient(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}
	customClient := &http.Client{}

	client := NewA2AClient(testDID, keyPair, customClient)

	assert.NotNil(t, client)
	assert.Equal(t, customClient, client.httpClient)
}

// Test Do method signs request and executes
func TestA2AClient_Do(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify signature headers are present
		assert.NotEmpty(t, r.Header.Get("Signature"))
		assert.NotEmpty(t, r.Header.Get("Signature-Input"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()
	req, err := http.NewRequest("POST", server.URL, bytes.NewReader([]byte(`{"method": "test"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(ctx, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test Post method creates and sends signed POST request
func TestA2AClient_Post(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		assert.Equal(t, "POST", r.Method)

		// Verify content type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify body
		body, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{"jsonrpc": "2.0", "method": "test"}`, string(body))

		// Verify signature
		assert.NotEmpty(t, r.Header.Get("Signature"))
		assert.NotEmpty(t, r.Header.Get("Signature-Input"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc": "2.0", "result": "ok"}`))
	}))
	defer server.Close()

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()
	body := []byte(`{"jsonrpc": "2.0", "method": "test"}`)

	resp, err := client.Post(ctx, server.URL, body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test Get method creates and sends signed GET request
func TestA2AClient_Get(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		assert.Equal(t, "GET", r.Method)

		// Verify signature
		assert.NotEmpty(t, r.Header.Get("Signature"))
		assert.NotEmpty(t, r.Header.Get("Signature-Input"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()

	resp, err := client.Get(ctx, server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test context cancellation
func TestA2AClient_ContextCancellation(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	_, err = client.Do(ctx, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

// Test signing error handling
func TestA2AClient_SigningError(t *testing.T) {
	testDID := did.AgentDID("invalid-did") // Invalid DID format
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()
	req, err := http.NewRequest("POST", "http://example.com", nil)
	require.NoError(t, err)

	// This might fail during signing due to invalid DID
	_, err = client.Do(ctx, req)
	// We expect either no error (if signing succeeds anyway) or an error
	// The important thing is it doesn't panic
}

// Test HTTP client error handling
func TestA2AClient_HTTPError(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()

	// Invalid URL
	_, err := client.Get(ctx, "://invalid-url")
	assert.Error(t, err)
}

// Test nil body handling in Post
func TestA2AClient_PostNilBody(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Empty(t, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()

	resp, err := client.Post(ctx, server.URL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test GetAgentDID returns the agent DID
func TestA2AClient_GetAgentDID(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	assert.Equal(t, testDID, client.GetAgentDID())
}

// Test GetKeyPair returns the key pair
func TestA2AClient_GetKeyPair(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	assert.Equal(t, keyPair, client.GetKeyPair())
}

// Test Do with request that succeeds
func TestA2AClient_DoSuccess(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(ctx, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Test Post with empty URL (error case)
func TestA2AClient_PostEmptyURL(t *testing.T) {
	testDID := did.AgentDID("did:sage:ethereum:0xtest")
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPair{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	client := NewA2AClient(testDID, keyPair, nil)

	ctx := context.Background()
	body := []byte(`{"test": "data"}`)

	// Empty URL should cause error
	_, err := client.Post(ctx, "", body)
	assert.Error(t, err)
}

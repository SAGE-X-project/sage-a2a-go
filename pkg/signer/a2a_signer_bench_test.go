package signer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"testing"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// mockKeyPairBench for benchmarking
type mockKeyPairBench struct {
	pubKey  *ecdsa.PublicKey
	privKey *ecdsa.PrivateKey
}

func (m *mockKeyPairBench) ID() string {
	return "bench-key-id"
}

func (m *mockKeyPairBench) PublicKey() stdcrypto.PublicKey {
	return m.pubKey
}

func (m *mockKeyPairBench) PrivateKey() stdcrypto.PrivateKey {
	return m.privKey
}

func (m *mockKeyPairBench) Type() crypto.KeyType {
	return crypto.KeyTypeSecp256k1
}

func (m *mockKeyPairBench) Sign(data []byte) ([]byte, error) {
	return []byte("mock-signature-bench"), nil
}

func (m *mockKeyPairBench) Verify(data, signature []byte) error {
	return nil
}

// Benchmark HTTP request signing
func BenchmarkSignRequest(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPairBench{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	signer := NewDefaultA2ASigner()
	body := []byte(`{"task":"benchmark"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "https://bench.example.com/task", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_ = signer.SignRequest(ctx, req, testDID, keyPair)
	}
}

// Benchmark signing with custom options
func BenchmarkSignRequestWithOptions(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPairBench{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	signer := NewDefaultA2ASigner()
	body := []byte(`{"task":"benchmark"}`)

	opts := &SigningOptions{
		Components: []string{"@method", "@target-uri", "content-type"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "https://bench.example.com/task", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		_ = signer.SignRequestWithOptions(ctx, req, testDID, keyPair, opts)
	}
}

// Benchmark signature base string building
func BenchmarkBuildSignatureBase(b *testing.B) {
	signer := &DefaultA2ASigner{}
	req, _ := http.NewRequest("POST", "https://bench.example.com/task", nil)
	req.Header.Set("Content-Type", "application/json")

	components := []string{"@method", "@target-uri", "content-type"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.buildSignatureBase(req, components)
	}
}

// Benchmark signature input building
func BenchmarkBuildSignatureInput(b *testing.B) {
	signer := &DefaultA2ASigner{}
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	components := []string{"@method", "@target-uri", "content-type"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.buildSignatureInput(components, testDID, "ES256K", 1234567890, 0, "")
	}
}

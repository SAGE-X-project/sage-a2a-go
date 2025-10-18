package protocol

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Benchmark Agent Card creation
func BenchmarkAgentCardBuilder(b *testing.B) {
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").
			WithDescription("Benchmark agent").
			WithCapabilities("task.execute", "messaging.send").
			WithMetadata("version", "1.0.0").
			Build()
	}
}

// Benchmark Agent Card validation
func BenchmarkAgentCardValidation(b *testing.B) {
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")
	card := NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = card.Validate()
	}
}

// Benchmark capability check
func BenchmarkAgentCardHasCapability(b *testing.B) {
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")
	card := NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").
		WithCapabilities("task.execute", "messaging.send", "data.process").
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = card.HasCapability("task.execute")
	}
}

// Benchmark expiration check
func BenchmarkAgentCardIsExpired(b *testing.B) {
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")
	card := NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").
		WithExpiresAt(time.Now().Add(365 * 24 * time.Hour)).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = card.IsExpired()
	}
}

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

// Benchmark Agent Card signing
func BenchmarkSignAgentCard(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	card := NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").
		WithDescription("Benchmark agent").
		Build()

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPairBench{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	signer := NewDefaultAgentCardSigner(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.SignAgentCard(ctx, card, keyPair)
	}
}

// Benchmark Agent Card verification (signature parsing only)
func BenchmarkVerifyAgentCardWithKey(b *testing.B) {
	ctx := context.Background()
	testDID := did.AgentDID("did:sage:ethereum:0xbenchmark")

	card := NewAgentCardBuilder(testDID, "BenchAgent", "https://bench.example.com").Build()

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyPair := &mockKeyPairBench{
		pubKey:  &privKey.PublicKey,
		privKey: privKey,
	}

	signer := NewDefaultAgentCardSigner(nil)
	signedCard, _ := signer.SignAgentCard(ctx, card, keyPair)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.VerifyAgentCardWithKey(ctx, signedCard, &privKey.PublicKey)
	}
}

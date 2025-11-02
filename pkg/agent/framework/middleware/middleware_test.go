package middleware_test

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/did"
	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/middleware"
)

func TestNewDIDAuth(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create resolver for tests
	resolver, err := did.NewResolver(did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	})
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	tests := []struct {
		name    string
		config  middleware.Config
		wantErr bool
	}{
		{
			name: "valid config with required auth",
			config: middleware.Config{
				Resolver: resolver,
				Optional: false,
			},
			wantErr: false,
		},
		{
			name: "valid config with optional auth",
			config: middleware.Config{
				Resolver: resolver,
				Optional: true,
			},
			wantErr: false,
		},
		{
			name: "nil resolver",
			config: middleware.Config{
				Resolver: nil,
				Optional: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewDIDAuth(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDIDAuth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && auth == nil {
				t.Error("NewDIDAuth() returned nil for valid config")
			}
		})
	}
}

func TestGetUnderlying(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resolver, err := did.NewResolver(did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	})
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	t.Run("get underlying middleware", func(t *testing.T) {
		auth, err := middleware.NewDIDAuth(middleware.Config{
			Resolver: resolver,
			Optional: false,
		})
		if err != nil {
			t.Fatalf("Failed to create DIDAuth: %v", err)
		}

		underlying := auth.GetUnderlying()
		if underlying == nil {
			t.Error("GetUnderlying() returned nil")
		}
	})
}

func TestComputeContentDigest(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "empty body",
			body:     []byte{},
			expected: computeExpectedDigest([]byte{}),
		},
		{
			name:     "simple text",
			body:     []byte("hello world"),
			expected: computeExpectedDigest([]byte("hello world")),
		},
		{
			name:     "JSON payload",
			body:     []byte(`{"key":"value"}`),
			expected: computeExpectedDigest([]byte(`{"key":"value"}`)),
		},
		{
			name:     "binary data",
			body:     []byte{0x00, 0x01, 0x02, 0x03},
			expected: computeExpectedDigest([]byte{0x00, 0x01, 0x02, 0x03}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest := middleware.ComputeContentDigest(tt.body)
			if digest != tt.expected {
				t.Errorf("ComputeContentDigest() = %v, want %v", digest, tt.expected)
			}

			// Verify format
			if len(digest) < 10 {
				t.Error("Digest is too short")
			}
			if digest[:9] != "sha-256=:" {
				t.Error("Digest does not start with correct prefix")
			}
			if digest[len(digest)-1] != ':' {
				t.Error("Digest does not end with ':'")
			}
		})
	}
}

func TestComputeContentDigestConsistency(t *testing.T) {
	t.Run("same input produces same output", func(t *testing.T) {
		body := []byte("test data")
		digest1 := middleware.ComputeContentDigest(body)
		digest2 := middleware.ComputeContentDigest(body)

		if digest1 != digest2 {
			t.Error("Same input produced different digests")
		}
	})

	t.Run("different inputs produce different outputs", func(t *testing.T) {
		body1 := []byte("test data 1")
		body2 := []byte("test data 2")

		digest1 := middleware.ComputeContentDigest(body1)
		digest2 := middleware.ComputeContentDigest(body2)

		if digest1 == digest2 {
			t.Error("Different inputs produced same digest")
		}
	})
}

func TestDIDAuthLifecycle(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resolver, err := did.NewResolver(did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	})
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	t.Run("create and use middleware", func(t *testing.T) {
		// Create middleware with required auth
		auth, err := middleware.NewDIDAuth(middleware.Config{
			Resolver: resolver,
			Optional: false,
		})
		if err != nil {
			t.Fatalf("Failed to create DIDAuth: %v", err)
		}

		// Verify it has underlying implementation
		if auth.GetUnderlying() == nil {
			t.Error("Middleware has no underlying implementation")
		}

		// Create middleware with optional auth
		authOptional, err := middleware.NewDIDAuth(middleware.Config{
			Resolver: resolver,
			Optional: true,
		})
		if err != nil {
			t.Fatalf("Failed to create optional DIDAuth: %v", err)
		}

		if authOptional.GetUnderlying() == nil {
			t.Error("Optional middleware has no underlying implementation")
		}
	})
}

// Helper function to compute expected digest for tests
func computeExpectedDigest(body []byte) string {
	sum := sha256.Sum256(body)
	b64 := base64.StdEncoding.EncodeToString(sum[:])
	return fmt.Sprintf("sha-256=:%s:", b64)
}

// Benchmark tests
func BenchmarkComputeContentDigest(b *testing.B) {
	body := []byte(`{"jsonrpc":"2.0","method":"test","params":{},"id":1}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = middleware.ComputeContentDigest(body)
	}
}

func BenchmarkComputeContentDigestLarge(b *testing.B) {
	// Create a larger payload (~1KB)
	body := make([]byte, 1024)
	for i := range body {
		body[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = middleware.ComputeContentDigest(body)
	}
}

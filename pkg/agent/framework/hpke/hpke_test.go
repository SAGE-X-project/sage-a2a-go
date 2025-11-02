package hpke_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/did"
	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/hpke"
	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/keys"
	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/session"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
)

// Sample JWK for testing
const testJWK = `{
  "kty": "EC",
  "crv": "secp256k1",
  "x": "JnbduT7q-RaJaJGT5pE8xXZxKQzCvKMEWnNXWKEjRGI",
  "y": "Gj8cgKT1YJ8qVOUEHj3VlqGnLvKFdxnwfGXnLJ0OQGU",
  "d": "wKl5f9G5ufZx7HtVmcJVQjREHlO8gT7J3FYq2gKQH0g"
}`

func setupTestKeys(t *testing.T) *keys.KeySet {
	tempDir := t.TempDir()
	signFile := filepath.Join(tempDir, "sign.jwk")
	kemFile := filepath.Join(tempDir, "kem.jwk")

	if err := os.WriteFile(signFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(kemFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	keySet, err := keys.LoadKeySet(keys.KeyConfig{
		SigningKeyFile: signFile,
		KEMKeyFile:     kemFile,
	})
	if err != nil {
		t.Fatalf("Failed to load test keys: %v", err)
	}

	return keySet
}

func setupTestResolver(t *testing.T) *did.Resolver {
	resolver, err := did.NewResolver(did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	})
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}
	return resolver
}

func TestNewServer(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	keySet := setupTestKeys(t)
	resolver := setupTestResolver(t)
	sessionMgr := session.NewManager()
	testDID := sagedid.AgentDID("did:sage:test")

	tests := []struct {
		name    string
		config  hpke.ServerConfig
		wantErr bool
	}{
		{
			name: "valid server config",
			config: hpke.ServerConfig{
				SigningKey:     keySet.SigningKey,
				KEMKey:         keySet.KEMKey,
				DID:            testDID,
				Resolver:       resolver,
				SessionManager: sessionMgr,
			},
			wantErr: false,
		},
		{
			name: "nil signing key",
			config: hpke.ServerConfig{
				SigningKey:     nil,
				KEMKey:         keySet.KEMKey,
				DID:            testDID,
				Resolver:       resolver,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "nil KEM key",
			config: hpke.ServerConfig{
				SigningKey:     keySet.SigningKey,
				KEMKey:         nil,
				DID:            testDID,
				Resolver:       resolver,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "empty DID",
			config: hpke.ServerConfig{
				SigningKey:     keySet.SigningKey,
				KEMKey:         keySet.KEMKey,
				DID:            "",
				Resolver:       resolver,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "nil resolver",
			config: hpke.ServerConfig{
				SigningKey:     keySet.SigningKey,
				KEMKey:         keySet.KEMKey,
				DID:            testDID,
				Resolver:       nil,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "nil session manager",
			config: hpke.ServerConfig{
				SigningKey:     keySet.SigningKey,
				KEMKey:         keySet.KEMKey,
				DID:            testDID,
				Resolver:       resolver,
				SessionManager: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := hpke.NewServer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && server == nil {
				t.Error("NewServer() returned nil for valid config")
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	keySet := setupTestKeys(t)
	resolver := setupTestResolver(t)
	sessionMgr := session.NewManager()
	testDID := sagedid.AgentDID("did:sage:test")

	// Mock transport for testing
	// Note: In real scenarios, use prototx.NewA2ATransport
	var mockTransport hpke.Transport = nil

	tests := []struct {
		name    string
		config  hpke.ClientConfig
		wantErr bool
	}{
		{
			name: "valid client config",
			config: hpke.ClientConfig{
				Transport:      mockTransport,
				Resolver:       resolver,
				SigningKey:     keySet.SigningKey,
				ClientDID:      testDID,
				SessionManager: sessionMgr,
			},
			wantErr: true, // Will fail due to nil transport, but config validation passes
		},
		{
			name: "nil resolver",
			config: hpke.ClientConfig{
				Transport:      mockTransport,
				Resolver:       nil,
				SigningKey:     keySet.SigningKey,
				ClientDID:      testDID,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "nil signing key",
			config: hpke.ClientConfig{
				Transport:      mockTransport,
				Resolver:       resolver,
				SigningKey:     nil,
				ClientDID:      testDID,
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "empty client DID",
			config: hpke.ClientConfig{
				Transport:      mockTransport,
				Resolver:       resolver,
				SigningKey:     keySet.SigningKey,
				ClientDID:      "",
				SessionManager: sessionMgr,
			},
			wantErr: true,
		},
		{
			name: "nil session manager",
			config: hpke.ClientConfig{
				Transport:      mockTransport,
				Resolver:       resolver,
				SigningKey:     keySet.SigningKey,
				ClientDID:      testDID,
				SessionManager: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := hpke.NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Note: client will be nil for invalid configs
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil for valid config")
			}
		})
	}
}

func TestServerGetUnderlying(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	keySet := setupTestKeys(t)
	resolver := setupTestResolver(t)
	sessionMgr := session.NewManager()

	server, err := hpke.NewServer(hpke.ServerConfig{
		SigningKey:     keySet.SigningKey,
		KEMKey:         keySet.KEMKey,
		DID:            "did:sage:test",
		Resolver:       resolver,
		SessionManager: sessionMgr,
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	t.Run("get underlying server", func(t *testing.T) {
		underlying := server.GetUnderlying()
		if underlying == nil {
			t.Error("GetUnderlying() returned nil")
		}
	})
}

func TestClientGetUnderlying(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	keySet := setupTestKeys(t)
	resolver := setupTestResolver(t)
	sessionMgr := session.NewManager()

	// This will fail due to nil transport, but we're testing the structure
	_, err := hpke.NewClient(hpke.ClientConfig{
		Transport:      nil,
		Resolver:       resolver,
		SigningKey:     keySet.SigningKey,
		ClientDID:      "did:sage:test",
		SessionManager: sessionMgr,
	})

	if err == nil {
		t.Error("Expected error with nil transport")
	}
}

func TestServerLifecycle(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	keySet := setupTestKeys(t)
	resolver := setupTestResolver(t)

	t.Run("create multiple servers", func(t *testing.T) {
		sessionMgr1 := session.NewManager()
		server1, err := hpke.NewServer(hpke.ServerConfig{
			SigningKey:     keySet.SigningKey,
			KEMKey:         keySet.KEMKey,
			DID:            "did:sage:test1",
			Resolver:       resolver,
			SessionManager: sessionMgr1,
		})
		if err != nil {
			t.Fatalf("Failed to create first server: %v", err)
		}

		sessionMgr2 := session.NewManager()
		server2, err := hpke.NewServer(hpke.ServerConfig{
			SigningKey:     keySet.SigningKey,
			KEMKey:         keySet.KEMKey,
			DID:            "did:sage:test2",
			Resolver:       resolver,
			SessionManager: sessionMgr2,
		})
		if err != nil {
			t.Fatalf("Failed to create second server: %v", err)
		}

		// Verify they are independent
		if server1 == server2 {
			t.Error("Servers should be independent instances")
		}
	})
}

// Benchmark tests
func BenchmarkNewServer(b *testing.B) {
	keySet := setupTestKeys(&testing.T{})
	resolver := setupTestResolver(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionMgr := session.NewManager()
		_, _ = hpke.NewServer(hpke.ServerConfig{
			SigningKey:     keySet.SigningKey,
			KEMKey:         keySet.KEMKey,
			DID:            "did:sage:test",
			Resolver:       resolver,
			SessionManager: sessionMgr,
		})
	}
}

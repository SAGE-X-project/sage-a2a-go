package did_test

import (
	"os"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/did"
)

func TestNewResolver(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		config  did.Config
		wantErr bool
	}{
		{
			name: "valid config with defaults",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
				PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			},
			wantErr: false,
		},
		{
			name: "empty RPC endpoint",
			config: did.Config{
				RPCEndpoint:     "",
				ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
				PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			},
			wantErr: true,
		},
		{
			name: "empty contract address",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "",
				PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			},
			wantErr: true,
		},
		{
			name: "empty private key",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
				PrivateKey:      "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := did.NewResolver(tt.config)
			// Skip if Ethereum node is not available (expected error for valid config)
			if err != nil && !tt.wantErr && (err.Error() == "create registry client: failed to create AgentCard client: failed to get network ID: Internal error" ||
				err.Error() == "dial tcp 127.0.0.1:8545: connect: connection refused") {
				t.Skipf("Ethereum node not available: %v", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resolver == nil {
				t.Error("NewResolver() returned nil resolver for valid config")
			}
		})
	}
}

func TestNewResolverFromEnv(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		envSetup func()
		cleanup  func()
		wantErr  bool
	}{
		{
			name: "with default values",
			envSetup: func() {
				// Use defaults
			},
			cleanup: func() {},
			wantErr: false,
		},
		{
			name: "with custom RPC endpoint",
			envSetup: func() {
				os.Setenv("ETH_RPC_URL", "http://localhost:8545")
			},
			cleanup: func() {
				os.Unsetenv("ETH_RPC_URL")
			},
			wantErr: false,
		},
		{
			name: "with custom registry address",
			envSetup: func() {
				os.Setenv("SAGE_REGISTRY_ADDRESS", "0x1234567890123456789012345678901234567890")
			},
			cleanup: func() {
				os.Unsetenv("SAGE_REGISTRY_ADDRESS")
			},
			wantErr: false,
		},
		{
			name: "with custom private key",
			envSetup: func() {
				os.Setenv("SAGE_EXTERNAL_KEY", "0x0000000000000000000000000000000000000000000000000000000000000001")
			},
			cleanup: func() {
				os.Unsetenv("SAGE_EXTERNAL_KEY")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			defer tt.cleanup()

			resolver, err := did.NewResolverFromEnv()
			// Skip if Ethereum node is not available (expected error for valid config)
			if err != nil && !tt.wantErr && (err.Error() == "create registry client: failed to create AgentCard client: failed to get network ID: Internal error" ||
				err.Error() == "dial tcp 127.0.0.1:8545: connect: connection refused") {
				t.Skipf("Ethereum node not available: %v", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResolverFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resolver == nil {
				t.Error("NewResolverFromEnv() returned nil resolver")
			}
		})
	}
}

func TestResolverGetMethods(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resolver, err := did.NewResolver(did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	})
	if err != nil {
		t.Skipf("Skipping test - cannot create resolver: %v", err)
	}

	t.Run("GetDIDClient", func(t *testing.T) {
		client := resolver.GetDIDClient()
		if client == nil {
			t.Error("GetDIDClient() returned nil")
		}
	})

	t.Run("GetKeyClient", func(t *testing.T) {
		client := resolver.GetKeyClient()
		if client == nil {
			t.Error("GetKeyClient() returned nil")
		}
	})

	t.Run("GetRegistryClient", func(t *testing.T) {
		client := resolver.GetRegistryClient()
		if client == nil {
			t.Error("GetRegistryClient() returned nil")
		}
	})
}

func TestResolverConfigValidation(t *testing.T) {
	// Skip if no Ethereum node available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		config  did.Config
		wantErr bool
	}{
		{
			name: "all fields provided",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
				PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			},
			wantErr: false,
		},
		{
			name: "invalid contract address format",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "invalid",
				PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			},
			wantErr: false, // Address validation happens at call time, not initialization
		},
		{
			name: "invalid private key format",
			config: did.Config{
				RPCEndpoint:     "http://127.0.0.1:8545",
				ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
				PrivateKey:      "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := did.NewResolver(tt.config)
			// Skip if Ethereum node is not available (expected error for valid config)
			if err != nil && !tt.wantErr && (err.Error() == "create registry client: failed to create AgentCard client: failed to get network ID: Internal error" ||
				err.Error() == "dial tcp 127.0.0.1:8545: connect: connection refused") {
				t.Skipf("Ethereum node not available: %v", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResolver() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewResolver(b *testing.B) {
	config := did.Config{
		RPCEndpoint:     "http://127.0.0.1:8545",
		ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		PrivateKey:      "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = did.NewResolver(config)
	}
}

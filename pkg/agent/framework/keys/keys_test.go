package keys_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/keys"
)

// Sample JWK for testing (secp256k1)
const testJWK = `{
  "kty": "EC",
  "crv": "secp256k1",
  "x": "JnbduT7q-RaJaJGT5pE8xXZxKQzCvKMEWnNXWKEjRGI",
  "y": "Gj8cgKT1YJ8qVOUEHj3VlqGnLvKFdxnwfGXnLJ0OQGU",
  "d": "wKl5f9G5ufZx7HtVmcJVQjREHlO8gT7J3FYq2gKQH0g"
}`

func TestLoadFromJWKBytes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid JWK",
			data:    []byte(testJWK),
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			data:    []byte("not json"),
			wantErr: true,
		},
		{
			name:    "invalid JWK structure",
			data:    []byte(`{"invalid": "jwk"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kp, err := keys.LoadFromJWKBytes(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromJWKBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && kp == nil {
				t.Error("LoadFromJWKBytes() returned nil keypair for valid input")
			}
		})
	}
}

func TestLoadFromJWKFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create valid JWK file
	validFile := filepath.Join(tempDir, "valid.jwk")
	if err := os.WriteFile(validFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create invalid JWK file
	invalidFile := filepath.Join(tempDir, "invalid.jwk")
	if err := os.WriteFile(invalidFile, []byte("invalid"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    validFile,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tempDir, "nonexistent.jwk"),
			wantErr: true,
		},
		{
			name:    "invalid JWK file",
			path:    invalidFile,
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kp, err := keys.LoadFromJWKFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromJWKFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && kp == nil {
				t.Error("LoadFromJWKFile() returned nil keypair for valid input")
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "test.jwk")
	if err := os.WriteFile(validFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		envVar   string
		envValue string
		wantErr  bool
	}{
		{
			name:     "valid environment variable",
			envVar:   "TEST_JWK_FILE",
			envValue: validFile,
			wantErr:  false,
		},
		{
			name:     "unset environment variable",
			envVar:   "UNSET_JWK_FILE",
			envValue: "",
			wantErr:  true,
		},
		{
			name:     "invalid file path in env",
			envVar:   "INVALID_JWK_FILE",
			envValue: "/nonexistent/file.jwk",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			kp, err := keys.LoadFromEnv(tt.envVar)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && kp == nil {
				t.Error("LoadFromEnv() returned nil keypair for valid input")
			}
		})
	}
}

func TestLoadKeySet(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	signFile := filepath.Join(tempDir, "sign.jwk")
	kemFile := filepath.Join(tempDir, "kem.jwk")

	if err := os.WriteFile(signFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(kemFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		config  keys.KeyConfig
		wantErr bool
	}{
		{
			name: "valid key set",
			config: keys.KeyConfig{
				SigningKeyFile: signFile,
				KEMKeyFile:     kemFile,
			},
			wantErr: false,
		},
		{
			name: "missing signing key file",
			config: keys.KeyConfig{
				SigningKeyFile: "",
				KEMKeyFile:     kemFile,
			},
			wantErr: true,
		},
		{
			name: "missing KEM key file",
			config: keys.KeyConfig{
				SigningKeyFile: signFile,
				KEMKeyFile:     "",
			},
			wantErr: true,
		},
		{
			name: "non-existent signing key",
			config: keys.KeyConfig{
				SigningKeyFile: "/nonexistent/sign.jwk",
				KEMKeyFile:     kemFile,
			},
			wantErr: true,
		},
		{
			name: "non-existent KEM key",
			config: keys.KeyConfig{
				SigningKeyFile: signFile,
				KEMKeyFile:     "/nonexistent/kem.jwk",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keySet, err := keys.LoadKeySet(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadKeySet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if keySet == nil {
					t.Error("LoadKeySet() returned nil keySet for valid input")
				}
				if keySet.SigningKey == nil {
					t.Error("LoadKeySet() returned nil SigningKey")
				}
				if keySet.KEMKey == nil {
					t.Error("LoadKeySet() returned nil KEMKey")
				}
			}
		})
	}
}

func TestLoadKeySetFromEnv(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	signFile := filepath.Join(tempDir, "sign.jwk")
	kemFile := filepath.Join(tempDir, "kem.jwk")

	if err := os.WriteFile(signFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(kemFile, []byte(testJWK), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		signEnv    string
		signValue  string
		kemEnv     string
		kemValue   string
		wantErr    bool
	}{
		{
			name:      "valid environment variables",
			signEnv:   "TEST_SIGN_KEY",
			signValue: signFile,
			kemEnv:    "TEST_KEM_KEY",
			kemValue:  kemFile,
			wantErr:   false,
		},
		{
			name:      "unset signing key env",
			signEnv:   "UNSET_SIGN_KEY",
			signValue: "",
			kemEnv:    "TEST_KEM_KEY",
			kemValue:  kemFile,
			wantErr:   true,
		},
		{
			name:      "unset KEM key env",
			signEnv:   "TEST_SIGN_KEY",
			signValue: signFile,
			kemEnv:    "UNSET_KEM_KEY",
			kemValue:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.signValue != "" {
				os.Setenv(tt.signEnv, tt.signValue)
				defer os.Unsetenv(tt.signEnv)
			}
			if tt.kemValue != "" {
				os.Setenv(tt.kemEnv, tt.kemValue)
				defer os.Unsetenv(tt.kemEnv)
			}

			keySet, err := keys.LoadKeySetFromEnv(tt.signEnv, tt.kemEnv)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadKeySetFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if keySet == nil {
					t.Error("LoadKeySetFromEnv() returned nil keySet")
				}
				if keySet.SigningKey == nil {
					t.Error("LoadKeySetFromEnv() returned nil SigningKey")
				}
				if keySet.KEMKey == nil {
					t.Error("LoadKeySetFromEnv() returned nil KEMKey")
				}
			}
		})
	}
}

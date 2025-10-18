package verifier

import (
	"context"
	"crypto"
	"fmt"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// EthereumClient defines the interface for interacting with SAGE DID registry
type EthereumClient interface {
	// ResolveAllPublicKeys resolves all verified public keys for an agent
	ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error)

	// ResolvePublicKeyByType resolves a public key of specific type
	ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error)
}

// DefaultKeySelector implements KeySelector with protocol-based selection logic
type DefaultKeySelector struct {
	client EthereumClient
}

// NewDefaultKeySelector creates a new DefaultKeySelector
func NewDefaultKeySelector(client EthereumClient) *DefaultKeySelector {
	return &DefaultKeySelector{
		client: client,
	}
}

// SelectKey selects the appropriate key for the given protocol
func (s *DefaultKeySelector) SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error) {
	// Check context first
	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("context error: %w", err)
	}

	// Determine preferred key type based on protocol
	var preferredKeyType did.KeyType
	var hasPreference bool

	switch protocol {
	case "ethereum":
		preferredKeyType = did.KeyTypeECDSA
		hasPreference = true
	case "solana":
		preferredKeyType = did.KeyTypeEd25519
		hasPreference = true
	default:
		// No preference, use first verified key
		hasPreference = false
	}

	// If we have a preference, try to get that key type first
	if hasPreference {
		pubKey, err := s.client.ResolvePublicKeyByType(ctx, agentDID, preferredKeyType)
		if err == nil {
			// Success! Return the preferred key
			return pubKey.(crypto.PublicKey), preferredKeyType, nil
		}
		// If preferred key not found, fallback to first available key
	}

	// Fallback: Get first available verified key
	return s.selectFirstKey(ctx, agentDID)
}

// selectFirstKey selects the first available verified key
func (s *DefaultKeySelector) selectFirstKey(ctx context.Context, agentDID did.AgentDID) (crypto.PublicKey, did.KeyType, error) {
	keys, err := s.client.ResolveAllPublicKeys(ctx, agentDID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve keys: %w", err)
	}

	if len(keys) == 0 {
		return nil, 0, fmt.Errorf("no verified keys found for DID: %s", agentDID)
	}

	// Get first key
	firstKey := keys[0]

	// Try to resolve by type first (more efficient, especially in production)
	pubKey, err := s.client.ResolvePublicKeyByType(ctx, agentDID, firstKey.Type)
	if err == nil {
		return pubKey.(crypto.PublicKey), firstKey.Type, nil
	}

	// Fallback to unmarshal if ResolvePublicKeyByType fails
	keyTypeStr := keyTypeToString(firstKey.Type)
	pubKey, err = did.UnmarshalPublicKey(firstKey.KeyData, keyTypeStr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	return pubKey.(crypto.PublicKey), firstKey.Type, nil
}

// keyTypeToString converts KeyType to string for UnmarshalPublicKey
func keyTypeToString(keyType did.KeyType) string {
	switch keyType {
	case did.KeyTypeECDSA:
		return "secp256k1"
	case did.KeyTypeEd25519:
		return "ed25519"
	default:
		return "unknown"
	}
}

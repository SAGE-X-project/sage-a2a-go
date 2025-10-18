package verifier

import (
	"context"
	"crypto"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// KeySelector selects the appropriate cryptographic key for an agent
// based on the protocol or explicit preference
type KeySelector interface {
	// SelectKey selects a key for the given agent DID and protocol
	// protocol: "ethereum", "solana", or empty string for default selection
	// Returns: public key, key type, error
	SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error)
}

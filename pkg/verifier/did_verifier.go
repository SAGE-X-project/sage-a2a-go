package verifier

import (
	"context"
	"crypto"
	"net/http"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DIDVerifier verifies HTTP signatures using SAGE DIDs
type DIDVerifier interface {
	// VerifyHTTPSignature verifies the HTTP signature in the request
	// using the public key resolved from the agent DID
	VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error

	// ResolvePublicKey resolves a public key for the given DID
	// keyType: optional preferred key type (nil for automatic selection)
	ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error)

	// VerifyHTTPSignatureWithKeyID verifies HTTP signature and extracts DID from keyid parameter
	// Returns the verified agent DID
	VerifyHTTPSignatureWithKeyID(ctx context.Context, req *http.Request) (did.AgentDID, error)
}

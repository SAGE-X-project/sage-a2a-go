package signer

import (
	"context"
	"net/http"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// A2ASigner signs HTTP messages for A2A protocol with DID identity
type A2ASigner interface {
	// SignRequest signs an HTTP request with the agent's key
	// The DID is included in the signature as the keyid parameter
	SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error

	// SignRequestWithOptions signs an HTTP request with custom options
	SignRequestWithOptions(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair, opts *SigningOptions) error
}

// SigningOptions contains options for signing HTTP requests
type SigningOptions struct {
	// Components are the signature components to include (e.g., "@method", "@target-uri")
	Components []string

	// Created is the timestamp when the signature was created (Unix timestamp)
	// If 0, current time is used
	Created int64

	// Expires is the timestamp when the signature expires (Unix timestamp)
	// If 0, no expiration
	Expires int64

	// Nonce is an optional nonce value for preventing replay attacks
	Nonce string

	// Algorithm override (if empty, determined from key type)
	Algorithm string
}

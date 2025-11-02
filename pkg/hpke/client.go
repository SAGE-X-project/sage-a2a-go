// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
//
// sage-a2a-go is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sage-a2a-go is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with sage-a2a-go.  If not, see <https://www.gnu.org/licenses/>.

package hpke

import (
	"context"
	"fmt"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/sage-x-project/sage/pkg/agent/did"
	sagehpke "github.com/sage-x-project/sage/pkg/agent/hpke"
	sagesession "github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// Client provides a simplified wrapper around SAGE's HPKE client
// for establishing end-to-end encrypted sessions with peer agents.
//
// Usage:
//
//	client := hpke.NewClient(myDID, myKeyPair, transport, resolver, sessionMgr)
//	kid, err := client.InitializeSession(ctx, "context-123", peerDID)
//	if err != nil {
//	    // handle error
//	}
//	// Use kid to encrypt/decrypt messages via session manager
type Client struct {
	sageClient *sagehpke.Client
	clientDID  identity.AgentDID
	keyPair    crypto.KeyPair
	transport  transport.MessageTransport
	resolver   did.Resolver
	sessMgr    *sagesession.Manager
}

// ClientOptions configures HPKE client behavior
type ClientOptions struct {
	// InfoBuilder customizes HPKE info/export context generation
	// If nil, uses SAGE's DefaultInfoBuilder
	InfoBuilder sagehpke.InfoBuilder

	// CookieSource provides optional cookie attachment for requests
	// If nil, no cookies are attached
	CookieSource sagehpke.CookieSource
}

// NewClient creates a new HPKE client for establishing encrypted sessions.
//
// Parameters:
//   - clientDID: This agent's DID (did:sage:ethereum:0x...)
//   - keyPair: This agent's Ed25519 signing key pair
//   - transport: Message transport for sending HPKE init requests
//   - resolver: DID resolver for looking up peer KEM keys
//   - sessionMgr: Session manager for storing derived session keys
//   - opts: Optional configuration (can be nil for defaults)
func NewClient(
	clientDID identity.AgentDID,
	keyPair crypto.KeyPair,
	transport transport.MessageTransport,
	resolver did.Resolver,
	sessionMgr *sagesession.Manager,
	opts *ClientOptions,
) (*Client, error) {
	if clientDID == "" {
		return nil, fmt.Errorf("clientDID is required")
	}
	if keyPair == nil {
		return nil, fmt.Errorf("keyPair is required")
	}
	if transport == nil {
		return nil, fmt.Errorf("transport is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("resolver is required")
	}
	if sessionMgr == nil {
		return nil, fmt.Errorf("sessionMgr is required")
	}

	// Set defaults
	if opts == nil {
		opts = &ClientOptions{}
	}

	// Create SAGE HPKE client
	sageClient := sagehpke.NewClient(
		transport,
		resolver,
		keyPair,
		string(clientDID),
		opts.InfoBuilder, // nil is handled by SAGE
		sessionMgr,
	)

	// Attach cookie source if provided
	if opts.CookieSource != nil {
		sageClient = sageClient.WithCookieSource(opts.CookieSource)
	}

	return &Client{
		sageClient: sageClient,
		clientDID:  clientDID,
		keyPair:    keyPair,
		transport:  transport,
		resolver:   resolver,
		sessMgr:    sessionMgr,
	}, nil
}

// InitializeSession establishes an end-to-end encrypted session with a peer agent.
//
// This performs the HPKE initialization protocol:
//  1. Resolves peer's X25519 KEM public key from DID registry
//  2. Performs HPKE Base sender-side key derivation (RFC 9180)
//  3. Generates ephemeral X25519 key for additional DH exchange
//  4. Sends signed HPKE-init message to peer
//  5. Verifies peer's signature and ackTag
//  6. Derives and stores session key in session manager
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - contextID: Application-specific context identifier (e.g., conversation ID)
//   - peerDID: Target peer's DID (did:sage:ethereum:0x...)
//
// Returns:
//   - kid: Session key identifier for use with session manager
//   - error: Any errors during initialization
//
// Security:
//   - Provides forward secrecy via ephemeral keys
//   - Mutual authentication via DID signatures
//   - Anti-replay protection via nonce and ackTag
//   - TOFU (Trust On First Use) public key pinning
func (c *Client) InitializeSession(ctx context.Context, contextID string, peerDID identity.AgentDID) (kid string, err error) {
	// Convert identity.AgentDID to did.AgentDID for SAGE
	sagePeerDID := did.AgentDID(peerDID)
	sageClientDID := did.AgentDID(c.clientDID)

	// Call SAGE HPKE client
	kid, err = c.sageClient.Initialize(ctx, contextID, string(sageClientDID), string(sagePeerDID))
	if err != nil {
		return "", fmt.Errorf("HPKE initialization failed: %w", err)
	}

	return kid, nil
}

// Note: For encryption/decryption operations, use the session manager directly:
//   sessionMgr := client.GetSessionManager()
//   ciphertext, err := sessionMgr.Encrypt(ctx, kid, plaintext)
//   plaintext, err := sessionMgr.Decrypt(ctx, kid, ciphertext)

// GetSessionManager returns the underlying SAGE session manager for advanced usage
func (c *Client) GetSessionManager() *sagesession.Manager {
	return c.sessMgr
}

// GetSAGEClient returns the underlying SAGE HPKE client for advanced usage
// This allows interoperability with existing SAGE code
func (c *Client) GetSAGEClient() *sagehpke.Client {
	return c.sageClient
}

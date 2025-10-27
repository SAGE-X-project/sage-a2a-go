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

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

// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
// Licensed under the LGPL v3 or later: https://www.gnu.org/licenses/
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
	"bytes"
	"context"
	gocrypto "crypto"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DefaultA2ASigner implements RFC9421-style HTTP Message Signatures.
type DefaultA2ASigner struct{}

// NewDefaultA2ASigner creates a new signer.
func NewDefaultA2ASigner() *DefaultA2ASigner { return &DefaultA2ASigner{} }

// SignRequest signs an HTTP request with default options.
// Default components: ["@method", "@path", "@query", "content-digest"]
func (s *DefaultA2ASigner) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	opts := &SigningOptions{
		Components: []string{"@method", "@path", "@query", "content-digest"},
		Created:    0, // now
	}
	return s.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
}

// SignRequestWithOptions signs an HTTP request with custom options,
// delegating the actual signing to rfc9421.HTTPVerifier.
func (s *DefaultA2ASigner) SignRequestWithOptions(
	ctx context.Context,
	req *http.Request,
	agentDID did.AgentDID,
	keyPair sagecrypto.KeyPair,
	opts *SigningOptions,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if keyPair == nil {
		return fmt.Errorf("key pair cannot be nil")
	}
	if strings.TrimSpace(string(agentDID)) == "" {
		return fmt.Errorf("DID cannot be empty")
	}
	if opts == nil {
		opts = &SigningOptions{Components: []string{"@method", "@path", "@query", "content-digest"}}
	}

	// Create defensive copy of Components to avoid mutating caller's slice
	components := make([]string, len(opts.Components))
	copy(components, opts.Components)

	if len(components) == 0 {
		components = []string{"@method", "@path", "@query", "content-digest"}
	}

	if !includes(components, "content-digest") {
		components = append(components, "content-digest")
	}

	// Always recompute Content-Digest to prevent digest injection attacks
	if err := ensureContentDigestHeader(req); err != nil {
		return fmt.Errorf("compute content-digest: %w", err)
	}

	created := opts.Created
	if created == 0 {
		created = time.Now().Unix()
	} else {
		// Validate user-supplied timestamp
		if err := validateTimestamp(created); err != nil {
			return fmt.Errorf("invalid timestamp: %w", err)
		}
	}
	alg, err := s.getAlgorithm(keyPair.Type())
	if err != nil {
		return fmt.Errorf("%w (did: %s, url: %s)", err, agentDID, req.URL.String())
	}
	if opts.Algorithm != "" {
		// Validate user-supplied algorithm against whitelist
		if err := s.validateAlgorithm(opts.Algorithm, keyPair.Type()); err != nil {
			return fmt.Errorf("invalid algorithm: %w", err)
		}
		alg = opts.Algorithm
	}

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: quoteComponents(components), // Use copied components, not original
		KeyID:             string(agentDID),
		Algorithm:         alg,
		Created:           created,
		Expires:           opts.Expires,
		Nonce:             opts.Nonce,
	}

	// Get standard crypto.Signer
	priv := keyPair.PrivateKey()
	signer, ok := priv.(gocrypto.Signer)
	if !ok {
		return fmt.Errorf("private key does not implement crypto.Signer: %T (did: %s, url: %s)",
			priv, agentDID, req.URL.String())
	}

	// RFC 9421 sign with "sig1" label
	httpv := rfc9421.NewHTTPVerifier()
	if err := httpv.SignRequest(req, "sig1", params, signer); err != nil {
		return fmt.Errorf("rfc9421 signing failed for %s (did: %s): %w",
			req.URL.String(), agentDID, err)
	}

	return nil
}

func includes(list []string, v string) bool {
	lv := strings.ToLower(v)
	for _, e := range list {
		if strings.ToLower(e) == lv {
			return true
		}
	}
	return false
}

func quoteComponents(components []string) []string {
	out := make([]string, 0, len(components))
	for _, c := range components {
		c = strings.ToLower(strings.TrimSpace(c))

		// Skip empty strings
		if c == "" {
			continue
		}

		// If already properly quoted, keep as-is
		if len(c) > 0 && c[0] == '"' && c[len(c)-1] == '"' {
			out = append(out, c)
			continue
		}

		// Quote the component
		out = append(out, fmt.Sprintf(`"%s"`, c))
	}
	return out
}

const (
	// maxBodySize is the maximum allowed request body size (10MB)
	maxBodySize = 10 * 1024 * 1024

	// maxClockSkew is the maximum allowed clock difference (60 seconds)
	maxClockSkew = 60

	// maxValidAge is the maximum age for signatures (5 minutes)
	maxValidAge = 300
)

// allowedAlgorithms defines the whitelist of permitted algorithms per key type
var allowedAlgorithms = map[sagecrypto.KeyType][]string{
	sagecrypto.KeyTypeSecp256k1: {"es256k"},
	sagecrypto.KeyTypeEd25519:   {"ed25519"},
}

// Ensure Content-Digest over entire body (sha-256, base64, RFC9421 syntax)
// This function buffers the entire request body in memory with a size limit.
// It also removes Transfer-Encoding header to avoid conflicts with Content-Length.
func ensureContentDigestHeader(req *http.Request) error {
	// Remove Transfer-Encoding header to avoid conflict with Content-Length
	req.Header.Del("Transfer-Encoding")

	var body []byte
	if req.Body != nil {
		var err error
		// Use LimitReader to enforce maximum size
		limitedReader := io.LimitReader(req.Body, maxBodySize+1)
		body, err = io.ReadAll(limitedReader)
		if err != nil {
			return err
		}
		// Check if body exceeds limit
		if len(body) > maxBodySize {
			return fmt.Errorf("request body exceeds maximum size of %d bytes", maxBodySize)
		}
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	h := sha256.Sum256(body)
	d := base64.StdEncoding.EncodeToString(h[:])
	req.Header.Set("Content-Digest", "sha-256=:"+d+":")
	return nil
}

func (s *DefaultA2ASigner) getAlgorithm(k sagecrypto.KeyType) (string, error) {
	switch k {
	case sagecrypto.KeyTypeSecp256k1:
		return "es256k", nil
	case sagecrypto.KeyTypeEd25519:
		return "ed25519", nil
	default:
		return "", fmt.Errorf("unsupported key type: %v", k)
	}
}

// validateAlgorithm checks if the algorithm is allowed for the given key type
func (s *DefaultA2ASigner) validateAlgorithm(alg string, keyType sagecrypto.KeyType) error {
	allowed, ok := allowedAlgorithms[keyType]
	if !ok {
		return fmt.Errorf("unsupported key type: %v", keyType)
	}

	algLower := strings.ToLower(alg)
	for _, a := range allowed {
		if a == algLower {
			return nil
		}
	}

	return fmt.Errorf("algorithm %s not allowed for key type %v (allowed: %v)",
		alg, keyType, allowed)
}

// validateTimestamp checks if the timestamp is within acceptable range
func validateTimestamp(created int64) error {
	now := time.Now().Unix()

	// Check if timestamp is too far in the past
	if created < now-maxValidAge {
		return fmt.Errorf("signature timestamp too old (max age: %d seconds)", maxValidAge)
	}

	// Check if timestamp is in the future (allowing for clock skew)
	if created > now+maxClockSkew {
		return fmt.Errorf("signature timestamp in the future (max skew: %d seconds)", maxClockSkew)
	}

	return nil
}

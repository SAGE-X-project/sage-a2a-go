// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
// Licensed under the LGPL v3 or later: https://www.gnu.org/licenses/

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
	if opts == nil || len(opts.Components) == 0 {
		opts = &SigningOptions{Components: []string{"@method", "@path", "@query", "content-digest"}}
	}

	if !includes(opts.Components, "content-digest") {
		opts.Components = append(opts.Components, "content-digest")
	}
	if strings.TrimSpace(req.Header.Get("Content-Digest")) == "" {
		if err := ensureContentDigestHeader(req); err != nil {
			return fmt.Errorf("compute content-digest: %w", err)
		}
	}

	created := opts.Created
	if created == 0 {
		created = time.Now().Unix()
	}
	alg := s.getAlgorithm(keyPair.Type())
	if opts.Algorithm != "" {
		alg = opts.Algorithm
	}

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: quoteComponents(opts.Components),
		KeyID:             string(agentDID),
		Algorithm:         alg,
		Created:           created,
		Expires:           opts.Expires,
		Nonce:             opts.Nonce,
	}

	// 표준 crypto.Signer 확보
	priv := keyPair.PrivateKey()
	signer, ok := priv.(gocrypto.Signer)
	if !ok {
		return fmt.Errorf("private key does not implement crypto.Signer: %T", priv)
	}

	// RFC 9421 sign "sig1"
	httpv := rfc9421.NewHTTPVerifier()
	if err := httpv.SignRequest(req, "sig1", params, signer); err != nil {
		return fmt.Errorf("rfc9421 signing failed: %w", err)
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
		if len(c) > 0 && c[0] == '"' && c[len(c)-1] == '"' {
			out = append(out, c)
			continue
		}
		out = append(out, fmt.Sprintf(`"%s"`, c))
	}
	return out
}

// Ensure Content-Digest over entire body (sha-256, base64, RFC9421 syntax)
func ensureContentDigestHeader(req *http.Request) error {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return err
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

func (s *DefaultA2ASigner) getAlgorithm(k sagecrypto.KeyType) string {
	switch k {
	case sagecrypto.KeyTypeSecp256k1:
		return "es256k"
	case sagecrypto.KeyTypeEd25519:
		return "ed25519"
	default:
		return ""
	}
}

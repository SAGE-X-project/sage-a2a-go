// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
// Licensed under the LGPL v3 or later: https://www.gnu.org/licenses/

package signer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DefaultA2ASigner implements RFC9421-style HTTP Message Signatures.
type DefaultA2ASigner struct{}

// NewDefaultA2ASigner creates a new signer.
func NewDefaultA2ASigner() *DefaultA2ASigner { return &DefaultA2ASigner{} }

// SignRequest signs an HTTP request with default options.
// Default components: ["@method", "@path", "@query", "content-digest"]
func (s *DefaultA2ASigner) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error {
	opts := &SigningOptions{
		Components: []string{"@method", "@path", "@query", "content-digest"},
		Created:    0, // now
	}
	return s.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
}

// SignRequestWithOptions signs an HTTP request with custom options.
func (s *DefaultA2ASigner) SignRequestWithOptions(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair, opts *SigningOptions) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if keyPair == nil {
		return fmt.Errorf("key pair cannot be nil")
	}
	if agentDID == "" {
		return fmt.Errorf("DID cannot be empty")
	}
	if opts == nil || len(opts.Components) == 0 {
		opts = &SigningOptions{Components: []string{"@method", "@path", "@query", "content-digest"}}
	}

	// Ensure Content-Digest if requested
	if includes(opts.Components, "content-digest") && strings.TrimSpace(req.Header.Get("Content-Digest")) == "" {
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

	// (1) Signature-Input params (same string used in base & header)
	params := s.buildSignatureParams(opts.Components, agentDID, alg, created, opts.Expires, opts.Nonce)

	// (2) Signature Base (includes "@signature-params" as last line)
	base, err := s.buildSignatureBaseWithParams(req, opts.Components, params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}

	// (3) Sign sha-256(base)
	sum := sha256.Sum256([]byte(base))
	sigRaw, err := keyPair.Sign(sum[:])
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// (4) Normalize to r||s (64 bytes)
	sig64, err := toRaw64(sigRaw)
	if err != nil {
		return fmt.Errorf("force raw64: %w", err)
	}

	// (5) Headers
	req.Header.Set("Signature-Input", "sig1="+params)
	req.Header.Set("Signature", s.buildSignatureHeader(sig64))
	return nil
}

// ===== helpers =====

func includes(list []string, v string) bool {
	lv := strings.ToLower(v)
	for _, e := range list {
		if strings.ToLower(e) == lv {
			return true
		}
	}
	return false
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

// Serialize signature parameters (without "sig1=").
// Order kept to match verifier: (components);keyid;alg;created;expires;nonce
func (s *DefaultA2ASigner) buildSignatureParams(components []string, agentDID did.AgentDID, algorithm string, created, expires int64, nonce string) string {
	quoted := make([]string, len(components))
	for i, c := range components {
		quoted[i] = fmt.Sprintf(`"%s"`, c)
	}
	parts := []string{fmt.Sprintf("(%s)", strings.Join(quoted, " "))}
	parts = append(parts, fmt.Sprintf(`keyid="%s"`, agentDID))
	if algorithm != "" {
		parts = append(parts, fmt.Sprintf(`alg="%s"`, algorithm))
	}
	if created > 0 {
		parts = append(parts, fmt.Sprintf("created=%d", created))
	}
	if expires > 0 {
		parts = append(parts, fmt.Sprintf("expires=%d", expires))
	}
	if nonce != "" {
		parts = append(parts, fmt.Sprintf(`nonce="%s"`, nonce))
	}
	return strings.Join(parts, ";")
}

// Build base with @signature-params last.
// "@query" ALWAYS included as "?" + RawQuery (empty -> "?")
func (s *DefaultA2ASigner) buildSignatureBaseWithParams(req *http.Request, components []string, params string) (string, error) {
	var lines []string

	for _, comp := range components {
		switch strings.ToLower(comp) {
		case "@method":
			lines = append(lines, fmt.Sprintf(`"%s": %s`, "@method", strings.ToUpper(req.Method)))
		case "@path":
			p := req.URL.Path
			if p == "" {
				p = "/"
			}
			lines = append(lines, fmt.Sprintf(`"%s": %s`, "@path", p))
		case "@query":
			q := "?" + req.URL.RawQuery
			lines = append(lines, fmt.Sprintf(`"%s": %s`, "@query", q))
		case "content-digest":
			if v := req.Header.Get("Content-Digest"); v != "" {
				lines = append(lines, fmt.Sprintf(`"%s": %s`, "content-digest", v))
			}
		default:
			if v := req.Header.Get(comp); v != "" {
				lines = append(lines, fmt.Sprintf(`"%s": %s`, strings.ToLower(comp), v))
			}
		}
	}

	lines = append(lines, fmt.Sprintf(`"%s": %s`, "@signature-params", params))
	return strings.Join(lines, "\n"), nil
}

// Signature: sig1=:<base64(r||s)>:
func (s *DefaultA2ASigner) buildSignatureHeader(sigRaw64 []byte) string {
	return fmt.Sprintf("sig1=:%s:", base64.StdEncoding.EncodeToString(sigRaw64))
}

func (s *DefaultA2ASigner) getAlgorithm(k crypto.KeyType) string {
	switch k {
	case crypto.KeyTypeSecp256k1:
		return "es256k"
	case crypto.KeyTypeEd25519:
		return "ed25519"
	default:
		return ""
	}
}

// toRaw64 coercs ECDSA signatures to r||s (64 bytes).
// - 65 bytes: r(32)||s(32)||v(1) → drop v
// - 64 bytes: pass-through
// - DER: ASN.1 decode then left-pad r, s to 32 bytes
func toRaw64(sig []byte) ([]byte, error) {
	switch len(sig) {
	case 64:
		return sig, nil
	case 65:
		return sig[:64], nil
	default:
		if len(sig) >= 8 && sig[0] == 0x30 {
			var ds struct{ R, S *big.Int }
			if _, err := asn1.Unmarshal(sig, &ds); err != nil {
				return nil, fmt.Errorf("asn.1 unmarshal: %w", err)
			}
			if ds.R == nil || ds.S == nil || ds.R.Sign() <= 0 || ds.S.Sign() <= 0 {
				return nil, fmt.Errorf("invalid DER r/s")
			}
			rb, sb := ds.R.Bytes(), ds.S.Bytes()
			if len(rb) > 32 || len(sb) > 32 {
				return nil, fmt.Errorf("r/s too large for 32-byte pads")
			}
			rp := make([]byte, 32)
			sp := make([]byte, 32)
			copy(rp[32-len(rb):], rb)
			copy(sp[32-len(sb):], sb)
			return append(rp, sp...), nil
		}
		return nil, fmt.Errorf("unsupported ECDSA signature format (len=%d)", len(sig))
	}
}

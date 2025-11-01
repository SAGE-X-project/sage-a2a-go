// pkg/verifier/default_did_verifier.go
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
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// PublicKeyClient is satisfied by pkg/agent/did/ethereum.EthereumClient
// (it has ResolvePublicKey and ResolveKEMKey that return interface{}).
type PublicKeyClient interface {
	ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error)
	ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error)
}

// SignatureVerifier as before …
type SignatureVerifier interface {
	VerifyHTTPRequest(req *http.Request, pubKey interface{}) error
}

// DefaultDIDVerifier now depends on:
//   - selector (uses AgentCardClient via DIDResolver to fetch V4 metadata)
//   - client   (uses EthereumClient to fetch specific key quickly when type is forced)
type DefaultDIDVerifier struct {
	client            PublicKeyClient // *ethereum.EthereumClient
	selector          KeySelector     // NewDefaultKeySelector(AgentCardClient)
	signatureVerifier SignatureVerifier
}

func NewDefaultDIDVerifier(client PublicKeyClient, selector KeySelector, signatureVerifier SignatureVerifier) *DefaultDIDVerifier {
	return &DefaultDIDVerifier{
		client:            client,
		selector:          selector,
		signatureVerifier: signatureVerifier,
	}
}

// ResolvePublicKey picks a key either by explicit KeyType or via selector policy.
func (v *DefaultDIDVerifier) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error) {
	// If the caller requests a specific key type, try a fast path.
	if keyType != nil {
		switch *keyType {
		case did.KeyTypeX25519:
			// HPKE/KEM key from chain (X25519 32 bytes)
			pk, err := v.client.ResolveKEMKey(ctx, agentDID)
			if err != nil {
				return nil, fmt.Errorf("resolve x25519 key: %w", err)
			}
			return pk.(crypto.PublicKey), nil

		case did.KeyTypeECDSA:
			// Default signing key on Ethereum (secp256k1)
			pk, err := v.client.ResolvePublicKey(ctx, agentDID)
			if err != nil {
				return nil, fmt.Errorf("resolve ecdsa key: %w", err)
			}
			return pk.(crypto.PublicKey), nil

		case did.KeyTypeEd25519:
			// Ask selector to pick Ed25519 from V4 metadata
			pk, _, err := v.selector.SelectKey(ctx, agentDID, "solana")
			if err != nil {
				return nil, fmt.Errorf("select ed25519: %w", err)
			}
			return pk, nil
		default:
			// Fall through to generic selection
		}
	}

	// Generic policy: selector decides (Ed25519 > ECDSA > first verified)
	pk, _, err := v.selector.SelectKey(ctx, agentDID, "")
	if err != nil {
		return nil, fmt.Errorf("select key: %w", err)
	}
	return pk, nil
}

// VerifyHTTPSignature verifies the HTTP signature in the request.
func (v *DefaultDIDVerifier) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	signatureInput := req.Header.Get("Signature-Input")
	signature := req.Header.Get("Signature")
	if signatureInput == "" || signature == "" {
		return fmt.Errorf("missing signature headers")
	}

	keyID, err := extractKeyID(signatureInput)
	if err != nil {
		return fmt.Errorf("failed to extract keyid: %w", err)
	}
	if !isValidDID(keyID) {
		return fmt.Errorf("invalid DID format in keyid: %s", keyID)
	}
	if keyID != string(agentDID) {
		return fmt.Errorf("keyid mismatch: expected %s, got %s", agentDID, keyID)
	}

	pubKey, err := v.ResolvePublicKey(ctx, agentDID, nil) // defaults to ECDSA
	if err != nil {
		return fmt.Errorf("failed to resolve public key: %w", err)
	}
	if v.signatureVerifier == nil {
		return fmt.Errorf("signature verifier not configured")
	}
	if err := v.signatureVerifier.VerifyHTTPRequest(req, pubKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	log.Println(("✅ Success verify"))
	return nil
}

// VerifyHTTPSignatureWithKeyID extracts DID from keyid and verifies the signature.
func (v *DefaultDIDVerifier) VerifyHTTPSignatureWithKeyID(ctx context.Context, req *http.Request) (did.AgentDID, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context error: %w", err)
	}
	sigInput := req.Header.Get("Signature-Input")
	if sigInput == "" {
		return "", fmt.Errorf("missing Signature-Input header")
	}
	keyID, err := extractKeyID(sigInput)
	if err != nil {
		return "", fmt.Errorf("failed to extract keyid: %w", err)
	}
	if !isValidDID(keyID) {
		return "", fmt.Errorf("invalid DID format in keyid: %s", keyID)
	}
	agentDID := did.AgentDID(keyID)
	if err := v.VerifyHTTPSignature(ctx, req, agentDID); err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}
	return agentDID, nil
}

// extractKeyID parses keyid from the Signature-Input header: sig1=(...);keyid="did:sage:ethereum:0x...";...
func extractKeyID(signatureInput string) (string, error) {
	re := regexp.MustCompile(`keyid="([^"]+)"`)
	m := re.FindStringSubmatch(signatureInput)
	if len(m) < 2 {
		return "", fmt.Errorf("keyid not found in Signature-Input header")
	}
	return m[1], nil
}

// isValidDID does a basic shape check. Tighten if needed.
func isValidDID(didStr string) bool {
	return strings.HasPrefix(didStr, "did:sage:")
}

// --- convenience ctor usage (example) ---
// ethCli, _ := ethdid.NewEthereumClient(&did.RegistryConfig{ RPCEndpoint: "...", ContractAddress: "...", PrivateKey: "" })
// sel := verifier.NewDefaultKeySelector(ethCli) // if you have a selector; or pass nil
// v := verifier.NewDefaultDIDVerifier(ethCli, sel, verifier.NewRFC9421Verifier())

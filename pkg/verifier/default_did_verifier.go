package verifier

import (
	"context"
	"crypto"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// SignatureVerifier defines the interface for verifying HTTP signatures
type SignatureVerifier interface {
	// VerifyHTTPRequest verifies an HTTP request signature
	VerifyHTTPRequest(req *http.Request, pubKey interface{}) error
}

// DefaultDIDVerifier implements DIDVerifier using SAGE DID resolution
type DefaultDIDVerifier struct {
	client           EthereumClient
	selector         KeySelector
	signatureVerifier SignatureVerifier
}

// NewDefaultDIDVerifier creates a new DefaultDIDVerifier
func NewDefaultDIDVerifier(client EthereumClient, selector KeySelector, signatureVerifier SignatureVerifier) *DefaultDIDVerifier {
	return &DefaultDIDVerifier{
		client:           client,
		selector:         selector,
		signatureVerifier: signatureVerifier,
	}
}

// ResolvePublicKey resolves a public key for the given DID
func (v *DefaultDIDVerifier) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error) {
	if keyType != nil {
		// Resolve specific key type
		pubKey, err := v.client.ResolvePublicKeyByType(ctx, agentDID, *keyType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve key type %d: %w", *keyType, err)
		}
		return pubKey.(crypto.PublicKey), nil
	}

	// Use KeySelector for automatic key selection
	pubKey, _, err := v.selector.SelectKey(ctx, agentDID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to select key: %w", err)
	}

	return pubKey, nil
}

// VerifyHTTPSignature verifies the HTTP signature in the request
func (v *DefaultDIDVerifier) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
	// Check context first
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// Check if signature headers exist
	signatureInput := req.Header.Get("Signature-Input")
	signature := req.Header.Get("Signature")

	if signatureInput == "" || signature == "" {
		return fmt.Errorf("missing signature headers")
	}

	// Extract keyid from Signature-Input header and validate it matches agentDID
	keyID, err := extractKeyID(signatureInput)
	if err != nil {
		return fmt.Errorf("failed to extract keyid: %w", err)
	}

	// Validate that keyID is a valid DID
	if !isValidDID(keyID) {
		return fmt.Errorf("invalid DID format in keyid: %s", keyID)
	}

	// Check if keyID matches the provided agentDID
	if keyID != string(agentDID) {
		return fmt.Errorf("keyid mismatch: expected %s, got %s", agentDID, keyID)
	}

	// Resolve public key for the DID
	pubKey, err := v.ResolvePublicKey(ctx, agentDID, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve public key: %w", err)
	}

	// Verify signature using the signature verifier
	if v.signatureVerifier == nil {
		return fmt.Errorf("signature verifier not configured")
	}

	if err := v.signatureVerifier.VerifyHTTPRequest(req, pubKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// VerifyHTTPSignatureWithKeyID extracts DID from keyid and verifies the signature
func (v *DefaultDIDVerifier) VerifyHTTPSignatureWithKeyID(ctx context.Context, req *http.Request) (did.AgentDID, error) {
	// Check context first
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context error: %w", err)
	}

	// Check if signature headers exist
	signatureInput := req.Header.Get("Signature-Input")
	if signatureInput == "" {
		return "", fmt.Errorf("missing Signature-Input header")
	}

	// Extract keyid (DID) from Signature-Input header
	keyID, err := extractKeyID(signatureInput)
	if err != nil {
		return "", fmt.Errorf("failed to extract keyid: %w", err)
	}

	// Validate that keyID is a valid DID
	if !isValidDID(keyID) {
		return "", fmt.Errorf("invalid DID format in keyid: %s", keyID)
	}

	agentDID := did.AgentDID(keyID)

	// Verify the signature
	if err := v.VerifyHTTPSignature(ctx, req, agentDID); err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	return agentDID, nil
}

// extractKeyID extracts the keyid parameter from Signature-Input header
// Format: sig1=(...);keyid="did:sage:ethereum:0x...";...
func extractKeyID(signatureInput string) (string, error) {
	// Match keyid="..." pattern
	re := regexp.MustCompile(`keyid="([^"]+)"`)
	matches := re.FindStringSubmatch(signatureInput)

	if len(matches) < 2 {
		return "", fmt.Errorf("keyid not found in Signature-Input header")
	}

	return matches[1], nil
}

// isValidDID validates that a string is a valid DID format
// Valid formats:
// - did:sage:ethereum:0x{address}
// - did:sage:ethereum:0x{address}:{nonce}
func isValidDID(didStr string) bool {
	// Basic DID format check
	if !strings.HasPrefix(didStr, "did:sage:") {
		return false
	}

	// More strict validation could be added here
	// For now, just check the prefix
	return true
}

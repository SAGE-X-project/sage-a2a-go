package signer

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DefaultA2ASigner implements A2ASigner with RFC9421 HTTP Message Signatures
type DefaultA2ASigner struct{}

// NewDefaultA2ASigner creates a new DefaultA2ASigner
func NewDefaultA2ASigner() *DefaultA2ASigner {
	return &DefaultA2ASigner{}
}

// SignRequest signs an HTTP request with the agent's key using default options
func (s *DefaultA2ASigner) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error {
	// Use default signing options
	opts := &SigningOptions{
		Components: []string{"@method", "@target-uri"},
		Created:    0, // Will use current time
	}

	return s.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
}

// SignRequestWithOptions signs an HTTP request with custom options
func (s *DefaultA2ASigner) SignRequestWithOptions(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair, opts *SigningOptions) error {
	// Check context
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	// Validate inputs
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if keyPair == nil {
		return fmt.Errorf("key pair cannot be nil")
	}

	if agentDID == "" {
		return fmt.Errorf("DID cannot be empty")
	}

	// Use default options if nil
	if opts == nil {
		opts = &SigningOptions{
			Components: []string{"@method", "@target-uri"},
		}
	}

	// Set created timestamp if not provided
	created := opts.Created
	if created == 0 {
		created = time.Now().Unix()
	}

	// Build signature base
	signatureBase, err := s.buildSignatureBase(req, opts.Components)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}

	// Sign the signature base
	signature, err := keyPair.Sign([]byte(signatureBase))
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Determine algorithm from key type
	algorithm := s.getAlgorithm(keyPair.Type())

	// Build Signature-Input header
	signatureInput := s.buildSignatureInput(opts.Components, agentDID, algorithm, created, opts.Expires, opts.Nonce)

	// Build Signature header
	signatureHeader := s.buildSignatureHeader(signature)

	// Set headers
	req.Header.Set("Signature-Input", signatureInput)
	req.Header.Set("Signature", signatureHeader)

	return nil
}

// buildSignatureBase creates the signature base string according to RFC9421
func (s *DefaultA2ASigner) buildSignatureBase(req *http.Request, components []string) (string, error) {
	var lines []string

	for _, component := range components {
		var line string

		switch component {
		case "@method":
			line = fmt.Sprintf(`"%s": %s`, component, req.Method)
		case "@target-uri":
			targetURI := req.URL.String()
			if targetURI == "" {
				targetURI = req.RequestURI
			}
			line = fmt.Sprintf(`"%s": %s`, component, targetURI)
		case "@authority":
			line = fmt.Sprintf(`"%s": %s`, component, req.Host)
		default:
			// Header component
			headerValue := req.Header.Get(component)
			if headerValue == "" {
				// Skip missing headers
				continue
			}
			line = fmt.Sprintf(`"%s": %s`, strings.ToLower(component), headerValue)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// buildSignatureInput creates the Signature-Input header value
func (s *DefaultA2ASigner) buildSignatureInput(components []string, agentDID did.AgentDID, algorithm string, created int64, expires int64, nonce string) string {
	// Format components
	componentList := make([]string, len(components))
	for i, comp := range components {
		componentList[i] = fmt.Sprintf(`"%s"`, comp)
	}

	// Build parameter string
	params := []string{
		fmt.Sprintf("(%s)", strings.Join(componentList, " ")),
	}

	if created > 0 {
		params = append(params, fmt.Sprintf("created=%d", created))
	}

	if expires > 0 {
		params = append(params, fmt.Sprintf("expires=%d", expires))
	}

	if nonce != "" {
		params = append(params, fmt.Sprintf(`nonce="%s"`, nonce))
	}

	params = append(params, fmt.Sprintf(`keyid="%s"`, agentDID))

	if algorithm != "" {
		params = append(params, fmt.Sprintf(`alg="%s"`, algorithm))
	}

	return fmt.Sprintf("sig1=%s", strings.Join(params, ";"))
}

// buildSignatureHeader creates the Signature header value
func (s *DefaultA2ASigner) buildSignatureHeader(signature []byte) string {
	encoded := base64.StdEncoding.EncodeToString(signature)
	return fmt.Sprintf("sig1=:%s:", encoded)
}

// getAlgorithm returns the algorithm identifier for the given key type
func (s *DefaultA2ASigner) getAlgorithm(keyType crypto.KeyType) string {
	switch keyType {
	case crypto.KeyTypeSecp256k1:
		return "ecdsa-p256-sha256"
	case crypto.KeyTypeEd25519:
		return "ed25519"
	default:
		return ""
	}
}

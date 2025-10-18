package verifier

import (
	"crypto"
	"net/http"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
)

// RFC9421Verifier implements SignatureVerifier using SAGE's RFC9421 HTTP verifier
type RFC9421Verifier struct {
	verifier *rfc9421.HTTPVerifier
	options  *rfc9421.HTTPVerificationOptions
}

// NewRFC9421Verifier creates a new RFC9421Verifier with default options
func NewRFC9421Verifier() *RFC9421Verifier {
	return &RFC9421Verifier{
		verifier: rfc9421.NewHTTPVerifier(),
		options:  rfc9421.DefaultHTTPVerificationOptions(),
	}
}

// VerifyHTTPRequest verifies an HTTP request signature using RFC9421
func (v *RFC9421Verifier) VerifyHTTPRequest(req *http.Request, pubKey interface{}) error {
	// Convert interface{} to crypto.PublicKey
	cryptoPubKey, ok := pubKey.(crypto.PublicKey)
	if !ok {
		// Try to use as-is if it's already a valid public key type
		cryptoPubKey = pubKey
	}

	// Use SAGE's RFC9421 HTTP verifier
	return v.verifier.VerifyRequest(req, cryptoPubKey, v.options)
}

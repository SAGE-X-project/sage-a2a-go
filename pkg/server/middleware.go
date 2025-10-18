package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

type contextKey string

const agentDIDKey contextKey = "agent_did"

// ErrorHandler handles verification errors
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// DIDAuthMiddleware provides HTTP middleware for DID signature verification
type DIDAuthMiddleware struct {
	verifier     verifier.DIDVerifier
	errorHandler ErrorHandler
	optional     bool
}

// NewDIDAuthMiddleware creates a new DID authentication middleware
func NewDIDAuthMiddleware(client verifier.EthereumClient) *DIDAuthMiddleware {
	selector := verifier.NewDefaultKeySelector(client)
	sigVerifier := verifier.NewRFC9421Verifier()
	didVerifier := verifier.NewDefaultDIDVerifier(client, selector, sigVerifier)

	return &DIDAuthMiddleware{
		verifier:     didVerifier,
		errorHandler: defaultErrorHandler,
		optional:     false,
	}
}

// NewDIDAuthMiddlewareWithVerifier creates middleware with a custom verifier
func NewDIDAuthMiddlewareWithVerifier(didVerifier verifier.DIDVerifier) *DIDAuthMiddleware {
	return &DIDAuthMiddleware{
		verifier:     didVerifier,
		errorHandler: defaultErrorHandler,
		optional:     false,
	}
}

// SetErrorHandler sets a custom error handler
func (m *DIDAuthMiddleware) SetErrorHandler(handler ErrorHandler) {
	m.errorHandler = handler
}

// SetOptional sets whether signature verification is optional
// If true, requests without signatures are allowed to pass through
func (m *DIDAuthMiddleware) SetOptional(optional bool) {
	m.optional = optional
}

// Wrap wraps an HTTP handler with DID authentication
func (m *DIDAuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip verification for OPTIONS requests (CORS preflight)
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if signature headers are present
		signatureInput := r.Header.Get("Signature-Input")
		signature := r.Header.Get("Signature")

		if signatureInput == "" || signature == "" {
			if m.optional {
				// Allow request to proceed without DID in context
				next.ServeHTTP(w, r)
				return
			}
			m.errorHandler(w, r, fmt.Errorf("missing signature headers"))
			return
		}

		// Read body to preserve it for handler
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body.Close()
		}

		// Restore body for verification
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Extract and verify DID signature
		ctx := r.Context()
		agentDID, err := m.verifier.VerifyHTTPSignatureWithKeyID(ctx, r)
		if err != nil {
			// Restore body even on error
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			m.errorHandler(w, r, fmt.Errorf("signature verification failed: %w", err))
			return
		}

		// Restore body for handler
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Add DID to context
		ctx = context.WithValue(ctx, agentDIDKey, agentDID)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// GetAgentDIDFromContext extracts the agent DID from request context
func GetAgentDIDFromContext(ctx context.Context) (did.AgentDID, bool) {
	agentDID, ok := ctx.Value(agentDIDKey).(did.AgentDID)
	return agentDID, ok
}

// defaultErrorHandler is the default error handler
func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("Unauthorized: %s", err.Error()), http.StatusUnauthorized)
}

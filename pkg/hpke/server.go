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

package hpke

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	sagehpke "github.com/sage-x-project/sage/pkg/agent/hpke"
	sagesession "github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Server handles HPKE-encrypted messages (wraps SAGE HPKE Server)
// This provides a simplified API for sage-multi-agent users
type Server struct {
	sageServer *sagehpke.Server    // Underlying SAGE HPKE server
	serverDID  identity.AgentDID   // Server's DID
	sessMgr    *sagesession.Manager
	kemKey     crypto.KeyPair      // X25519 KEM key pair
}

// ServerOptions configures the HPKE server
type ServerOptions struct {
	// AllowedSuites specifies allowed HPKE cipher suites
	AllowedSuites []string

	// MaxSkew is the maximum allowed time skew for timestamp validation
	MaxSkew time.Duration

	// KEM is the X25519 KEM key pair (optional, will be generated if not provided)
	KEM crypto.KeyPair

	// Transport is the message transport for sending responses (optional)
	Transport transport.MessageTransport

	// InfoBuilder builds HPKE info fields (optional, uses default if not provided)
	InfoBuilder sagehpke.InfoBuilder

	// Binder issues custom key IDs (optional)
	Binder sagehpke.KeyIDBinder

	// Cookies provides anti-DoS cookie verification (optional)
	Cookies sagehpke.CookieVerifier
}

// DefaultServerOptions returns default HPKE server options
func DefaultServerOptions() *ServerOptions {
	return &ServerOptions{
		MaxSkew: 2 * time.Minute,
	}
}

// NewServer creates a new HPKE server wrapping SAGE's implementation
func NewServer(
	signingKey crypto.KeyPair,
	sessionMgr *sagesession.Manager,
	serverDID string,
	resolver did.Resolver,
	opts *ServerOptions,
) (*Server, error) {
	if signingKey == nil {
		return nil, fmt.Errorf("signing key is required")
	}
	if sessionMgr == nil {
		return nil, fmt.Errorf("session manager is required")
	}
	if serverDID == "" {
		return nil, fmt.Errorf("server DID is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("DID resolver is required")
	}

	if opts == nil {
		opts = DefaultServerOptions()
	}

	// Generate KEM key if not provided
	kemKey := opts.KEM
	if kemKey == nil {
		var err error
		kemKey, err = crypto.GenerateX25519KeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate KEM key: %w", err)
		}
	}

	// Convert to SAGE types
	sageSigningKey := sagecrypto.KeyPair(signingKey)
	sageKEMKey := sagecrypto.KeyPair(kemKey)

	// Build SAGE server options
	sageOpts := &sagehpke.ServerOpts{
		AllowedSuites: opts.AllowedSuites,
		MaxSkew:       opts.MaxSkew,
		KEM:           sageKEMKey,
		Transport:     opts.Transport,
		Info:          opts.InfoBuilder,
		Binder:        opts.Binder,
		Cookies:       opts.Cookies,
	}

	// Create underlying SAGE HPKE server
	sageServer := sagehpke.NewServer(sageSigningKey, sessionMgr, serverDID, resolver, sageOpts)

	return &Server{
		sageServer: sageServer,
		serverDID:  identity.AgentDID(serverDID),
		sessMgr:    sessionMgr,
		kemKey:     kemKey,
	}, nil
}

// HandleMessage processes an HPKE-encrypted message
// This wraps SAGE's HandleMessage and provides a simplified return signature
func (s *Server) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	return s.sageServer.HandleMessage(ctx, msg)
}

// MessagesHandler returns an HTTP handler for the /messages endpoint
// This provides HTTP transport for HPKE messages
func (s *Server) MessagesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse incoming SecureMessage
		var msg transport.SecureMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		// Handle message using SAGE server
		resp, err := s.HandleMessage(r.Context(), &msg)
		if err != nil {
			// Return error response
			w.WriteHeader(http.StatusBadRequest)
			errorResp := map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errorResp)
			return
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

// GetKEMPublicKey returns the server's KEM public key
func (s *Server) GetKEMPublicKey() interface{} {
	return s.kemKey.PublicKey()
}

// GetDID returns the server's DID
func (s *Server) GetDID() identity.AgentDID {
	return s.serverDID
}

// GetSessionManager returns the session manager
func (s *Server) GetSessionManager() *sagesession.Manager {
	return s.sessMgr
}

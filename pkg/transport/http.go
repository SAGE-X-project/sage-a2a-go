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

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/sage-x-project/sage-a2a-go/pkg/signer"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// HTTPTransport provides HTTP transport for A2A messages with DID authentication
type HTTPTransport struct {
	client    *http.Client
	signer    signer.A2ASigner
	myDID     identity.AgentDID
	myKeyPair crypto.KeyPair
}

// HTTPTransportOptions configures the HTTP transport
type HTTPTransportOptions struct {
	// Timeout is the HTTP client timeout
	Timeout time.Duration

	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries int

	// Signer is the A2A message signer (optional, uses default if not provided)
	Signer signer.A2ASigner
}

// DefaultHTTPTransportOptions returns default HTTP transport options
func DefaultHTTPTransportOptions() *HTTPTransportOptions {
	return &HTTPTransportOptions{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
}

// NewHTTPTransport creates a new HTTP transport with DID authentication
func NewHTTPTransport(myDID identity.AgentDID, myKeyPair crypto.KeyPair, opts *HTTPTransportOptions) (*HTTPTransport, error) {
	if myDID == "" {
		return nil, fmt.Errorf("myDID is required")
	}
	if myKeyPair == nil {
		return nil, fmt.Errorf("myKeyPair is required")
	}

	if opts == nil {
		opts = DefaultHTTPTransportOptions()
	}

	// Use provided signer or default
	msgSigner := opts.Signer
	if msgSigner == nil {
		msgSigner = signer.NewDefaultA2ASigner()
	}

	client := &http.Client{
		Timeout: opts.Timeout,
	}

	return &HTTPTransport{
		client:    client,
		signer:    msgSigner,
		myDID:     myDID,
		myKeyPair: myKeyPair,
	}, nil
}

// SendSecureMessage sends an HPKE-encrypted message to the target URL with DID authentication
func (t *HTTPTransport) SendSecureMessage(ctx context.Context, targetURL string, msg *transport.SecureMessage) (*transport.Response, error) {
	// Serialize message to JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign request with DID authentication (RFC 9421)
	// Convert identity.AgentDID to did.AgentDID
	sageDID := did.AgentDID(t.myDID)
	if err := t.signer.SignRequest(ctx, req, sageDID, t.myKeyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var transportResp transport.Response
	if err := json.NewDecoder(resp.Body).Decode(&transportResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &transportResp, nil
}

// SendMessage sends a regular (non-encrypted) message with DID authentication
func (t *HTTPTransport) SendMessage(ctx context.Context, targetURL string, payload interface{}) (map[string]interface{}, error) {
	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign request with DID authentication (RFC 9421)
	// Convert identity.AgentDID to did.AgentDID
	sageDID := did.AgentDID(t.myDID)
	if err := t.signer.SignRequest(ctx, req, sageDID, t.myKeyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Get sends an HTTP GET request with DID authentication
func (t *HTTPTransport) Get(ctx context.Context, targetURL string) (map[string]interface{}, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request with DID authentication (RFC 9421)
	// Convert identity.AgentDID to did.AgentDID
	sageDID := did.AgentDID(t.myDID)
	if err := t.signer.SignRequest(ctx, req, sageDID, t.myKeyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// SetTimeout updates the HTTP client timeout
func (t *HTTPTransport) SetTimeout(timeout time.Duration) {
	t.client.Timeout = timeout
}

// GetClient returns the underlying HTTP client
func (t *HTTPTransport) GetClient() *http.Client {
	return t.client
}

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

package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/sage-x-project/sage-a2a-go/pkg/signer"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// A2AClient is an HTTP client that automatically signs requests with DID authentication
type A2AClient struct {
	agentDID   did.AgentDID
	keyPair    crypto.KeyPair
	signer     signer.A2ASigner
	httpClient *http.Client
}

// NewA2AClient creates a new A2A client with automatic DID signing
// If httpClient is nil, http.DefaultClient is used
func NewA2AClient(agentDID did.AgentDID, keyPair crypto.KeyPair, httpClient *http.Client) *A2AClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &A2AClient{
		agentDID:   agentDID,
		keyPair:    keyPair,
		signer:     signer.NewDefaultA2ASigner(),
		httpClient: httpClient,
	}
}

// Do executes an HTTP request with automatic DID signature
func (c *A2AClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Check context first
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error: %w", err)
	}

	// Sign the request with DID
	if err := c.signer.SignRequest(ctx, req, c.agentDID, c.keyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	return resp, nil
}

// Post sends a POST request with JSON body and automatic DID signature
func (c *A2AClient) Post(ctx context.Context, url string, body []byte) (*http.Response, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.Do(ctx, req)
}

// Get sends a GET request with automatic DID signature
func (c *A2AClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	return c.Do(ctx, req)
}

// GetAgentDID returns the agent DID
func (c *A2AClient) GetAgentDID() did.AgentDID {
	return c.agentDID
}

// GetKeyPair returns the key pair
func (c *A2AClient) GetKeyPair() crypto.KeyPair {
	return c.keyPair
}

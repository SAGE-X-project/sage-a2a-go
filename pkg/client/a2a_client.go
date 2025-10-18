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

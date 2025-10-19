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
	"iter"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/sage-x-project/sage-a2a-go/pkg/signer"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// DIDHTTPTransport implements a2aclient.Transport for HTTP/JSON-RPC 2.0
// with automatic DID signature on all requests using RFC 9421.
//
// This transport provides:
//   - HTTP/JSON-RPC 2.0 protocol support (required by A2A spec)
//   - Automatic DID signing on all outgoing requests
//   - RFC 9421 HTTP Message Signatures
//   - Compatible with a2a-go client infrastructure
type DIDHTTPTransport struct {
	baseURL    string
	agentDID   did.AgentDID
	keyPair    crypto.KeyPair
	signer     signer.A2ASigner
	httpClient *http.Client
}

// NewDIDHTTPTransport creates a new DID-authenticated HTTP transport.
//
// Parameters:
//   - baseURL: The base URL of the A2A agent (e.g., "https://agent.example.com")
//   - agentDID: Your agent's DID for signing requests
//   - keyPair: Your agent's private key for signing
//   - httpClient: Optional HTTP client (nil to use http.DefaultClient)
func NewDIDHTTPTransport(
	baseURL string,
	agentDID did.AgentDID,
	keyPair crypto.KeyPair,
	httpClient *http.Client,
) a2aclient.Transport {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &DIDHTTPTransport{
		baseURL:    baseURL,
		agentDID:   agentDID,
		keyPair:    keyPair,
		signer:     signer.NewDefaultA2ASigner(),
		httpClient: httpClient,
	}
}

// ========================================
// JSON-RPC 2.0 Helper Methods
// ========================================

// jsonRPCRequest represents a JSON-RPC 2.0 request
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int    `json:"id"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// jsonRPCError represents a JSON-RPC 2.0 error
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// call makes a JSON-RPC 2.0 call with DID signature and returns the raw result
func (t *DIDHTTPTransport) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	// Create JSON-RPC request
	rpcReq := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1, // TODO: use atomic counter for ID
	}

	// Marshal request body
	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+"/rpc", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign request with DID
	if err := t.signer.SignRequest(ctx, req, t.agentDID, t.keyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request with DID: %w", err)
	}

	// Execute HTTP request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s: %s", resp.StatusCode, resp.Status, string(respBody))
	}

	// Parse JSON-RPC response
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// ========================================
// A2A Protocol Methods (a2aclient.Transport interface)
// ========================================

// GetTask implements the 'tasks/get' protocol method.
func (t *DIDHTTPTransport) GetTask(ctx context.Context, query *a2a.TaskQueryParams) (*a2a.Task, error) {
	result, err := t.call(ctx, "tasks/get", query)
	if err != nil {
		return nil, err
	}

	var task a2a.Task
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Task: %w", err)
	}

	return &task, nil
}

// CancelTask implements the 'tasks/cancel' protocol method.
func (t *DIDHTTPTransport) CancelTask(ctx context.Context, id *a2a.TaskIDParams) (*a2a.Task, error) {
	result, err := t.call(ctx, "tasks/cancel", id)
	if err != nil {
		return nil, err
	}

	var task a2a.Task
	if err := json.Unmarshal(result, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Task: %w", err)
	}

	return &task, nil
}

// SendMessage implements the 'message/send' protocol method (non-streaming).
func (t *DIDHTTPTransport) SendMessage(ctx context.Context, message *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	result, err := t.call(ctx, "message/send", message)
	if err != nil {
		return nil, err
	}

	// Result can be either Task or Message
	// Distinguish by checking for "id" (Task) vs "messageId" (Message) field
	var raw map[string]interface{}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// Check if it's a Message (has "messageId" field)
	if _, hasMessageID := raw["messageId"]; hasMessageID {
		var msg a2a.Message
		if err := json.Unmarshal(result, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Message: %w", err)
		}
		return &msg, nil
	}

	// Otherwise, try Task (has "id" field)
	if _, hasID := raw["id"]; hasID {
		var task a2a.Task
		if err := json.Unmarshal(result, &task); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Task: %w", err)
		}
		return &task, nil
	}

	return nil, fmt.Errorf("result is neither Task nor Message")
}

// ResubscribeToTask implements the 'tasks/resubscribe' protocol method.
// Note: HTTP transport uses Server-Sent Events (SSE) for streaming.
func (t *DIDHTTPTransport) ResubscribeToTask(ctx context.Context, id *a2a.TaskIDParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// TODO: Implement SSE streaming for HTTP transport
		// For now, return error
		yield(nil, fmt.Errorf("ResubscribeToTask: SSE streaming not implemented yet"))
	}
}

// SendStreamingMessage implements the 'message/stream' protocol method (streaming).
// Note: HTTP transport uses Server-Sent Events (SSE) for streaming.
func (t *DIDHTTPTransport) SendStreamingMessage(ctx context.Context, message *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// TODO: Implement SSE streaming for HTTP transport
		// For now, return error
		yield(nil, fmt.Errorf("SendStreamingMessage: SSE streaming not implemented yet"))
	}
}

// GetTaskPushConfig implements the 'tasks/pushNotificationConfig/get' protocol method.
func (t *DIDHTTPTransport) GetTaskPushConfig(ctx context.Context, params *a2a.GetTaskPushConfigParams) (*a2a.TaskPushConfig, error) {
	result, err := t.call(ctx, "tasks/pushNotificationConfig/get", params)
	if err != nil {
		return nil, err
	}

	var config a2a.TaskPushConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TaskPushConfig: %w", err)
	}

	return &config, nil
}

// ListTaskPushConfig implements the 'tasks/pushNotificationConfig/list' protocol method.
func (t *DIDHTTPTransport) ListTaskPushConfig(ctx context.Context, params *a2a.ListTaskPushConfigParams) ([]*a2a.TaskPushConfig, error) {
	result, err := t.call(ctx, "tasks/pushNotificationConfig/list", params)
	if err != nil {
		return nil, err
	}

	var configs []*a2a.TaskPushConfig
	if err := json.Unmarshal(result, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TaskPushConfig list: %w", err)
	}

	return configs, nil
}

// SetTaskPushConfig implements the 'tasks/pushNotificationConfig/set' protocol method.
func (t *DIDHTTPTransport) SetTaskPushConfig(ctx context.Context, config *a2a.TaskPushConfig) (*a2a.TaskPushConfig, error) {
	result, err := t.call(ctx, "tasks/pushNotificationConfig/set", config)
	if err != nil {
		return nil, err
	}

	var resultConfig a2a.TaskPushConfig
	if err := json.Unmarshal(result, &resultConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TaskPushConfig: %w", err)
	}

	return &resultConfig, nil
}

// DeleteTaskPushConfig implements the 'tasks/pushNotificationConfig/delete' protocol method.
func (t *DIDHTTPTransport) DeleteTaskPushConfig(ctx context.Context, params *a2a.DeleteTaskPushConfigParams) error {
	_, err := t.call(ctx, "tasks/pushNotificationConfig/delete", params)
	return err
}

// GetAgentCard implements agent card retrieval.
// For HTTP transport, this fetches from the well-known URL.
func (t *DIDHTTPTransport) GetAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	// Fetch agent card from well-known URL
	url := t.baseURL + "/.well-known/agent-card.json"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request with DID
	if err := t.signer.SignRequest(ctx, req, t.agentDID, t.keyPair); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &card, nil
}

// Destroy cleans up resources (HTTP client doesn't need cleanup).
func (t *DIDHTTPTransport) Destroy() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

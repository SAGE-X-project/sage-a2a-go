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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// init initializes crypto package for tests
func init() {
	// Register key generators
	crypto.SetKeyGenerators(
		func() (crypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
	)

	// Register storage constructors
	crypto.SetStorageConstructors(
		func() crypto.KeyStorage { return storage.NewMemoryKeyStorage() },
	)

	// Register format constructors
	crypto.SetFormatConstructors(
		func() crypto.KeyExporter { return formats.NewJWKExporter() },
		func() crypto.KeyExporter { return formats.NewPEMExporter() },
		func() crypto.KeyImporter { return formats.NewJWKImporter() },
		func() crypto.KeyImporter { return formats.NewPEMImporter() },
	)
}

// setupTestTransport creates a test DID HTTP transport with a mock server
func setupTestTransport(t *testing.T, handler http.HandlerFunc) (*DIDHTTPTransport, *httptest.Server) {
	server := httptest.NewServer(handler)

	// Create test DID and keypair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")

	transport := NewDIDHTTPTransport(server.URL, agentDID, keyPair, nil).(*DIDHTTPTransport)

	return transport, server
}

// mockJSONRPCResponse creates a mock JSON-RPC 2.0 response
func mockJSONRPCResponse(result interface{}) []byte {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
	}

	resultJSON, _ := json.Marshal(result)
	resp.Result = resultJSON

	respJSON, _ := json.Marshal(resp)
	return respJSON
}

// mockJSONRPCError creates a mock JSON-RPC 2.0 error response
func mockJSONRPCError(code int, message string) []byte {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}

	respJSON, _ := json.Marshal(resp)
	return respJSON
}

func TestNewDIDHTTPTransport(t *testing.T) {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")
	baseURL := "https://example.com"

	transport := NewDIDHTTPTransport(baseURL, agentDID, keyPair, nil).(*DIDHTTPTransport)

	assert.Equal(t, baseURL, transport.baseURL)
	assert.Equal(t, agentDID, transport.agentDID)
	assert.NotNil(t, transport.signer)
	assert.NotNil(t, transport.httpClient)
}

func TestDIDHTTPTransport_GetTask(t *testing.T) {
	expectedTask := &a2a.Task{
		ID: "task-123",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/rpc", r.URL.Path)

		// Verify Content-Type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify Signature headers exist
		assert.NotEmpty(t, r.Header.Get("Signature-Input"))
		assert.NotEmpty(t, r.Header.Get("Signature"))

		// Verify request body
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "2.0", req.JSONRPC)
		assert.Equal(t, "tasks/get", req.Method)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedTask))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	task, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.NoError(t, err)
	assert.Equal(t, expectedTask.ID, task.ID)
	assert.Equal(t, expectedTask.Status.State, task.Status.State)
}

func TestDIDHTTPTransport_CancelTask(t *testing.T) {
	expectedTask := &a2a.Task{
		ID: "task-123",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateCanceled,
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/cancel", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedTask))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	task, err := transport.CancelTask(ctx, &a2a.TaskIDParams{ID: "task-123"})

	require.NoError(t, err)
	assert.Equal(t, expectedTask.ID, task.ID)
	assert.Equal(t, a2a.TaskStateCanceled, task.Status.State)
}

func TestDIDHTTPTransport_SendMessage_ReturnsTask(t *testing.T) {
	expectedTask := &a2a.Task{
		ID: "task-456",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateWorking,
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "message/send", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedTask))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	message := &a2a.MessageSendParams{
		Message: &a2a.Message{
			ID:   "msg-123",
			Role: a2a.MessageRoleUser,
			Parts: []a2a.Part{
				&a2a.TextPart{Text: "Hello"},
			},
		},
	}

	result, err := transport.SendMessage(ctx, message)

	require.NoError(t, err)
	task, ok := result.(*a2a.Task)
	require.True(t, ok, "result should be a Task")
	assert.Equal(t, expectedTask.ID, task.ID)
	assert.Equal(t, expectedTask.Status.State, task.Status.State)
}

func TestDIDHTTPTransport_SendMessage_ReturnsMessage(t *testing.T) {
	// Create expected message using NewMessage to ensure proper initialization
	expectedMessage := a2a.NewMessage(
		a2a.MessageRoleAgent,
		&a2a.TextPart{Text: "Response"},
	)
	expectedMessage.ID = "msg-789"

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedMessage))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	message := &a2a.MessageSendParams{
		Message: a2a.NewMessage(
			a2a.MessageRoleUser,
			&a2a.TextPart{Text: "Hello"},
		),
	}

	result, err := transport.SendMessage(ctx, message)

	require.NoError(t, err)

	// Debug: print result type
	t.Logf("Result type: %T", result)

	// Check if it's a Message
	msg, ok := result.(*a2a.Message)
	if !ok {
		// Try Task
		task, isTask := result.(*a2a.Task)
		if isTask {
			t.Fatalf("Expected Message but got Task: %+v", task)
		}
		t.Fatalf("Expected Message but got unknown type: %T", result)
	}

	assert.Equal(t, expectedMessage.ID, msg.ID)
	assert.Equal(t, expectedMessage.Role, msg.Role)
}

func TestDIDHTTPTransport_GetTaskPushConfig(t *testing.T) {
	expectedConfig := &a2a.TaskPushConfig{
		TaskID: "task-123",
		Config: a2a.PushConfig{
			URL: "https://callback.example.com",
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/pushNotificationConfig/get", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedConfig))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	config, err := transport.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigParams{TaskID: "task-123"})

	require.NoError(t, err)
	assert.Equal(t, expectedConfig.TaskID, config.TaskID)
	assert.Equal(t, expectedConfig.Config.URL, config.Config.URL)
}

func TestDIDHTTPTransport_ListTaskPushConfig(t *testing.T) {
	expectedConfigs := []*a2a.TaskPushConfig{
		{TaskID: "task-123", Config: a2a.PushConfig{URL: "https://callback1.example.com"}},
		{TaskID: "task-456", Config: a2a.PushConfig{URL: "https://callback2.example.com"}},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/pushNotificationConfig/list", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedConfigs))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	configs, err := transport.ListTaskPushConfig(ctx, &a2a.ListTaskPushConfigParams{})

	require.NoError(t, err)
	assert.Len(t, configs, 2)
	assert.Equal(t, expectedConfigs[0].TaskID, configs[0].TaskID)
	assert.Equal(t, expectedConfigs[1].TaskID, configs[1].TaskID)
}

func TestDIDHTTPTransport_SetTaskPushConfig(t *testing.T) {
	inputConfig := &a2a.TaskPushConfig{
		TaskID: "task-789",
		Config: a2a.PushConfig{
			URL: "https://callback.example.com",
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/pushNotificationConfig/set", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(inputConfig))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	config, err := transport.SetTaskPushConfig(ctx, inputConfig)

	require.NoError(t, err)
	assert.Equal(t, inputConfig.TaskID, config.TaskID)
	assert.Equal(t, inputConfig.Config.URL, config.Config.URL)
}

func TestDIDHTTPTransport_DeleteTaskPushConfig(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/pushNotificationConfig/delete", req.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(nil))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	err := transport.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigParams{TaskID: "task-123"})

	require.NoError(t, err)
}

func TestDIDHTTPTransport_GetAgentCard(t *testing.T) {
	expectedCard := &a2a.AgentCard{
		Name:               "Test Agent",
		Description:        "A test agent",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify GET request to well-known URL
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/.well-known/agent-card.json", r.URL.Path)

		// Verify Signature headers exist
		assert.NotEmpty(t, r.Header.Get("Signature-Input"))
		assert.NotEmpty(t, r.Header.Get("Signature"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedCard)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	card, err := transport.GetAgentCard(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedCard.Name, card.Name)
	assert.Equal(t, expectedCard.Description, card.Description)
	assert.Equal(t, expectedCard.PreferredTransport, card.PreferredTransport)
}

func TestDIDHTTPTransport_JSONRPCError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCError(-32600, "Invalid Request"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON-RPC error")
	assert.Contains(t, err.Error(), "-32600")
	assert.Contains(t, err.Error(), "Invalid Request")
}

func TestDIDHTTPTransport_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error")
	assert.Contains(t, err.Error(), "500")
}

func TestDIDHTTPTransport_Destroy(t *testing.T) {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")
	transport := NewDIDHTTPTransport("https://example.com", agentDID, keyPair, nil).(*DIDHTTPTransport)

	err = transport.Destroy()
	assert.NoError(t, err)
}

func TestDIDHTTPTransport_ResubscribeToTask_NotImplemented(t *testing.T) {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")
	transport := NewDIDHTTPTransport("https://example.com", agentDID, keyPair, nil).(*DIDHTTPTransport)

	ctx := context.Background()
	seq := transport.ResubscribeToTask(ctx, &a2a.TaskIDParams{ID: "task-123"})

	// Iterate and expect error
	for event, err := range seq {
		assert.Nil(t, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE streaming not implemented yet")
		break
	}
}

func TestDIDHTTPTransport_SendStreamingMessage_NotImplemented(t *testing.T) {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")
	transport := NewDIDHTTPTransport("https://example.com", agentDID, keyPair, nil).(*DIDHTTPTransport)

	ctx := context.Background()
	message := &a2a.MessageSendParams{
		Message: &a2a.Message{
			ID:   "msg-123",
			Role: a2a.MessageRoleUser,
			Parts: []a2a.Part{
				&a2a.TextPart{Text: "Hello"},
			},
		},
	}
	seq := transport.SendStreamingMessage(ctx, message)

	// Iterate and expect error
	for event, err := range seq {
		assert.Nil(t, event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SSE streaming not implemented yet")
		break
	}
}

func ExampleNewDIDHTTPTransport() {
	// Create DID and keypair
	keyPair, _ := crypto.GenerateSecp256k1KeyPair()
	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Create transport
	transport := NewDIDHTTPTransport(
		"https://agent.example.com",
		agentDID,
		keyPair,
		nil, // use default HTTP client
	)

	fmt.Printf("Transport created for %s\n", agentDID)
	transport.Destroy()
}

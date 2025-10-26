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
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/sage-x-project/sage-a2a-go/pkg/protocol"
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
		func() (crypto.KeyPair, error) { return keys.GenerateP256KeyPair() },
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

// testCallInterceptor is a test implementation of CallInterceptor
type testCallInterceptor struct {
	called *bool
}

func (i *testCallInterceptor) Before(ctx context.Context, req *a2aclient.Request) (context.Context, error) {
	*i.called = true
	return ctx, nil
}

func (i *testCallInterceptor) After(ctx context.Context, resp *a2aclient.Response) error {
	return nil
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

// ========================================
// A2A Protocol v0.4.0 Tests
// ========================================

func TestDIDHTTPTransport_ListTasks(t *testing.T) {
	// Create expected tasks
	expectedTasks := []*a2a.Task{
		{
			ID:        "task-123",
			ContextID: "context-abc",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateWorking,
			},
		},
		{
			ID:        "task-456",
			ContextID: "context-abc",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateCompleted,
			},
		},
	}

	expectedResult := &protocol.ListTasksResult{
		Tasks:         expectedTasks,
		TotalSize:     2,
		PageSize:      10,
		NextPageToken: "",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tasks/list", req.Method)

		// Verify params structure
		if req.Params != nil {
			params, ok := req.Params.(map[string]interface{})
			require.True(t, ok)
			t.Logf("Received params: %+v", params)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedResult))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &protocol.ListTasksParams{
		ContextID:     "context-abc",
		Status:        a2a.TaskStateWorking,
		PageSize:      10,
		HistoryLength: 5,
	}

	result, err := transport.ListTasks(ctx, params)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Tasks, 2)
	assert.Equal(t, expectedTasks[0].ID, result.Tasks[0].ID)
	assert.Equal(t, expectedTasks[1].ID, result.Tasks[1].ID)
	assert.Equal(t, 2, result.TotalSize)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, "", result.NextPageToken)
}

func TestDIDHTTPTransport_ListTasks_WithPagination(t *testing.T) {
	expectedResult := &protocol.ListTasksResult{
		Tasks: []*a2a.Task{
			{
				ID:        "task-1",
				ContextID: "ctx-1",
				Status:    a2a.TaskStatus{State: a2a.TaskStateWorking},
			},
		},
		TotalSize:     100,
		PageSize:      1,
		NextPageToken: "token-abc123",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedResult))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &protocol.ListTasksParams{
		PageSize:  1,
		PageToken: "prev-token",
	}

	result, err := transport.ListTasks(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, "token-abc123", result.NextPageToken)
	assert.Equal(t, 100, result.TotalSize)
}

func TestDIDHTTPTransport_ListTasks_EmptyResult(t *testing.T) {
	expectedResult := &protocol.ListTasksResult{
		Tasks:         []*a2a.Task{},
		TotalSize:     0,
		PageSize:      50,
		NextPageToken: "",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(expectedResult))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &protocol.ListTasksParams{
		ContextID: "non-existent",
	}

	result, err := transport.ListTasks(ctx, params)

	require.NoError(t, err)
	assert.Empty(t, result.Tasks)
	assert.Equal(t, 0, result.TotalSize)
}

// ========================================
// SSE Streaming Tests
// ========================================

// mockSSEResponse creates a mock SSE stream with multiple events
func mockSSEResponse(events []string) string {
	var response string
	for _, event := range events {
		response += "data: " + event + "\n\n"
	}
	return response
}

// TestDIDHTTPTransport_SendStreamingMessage_Success tests successful SSE streaming
func TestDIDHTTPTransport_SendStreamingMessage_Success(t *testing.T) {
	// Create mock SSE events
	message1 := &a2a.Message{
		ID:    "msg-1",
		Role:  a2a.MessageRoleAgent,
		Parts: []a2a.Part{&a2a.TextPart{Text: "Hello"}},
	}

	task1 := &a2a.Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status:    a2a.TaskStatus{State: a2a.TaskStateWorking},
	}

	// Create JSON-RPC wrapped responses
	rpcResp1, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"message": message1,
		},
	})

	rpcResp2, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"task": task1,
		},
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify Accept header
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		// Write SSE events
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp1))
		w.(http.Flusher).Flush()

		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp2))
		w.(http.Flusher).Flush()
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Collect events
	var events []a2a.Event
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		require.NoError(t, err)
		events = append(events, event)
	}

	// Verify we received both events
	require.Len(t, events, 2)

	// Verify first event is Message
	msg, ok := events[0].(*a2a.Message)
	require.True(t, ok, "First event should be Message")
	assert.Equal(t, "msg-1", msg.ID)

	// Verify second event is Task
	task, ok := events[1].(*a2a.Task)
	require.True(t, ok, "Second event should be Task")
	assert.Equal(t, a2a.TaskID("task-1"), task.ID)
}

// TestDIDHTTPTransport_SendStreamingMessage_AllEventTypes tests all A2A event types
func TestDIDHTTPTransport_SendStreamingMessage_AllEventTypes(t *testing.T) {
	// Message event
	message := &a2a.Message{
		ID:    "msg-1",
		Role:  a2a.MessageRoleAgent,
		Parts: []a2a.Part{&a2a.TextPart{Text: "Response"}},
	}

	// Task event
	task := &a2a.Task{
		ID:        "task-1",
		ContextID: "ctx-1",
		Status:    a2a.TaskStatus{State: a2a.TaskStateWorking},
	}

	// TaskStatusUpdateEvent
	statusUpdate := &a2a.TaskStatusUpdateEvent{
		TaskID: "task-1",
		Status: a2a.TaskStatus{State: a2a.TaskStateCompleted},
	}

	// TaskArtifactUpdateEvent
	artifactUpdate := &a2a.TaskArtifactUpdateEvent{
		TaskID: "task-1",
		Artifact: &a2a.Artifact{
			ID:   "artifact-1",
			Name: "output.txt",
		},
	}

	// Create JSON-RPC responses for each event type
	events := []map[string]interface{}{
		{"message": message},
		{"task": task},
		{"statusUpdate": statusUpdate},
		{"artifactUpdate": artifactUpdate},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, eventData := range events {
			rpcResp, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  eventData,
			})
			fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
			w.(http.Flusher).Flush()
		}
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Collect all events
	var receivedEvents []a2a.Event
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		require.NoError(t, err)
		receivedEvents = append(receivedEvents, event)
	}

	// Verify we received all 4 event types
	require.Len(t, receivedEvents, 4)

	// Verify each event type
	_, ok := receivedEvents[0].(*a2a.Message)
	assert.True(t, ok, "Event 0 should be Message")

	_, ok = receivedEvents[1].(*a2a.Task)
	assert.True(t, ok, "Event 1 should be Task")

	_, ok = receivedEvents[2].(*a2a.TaskStatusUpdateEvent)
	assert.True(t, ok, "Event 2 should be TaskStatusUpdateEvent")

	_, ok = receivedEvents[3].(*a2a.TaskArtifactUpdateEvent)
	assert.True(t, ok, "Event 3 should be TaskArtifactUpdateEvent")
}

// TestDIDHTTPTransport_SendStreamingMessage_ContextCancellation tests context cancellation
func TestDIDHTTPTransport_SendStreamingMessage_ContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send infinite stream (will be cancelled)
		for i := 0; i < 1000; i++ {
			message := &a2a.Message{
				ID:    fmt.Sprintf("msg-%d", i),
				Role:  a2a.MessageRoleAgent,
				Parts: []a2a.Part{&a2a.TextPart{Text: "Data"}},
			}
			rpcResp, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  map[string]interface{}{"message": message},
			})
			fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
			w.(http.Flusher).Flush()
		}
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Cancel after receiving 5 events
	eventCount := 0
	var lastError error

	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			lastError = err
			break
		}

		eventCount++
		if eventCount >= 5 {
			cancel()
		}

		if event == nil {
			break
		}
	}

	// Should have received context.Canceled error
	assert.Error(t, lastError)
	assert.Equal(t, context.Canceled, lastError)
	assert.GreaterOrEqual(t, eventCount, 5)
}

// TestDIDHTTPTransport_ResubscribeToTask_Success tests task resubscription
func TestDIDHTTPTransport_ResubscribeToTask_Success(t *testing.T) {
	// Create backfill events (messages that occurred while disconnected)
	backfillMessage := &a2a.Message{
		ID:    "msg-backfill",
		Role:  a2a.MessageRoleAgent,
		Parts: []a2a.Part{&a2a.TextPart{Text: "Missed message"}},
	}

	statusUpdate := &a2a.TaskStatusUpdateEvent{
		TaskID: "task-123",
		Status: a2a.TaskStatus{State: a2a.TaskStateCompleted},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a resubscribe request
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "tasks/resubscribe", req.Method)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send backfill events
		rpcResp1, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"message": backfillMessage},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp1))
		w.(http.Flusher).Flush()

		rpcResp2, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"statusUpdate": statusUpdate},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp2))
		w.(http.Flusher).Flush()
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.TaskIDParams{ID: "task-123"}

	// Collect backfill events
	var events []a2a.Event
	for event, err := range transport.ResubscribeToTask(ctx, params) {
		require.NoError(t, err)
		events = append(events, event)
	}

	// Verify backfill events
	require.Len(t, events, 2)

	msg, ok := events[0].(*a2a.Message)
	require.True(t, ok)
	assert.Equal(t, "msg-backfill", msg.ID)

	status, ok := events[1].(*a2a.TaskStatusUpdateEvent)
	require.True(t, ok)
	assert.Equal(t, a2a.TaskID("task-123"), status.TaskID)
}

// TestDIDHTTPTransport_SSE_ErrorHandling tests various error scenarios
func TestDIDHTTPTransport_SSE_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
		body        string
		expectError string
	}{
		{
			name:        "Wrong Content-Type",
			contentType: "application/json",
			statusCode:  http.StatusOK,
			body:        "data: test\n\n",
			expectError: "unexpected Content-Type",
		},
		{
			name:        "HTTP Error Status",
			contentType: "text/event-stream",
			statusCode:  http.StatusInternalServerError,
			body:        "",
			expectError: "HTTP error: 500",
		},
		{
			name:        "Malformed JSON-RPC",
			contentType: "text/event-stream",
			statusCode:  http.StatusOK,
			body:        "data: {invalid json}\n\n",
			expectError: "failed to parse SSE JSON-RPC response",
		},
		{
			name:        "JSON-RPC Error",
			contentType: "text/event-stream",
			statusCode:  http.StatusOK,
			body: `data: {"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}

`,
			expectError: "JSON-RPC error in SSE stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}

			transport, server := setupTestTransport(t, handler)
			defer server.Close()

			ctx := context.Background()
			params := &a2a.MessageSendParams{
				Message: &a2a.Message{
					Role:  a2a.MessageRoleUser,
					Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
				},
			}

			// Should get error
			var gotError error
			for _, err := range transport.SendStreamingMessage(ctx, params) {
				if err != nil {
					gotError = err
					break
				}
			}

			require.Error(t, gotError)
			assert.Contains(t, gotError.Error(), tt.expectError)
		})
	}
}

// TestDIDHTTPTransport_SSE_UnknownEventType tests handling of unknown event types
func TestDIDHTTPTransport_SSE_UnknownEventType(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send unknown event type
		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"unknownField": map[string]string{"foo": "bar"},
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Should get error for unknown event type
	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "unknown SSE event type")
}

// TestDIDHTTPTransport_SSE_MultilineData tests SSE with message containing newlines
func TestDIDHTTPTransport_SSE_MultilineData(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Create a message with text containing newlines
		// This tests that JSON escaping works correctly through SSE
		message := &a2a.Message{
			ID:    "msg-1",
			Role:  a2a.MessageRoleAgent,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Line 1\nLine 2\nLine 3"}},
		}

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"message": message},
		})

		// Send as single SSE data line (JSON already escapes \n properly)
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
		w.(http.Flusher).Flush()
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Collect events
	var events []a2a.Event
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		require.NoError(t, err)
		events = append(events, event)
	}

	// Should successfully parse message with newlines in text
	require.Len(t, events, 1)
	msg, ok := events[0].(*a2a.Message)
	require.True(t, ok)
	assert.Equal(t, "msg-1", msg.ID)

	// Verify message has parts (content preserved through SSE+JSON)
	require.NotEmpty(t, msg.Parts)
}

// TestDIDHTTPTransport_SSE_DIDSignature tests that SSE requests are DID-signed
func TestDIDHTTPTransport_SSE_DIDSignature(t *testing.T) {
	var capturedHeaders http.Header

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Capture headers for verification
		capturedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send minimal valid event
		message := &a2a.Message{ID: "msg-1", Role: a2a.MessageRoleAgent}
		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"message": message},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	// Execute request
	for range transport.SendStreamingMessage(ctx, params) {
		break // Just need to trigger the request
	}

	// Verify DID signature headers are present
	assert.NotEmpty(t, capturedHeaders.Get("Signature"), "Should have Signature header")
	assert.NotEmpty(t, capturedHeaders.Get("Signature-Input"), "Should have Signature-Input header")
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
	assert.Equal(t, "text/event-stream", capturedHeaders.Get("Accept"))

	// Verify signature-input contains DID
	sigInput := capturedHeaders.Get("Signature-Input")
	assert.Contains(t, sigInput, "did:sage:ethereum")
}

// ========================================
// Additional Error Coverage Tests
// ========================================

func TestDIDHTTPTransport_GetTask_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Invalid task JSON
		w.Write(mockJSONRPCResponse("invalid task data"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal Task")
}

func TestDIDHTTPTransport_CancelTask_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse([]int{1, 2, 3})) // Invalid data
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.CancelTask(ctx, &a2a.TaskIDParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal Task")
}

func TestDIDHTTPTransport_SendMessage_InvalidResult(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Result with neither "id" nor "messageId"
		invalidResult := map[string]interface{}{
			"someOtherField": "value",
		}
		w.Write(mockJSONRPCResponse(invalidResult))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			ID:    "msg-123",
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Hello"}},
		},
	}

	_, err := transport.SendMessage(ctx, params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "result is neither Task nor Message")
}

func TestDIDHTTPTransport_GetTaskPushConfig_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse("invalid config"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTaskPushConfig(ctx, &a2a.GetTaskPushConfigParams{
		TaskID: "task-123",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal TaskPushConfig")
}

func TestDIDHTTPTransport_ListTaskPushConfig_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse("not an array"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.ListTaskPushConfig(ctx, &a2a.ListTaskPushConfigParams{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal TaskPushConfig list")
}

func TestDIDHTTPTransport_SetTaskPushConfig_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse(123)) // Number instead of object
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.SetTaskPushConfig(ctx, &a2a.TaskPushConfig{
		TaskID: "task-123",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal TaskPushConfig")
}

func TestDIDHTTPTransport_ListTasks_UnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mockJSONRPCResponse("invalid list result"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.ListTasks(ctx, &protocol.ListTasksParams{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal ListTasksResult")
}

func TestDIDHTTPTransport_GetAgentCard_HTTPError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetAgentCard(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error: 404")
}

func TestDIDHTTPTransport_GetAgentCard_InvalidJSON(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetAgentCard(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode agent card")
}

// ========================================
// SSE Error Coverage Tests
// ========================================

func TestDIDHTTPTransport_SSE_InvalidEventStructure(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send SSE with invalid result structure
		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "this should be an object, not a string",
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "failed to parse SSE result structure")
}

func TestDIDHTTPTransport_SSE_InvalidMessageUnmarshal(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send invalid message structure
		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"message": "invalid message - should be object",
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "failed to parse Message from SSE")
}

func TestDIDHTTPTransport_SSE_InvalidTaskUnmarshal(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": []int{1, 2, 3}, // Invalid task
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "failed to parse Task from SSE")
}

func TestDIDHTTPTransport_SSE_InvalidStatusUpdateUnmarshal(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"statusUpdate": "invalid",
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "failed to parse TaskStatusUpdateEvent from SSE")
}

func TestDIDHTTPTransport_SSE_InvalidArtifactUpdateUnmarshal(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"artifactUpdate": false,
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	var gotError error
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = err
			break
		}
	}

	require.Error(t, gotError)
	assert.Contains(t, gotError.Error(), "failed to parse TaskArtifactUpdateEvent from SSE")
}

func TestDIDHTTPTransport_SSE_ReadError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Write partial event then close connection
		fmt.Fprint(w, "data: incomplete")
		// Don't flush, just close
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			// Stream should end (may or may not have error depending on timing)
			break
		}
		eventCount++
	}

	// Should have ended (either normally or with error)
	// If we get here without hanging, test passes
	assert.GreaterOrEqual(t, eventCount, 0)
}

// ============================================================
// Factory Function Tests
// ============================================================

func TestWithDIDHTTPTransport(t *testing.T) {
	// Setup test server with agent card
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			card := &a2a.AgentCard{
				Name:               "Test Agent",
				URL:                r.URL.Scheme + "://" + r.Host,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
		default:
			// Mock JSON-RPC endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"id":          "task-123",
					"state":       "working",
					"description": "Test task",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Create test credentials
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	agentDID := did.AgentDID("did:sage:test:0x1234")

	// Create agent card
	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                server.URL,
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	// Test factory option
	ctx := context.Background()
	client, err := a2aclient.NewFromCard(
		ctx,
		card,
		WithDIDHTTPTransport(agentDID, keyPair, nil),
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Destroy()

	// Verify transport is working
	task, err := client.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskID("task-123"), task.ID)
}

func TestNewDIDAuthenticatedClient(t *testing.T) {
	// Setup test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			card := &a2a.AgentCard{
				Name:               "Test Agent",
				URL:                r.URL.Scheme + "://" + r.Host,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"id":          "task-456",
					"state":       "completed",
					"description": "Test task",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Create credentials
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	agentDID := did.AgentDID("did:sage:test:0xabcd")

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                server.URL,
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	// Test convenience function
	ctx := context.Background()
	client, err := NewDIDAuthenticatedClient(ctx, agentDID, keyPair, card)

	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Destroy()

	// Verify it works
	task, err := client.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-456"})
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskID("task-456"), task.ID)
}

func TestNewDIDAuthenticatedClientWithInterceptors(t *testing.T) {
	// Setup test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			card := &a2a.AgentCard{
				Name:               "Test Agent",
				URL:                r.URL.Scheme + "://" + r.Host,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"id":    "task-789",
					"state": "working",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	agentDID := did.AgentDID("did:sage:test:0xefgh")

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                server.URL,
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	// Create a test interceptor
	interceptorCalled := false
	testInterceptor := &testCallInterceptor{called: &interceptorCalled}

	// Test with interceptor
	ctx := context.Background()
	client, err := NewDIDAuthenticatedClientWithInterceptors(
		ctx,
		agentDID,
		keyPair,
		card,
		testInterceptor,
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Destroy()

	// Make a call to trigger interceptor
	_, err = client.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-789"})
	require.NoError(t, err)

	// Verify interceptor was called
	assert.True(t, interceptorCalled, "Interceptor should have been called")
}

func TestNewDIDAuthenticatedClientWithConfig(t *testing.T) {
	// Setup test server
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/agent.json":
			card := &a2a.AgentCard{
				Name:               "Test Agent",
				URL:                r.URL.Scheme + "://" + r.Host,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(card)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"id":    "task-999",
					"state": "completed",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	agentDID := did.AgentDID("did:sage:test:0xconfig")

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                server.URL,
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	// Create custom config
	config := a2aclient.Config{
		AcceptedOutputModes: []string{"application/json"},
	}

	// Test with custom config
	ctx := context.Background()
	client, err := NewDIDAuthenticatedClientWithConfig(
		ctx,
		agentDID,
		keyPair,
		card,
		config,
	)

	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Destroy()

	// Verify it works
	task, err := client.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-999"})
	require.NoError(t, err)
	assert.Equal(t, a2a.TaskID("task-999"), task.ID)
}

// ============================================================
// Additional Coverage Tests for 90% Target
// ============================================================

func TestDIDHTTPTransport_Call_JSONRPCError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return JSON-RPC error response
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32600,
				"message": "Invalid Request",
			},
		}
		json.NewEncoder(w).Encode(resp)
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

func TestDIDHTTPTransport_SendMessage_MessageUnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return result with messageId but invalid Message structure
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"messageId": "msg-123",
				"parts":     "invalid-should-be-array",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	_, err := transport.SendMessage(ctx, params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal Message")
}

func TestDIDHTTPTransport_SendMessage_TaskUnmarshalError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return result with id but invalid Task structure (status should be object)
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"id":     "task-123",
				"status": "invalid-should-be-object",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	_, err := transport.SendMessage(ctx, params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal Task")
}

func TestDIDHTTPTransport_SendMessage_NeitherTaskNorMessage(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return result without id or messageId
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"unknown": "field",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	_, err := transport.SendMessage(ctx, params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "result is neither Task nor Message")
}

func TestDIDHTTPTransport_SSE_InvalidLineWithoutColon(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send invalid line without colon (should be skipped)
		fmt.Fprintf(w, "invalid line without colon\n")

		// Send valid event
		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": map[string]interface{}{
					"id":    "task-123",
					"state": "working",
				},
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event != nil {
			eventCount++
		}
	}

	// Should receive at least 1 valid event (invalid line should be skipped)
	assert.GreaterOrEqual(t, eventCount, 1)
}

func TestDIDHTTPTransport_SSE_DataWithLeadingSpace(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": map[string]interface{}{
					"id":    "task-123",
					"state": "working",
				},
			},
		})

		// SSE spec: leading space after colon should be removed
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event != nil {
			eventCount++
		}
	}

	// Should successfully parse data with leading space
	assert.GreaterOrEqual(t, eventCount, 1)
}

func TestDIDHTTPTransport_SSE_EventAndIDFields(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": map[string]interface{}{
					"id":    "task-123",
					"state": "working",
				},
			},
		})

		// Send with event and id fields
		fmt.Fprintf(w, "event: message\n")
		fmt.Fprintf(w, "id: event-123\n")
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event != nil {
			eventCount++
		}
	}

	// Should successfully parse event with event and id fields
	assert.GreaterOrEqual(t, eventCount, 1)
}

func TestDIDHTTPTransport_SSE_RetryField(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": map[string]interface{}{
					"id":    "task-123",
					"state": "working",
				},
			},
		})

		// Send with retry field
		fmt.Fprintf(w, "retry: 5000\n")
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event != nil {
			eventCount++
		}
	}

	// Should successfully parse event with retry field
	assert.GreaterOrEqual(t, eventCount, 1)
}

func TestDIDHTTPTransport_SSE_UnknownField(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		rpcResp, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]interface{}{
				"task": map[string]interface{}{
					"id":    "task-123",
					"state": "working",
				},
			},
		})

		// Send with unknown field (should be ignored per SSE spec)
		fmt.Fprintf(w, "unknown: field\n")
		fmt.Fprintf(w, "data: %s\n\n", string(rpcResp))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	eventCount := 0
	for event, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if event != nil {
			eventCount++
		}
	}

	// Should successfully parse event (unknown field should be ignored)
	assert.GreaterOrEqual(t, eventCount, 1)
}

func TestDIDHTTPTransport_Call_HTTPStatusNotOK(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error: 500")
}

func TestDIDHTTPTransport_Call_InvalidJSONResponse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	_, err := transport.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON-RPC response")
}

func TestDIDHTTPTransport_SSE_HTTPStatusNotOK(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	gotError := false
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = true
			assert.Contains(t, err.Error(), "HTTP error: 503")
			break
		}
	}

	assert.True(t, gotError, "Expected HTTP error")
}

func TestDIDHTTPTransport_SSE_WrongContentType(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Test"}},
		},
	}

	gotError := false
	for _, err := range transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			gotError = true
			assert.Contains(t, err.Error(), "unexpected Content-Type")
			break
		}
	}

	assert.True(t, gotError, "Expected Content-Type error")
}

func TestDIDHTTPTransport_ListTasks_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := mockJSONRPCResponse(map[string]interface{}{
			"tasks": []interface{}{
				map[string]interface{}{
					"id":    "task-1",
					"state": "working",
				},
				map[string]interface{}{
					"id":    "task-2",
					"state": "completed",
				},
			},
		})
		w.Write(resp)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	result, err := transport.ListTasks(ctx, &protocol.ListTasksParams{})

	require.NoError(t, err)
	assert.Len(t, result.Tasks, 2)
	assert.Equal(t, a2a.TaskID("task-1"), result.Tasks[0].ID)
	assert.Equal(t, a2a.TaskID("task-2"), result.Tasks[1].ID)
}

func TestDIDHTTPTransport_GetAgentCard_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/agent-card.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			card := &a2a.AgentCard{
				Name:               "Test Agent",
				URL:                "https://test.example.com",
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			}
			json.NewEncoder(w).Encode(card)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentDID := did.AgentDID("did:sage:test:0x1234")
	transport := NewDIDHTTPTransport(server.URL, agentDID, keyPair, nil).(*DIDHTTPTransport)

	ctx := context.Background()
	card, err := transport.GetAgentCard(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Test Agent", card.Name)
}

func TestDIDHTTPTransport_DeleteTaskPushConfig_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := mockJSONRPCResponse(map[string]interface{}{
			"success": true,
		})
		w.Write(resp)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	ctx := context.Background()
	err := transport.DeleteTaskPushConfig(ctx, &a2a.DeleteTaskPushConfigParams{
		TaskID:   "task-123",
		ConfigID: "config-1",
	})

	require.NoError(t, err)
}

func TestDIDHTTPTransport_Destroy_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}

	transport, server := setupTestTransport(t, handler)
	defer server.Close()

	err := transport.Destroy()
	require.NoError(t, err)
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

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/sage-x-project/sage-a2a-go/pkg/protocol"
	"github.com/sage-x-project/sage-a2a-go/pkg/transport"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize crypto subsystem
	crypto.SetKeyGenerators(
		func() (crypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateP256KeyPair() },
	)
	crypto.SetStorageConstructors(
		func() crypto.KeyStorage { return storage.NewMemoryKeyStorage() },
	)
	crypto.SetFormatConstructors(
		func() crypto.KeyExporter { return formats.NewJWKExporter() },
		func() crypto.KeyExporter { return formats.NewPEMExporter() },
		func() crypto.KeyImporter { return formats.NewJWKImporter() },
		func() crypto.KeyImporter { return formats.NewPEMImporter() },
	)
}

// TestE2E_FullHTTPCycle tests the complete HTTP request/response cycle with DID authentication
func TestE2E_FullHTTPCycle(t *testing.T) {
	// Setup: Create client DID and keypair
	clientKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	clientDID := did.AgentDID("did:sage:ethereum:0xclient")

	// Create simple HTTP server (without DID verification for basic e2e test)
	mux := http.NewServeMux()

	// Add well-known agent card endpoint
	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		card := map[string]interface{}{
			"name":        "Test Agent",
			"description": "E2E Test Agent",
			"url":         "http://localhost",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	})

	// Add JSON-RPC endpoint
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Parse JSON-RPC request
		var rpcReq map[string]interface{}
		if err := json.Unmarshal(body, &rpcReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, ok := rpcReq["method"].(string)
		if !ok {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		// Handle different methods
		var result interface{}
		switch method {
		case "message/send":
			// Create a proper Message object using NewMessage
			msg := a2a.NewMessage(
				a2a.MessageRoleAgent,
				&a2a.TextPart{Text: "Hello from server!"},
			)
			msg.ID = "msg-123"
			result = msg
		default:
			http.Error(w, fmt.Sprintf("Unknown method: %s", method), http.StatusBadRequest)
			return
		}

		// Marshal result to JSON
		resultJSON, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "Failed to marshal result", http.StatusInternalServerError)
			return
		}

		// Create JSON-RPC response
		rpcResp := map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  json.RawMessage(resultJSON),
			"id":      rpcReq["id"],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rpcResp)
	})

	// Start test server
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("SendMessage_Success", func(t *testing.T) {
		// Create client
		client, err := transport.NewDIDAuthenticatedClient(
			context.Background(),
			clientDID,
			clientKeyPair,
			&a2a.AgentCard{
				URL:                testServer.URL,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			},
		)
		require.NoError(t, err)
		defer client.Destroy()

		// Send message
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		message := a2a.NewMessage(
			a2a.MessageRoleUser,
			&a2a.TextPart{Text: "Hello from client!"},
		)

		result, err := client.SendMessage(ctx, &a2a.MessageSendParams{
			Message: message,
		})

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)

		// Check response message
		if msg, ok := result.(*a2a.Message); ok {
			assert.Equal(t, "msg-123", msg.ID)
			assert.Equal(t, a2a.MessageRoleAgent, msg.Role)
			require.Len(t, msg.Parts, 1)

			// After bug fix, Parts should be pointer types
			if textPart, ok := msg.Parts[0].(*a2a.TextPart); ok {
				assert.Equal(t, "Hello from server!", textPart.Text)
			} else {
				t.Fatalf("Expected *a2a.TextPart (pointer), got %T", msg.Parts[0])
			}
		} else {
			t.Fatalf("Expected *a2a.Message, got %T", result)
		}
	})

	t.Run("GetAgentCard_Success", func(t *testing.T) {
		// Create client
		client, err := transport.NewDIDAuthenticatedClient(
			context.Background(),
			clientDID,
			clientKeyPair,
			&a2a.AgentCard{
				URL:                testServer.URL,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			},
		)
		require.NoError(t, err)
		defer client.Destroy()

		// Get agent card
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		card, err := client.GetAgentCard(ctx)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, card)
		assert.Equal(t, "Test Agent", card.Name)
		assert.Equal(t, "E2E Test Agent", card.Description)
	})

	t.Run("Timeout_HandledCorrectly", func(t *testing.T) {
		// Create client
		client, err := transport.NewDIDAuthenticatedClient(
			context.Background(),
			clientDID,
			clientKeyPair,
			&a2a.AgentCard{
				URL:                testServer.URL,
				PreferredTransport: a2a.TransportProtocolJSONRPC,
			},
		)
		require.NoError(t, err)
		defer client.Destroy()

		// Use very short timeout to trigger timeout error
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		message := a2a.NewMessage(
			a2a.MessageRoleUser,
			&a2a.TextPart{Text: "This will timeout"},
		)

		_, err = client.SendMessage(ctx, &a2a.MessageSendParams{
			Message: message,
		})

		// Should fail due to timeout
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}

// TestE2E_SSEStreaming tests Server-Sent Events streaming functionality
func TestE2E_SSEStreaming(t *testing.T) {
	// Setup: Create client DID and keypair
	clientKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	clientDID := did.AgentDID("did:sage:ethereum:0xclient")

	// Create HTTP server with SSE endpoint
	mux := http.NewServeMux()

	// Add well-known agent card endpoint
	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		card := map[string]interface{}{
			"name":        "Test Agent",
			"description": "E2E Test Agent",
			"url":         "http://localhost",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	})

	// Add SSE streaming endpoint (sends JSON-RPC wrapped responses)
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Check if client wants SSE
		if r.Header.Get("Accept") == "text/event-stream" {
			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming not supported", http.StatusInternalServerError)
				return
			}

			// Send 3 messages wrapped in JSON-RPC
			for i := 1; i <= 3; i++ {
				msg := a2a.NewMessage(
					a2a.MessageRoleAgent,
					&a2a.TextPart{Text: fmt.Sprintf("Stream message %d", i)},
				)
				msg.ID = fmt.Sprintf("msg-%d", i)

				// Wrap in JSON-RPC response
				rpcResp := map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      1,
					"result": map[string]interface{}{
						"message": msg,
					},
				}

				rpcJSON, _ := json.Marshal(rpcResp)
				fmt.Fprintf(w, "data: %s\n\n", rpcJSON)
				flusher.Flush()

				// Small delay between messages
				time.Sleep(10 * time.Millisecond)
			}
			return
		}

		// Handle regular JSON-RPC request (non-streaming)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		var rpcReq map[string]interface{}
		if err := json.Unmarshal(body, &rpcReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, ok := rpcReq["method"].(string)
		if !ok {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		var result interface{}
		switch method {
		case "message/send":
			msg := a2a.NewMessage(
				a2a.MessageRoleAgent,
				&a2a.TextPart{Text: "Hello from server!"},
			)
			msg.ID = "msg-123"
			result = msg
		default:
			http.Error(w, fmt.Sprintf("Unknown method: %s", method), http.StatusBadRequest)
			return
		}

		resultJSON, _ := json.Marshal(result)
		rpcResp := map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  json.RawMessage(resultJSON),
			"id":      rpcReq["id"],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rpcResp)
	})

	// Start test server
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("StreamMessage_Success", func(t *testing.T) {
		// Create transport directly for streaming
		httpTransport := transport.NewDIDHTTPTransport(
			testServer.URL,
			clientDID,
			clientKeyPair,
			nil, // use default HTTP client
		)

		// Create streaming context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		message := a2a.NewMessage(
			a2a.MessageRoleUser,
			&a2a.TextPart{Text: "Start streaming"},
		)

		// Stream messages using iterator
		messageCount := 0
		for event, err := range httpTransport.SendStreamingMessage(ctx, &a2a.MessageSendParams{
			Message: message,
		}) {
			if err != nil {
				t.Logf("Stream error: %v", err)
				break
			}

			messageCount++
			t.Logf("Received stream event %d: %T", messageCount, event)

			// Check if it's a Message
			if msg, ok := event.(*a2a.Message); ok {
				require.NotEmpty(t, msg.ID)
				require.Equal(t, a2a.MessageRoleAgent, msg.Role)
				require.Len(t, msg.Parts, 1)

				// After bug fix, Parts should be pointer types
				if textPart, ok := msg.Parts[0].(*a2a.TextPart); ok {
					assert.Contains(t, textPart.Text, "Stream message")
				} else {
					t.Fatalf("Expected *a2a.TextPart (pointer), got %T", msg.Parts[0])
				}
			}
		}

		// Verify we received messages
		assert.Equal(t, 3, messageCount, "Should have received 3 streamed messages")
	})
}

// TestE2E_TaskOperations tests Task-related operations (CreateTask, GetTask)
func TestE2E_TaskOperations(t *testing.T) {
	// Setup: Create client DID and keypair
	clientKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	clientDID := did.AgentDID("did:sage:ethereum:0xclient")

	// Create HTTP server
	mux := http.NewServeMux()

	// Add well-known agent card endpoint
	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		card := map[string]interface{}{
			"name":        "Test Agent",
			"description": "E2E Test Agent",
			"url":         "http://localhost",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	})

	// Add JSON-RPC endpoint for task operations
	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		var rpcReq map[string]interface{}
		if err := json.Unmarshal(body, &rpcReq); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		method, ok := rpcReq["method"].(string)
		if !ok {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		var result interface{}
		switch method {
		case "tasks/get":
			// Return existing task
			task := &a2a.Task{
				ID:        a2a.TaskID("task-456"),
				ContextID: "ctx-123",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCompleted,
				},
			}
			result = task
		case "tasks/list":
			// Return list of tasks
			taskList := &protocol.ListTasksResult{
				Tasks: []*a2a.Task{
					{
						ID:        a2a.TaskID("task-1"),
						ContextID: "ctx-123",
						Status: a2a.TaskStatus{
							State: a2a.TaskStateWorking,
						},
					},
					{
						ID:        a2a.TaskID("task-2"),
						ContextID: "ctx-123",
						Status: a2a.TaskStatus{
							State: a2a.TaskStateCompleted,
						},
					},
				},
				TotalSize: 2,
				PageSize:  10,
			}
			result = taskList
		case "tasks/cancel":
			// Return cancelled task
			task := &a2a.Task{
				ID:        a2a.TaskID("task-456"),
				ContextID: "ctx-123",
				Status: a2a.TaskStatus{
					State: a2a.TaskStateCanceled,
				},
			}
			result = task
		default:
			http.Error(w, fmt.Sprintf("Unknown method: %s", method), http.StatusBadRequest)
			return
		}

		resultJSON, _ := json.Marshal(result)
		rpcResp := map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  json.RawMessage(resultJSON),
			"id":      rpcReq["id"],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rpcResp)
	})

	// Start test server
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("GetTask_Success", func(t *testing.T) {
		// Create transport directly
		httpTransport := transport.NewDIDHTTPTransport(
			testServer.URL,
			clientDID,
			clientKeyPair,
			nil,
		)

		// Get task
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := httpTransport.GetTask(ctx, &a2a.TaskQueryParams{
			ID: "task-456",
		})

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, a2a.TaskID("task-456"), result.ID)
		assert.Equal(t, a2a.TaskStateCompleted, result.Status.State)
	})

	t.Run("ListTasks_Success", func(t *testing.T) {
		// Create transport directly (cast to concrete type for ListTasks method)
		httpTransport := transport.NewDIDHTTPTransport(
			testServer.URL,
			clientDID,
			clientKeyPair,
			nil,
		).(*transport.DIDHTTPTransport)

		// List tasks
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := httpTransport.ListTasks(ctx, &protocol.ListTasksParams{
			ContextID: "ctx-123",
		})

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Tasks, 2)
		assert.Equal(t, a2a.TaskID("task-1"), result.Tasks[0].ID)
		assert.Equal(t, a2a.TaskID("task-2"), result.Tasks[1].ID)
	})

	t.Run("CancelTask_Success", func(t *testing.T) {
		// Create transport directly
		httpTransport := transport.NewDIDHTTPTransport(
			testServer.URL,
			clientDID,
			clientKeyPair,
			nil,
		)

		// Cancel task
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := httpTransport.CancelTask(ctx, &a2a.TaskIDParams{
			ID: "task-456",
		})

		// Verify
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, a2a.TaskID("task-456"), result.ID)
		assert.Equal(t, a2a.TaskStateCanceled, result.Status.State)
	})
}

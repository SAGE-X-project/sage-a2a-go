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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
)

// sseEvent represents a single Server-Sent Event
type sseEvent struct {
	// Event type (optional)
	Event string
	// Data payload
	Data []byte
	// ID for event stream tracking (optional)
	ID string
	// Retry interval in milliseconds (optional)
	Retry int
}

// parseSSEStream reads and parses Server-Sent Events from an HTTP response.
// It returns an iterator that yields a2a.Event and error pairs.
//
// SSE Format (per W3C spec):
//
//	event: message
//	data: {"jsonrpc":"2.0","id":1,"result":{...}}
//	id: 123
//
//	data: {"jsonrpc":"2.0","id":2,"result":{...}}
//
// The function handles:
//   - Multi-line data fields (concatenated with \n)
//   - Event type specification
//   - Event IDs for resumption
//   - Context cancellation
//   - Connection errors
func parseSSEStream(ctx context.Context, resp *http.Response) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		var currentEvent sseEvent
		var dataBuffer bytes.Buffer

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			// Read a line from the stream
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					return
				}
				yield(nil, fmt.Errorf("error reading SSE stream: %w", err))
				return
			}

			// Remove trailing newline
			line = bytes.TrimRight(line, "\r\n")

			// Empty line means end of event
			if len(line) == 0 {
				if dataBuffer.Len() > 0 {
					currentEvent.Data = dataBuffer.Bytes()
					dataBuffer.Reset()

					// Parse the JSON-RPC response from the SSE data
					event, err := parseSSEData(currentEvent.Data)
					if err != nil {
						if !yield(nil, err) {
							return
						}
						// Reset for next event
						currentEvent = sseEvent{}
						continue
					}

					// Yield the event
					if !yield(event, nil) {
						return
					}

					// Reset for next event
					currentEvent = sseEvent{}
				}
				continue
			}

			// Parse field
			colonIndex := bytes.IndexByte(line, ':')
			if colonIndex == -1 {
				// Invalid line, skip
				continue
			}

			field := string(line[:colonIndex])
			value := line[colonIndex+1:]

			// Remove leading space from value (per SSE spec)
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}

			// Handle different field types
			switch field {
			case "event":
				currentEvent.Event = string(value)
			case "data":
				// Accumulate data (may be multi-line)
				if dataBuffer.Len() > 0 {
					dataBuffer.WriteByte('\n')
				}
				dataBuffer.Write(value)
			case "id":
				currentEvent.ID = string(value)
			case "retry":
				// Retry interval in milliseconds (not currently used)
				// Could be used for automatic reconnection logic
			default:
				// Unknown field, ignore per SSE spec
			}
		}
	}
}

// parseSSEData parses the JSON-RPC response from SSE data and extracts the A2A event.
//
// The data contains a JSON-RPC response with one of these result types:
//   - Message
//   - Task
//   - TaskStatusUpdateEvent
//   - TaskArtifactUpdateEvent
func parseSSEData(data []byte) (a2a.Event, error) {
	// Parse JSON-RPC response wrapper
	var rpcResp struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Result  json.RawMessage `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse SSE JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error in SSE stream: %d - %s",
			rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// The result can be Message, Task, TaskStatusUpdateEvent, or TaskArtifactUpdateEvent
	// We need to determine which type it is by trying to unmarshal

	// Try to determine the type by looking for discriminating fields
	var typeCheck struct {
		Message        json.RawMessage `json:"message"`
		Task           json.RawMessage `json:"task"`
		StatusUpdate   json.RawMessage `json:"statusUpdate"`
		ArtifactUpdate json.RawMessage `json:"artifactUpdate"`
	}

	if err := json.Unmarshal(rpcResp.Result, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse SSE result structure: %w", err)
	}

	// Determine which field is present and parse accordingly
	if typeCheck.Message != nil {
		var msg a2a.Message
		if err := json.Unmarshal(typeCheck.Message, &msg); err != nil {
			return nil, fmt.Errorf("failed to parse Message from SSE: %w", err)
		}
		return &msg, nil
	}

	if typeCheck.Task != nil {
		var task a2a.Task
		if err := json.Unmarshal(typeCheck.Task, &task); err != nil {
			return nil, fmt.Errorf("failed to parse Task from SSE: %w", err)
		}
		return &task, nil
	}

	if typeCheck.StatusUpdate != nil {
		var statusEvent a2a.TaskStatusUpdateEvent
		if err := json.Unmarshal(typeCheck.StatusUpdate, &statusEvent); err != nil {
			return nil, fmt.Errorf("failed to parse TaskStatusUpdateEvent from SSE: %w", err)
		}
		return &statusEvent, nil
	}

	if typeCheck.ArtifactUpdate != nil {
		var artifactEvent a2a.TaskArtifactUpdateEvent
		if err := json.Unmarshal(typeCheck.ArtifactUpdate, &artifactEvent); err != nil {
			return nil, fmt.Errorf("failed to parse TaskArtifactUpdateEvent from SSE: %w", err)
		}
		return &artifactEvent, nil
	}

	return nil, fmt.Errorf("unknown SSE event type in result")
}

// callSSE makes an HTTP request expecting an SSE stream response.
// It returns an iterator of A2A events.
func (t *DIDHTTPTransport) callSSE(ctx context.Context, method string, params any) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// Create JSON-RPC request
		rpcReq := jsonRPCRequest{
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
			ID:      1,
		}

		// Marshal request body
		body, err := json.Marshal(rpcReq)
		if err != nil {
			yield(nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err))
			return
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", t.baseURL+"/rpc", bytes.NewReader(body))
		if err != nil {
			yield(nil, fmt.Errorf("failed to create HTTP request: %w", err))
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		// Sign request with DID
		if err := t.signer.SignRequest(ctx, req, t.agentDID, t.keyPair); err != nil {
			yield(nil, fmt.Errorf("failed to sign request with DID: %w", err))
			return
		}

		// Execute HTTP request
		resp, err := t.httpClient.Do(req)
		if err != nil {
			yield(nil, fmt.Errorf("HTTP request failed: %w", err))
			return
		}

		// Verify Content-Type is text/event-stream
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/event-stream") {
			resp.Body.Close()
			yield(nil, fmt.Errorf("unexpected Content-Type: %s, expected text/event-stream", contentType))
			return
		}

		// Check HTTP status
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			yield(nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status))
			return
		}

		// Parse SSE stream
		for event, err := range parseSSEStream(ctx, resp) {
			if !yield(event, err) {
				return
			}
		}
	}
}

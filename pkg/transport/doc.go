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

// Package transport provides DID-authenticated transport implementations for a2a-go.
//
// This package implements the a2aclient.Transport interface to provide HTTP/JSON-RPC 2.0
// support with automatic DID signature on all requests using RFC 9421 HTTP Message Signatures.
//
// # Key Features
//
//   - HTTP/JSON-RPC 2.0 protocol support (required by A2A specification)
//   - Automatic DID signing on all outgoing requests
//   - RFC 9421 HTTP Message Signatures
//   - Seamless integration with a2a-go client infrastructure
//   - Compatible with all a2a-go features (interceptors, config, etc.)
//
// # Usage
//
// The simplest way to use this package is via the convenience functions:
//
//	client, err := transport.NewDIDAuthenticatedClient(
//	    ctx,
//	    myAgentDID,
//	    myKeyPair,
//	    targetAgentCard,
//	)
//	if err != nil {
//	    return err
//	}
//	defer client.Destroy()
//
//	// Use standard a2a-go client methods
//	task, err := client.SendMessage(ctx, message)
//
// For more control, use the factory option directly:
//
//	client, err := a2aclient.NewFromCard(
//	    ctx,
//	    agentCard,
//	    transport.WithDIDHTTPTransport(myDID, myKeyPair, nil),
//	    a2aclient.WithConfig(customConfig),
//	    a2aclient.WithInterceptors(loggingInterceptor),
//	)
//
// # Architecture
//
// The transport layer sits between a2a-go's Client and the actual HTTP/HTTPS
// network layer:
//
//	a2aclient.Client (a2a-go)
//	    └─→ CallInterceptors
//	        └─→ DIDHTTPTransport (sage-a2a-go)
//	            └─→ HTTP/JSON-RPC 2.0 + DID Signatures
//	                └─→ Network
//
// # Protocol Support
//
// The DIDHTTPTransport implements all required A2A protocol methods:
//
//   - GetTask, CancelTask
//   - SendMessage, SendStreamingMessage (via SSE)
//   - ResubscribeToTask (via SSE)
//   - GetTaskPushConfig, ListTaskPushConfig, SetTaskPushConfig, DeleteTaskPushConfig
//   - GetAgentCard
//
// # Security
//
// All HTTP requests are automatically signed with the agent's DID using RFC 9421:
//
//   - Signature-Input header with DID as keyid
//   - Signature header with base64-encoded signature
//   - Covers HTTP method, target URI, and request body
//   - Timestamp included for replay attack prevention
//
// # Future Enhancements
//
//   - Server-Sent Events (SSE) for streaming methods
//   - WebSocket transport
//   - Request/response logging
//   - Metrics and observability
package transport

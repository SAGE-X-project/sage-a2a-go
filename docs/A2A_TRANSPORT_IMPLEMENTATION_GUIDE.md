# A2A/gRPC Transport Implementation Guide

**Target Project:** sage-a2a-go
**SAGE Version:** 1.3.0+
**Last Updated:** 2025-10-26

## Overview

This document provides a comprehensive guide for implementing gRPC/A2A transport in the `sage-a2a-go` project. SAGE provides the transport abstraction interface, and sage-a2a-go implements the concrete gRPC/A2A transport adapter.

### Project Responsibilities

**SAGE (Core Framework):**
- ✅ Transport interface definition (`MessageTransport`)
- ✅ Message types (`SecureMessage`, `Response`)
- ✅ Transport selector with gRPC scheme support (`grpc://`)
- ✅ Registration mechanism for transport factories

**sage-a2a-go (A2A Integration):**
- ❌ gRPC service definition (protobuf)
- ❌ gRPC server implementation
- ❌ gRPC client implementation (MessageTransport adapter)
- ❌ Transport factory registration
- ❌ A2A protocol-specific features

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  SAGE Security Layer                                     │
│  (handshake, HPKE, session management)                  │
└────────────────────┬────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────┐
│  transport.MessageTransport interface (SAGE)            │
│  - Send(ctx, SecureMessage) → Response                  │
└────────────────────┬────────────────────────────────────┘
                     │ implemented by
                     ▼
┌─────────────────────────────────────────────────────────┐
│  A2ATransport (sage-a2a-go)                             │
│  - Wraps gRPC client                                    │
│  - Converts SecureMessage ↔ protobuf                    │
└────────────────────┬────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────┐
│  gRPC Client/Server (sage-a2a-go)                       │
│  - A2A protocol implementation                          │
│  - Protobuf message definitions                         │
└─────────────────────────────────────────────────────────┘
```

## SAGE Transport Interface

### MessageTransport Interface

All transport implementations must satisfy this interface:

```go
// From: github.com/sage-x-project/sage/pkg/agent/transport

type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}
```

### SecureMessage Type

Input message prepared by SAGE security layer:

```go
type SecureMessage struct {
    // Message identifiers
    ID        string  // Unique message ID (UUID)
    ContextID string  // Conversation context ID
    TaskID    string  // Task identifier

    // Security payload (already encrypted by SAGE)
    Payload []byte  // Encrypted message content

    // DID and authentication
    DID       string  // Sender DID (did:sage:ethereum:...)
    Signature []byte  // Message signature

    // Additional metadata
    Metadata map[string]string  // Custom headers

    // Message role
    Role string  // "user" or "agent"
}
```

**Important:** The `Payload` is already encrypted by SAGE's HPKE layer. The transport only handles transmission.

### Response Type

Output response to SAGE:

```go
type Response struct {
    Success   bool    // Whether message was delivered
    MessageID string  // Echo of sent message ID
    TaskID    string  // Echo of task ID
    Data      []byte  // Response payload (encrypted)
    Error     error   // Transport error if any
}
```

## Implementation Steps

### Step 1: Define Protobuf Service

Create `proto/a2a/messaging.proto` in sage-a2a-go:

```protobuf
syntax = "proto3";

package a2a.v1;

option go_package = "github.com/a2aproject/sage-a2a-go/gen/a2a/v1;a2av1";

// A2A Messaging Service
service A2AMessaging {
    // SendSecureMessage sends a SAGE encrypted message
    rpc SendSecureMessage(SecureMessageRequest) returns (SecureMessageResponse);

    // StreamMessages enables bidirectional streaming (optional)
    rpc StreamMessages(stream SecureMessageRequest) returns (stream SecureMessageResponse);
}

// SecureMessageRequest maps to SAGE's SecureMessage
message SecureMessageRequest {
    // Message identifiers
    string message_id = 1;
    string context_id = 2;
    string task_id = 3;

    // Encrypted payload (from SAGE)
    bytes payload = 4;

    // DID and signature
    string sender_did = 5;
    bytes signature = 6;

    // Metadata
    map<string, string> metadata = 7;

    // Role
    string role = 8;  // "user" or "agent"
}

// SecureMessageResponse maps to SAGE's Response
message SecureMessageResponse {
    // Status
    bool success = 1;

    // Message tracking
    string message_id = 2;
    string task_id = 3;

    // Response data (encrypted)
    bytes data = 4;

    // Error information
    string error_message = 5;
}
```

### Step 2: Generate gRPC Code

Add to `sage-a2a-go/Makefile`:

```makefile
# Generate protobuf and gRPC code
.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/a2a/messaging.proto

.PHONY: generate
generate: proto
```

Run:
```bash
cd sage-a2a-go
make generate
```

### Step 3: Implement gRPC Server

Create `sage-a2a-go/pkg/transport/grpc/server.go`:

```go
package grpc

import (
    "context"
    "fmt"

    a2av1 "github.com/a2aproject/sage-a2a-go/gen/a2a/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Server implements the A2A gRPC server
type Server struct {
    a2av1.UnimplementedA2AMessagingServer

    // Handler processes incoming secure messages
    // This is typically connected to SAGE's message handler
    handler MessageHandler
}

// MessageHandler processes incoming SAGE secure messages
type MessageHandler interface {
    HandleSecureMessage(ctx context.Context, req *a2av1.SecureMessageRequest) (*a2av1.SecureMessageResponse, error)
}

// NewServer creates a new A2A gRPC server
func NewServer(handler MessageHandler) *Server {
    return &Server{
        handler: handler,
    }
}

// SendSecureMessage implements the A2A messaging service
func (s *Server) SendSecureMessage(ctx context.Context, req *a2av1.SecureMessageRequest) (*a2av1.SecureMessageResponse, error) {
    // Validate request
    if req.MessageId == "" {
        return nil, status.Error(codes.InvalidArgument, "message_id is required")
    }
    if req.SenderDid == "" {
        return nil, status.Error(codes.InvalidArgument, "sender_did is required")
    }
    if len(req.Payload) == 0 {
        return nil, status.Error(codes.InvalidArgument, "payload is required")
    }

    // Delegate to handler (SAGE security layer)
    resp, err := s.handler.HandleSecureMessage(ctx, req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to handle message: %v", err)
    }

    return resp, nil
}

// StreamMessages implements bidirectional streaming (optional)
func (s *Server) StreamMessages(stream a2av1.A2AMessaging_StreamMessagesServer) error {
    for {
        req, err := stream.Recv()
        if err != nil {
            return err
        }

        resp, err := s.handler.HandleSecureMessage(stream.Context(), req)
        if err != nil {
            return status.Errorf(codes.Internal, "failed to handle message: %v", err)
        }

        if err := stream.Send(resp); err != nil {
            return err
        }
    }
}

// StartServer starts the gRPC server
func StartServer(addr string, handler MessageHandler) error {
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }

    grpcServer := grpc.NewServer()
    a2av1.RegisterA2AMessagingServer(grpcServer, NewServer(handler))

    return grpcServer.Serve(lis)
}
```

### Step 4: Implement Transport Adapter (Client)

Create `sage-a2a-go/pkg/transport/grpc/client.go`:

```go
package grpc

import (
    "context"
    "fmt"

    a2av1 "github.com/a2aproject/sage-a2a-go/gen/a2a/v1"
    "github.com/sage-x-project/sage/pkg/agent/transport"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// A2ATransport implements transport.MessageTransport using gRPC
type A2ATransport struct {
    client a2av1.A2AMessagingClient
    conn   *grpc.ClientConn
}

// NewA2ATransport creates a new gRPC transport
func NewA2ATransport(endpoint string) (transport.MessageTransport, error) {
    // Connect to gRPC server
    conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
    }

    client := a2av1.NewA2AMessagingClient(conn)

    return &A2ATransport{
        client: client,
        conn:   conn,
    }, nil
}

// Send implements transport.MessageTransport
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Convert SAGE SecureMessage to protobuf
    req := &a2av1.SecureMessageRequest{
        MessageId: msg.ID,
        ContextId: msg.ContextID,
        TaskId:    msg.TaskID,
        Payload:   msg.Payload,
        SenderDid: msg.DID,
        Signature: msg.Signature,
        Metadata:  msg.Metadata,
        Role:      msg.Role,
    }

    // Send via gRPC
    resp, err := t.client.SendSecureMessage(ctx, req)
    if err != nil {
        return &transport.Response{
            Success:   false,
            MessageID: msg.ID,
            TaskID:    msg.TaskID,
            Error:     err,
        }, nil
    }

    // Convert protobuf response to SAGE Response
    var respErr error
    if resp.ErrorMessage != "" {
        respErr = fmt.Errorf("%s", resp.ErrorMessage)
    }

    return &transport.Response{
        Success:   resp.Success,
        MessageID: resp.MessageId,
        TaskID:    resp.TaskId,
        Data:      resp.Data,
        Error:     respErr,
    }, nil
}

// Close closes the gRPC connection
func (t *A2ATransport) Close() error {
    return t.conn.Close()
}
```

### Step 5: Register with SAGE Transport Selector

Create `sage-a2a-go/pkg/transport/grpc/register.go`:

```go
package grpc

import (
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func init() {
    // Register gRPC transport factory with SAGE
    transport.DefaultSelector.RegisterFactory(transport.TransportGRPC, NewA2ATransport)
}
```

### Step 6: Integration in Application

In your `sage-a2a-go` application:

```go
package main

import (
    "context"
    "log"

    // Import to auto-register gRPC transport
    _ "github.com/a2aproject/sage-a2a-go/pkg/transport/grpc"

    "github.com/sage-x-project/sage/pkg/agent/transport"
    "github.com/sage-x-project/sage/pkg/agent/handshake"
)

func main() {
    // Create transport using SAGE selector
    trans, err := transport.SelectByURL("grpc://agent.example.com:50051")
    if err != nil {
        log.Fatal(err)
    }

    // Use with SAGE components
    client := handshake.NewClient(trans, keyPair)

    // Send messages
    resp, err := client.SendMessage(context.Background(), message)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: %+v", resp)
}
```

## Testing

### Unit Tests

Create `sage-a2a-go/pkg/transport/grpc/client_test.go`:

```go
package grpc

import (
    "context"
    "net"
    "testing"

    a2av1 "github.com/a2aproject/sage-a2a-go/gen/a2a/v1"
    "github.com/sage-x-project/sage/pkg/agent/transport"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
)

// Mock server for testing
type mockHandler struct{}

func (m *mockHandler) HandleSecureMessage(ctx context.Context, req *a2av1.SecureMessageRequest) (*a2av1.SecureMessageResponse, error) {
    return &a2av1.SecureMessageResponse{
        Success:   true,
        MessageId: req.MessageId,
        TaskId:    req.TaskId,
        Data:      []byte("test response"),
    }, nil
}

func TestA2ATransport_Send(t *testing.T) {
    // Setup in-memory gRPC server
    lis := bufconn.Listen(1024 * 1024)
    server := grpc.NewServer()
    a2av1.RegisterA2AMessagingServer(server, NewServer(&mockHandler{}))

    go server.Serve(lis)
    defer server.Stop()

    // Create client
    conn, err := grpc.DialContext(context.Background(), "",
        grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
            return lis.Dial()
        }),
        grpc.WithInsecure(),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer conn.Close()

    trans := &A2ATransport{
        client: a2av1.NewA2AMessagingClient(conn),
        conn:   conn,
    }

    // Test Send
    msg := &transport.SecureMessage{
        ID:        "test-msg-123",
        ContextID: "ctx-456",
        TaskID:    "task-789",
        Payload:   []byte("encrypted payload"),
        DID:       "did:sage:ethereum:0x123",
        Signature: []byte("signature"),
        Metadata:  map[string]string{"key": "value"},
        Role:      "user",
    }

    resp, err := trans.Send(context.Background(), msg)
    if err != nil {
        t.Fatalf("Send failed: %v", err)
    }

    if !resp.Success {
        t.Error("Expected success=true")
    }
    if resp.MessageID != msg.ID {
        t.Errorf("Expected message_id=%s, got %s", msg.ID, resp.MessageID)
    }
    if string(resp.Data) != "test response" {
        t.Errorf("Expected data='test response', got %s", string(resp.Data))
    }
}
```

### Integration Tests

Create `sage-a2a-go/tests/integration/transport_test.go`:

```go
package integration

import (
    "context"
    "testing"

    _ "github.com/a2aproject/sage-a2a-go/pkg/transport/grpc"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestGRPCTransport_Integration(t *testing.T) {
    // Start test server
    go startTestServer(t, ":50051")

    // Create transport via SAGE selector
    trans, err := transport.SelectByURL("grpc://localhost:50051")
    if err != nil {
        t.Fatalf("Failed to create transport: %v", err)
    }

    // Test message sending
    msg := &transport.SecureMessage{
        ID:      "integration-test-1",
        Payload: []byte("test payload"),
        DID:     "did:sage:test:123",
    }

    resp, err := trans.Send(context.Background(), msg)
    if err != nil {
        t.Fatalf("Send failed: %v", err)
    }

    if !resp.Success {
        t.Error("Expected successful response")
    }
}
```

## Error Handling

### Common Errors and Solutions

**1. Transport not registered**
```
Error: transport type "grpc" not registered (missing import or build tag?)
```
**Solution:** Import the grpc package to trigger auto-registration:
```go
import _ "github.com/a2aproject/sage-a2a-go/pkg/transport/grpc"
```

**2. Connection refused**
```
Error: failed to connect to localhost:50051: connection refused
```
**Solution:** Ensure gRPC server is running before creating client.

**3. Invalid payload**
```
Error: rpc error: code = InvalidArgument desc = payload is required
```
**Solution:** Verify SecureMessage.Payload is not empty.

## Performance Considerations

### Connection Pooling

```go
// Reuse transport connections
var (
    transportCache = make(map[string]transport.MessageTransport)
    cacheMu        sync.RWMutex
)

func GetOrCreateTransport(endpoint string) (transport.MessageTransport, error) {
    cacheMu.RLock()
    if trans, ok := transportCache[endpoint]; ok {
        cacheMu.RUnlock()
        return trans, nil
    }
    cacheMu.RUnlock()

    cacheMu.Lock()
    defer cacheMu.Unlock()

    // Double-check after acquiring write lock
    if trans, ok := transportCache[endpoint]; ok {
        return trans, nil
    }

    trans, err := transport.SelectByURL(endpoint)
    if err != nil {
        return nil, err
    }

    transportCache[endpoint] = trans
    return trans, nil
}
```

### gRPC Options

```go
// Optimized gRPC client options
func NewA2ATransport(endpoint string) (transport.MessageTransport, error) {
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(10 * 1024 * 1024), // 10MB
            grpc.MaxCallSendMsgSize(10 * 1024 * 1024), // 10MB
        ),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             3 * time.Second,
            PermitWithoutStream: true,
        }),
    }

    conn, err := grpc.Dial(endpoint, opts...)
    if err != nil {
        return nil, err
    }

    return &A2ATransport{
        client: a2av1.NewA2AMessagingClient(conn),
        conn:   conn,
    }, nil
}
```

## Security Considerations

### TLS Configuration

```go
import (
    "crypto/tls"
    "google.golang.org/grpc/credentials"
)

func NewA2ATransportTLS(endpoint string, certFile string) (transport.MessageTransport, error) {
    creds, err := credentials.NewClientTLSFromFile(certFile, "")
    if err != nil {
        return nil, err
    }

    conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
    if err != nil {
        return nil, err
    }

    return &A2ATransport{
        client: a2av1.NewA2AMessagingClient(conn),
        conn:   conn,
    }, nil
}
```

### Authentication

```go
// Add authentication metadata
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Add authentication token to context
    ctx = metadata.AppendToOutgoingContext(ctx,
        "authorization", "Bearer "+msg.DID,
        "x-message-signature", base64.StdEncoding.EncodeToString(msg.Signature),
    )

    // ... rest of Send implementation
}
```

## Project Structure

Recommended `sage-a2a-go` directory structure:

```
sage-a2a-go/
├── proto/
│   └── a2a/
│       └── messaging.proto          # Protobuf definitions
├── gen/
│   └── a2a/
│       └── v1/
│           ├── messaging.pb.go      # Generated protobuf code
│           └── messaging_grpc.pb.go # Generated gRPC code
├── pkg/
│   └── transport/
│       └── grpc/
│           ├── register.go          # Auto-registration
│           ├── client.go            # Transport adapter
│           ├── server.go            # gRPC server
│           └── client_test.go       # Tests
├── examples/
│   ├── client/
│   │   └── main.go                  # Example client
│   └── server/
│       └── main.go                  # Example server
├── tests/
│   └── integration/
│       └── transport_test.go        # Integration tests
├── Makefile                         # Build automation
├── go.mod
└── README.md
```

## Dependencies

Add to `sage-a2a-go/go.mod`:

```go
module github.com/a2aproject/sage-a2a-go

go 1.22

require (
    github.com/sage-x-project/sage v1.3.0
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
)
```

## Build Instructions

### Makefile

```makefile
.PHONY: all
all: generate test build

.PHONY: generate
generate:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/a2a/messaging.proto

.PHONY: test
test:
	go test -v ./...

.PHONY: test-integration
test-integration:
	go test -v -tags=integration ./tests/integration/...

.PHONY: build
build:
	go build -o bin/sage-a2a-client ./examples/client
	go build -o bin/sage-a2a-server ./examples/server

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf gen/
```

## Example Applications

### Server Example

`examples/server/main.go`:

```go
package main

import (
    "context"
    "log"

    a2av1 "github.com/a2aproject/sage-a2a-go/gen/a2a/v1"
    grpctransport "github.com/a2aproject/sage-a2a-go/pkg/transport/grpc"
)

type handler struct {
    // Add SAGE components here
}

func (h *handler) HandleSecureMessage(ctx context.Context, req *a2av1.SecureMessageRequest) (*a2av1.SecureMessageResponse, error) {
    log.Printf("Received message: %s from %s", req.MessageId, req.SenderDid)

    // Process with SAGE security layer
    // decrypt, verify signature, handle message

    return &a2av1.SecureMessageResponse{
        Success:   true,
        MessageId: req.MessageId,
        TaskId:    req.TaskId,
        Data:      []byte("processed response"),
    }, nil
}

func main() {
    h := &handler{}

    log.Println("Starting A2A gRPC server on :50051")
    if err := grpctransport.StartServer(":50051", h); err != nil {
        log.Fatal(err)
    }
}
```

### Client Example

`examples/client/main.go`:

```go
package main

import (
    "context"
    "log"

    _ "github.com/a2aproject/sage-a2a-go/pkg/transport/grpc"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func main() {
    // Create transport via SAGE selector
    trans, err := transport.SelectByURL("grpc://localhost:50051")
    if err != nil {
        log.Fatal(err)
    }

    // Send message
    msg := &transport.SecureMessage{
        ID:        "example-msg-1",
        ContextID: "ctx-1",
        TaskID:    "task-1",
        Payload:   []byte("encrypted payload"),
        DID:       "did:sage:ethereum:0x123",
        Signature: []byte("signature bytes"),
        Role:      "user",
    }

    resp, err := trans.Send(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Response: success=%v, data=%s", resp.Success, string(resp.Data))
}
```

## Verification Checklist

Before marking the implementation complete, verify:

- [ ] Protobuf definitions match SAGE's SecureMessage/Response types
- [ ] gRPC server implements A2AMessaging service
- [ ] Client implements transport.MessageTransport interface
- [ ] Transport factory registered with SAGE selector
- [ ] `grpc://` URLs work with transport.SelectByURL()
- [ ] Unit tests cover message conversion
- [ ] Integration tests verify end-to-end communication
- [ ] Error handling for all failure cases
- [ ] TLS support (optional but recommended)
- [ ] Documentation and examples provided

## References

### SAGE Documentation
- [Transport Layer README](../../pkg/agent/transport/README.md)
- [Transport Interface](../../pkg/agent/transport/interface.go)
- [HTTP Transport Implementation](../../pkg/agent/transport/http/) (reference example)
- [WebSocket Transport Implementation](../../pkg/agent/transport/websocket/) (reference example)

### gRPC Documentation
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Guide](https://protobuf.dev/programming-guides/proto3/)
- [gRPC Performance Best Practices](https://grpc.io/docs/guides/performance/)

### A2A Protocol
- [A2A Protocol Specification](https://github.com/a2aproject/a2a)
- [A2A Reference Implementation](https://github.com/a2aproject/a2a-go)

## Support

For questions or issues:
- SAGE Issues: https://github.com/sage-x-project/sage/issues
- sage-a2a-go Issues: https://github.com/a2aproject/sage-a2a-go/issues
- A2A Protocol Discussion: https://github.com/a2aproject/a2a/discussions

---

**Document Version:** 1.0
**SAGE Compatibility:** v1.3.0+
**Last Reviewed:** 2025-10-26

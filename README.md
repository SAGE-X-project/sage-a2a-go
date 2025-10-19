# SAGE-A2A-GO

**DID-Authenticated HTTP Transport for A2A Protocol**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![A2A Protocol](https://img.shields.io/badge/A2A-v0.3.0-green)](https://a2a-protocol.org)
[![SAGE Version](https://img.shields.io/badge/SAGE-v1.1.0-blue)](https://github.com/sage-x-project/sage)
[![License](https://img.shields.io/badge/license-LGPL--3.0-blue.svg)](LICENSE)

## Overview

`sage-a2a-go` provides **DID-authenticated HTTP/JSON-RPC 2.0 transport** for [a2a-go](https://github.com/a2aproject/a2a-go), enabling secure agent-to-agent communication with blockchain-anchored identity.

### What It Does

- ✅ **HTTP/JSON-RPC 2.0 Transport**: Required by A2A spec (a2a-go only has gRPC)
- ✅ **Automatic DID Signing**: All HTTP requests signed with RFC 9421
- ✅ **Blockchain Identity**: SAGE DIDs anchored on Ethereum/Solana/Kaia
- ✅ **Drop-in for a2a-go**: Use standard a2a-go Client with DID auth
- ✅ **Zero Code Duplication**: Wraps a2a-go, doesn't reimplement

### Why You Need This

| Feature | a2a-go | sage-a2a-go |
|---------|--------|-------------|
| A2A Client SDK | ✅ | ✅ (uses a2a-go) |
| gRPC Transport | ✅ | ✅ (from a2a-go) |
| HTTP/JSON-RPC Transport | ❌ | ✅ |
| DID Authentication | ❌ | ✅ |
| RFC 9421 Signatures | ❌ | ✅ |
| Blockchain Identity | ❌ | ✅ |

**sage-a2a-go = a2a-go + HTTP Transport + DID Auth**

## Architecture

```
┌──────────────────────────────────────────────────┐
│          Your Application                        │
│  (uses a2aclient.Client from a2a-go)             │
└──────────────────┬───────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────┐
│          a2a-go Client                           │
│  - GetTask, SendMessage, etc.                    │
│  - CallInterceptors                              │
│  - Config management                             │
└──────────┬──────────────────┬────────────────────┘
           │                  │
    ┌──────▼──────┐    ┌──────▼──────────────────┐
    │ gRPC        │    │ DIDHTTPTransport        │
    │ Transport   │    │ (sage-a2a-go)           │
    │ (a2a-go)    │    │ - HTTP/JSON-RPC 2.0     │
    └─────────────┘    │ - DID Signatures        │
                       │ - RFC 9421              │
                       └──────┬──────────────────┘
                              │
                       ┌──────▼──────────────────┐
                       │  SAGE DID Auth          │
                       │  - DIDVerifier          │
                       │  - A2ASigner            │
                       │  - KeySelector          │
                       └─────────────────────────┘
```

## Installation

```bash
go get github.com/sage-x-project/sage-a2a-go
go get github.com/a2aproject/a2a-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/a2aproject/a2a-go/a2a"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    ctx := context.Background()

    // Your agent's identity
    myDID := did.AgentDID("did:sage:ethereum:0x...")
    myKeyPair, _ := crypto.GenerateSecp256k1KeyPair()

    // Target agent's card
    targetCard := &a2a.AgentCard{
        Name: "Assistant Agent",
        URL:  "https://agent.example.com",
        // ...
    }

    // Create client with DID-authenticated HTTP transport
    client, err := transport.NewDIDAuthenticatedClient(
        ctx,
        myDID,
        myKeyPair,
        targetCard,
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Destroy()

    // Send message (automatically signed with DID)
    message := &a2a.MessageSendParams{
        Message: &a2a.Message{
            Role: a2a.RoleUser,
            Parts: []a2a.Part{
                &a2a.TextPart{Text: "Hello!"},
            },
        },
    }

    task, err := client.SendMessage(ctx, message)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Task: %+v", task)
}
```

### Advanced Usage with a2a-go Factory

```go
import (
    "github.com/a2aproject/a2a-go/a2aclient"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
)

// Use a2a-go's factory with DID HTTP transport
client, err := a2aclient.NewFromCard(
    ctx,
    agentCard,
    transport.WithDIDHTTPTransport(myDID, myKeyPair, nil),
    a2aclient.WithConfig(a2aclient.Config{
        AcceptedOutputModes: []string{"application/json"},
    }),
    a2aclient.WithInterceptors(loggingInterceptor),
)
```

### All A2A Protocol Methods Work

```go
// Task management
task, err := client.GetTask(ctx, &a2a.TaskQueryParams{ID: "task-123"})
task, err := client.CancelTask(ctx, &a2a.TaskIDParams{ID: "task-123"})

// Messaging
result, err := client.SendMessage(ctx, messageParams)

// Streaming (via Server-Sent Events)
for event, err := range client.SendStreamingMessage(ctx, messageParams) {
    // handle event
}

// Push notifications
config, err := client.GetTaskPushConfig(ctx, params)
configs, err := client.ListTaskPushConfig(ctx, params)
config, err := client.SetTaskPushConfig(ctx, config)
err := client.DeleteTaskPushConfig(ctx, params)

// Agent discovery
card, err := client.GetAgentCard(ctx)
```

## How It Works

### DID HTTP Transport

sage-a2a-go implements the `a2aclient.Transport` interface:

```go
type Transport interface {
    GetTask(ctx, query) (*Task, error)
    CancelTask(ctx, id) (*Task, error)
    SendMessage(ctx, msg) (SendMessageResult, error)
    SendStreamingMessage(ctx, msg) iter.Seq2[Event, error]
    // ... all A2A methods
}
```

**DIDHTTPTransport** adds:
1. HTTP/JSON-RPC 2.0 protocol support
2. Automatic DID signature on every request
3. RFC 9421 HTTP Message Signatures

### Request Flow

```
1. Client calls: client.SendMessage(ctx, message)
   ↓
2. a2a-go calls: transport.SendMessage(ctx, message)
   ↓
3. DIDHTTPTransport:
   - Marshals to JSON-RPC 2.0 request
   - Creates HTTP POST to /rpc endpoint
   - Signs with DID (RFC 9421)
   - Sends HTTP request
   - Parses JSON-RPC 2.0 response
   ↓
4. Returns result to client
```

### DID Signature Example

Every HTTP request includes:

```http
POST /rpc HTTP/1.1
Host: agent.example.com
Content-Type: application/json
Signature-Input: sig1=("@method" "@target-uri" "content-type");created=1234567890;keyid="did:sage:ethereum:0x...";alg="ecdsa-p256-sha256"
Signature: sig1=:base64signature:

{"jsonrpc":"2.0","method":"message/send","params":{...},"id":1}
```

## Components

### 1. DID HTTP Transport (`pkg/transport/`)

Implements HTTP/JSON-RPC 2.0 with DID signatures:
- `DIDHTTPTransport` - Main transport implementation
- `WithDIDHTTPTransport()` - Factory option for a2a-go
- `NewDIDAuthenticatedClient()` - Convenience function

### 2. DID Verification (`pkg/verifier/`)

**Existing components** (preserved from earlier work):
- `DIDVerifier` - Verify HTTP signatures using DIDs
- `KeySelector` - Protocol-aware key selection
- `RFC9421Verifier` - RFC 9421 implementation

### 3. HTTP Signing (`pkg/signer/`)

**Existing components** (preserved):
- `A2ASigner` - Sign HTTP requests with DID
- `DefaultA2ASigner` - RFC 9421 implementation

### 4. Agent Cards (`pkg/protocol/`)

**Existing components** (preserved):
- `AgentCard` - Agent metadata
- `AgentCardSigner` - Sign/verify cards with JWS

## Features

### ✅ Complete A2A Protocol Support

All 10 client methods from A2A v0.3.0 specification.

### ✅ Blockchain-Anchored Identity

DIDs stored on:
- Ethereum (ECDSA/secp256k1)
- Solana (Ed25519)
- Kaia

### ✅ RFC 9421 HTTP Signatures

- Signature-Input header
- Signature header
- DID as keyid parameter
- Timestamp for replay protection

### ✅ Multi-Key Support

- Protocol-aware key selection
- Up to 10 keys per agent
- Key rotation support

### ✅ Zero Code Duplication

- Wraps a2a-go instead of reimplementing
- Automatic updates when a2a-go updates
- Clean separation of concerns

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design and architecture
- **[Integration Guide](docs/INTEGRATION_GUIDE.md)** - Complete integration guide
- **[Design Documentation](docs/design.md)** - Technical design
- **[API Reference](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go)** - GoDoc

## Examples

Complete examples in [`cmd/examples/`](cmd/examples/):

### 1. Simple Agent
```bash
go run ./cmd/examples/simple-agent/main.go
```

### 2. Agent Communication
```bash
go run ./cmd/examples/agent-communication/main.go
```

### 3. Multi-Key Agent
```bash
go run ./cmd/examples/multi-key-agent/main.go
```

## Development

### Building
```bash
go build ./...
```

### Testing
```bash
go test ./...
```

### Coverage
```bash
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Version Management

sage-a2a-go tracks versions of its dependencies:

```go
import "github.com/sage-x-project/sage-a2a-go"

info := sagea2a.GetVersionInfo()
// info.A2AProtocolVersion = "0.3.0"
// info.SAGEVersion = "1.1.0"
```

### Updating a2a-go

When a2a-go updates:
```bash
go get -u github.com/a2aproject/a2a-go
go mod tidy
go test ./...
```

## Roadmap

### Current (v2.0.0)
- ✅ HTTP/JSON-RPC 2.0 transport
- ✅ DID signatures (RFC 9421)
- ✅ All A2A v0.3.0 methods (except streaming)

### Next (v2.1.0)
- [ ] Server-Sent Events (SSE) for streaming
- [ ] A2A v0.4.0 support (ListTasks)
- [ ] Performance optimizations

### Future (v3.0.0)
- [ ] WebSocket transport
- [ ] HTTP/2 support
- [ ] Metrics and observability

## Contributing

1. Fork the repository
2. Create feature branch
3. Write tests (TDD approach)
4. Implement feature
5. Submit Pull Request

### Commit Format
```
<type>: <subject>

Types: feat, fix, test, refactor, docs
Example: feat: add SSE streaming support
```

## Security

### Threat Mitigations
- **DID Spoofing**: Prevented by blockchain anchoring
- **Replay Attacks**: Timestamp validation
- **MITM**: TLS + HTTP signatures
- **Key Compromise**: Multi-key rotation

### Best Practices
- Use HTTPS/TLS 1.2+
- Validate signature timestamps
- Rotate keys regularly
- Use mTLS in production

## License

LGPL-3.0 - see [LICENSE](LICENSE)

**Dependencies**:
- a2a-go (Apache 2.0)
- SAGE (LGPL-3.0)

## References

- [A2A Protocol](https://a2a-protocol.org) - v0.3.0 specification
- [a2a-go](https://github.com/a2aproject/a2a-go) - Go SDK
- [SAGE](https://github.com/sage-x-project/sage) - DID infrastructure
- [RFC 9421](https://www.rfc-editor.org/rfc/rfc9421.html) - HTTP Signatures
- [DID Core](https://www.w3.org/TR/did-core/) - W3C Specification

## Support

- **GitHub Issues**: [Report bugs](https://github.com/sage-x-project/sage-a2a-go/issues)
- **Documentation**: [Complete guides](docs/)
- **Examples**: [`cmd/examples/`](cmd/examples/)

---

**sage-a2a-go**: Bringing DID authentication to A2A Protocol 🔐

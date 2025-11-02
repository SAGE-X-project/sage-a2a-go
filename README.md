# SAGE-A2A-GO

**DID-Authenticated HTTP Transport for A2A Protocol**

[![CI](https://github.com/sage-x-project/sage-a2a-go/workflows/CI/badge.svg)](https://github.com/sage-x-project/sage-a2a-go/actions)
[![codecov](https://codecov.io/gh/sage-x-project/sage-a2a-go/branch/main/graph/badge.svg)](https://codecov.io/gh/sage-x-project/sage-a2a-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/sage-x-project/sage-a2a-go)](https://goreportcard.com/report/github.com/sage-x-project/sage-a2a-go)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![A2A Protocol](https://img.shields.io/badge/A2A-v0.4.0-green)](https://a2a-protocol.org)
[![SAGE Version](https://img.shields.io/badge/SAGE-v1.5.2-blue)](https://github.com/sage-x-project/sage)
[![Release](https://img.shields.io/badge/release-v1.6.0-success)](https://github.com/SAGE-X-project/sage-a2a-go/releases/tag/v1.6.0)
[![License](https://img.shields.io/badge/license-LGPL--3.0-blue.svg)](LICENSE)

## Overview

`sage-a2a-go` provides **DID-authenticated HTTP/JSON-RPC 2.0 transport** for [a2a-go](https://github.com/a2aproject/a2a-go), enabling secure agent-to-agent communication with blockchain-anchored identity.

> **Note**: This project uses a [SAGE-X fork of a2a-go](https://github.com/SAGE-X-project/a2a-go) with critical bug fixes for Message Parts marshaling. See [Bug Fix](#-critical-bug-fix) section below.

### What It Does

- ✅ **HTTP/JSON-RPC 2.0 Transport**: Required by A2A spec (a2a-go only has gRPC)
- ✅ **Automatic DID Signing**: All HTTP requests signed with RFC 9421
- ✅ **Blockchain Identity**: SAGE DIDs anchored on Ethereum/Solana/Kaia
- ✅ **Drop-in for a2a-go**: Use standard a2a-go Client with DID auth
- ✅ **Zero Code Duplication**: Wraps a2a-go, doesn't reimplement
- ✅ **Bug Fixes**: Includes critical Message Parts marshaling fix

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

## Key Features

### 🚀 High-Level Agent Framework (NEW in v1.7.0)

Build SAGE protocol agents with **83% less code**:

```go
// Just 10 lines replaces 165 lines of boilerplate
agent, err := framework.NewAgentFromEnv(
    "payment", "PAYMENT", true, true,
)
```

**Benefits:**
- ✅ **Zero direct sage imports** - Framework handles all complexity
- ✅ **Environment-based config** - Easy deployment
- ✅ **Built-in HPKE** - Automatic encryption
- ✅ **DID authentication** - Signature verification included
- ✅ **Pure business logic** - Focus on what matters

👉 [Learn more about the Agent Framework](pkg/agent/framework/README.md)
👉 [See examples](examples/framework/)

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
```

The project automatically uses the [SAGE-X fork of a2a-go](https://github.com/SAGE-X-project/a2a-go) via `replace` directive in `go.mod`:

```go
// Use SAGE-X fork with bug fixes
replace github.com/a2aproject/a2a-go => github.com/SAGE-X-project/a2a-go v0.0.0-20251026124015-70634d9eddae
```

This ensures you get critical bug fixes for Message Parts marshaling automatically.

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/a2aproject/a2a-go/a2a"
    "github.com/sage-x-project/sage-a2a-go/pkg/agent"
    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
)

func main() {
    ctx := context.Background()

    // Generate key pair using sage-a2a-go crypto package
    myKeyPair, _ := crypto.GenerateSecp256k1KeyPair()

    // Your agent's identity
    myDID := identity.AgentDID("did:sage:ethereum:0x...")

    // Target agent's card
    targetCard := &a2a.AgentCard{
        Name: "Assistant Agent",
        URL:  "https://agent.example.com",
        // ...
    }

    // Create agent with DID authentication (unified API)
    myAgent, err := agent.NewAgent(ctx, myDID, myKeyPair, targetCard)
    if err != nil {
        log.Fatal(err)
    }
    defer myAgent.A2AClient.Destroy()

    // Send message (automatically signed with DID)
    message := &a2a.MessageSendParams{
        Message: &a2a.Message{
            Role: a2a.RoleUser,
            Parts: []a2a.Part{
                &a2a.TextPart{Text: "Hello!"},
            },
        },
    }

    task, err := myAgent.A2AClient.SendMessage(ctx, message)
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
    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
)

// Generate key pair
myKeyPair, _ := crypto.GenerateSecp256k1KeyPair()
myDID := identity.AgentDID("did:sage:ethereum:0x...")

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

## 🔧 Critical Bug Fix

### Message Parts Marshaling Issue

The official a2a-go library had a critical bug in message unmarshaling where `Message.Parts` would unmarshal as **value types** instead of **pointer types**, causing message transmission failures in agent-to-agent communication.

**Problem**:
```go
// After unmarshaling with official a2a-go
msg.Parts[0]  // Type: a2a.TextPart (value) ❌
```

**Solution**:
The [SAGE-X fork](https://github.com/SAGE-X-project/a2a-go) fixes this in `a2a/core.go:304-332`:

```go
// Fixed UnmarshalJSON implementation
case "text":
    var part TextPart
    if err := json.Unmarshal(rawMsg, &part); err != nil {
        return err
    }
    result[i] = &part  // Return pointer type ✅
```

**Impact**:
- ✅ Messages now transmit correctly between agents
- ✅ All 173 tests passing with strict pointer type validation
- ✅ E2E tests verify correct behavior

This project automatically uses the fixed fork, so you don't need to worry about this issue.

## Components

### Unified API Packages

sage-a2a-go provides a complete unified API so you **only need to import sage-a2a-go packages**:

#### 1. **pkg/crypto/** - Cryptographic Operations (v1.5.2)
Wraps SAGE crypto functionality with a simple interface:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"

// Generate key pairs
keyPair, _ := crypto.GenerateSecp256k1KeyPair()  // Ethereum
keyPair, _ := crypto.GenerateEd25519KeyPair()    // Solana
keyPair, _ := crypto.GenerateX25519KeyPair()     // HPKE/encryption

// Key types
crypto.KeyTypeSecp256k1
crypto.KeyTypeEd25519
crypto.KeyTypeX25519
```

#### 2. **pkg/identity/** - DID Management (v1.5.2)
Wraps SAGE DID functionality:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/identity"

// Work with DIDs
did := identity.AgentDID("did:sage:ethereum:0x...")
chain, address, _ := identity.ParseDID(did)

// Validate DIDs
err := identity.ValidateDID(string(did))

// Marshal/unmarshal public keys
keyData, _ := identity.MarshalPublicKey(pubKey)
pubKey, _ := identity.UnmarshalPublicKey(keyData, "secp256k1")
```

#### 3. **pkg/agent/** - High-Level Agent Builder (v1.5.2)
Combines SAGE identity with A2A communication:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent"

// Build complete agent with one function
agent, err := agent.NewAgent(ctx, myDID, myKeyPair, targetCard)

// Access A2A client
task, _ := agent.A2AClient.SendMessage(ctx, message)

// Agent structure
type Agent struct {
    DID       identity.AgentDID
    KeyPair   crypto.KeyPair
    A2AClient *a2aclient.Client
    Card      *a2a.AgentCard
}
```

#### 4. **pkg/hpke/** - HPKE Client Wrapper (v1.6.0)
Simplified HPKE (Hybrid Public Key Encryption) client for end-to-end encryption:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/hpke"

// Create HPKE client
hpkeClient, err := hpke.NewClient(
    clientDID,
    keyPair,
    transport,
    resolver,
    sessionMgr,
    nil, // optional ClientOptions
)

// Initialize encrypted session
sessionID, err := hpkeClient.InitializeSession(ctx, "context-123", peerDID)

// Access session manager
sessionMgr := hpkeClient.GetSessionManager()
```

#### 5. **pkg/registry/** - Registry Client Wrapper (v1.6.0)
Three-phase registration wrapper for agent registration:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/registry"

// Create registration client
client, err := registry.NewRegistrationClient(&registry.ClientConfig{
    RPCURL:          "https://ethereum-rpc.example.com",
    RegistryAddress: "0x1234...",
    PrivateKey:      "your-private-key",
})

// Phase 1: Commit (wait 1-60 minutes)
status, err := client.CommitRegistration(ctx, params)

// Phase 2: Register (wait 1 hour)
status, err = client.RegisterAgent(ctx, status)

// Phase 3: Activate
err = client.ActivateAgent(ctx, status)
```

#### 6. **pkg/session/** - Enhanced Session Manager (v1.6.0)
Advanced session management for multi-peer scenarios:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/session"

sessionMgr := session.NewManager()

// List sessions by peer DID
sessions := sessionMgr.ListByRemoteDID(ctx, peerDID)

// Get session count
count := sessionMgr.Count()

// Session metadata
err := sessionMgr.SetMetadata(ctx, sessionID, "request_id", "req-123")
value, err := sessionMgr.GetMetadata(ctx, sessionID, "request_id")

// Cleanup sessions
deleted := sessionMgr.DeleteByRemoteDID(ctx, peerDID)
```

**Philosophy**: Users should build agents using **only** sage-a2a-go imports. No need to directly import SAGE or A2A packages.

### Core Transport and Security

#### 7. **pkg/transport/** - DID HTTP Transport
Implements HTTP/JSON-RPC 2.0 with DID signatures:
- `DIDHTTPTransport` - Main transport implementation
- `WithDIDHTTPTransport()` - Factory option for a2a-go
- `NewDIDAuthenticatedClient()` - Convenience function

#### 8. **pkg/verifier/** - DID Verification
Verify HTTP signatures using DIDs:
- `DIDVerifier` - Verify HTTP signatures using DIDs
- `KeySelector` - Protocol-aware key selection (Ethereum/Solana/HPKE)
- `RFC9421Verifier` - RFC 9421 implementation

#### 9. **pkg/signer/** - HTTP Signing
Sign HTTP requests with DID:
- `A2ASigner` - Sign HTTP requests with DID
- `DefaultA2ASigner` - RFC 9421 implementation with security hardening

#### 10. **pkg/server/** - Server Middleware
DID authentication for HTTP servers:
- `DIDAuthMiddleware` - Middleware for verifying incoming requests
- Extracts and validates DID signatures
- Adds verified DID to request context

#### 11. **pkg/protocol/** - Agent Cards
Agent metadata and verification:
- `AgentCard` - Agent metadata
- `AgentCardSigner` - Sign/verify cards with JWS

## Features

### ✅ Complete A2A Protocol Support

All 10 client methods from A2A v0.4.0 specification with **92.1% test coverage** and comprehensive E2E tests.

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

## Testing

### Test Coverage

The project maintains **92.1% average test coverage** across all packages:

| Package | Coverage | Tests |
|---------|----------|-------|
| `pkg/server` | 100.0% | 🏆 Full coverage |
| `pkg/session` | 100.0% | 🏆 Full coverage (20 tests) |
| `pkg/client` | 92.3% | Unit + integration |
| `pkg/signer` | 92.2% | HTTP signing tests |
| `pkg/protocol` | 91.2% | Card validation |
| `pkg/verifier` | 88.0% | DID verification |
| `pkg/transport` | 87.2% | HTTP transport |
| `pkg/hpke` | 85.0% | HPKE client (9 tests) |
| `pkg/registry` | 83.0% | Registry client (6 tests) |
| **Total** | **92.1%** | **208 tests** |

### End-to-End Tests

Comprehensive E2E tests in `test/e2e/` verify real-world scenarios:

```bash
# Run E2E tests
go test ./test/e2e/... -v
```

**Test Coverage**:
- ✅ Full HTTP request/response cycle
- ✅ DID signature verification
- ✅ SSE streaming with multiple messages
- ✅ Task operations (get, list, cancel)
- ✅ Timeout handling
- ✅ Error propagation
- ✅ Message Parts pointer type validation

**9 test cases** covering:
1. `SendMessage_Success` - Complete message flow
2. `GetAgentCard_Success` - Agent card retrieval
3. `Timeout_HandledCorrectly` - Timeout scenarios
4. `StreamMessage_Success` - SSE streaming with 3 messages
5. `GetTask_Success` - Task retrieval
6. `ListTasks_Success` - Task listing with pagination
7. `CancelTask_Success` - Task cancellation

### Running Tests

```bash
# All tests
make test-all

# Unit tests only
make test

# With coverage report
make test-coverage

# E2E tests
go test ./test/e2e/... -v
```

## Documentation

### 📖 Getting Started
- **[Integration Guide](docs/INTEGRATION_GUIDE.md)** - Complete integration tutorial
- **[SSE Streaming Guide](docs/SSE_STREAMING_GUIDE.md)** - Real-time streaming with SSE
- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation

### 🏗️ Architecture & Design
- **[Architecture & Design](docs/ARCHITECTURE.md)** - Comprehensive system architecture, design principles, and implementation details
- **[GoDoc](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go)** - Generated API documentation

### 📋 Project Information
- **[Changelog](CHANGELOG.md)** - Version history and upgrade guide

## Examples

Complete examples in [`cmd/examples/`](cmd/examples/):

### 1. Simple Client ([README](cmd/examples/simple-client/))
Basic DID-authenticated A2A client.
```bash
go run ./cmd/examples/simple-client/main.go
```

### 2. Simple Agent ([README](cmd/examples/simple-agent/))
Create an agent with DID authentication.
```bash
go run ./cmd/examples/simple-agent/main.go
```

### 3. Chat Demo ([README](cmd/examples/chat-demo/))
Interactive chat application with SSE streaming.
```bash
go run ./cmd/examples/chat-demo/main.go
```

### 4. SSE Streaming ([README](cmd/examples/sse-streaming/))
Real-time message streaming with Server-Sent Events.
```bash
go run ./cmd/examples/sse-streaming/main.go
```

### 5. Agent Communication ([README](cmd/examples/agent-communication/))
Agent-to-agent communication example.
```bash
go run ./cmd/examples/agent-communication/main.go
```

### 6. Multi-Key Agent ([README](cmd/examples/multi-key-agent/))
Multi-protocol key management.
```bash
go run ./cmd/examples/multi-key-agent/main.go
```

## Development

This project uses **Make** for build automation. Run `make help` to see all available commands.

### Quick Start

```bash
# Display all available commands
make help

# Build the library
make build

# Run tests
make test

# Run tests with coverage report
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Quick development cycle (fmt + vet + test)
make dev
```

### Building

```bash
# Build library
make build

# Build example programs
make build-examples

# Install library locally
make install
```

### Testing

```bash
# Run unit tests
make test

# Run tests with verbose output
make test-verbose

# Generate coverage report (HTML)
make test-coverage

# Run integration tests
make test-integration

# Run all tests (unit + integration)
make test-all

# Run benchmarks
make bench
```

### Code Quality

```bash
# Format code
make fmt

# Check formatting
make fmt-check

# Run go vet
make vet

# Run linter
make lint

# Auto-fix linter issues
make lint-fix

# Run all quality checks
make check
```

### Dependencies

```bash
# Download dependencies
make deps

# Tidy go.mod and go.sum
make tidy

# Verify dependencies
make verify

# Update all dependencies
make deps-update
```

### Cleanup

```bash
# Clean build artifacts and test cache
make clean

# Clean only build artifacts
make clean-build

# Clean coverage reports
make clean-coverage
```

### CI/CD

```bash
# Run CI checks (format, vet, lint, test)
make ci

# Run full CI suite with coverage
make ci-full

# Run pre-commit checks
make pre-commit
```

### Manual Commands

If you prefer not to use Make:

```bash
# Build
go build ./...

# Test
go test ./...

# Coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Version Management

sage-a2a-go tracks versions of its dependencies:

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/version"

info := version.Get()
// info.SageA2AVersion = "1.0.0-dev"
// info.A2AProtocolVersion = "0.4.0"
// info.SAGEVersion = "1.5.2"
// info.A2AGoForkVersion = "v0.0.0-20251026124015-70634d9eddae"
```

### Updating Dependencies

When updating a2a-go (using SAGE-X fork):
```bash
# Update to latest fork version
go get -u github.com/SAGE-X-project/a2a-go
go mod tidy
go test ./...
```

The project uses the SAGE-X fork to ensure critical bug fixes are included. Monitor both repositories:
- **Official**: [a2aproject/a2a-go](https://github.com/a2aproject/a2a-go)
- **Fork (used)**: [SAGE-X-project/a2a-go](https://github.com/SAGE-X-project/a2a-go)

## Roadmap

### v1.6.0 (Current - Released 2025-11-02)
- ✅ HTTP/JSON-RPC 2.0 transport
- ✅ DID signatures (RFC 9421)
- ✅ A2A v0.4.0 protocol support
- ✅ Server-Sent Events (SSE) for streaming
- ✅ All core protocol methods (GetTask, SendMessage, ListTasks, etc.)
- ✅ DID authentication middleware for servers
- ✅ Unified API architecture (v1.5.2)
- ✅ HPKE Client wrapper (v1.6.0)
- ✅ Registry Client wrapper (v1.6.0)
- ✅ Enhanced Session Manager (v1.6.0)
- ✅ 92.1% test coverage (208 tests: Unit + Integration + E2E)
- ✅ 6 complete example programs
- ✅ Comprehensive documentation

### v1.7.0 (Planned)
- [ ] Enhanced error handling with custom error types
- [ ] Retry logic for network operations
- [ ] Connection pooling for HPKE sessions
- [ ] Metrics and observability hooks

### v2.0.0+ (Future)
- [ ] WebSocket transport
- [ ] HTTP/2 and HTTP/3 support
- [ ] Plugin architecture for custom transports
- [ ] GraphQL support for registry queries
- [ ] Multi-chain registry support (Solana, Polygon)
- [ ] WebAssembly compilation target
- [ ] Rate limiting and quota management

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

### Quick Start for Contributors

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests (TDD approach - maintain 90%+ coverage)
4. Implement feature
5. Run tests: `make test-all`
6. Run linter: `make lint`
7. Submit Pull Request

### Commit Format
```
<type>: <subject>

Types: feat, fix, test, refactor, docs, chore
Example: feat: add SSE streaming support
```

### Working with the a2a-go Fork

This project uses a SAGE-X fork of a2a-go. If you need to modify a2a-go:

1. Fork is at: https://github.com/SAGE-X-project/a2a-go
2. Local development: Use `replace` directive in `go.mod`
3. Submit fixes to the fork first
4. Update the fork version in this project

See [CONTRIBUTING.md](CONTRIBUTING.md) for full details.

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

- [A2A Protocol](https://a2a-protocol.org) - v0.4.0 specification
- [a2a-go (Official)](https://github.com/a2aproject/a2a-go) - Go SDK
- [a2a-go (SAGE-X Fork)](https://github.com/SAGE-X-project/a2a-go) - Fork with bug fixes (used by this project)
- [SAGE](https://github.com/sage-x-project/sage) - DID infrastructure
- [RFC 9421](https://www.rfc-editor.org/rfc/rfc9421.html) - HTTP Signatures
- [DID Core](https://www.w3.org/TR/did-core/) - W3C Specification

## Support

- **GitHub Issues**: [Report bugs](https://github.com/sage-x-project/sage-a2a-go/issues)
- **Documentation**: [Complete guides](docs/)
- **Examples**: [`cmd/examples/`](cmd/examples/)

---

**sage-a2a-go**: Bringing DID authentication to A2A Protocol 🔐

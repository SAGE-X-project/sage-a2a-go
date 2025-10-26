# sage-a2a-go Architecture

## Overview

sage-a2a-go provides **DID-authenticated HTTP/JSON-RPC 2.0 transport** for the A2A protocol by integrating:
- **A2A Protocol** (specification) - Agent-to-Agent protocol definitions
- **a2a-go** (SDK implementation) - Go SDK for A2A protocol
- **SAGE** (DID infrastructure) - Blockchain-anchored identity and cryptography

## Design Principles

### 1. Dependency-Based Architecture
```
sage-a2a-go (DID Auth Transport)
    ├─→ a2a-go (A2A SDK) - Direct dependency
    ├─→ SAGE (DID Core) - Direct dependency
    └─→ A2A Protocol - Indirect (via a2a-go)
```

**Benefits**:
- ✅ Automatic updates when a2a-go updates
- ✅ No code duplication
- ✅ Clean separation of concerns
- ✅ Easy to track A2A protocol versions

### 2. Transport Implementation Pattern
sage-a2a-go implements the `a2aclient.Transport` interface from a2a-go:

```go
// a2a-go defines the Transport interface
type Transport interface {
    GetTask(ctx, query) (*Task, error)
    CancelTask(ctx, id) (*Task, error)
    SendMessage(ctx, msg) (SendMessageResult, error)
    SendStreamingMessage(ctx, msg) iter.Seq2[Event, error]
    // ... all A2A protocol methods
}

// sage-a2a-go implements it with DID authentication
type DIDHTTPTransport struct {
    baseURL    string
    agentDID   did.AgentDID
    keyPair    crypto.KeyPair
    signer     signer.A2ASigner
    httpClient *http.Client
}
```

**Why Transport Implementation vs Wrapper?**
- ✅ More flexible - can change underlying transport protocol
- ✅ Cleaner integration with a2a-go factory system
- ✅ Automatic DID signing at HTTP layer (RFC 9421)
- ✅ Zero overhead - direct implementation, no wrapping

### 3. Factory-Based Integration
DID authentication is provided through a2a-go's factory system:

```go
// Factory option for a2a-go client
func WithDIDHTTPTransport(
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    httpClient *http.Client,
) a2aclient.FactoryOption

// Convenience function
func NewDIDAuthenticatedClient(
    ctx context.Context,
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    card *a2a.AgentCard,
) (*a2aclient.Client, error)
```

## Version Management Strategy

### A2A Protocol Version Tracking
```
sage-a2a-go/versions.go:
    const (
        Version = "2.0.0-alpha"
        A2AProtocolVersion = "0.3.0"  // Current supported version
        MinA2AProtocolVersion = "0.2.6"  // Minimum compatible version
        SAGEVersion = "1.1.0"
    )
```

### Dependency Version Pinning
```
go.mod:
    require (
        github.com/a2aproject/a2a-go v0.1.0
        github.com/sage-x-project/sage v1.1.0
    )

    replace github.com/a2aproject/a2a-go => ../a2a-go  // Local development
```

### Update Process
1. **A2A Protocol updates** (e.g., v0.3.0 → v0.4.0):
   - Wait for a2a-go to release compatible version
   - Update `go.mod` to new a2a-go version
   - Update `versions.go` constants
   - Implement new transport methods if added
   - Update tests and documentation

2. **a2a-go updates** (bug fixes, improvements):
   - Run `go get -u github.com/a2aproject/a2a-go`
   - Run tests to ensure compatibility
   - Update if needed

## Package Structure

```
sage-a2a-go/
├── pkg/
│   ├── transport/           # DID-authenticated HTTP transport ⭐ CORE
│   │   ├── did_http_transport.go  # DIDHTTPTransport (implements a2aclient.Transport)
│   │   ├── factory.go             # Factory options and convenience functions
│   │   ├── doc.go                 # Package documentation
│   │   └── *_test.go              # Tests
│   │
│   ├── client/              # Legacy HTTP wrapper (for reference)
│   │   ├── a2a_client.go          # Simple HTTP client with DID signing
│   │   ├── doc.go
│   │   └── *_test.go
│   │
│   ├── server/              # DID verification middleware
│   │   ├── middleware.go          # HTTP middleware for DID verification
│   │   ├── doc.go
│   │   └── *_test.go
│   │
│   ├── protocol/            # Agent Card management
│   │   ├── agent_card.go          # AgentCard struct and builder
│   │   ├── default_agent_card_signer.go  # JWS signing/verification
│   │   ├── doc.go
│   │   └── *_test.go
│   │
│   ├── signer/              # RFC9421 HTTP signing
│   │   ├── a2a_signer.go          # A2ASigner interface
│   │   ├── default_a2a_signer.go  # RFC 9421 implementation
│   │   ├── doc.go
│   │   └── *_test.go
│   │
│   └── verifier/            # DID verification
│       ├── did_verifier.go        # DIDVerifier interface
│       ├── default_did_verifier.go  # Implementation with blockchain resolution
│       ├── key_selector.go        # KeySelector interface
│       ├── default_key_selector.go  # Protocol-based key selection
│       ├── rfc9421_verifier.go    # RFC 9421 signature verification
│       ├── doc.go
│       └── *_test.go
│
├── cmd/
│   └── examples/            # Example applications
│       ├── simple-agent/
│       ├── agent-communication/
│       ├── multi-key-agent/
│       └── simple-client/
│
├── versions.go              # Version constants
├── go.mod                   # Dependencies
├── go.sum
├── README.md
└── LICENSE
```

## Component Responsibilities

### 1. Transport Package (`pkg/transport/`) ⭐ CORE

**Purpose**: Provide DID-authenticated HTTP/JSON-RPC 2.0 transport for a2a-go

**Key Types**:
```go
// DIDHTTPTransport implements a2aclient.Transport
type DIDHTTPTransport struct {
    baseURL    string
    agentDID   did.AgentDID
    keyPair    crypto.KeyPair
    signer     signer.A2ASigner
    httpClient *http.Client
}

// Implements all a2aclient.Transport methods:
- GetTask(ctx, query) (*a2a.Task, error)
- CancelTask(ctx, id) (*a2a.Task, error)
- SendMessage(ctx, msg) (a2a.SendMessageResult, error)
- SendStreamingMessage(ctx, msg) iter.Seq2[a2a.Event, error]
- ResubscribeToTask(ctx, id) iter.Seq2[a2a.Event, error]
- GetTaskPushConfig(ctx, params) (*a2a.TaskPushConfig, error)
- ListTaskPushConfig(ctx, params) ([]*a2a.TaskPushConfig, error)
- SetTaskPushConfig(ctx, config) (*a2a.TaskPushConfig, error)
- DeleteTaskPushConfig(ctx, params) error
- GetAgentCard(ctx) (*a2a.AgentCard, error)
```

**How It Works**:
1. Application calls a2a-go client method (e.g., `client.SendMessage()`)
2. a2a-go routes to transport (e.g., `transport.SendMessage()`)
3. DIDHTTPTransport:
   - Marshals request to JSON-RPC 2.0
   - Creates HTTP POST request to `/rpc`
   - **Signs request with DID** using RFC 9421
   - Sends HTTP request
   - Parses JSON-RPC 2.0 response
4. Returns result to a2a-go client
5. a2a-go returns to application

**Factory Functions**:
```go
// Factory option for a2a-go
func WithDIDHTTPTransport(did, key, client) a2aclient.FactoryOption

// Convenience functions
func NewDIDAuthenticatedClient(ctx, did, key, card) (*a2aclient.Client, error)
func NewDIDAuthenticatedClientWithConfig(ctx, did, key, card, config) (*a2aclient.Client, error)
func NewDIDAuthenticatedClientWithInterceptors(ctx, did, key, card, interceptors...) (*a2aclient.Client, error)
```

### 2. Verifier Package (`pkg/verifier/`)

**Purpose**: Verify HTTP signatures using SAGE DIDs

**Key Interfaces**:
```go
type DIDVerifier interface {
    VerifyHTTPSignature(ctx, req, did) error
    ResolvePublicKey(ctx, did, keyType) (crypto.PublicKey, error)
    VerifyHTTPSignatureWithKeyID(ctx, req) (did.AgentDID, error)
}

type KeySelector interface {
    SelectKey(ctx, did, protocol) (crypto.PublicKey, did.KeyType, error)
}
```

**Test Coverage**: 91-94%

### 3. Signer Package (`pkg/signer/`)

**Purpose**: Sign HTTP messages with DID identity

**Key Interface**:
```go
type A2ASigner interface {
    SignRequest(ctx, req, did, keyPair) error
    SignRequestWithOptions(ctx, req, did, keyPair, opts) error
}
```

**Features**:
- RFC 9421 HTTP Message Signatures
- DID as `keyid` parameter
- Configurable signature components
- Timestamp and nonce support

**Test Coverage**: 92.2%

### 4. Protocol Package (`pkg/protocol/`)

**Purpose**: Agent Card creation, signing, and verification

**Key Types**:
```go
type AgentCard struct {
    DID          string
    Name         string
    Description  string
    Endpoint     string
    Capabilities []string
    PublicKeys   []PublicKeyInfo
    CreatedAt    int64
    ExpiresAt    int64
    Metadata     map[string]interface{}
}

type AgentCardSigner interface {
    SignAgentCard(ctx, card, keyPair) (*SignedAgentCard, error)
    VerifyAgentCard(ctx, signedCard) error
    VerifyAgentCardWithKey(ctx, signedCard, publicKey) error
}
```

**Test Coverage**: 91.2%

### 5. Server Package (`pkg/server/`)

**Purpose**: HTTP middleware for DID verification (server-side)

**Key Component**:
```go
type DIDAuthMiddleware struct {
    verifier    verifier.DIDVerifier
    keySelector verifier.KeySelector
}

func (m *DIDAuthMiddleware) Middleware(next http.Handler) http.Handler
```

### 6. Client Package (`pkg/client/`) - Legacy

**Purpose**: Simple HTTP client wrapper (for reference/backward compatibility)

**Note**: For new projects, use `pkg/transport/DIDHTTPTransport` with a2a-go instead.

## Data Flow

### Client Request Flow

```
┌──────────────────────────────────────────────────┐
│ Application Code                                 │
│ client.SendMessage(ctx, message)                 │
└──────────────────┬───────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│ a2a-go Client (github.com/a2aproject/a2a-go)     │
│ - Handles A2A protocol logic                     │
│ - Calls transport.SendMessage()                  │
└──────────────────┬───────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│ DIDHTTPTransport (sage-a2a-go)                   │
│ 1. Marshal to JSON-RPC 2.0 request              │
│ 2. Create HTTP POST to /rpc                     │
│ 3. Sign with DID (RFC 9421) ← A2ASigner        │
│ 4. Send HTTP request                             │
│ 5. Parse JSON-RPC 2.0 response                  │
└──────────────────┬───────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│ HTTP Network                                     │
│ POST /rpc HTTP/1.1                               │
│ Signature-Input: sig1=(...);keyid="did:sage:…"  │
│ Signature: sig1=:base64signature:               │
└──────────────────────────────────────────────────┘
```

### Server Verification Flow

```
┌──────────────────────────────────────────────────┐
│ HTTP Request with DID Signature                  │
│ Signature-Input: keyid="did:sage:ethereum:0x…"  │
└──────────────────┬───────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│ DIDAuthMiddleware (sage-a2a-go)                  │
│ 1. Extract DID from keyid                        │
│ 2. Resolve public key from blockchain            │
│ 3. Verify RFC 9421 signature                     │
│ 4. Inject verified DID into context              │
└──────────────────┬───────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────┐
│ Application Handler                              │
│ - DID authenticated                               │
│ - Can trust sender identity                      │
└──────────────────────────────────────────────────┘
```

## Usage Examples

### Basic Client Usage

```go
import (
    "github.com/a2aproject/a2a-go/a2a"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Your agent's identity
myDID := did.AgentDID("did:sage:ethereum:0x...")
myKeyPair, _ := crypto.GenerateSecp256k1KeyPair()

// Target agent's card
targetCard := &a2a.AgentCard{
    Name: "Assistant",
    URL:  "https://agent.example.com",
}

// Create client with DID-authenticated transport
client, err := transport.NewDIDAuthenticatedClient(ctx, myDID, myKeyPair, targetCard)
if err != nil {
    log.Fatal(err)
}
defer client.Destroy()

// All requests are automatically signed with DID
task, err := client.SendMessage(ctx, &a2a.MessageSendParams{
    Message: &a2a.Message{
        Role: a2a.RoleUser,
        Parts: []a2a.Part{&a2a.TextPart{Text: "Hello"}},
    },
})
```

### Advanced Usage with a2a-go Factory

```go
import (
    "github.com/a2aproject/a2a-go/a2aclient"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
)

// Use a2a-go factory with custom options
client, err := a2aclient.NewFromCard(
    ctx,
    agentCard,
    transport.WithDIDHTTPTransport(myDID, myKeyPair, nil),
    a2aclient.WithConfig(a2aclient.Config{
        AcceptedOutputModes: []string{"application/json"},
    }),
    a2aclient.WithInterceptors(loggingInterceptor, metricsInterceptor),
)
```

### Server-Side Verification

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/server"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

// Setup DID verification middleware
client, _ := ethereum.NewEthereumClientV4(config)
selector := verifier.NewDefaultKeySelector(client)
didVerifier := verifier.NewDefaultDIDVerifier(client, selector, verifier.NewRFC9421Verifier())

middleware := server.NewDIDAuthMiddleware(didVerifier, selector)

// Apply to HTTP handler
http.Handle("/rpc", middleware.Middleware(http.HandlerFunc(handleRPC)))
```

## Testing Strategy

### Unit Tests
- Test transport methods independently
- Mock HTTP responses
- Verify DID signing on all requests
- Test JSON-RPC 2.0 marshaling

### Integration Tests
- Test with real a2aclient.Client
- Test with mock A2A server
- End-to-end DID authentication
- Multi-key scenarios

### Test Coverage
- **Overall**: 91.8%
- `pkg/verifier`: 93.1%
- `pkg/signer`: 92.2%
- `pkg/protocol`: 91.2%
- `pkg/transport`: TBD (new implementation)

## Performance Considerations

### Benchmarks
- DID signing: ~1-2ms per request (ECDSA/Ed25519)
- DID verification: ~2-3ms per request (with caching)
- Transport overhead: Minimal (direct implementation)
- JSON-RPC marshaling: ~0.1ms

### Optimization Opportunities
- Cache DID resolution results (TTL-based)
- Connection pooling (handled by http.Client)
- Batch signature verification (for servers)

## Security Considerations

### DID Signature Coverage
All HTTP requests include RFC 9421 signatures over:
- HTTP method (`@method`)
- Target URI (`@target-uri`)
- Content-Type header
- Request body (via Content-Digest)
- Timestamp (`created`)
- DID as `keyid` parameter

### Attack Mitigations
- **Replay attacks**: Timestamp validation (max 5 minutes)
- **Man-in-the-middle**: TLS + HTTP signatures
- **DID spoofing**: Blockchain-anchored identity
- **Key compromise**: Multi-key rotation support

## Maintenance Guidelines

### Regular Updates
- **Monthly**: Check for a2a-go updates
- **Quarterly**: Check for A2A protocol updates
- **After security fixes**: Immediate update

### Breaking Changes
When A2A introduces breaking changes:
1. Create feature branch
2. Update dependencies
3. Implement new transport methods
4. Update major version (semver)
5. Provide migration guide

### Deprecation Policy
- Deprecated features: 6 months notice
- Removed features: Major version bump
- Security issues: Immediate fix, patch release

## Roadmap

### Current (v2.0.0)
- ✅ HTTP/JSON-RPC 2.0 transport
- ✅ DID signatures (RFC 9421)
- ✅ All A2A v0.3.0 methods (except streaming)
- ✅ Agent Card signing/verification
- ✅ Multi-key support

### Next (v2.1.0)
- [ ] Server-Sent Events (SSE) for streaming methods
- [ ] A2A v0.4.0 support (when available)
- [ ] Performance optimizations
- [ ] Integration test suite

### Future (v3.0.0)
- [ ] gRPC transport (alongside HTTP)
- [ ] WebSocket transport
- [ ] HTTP/2 support
- [ ] Metrics and observability
- [ ] Multi-DID support

---

**Last Updated**: 2025-10-26
**Project Version**: 2.0.0-alpha
**A2A Protocol Version**: 0.3.0
**SAGE Version**: 1.1.0
**a2a-go Version**: 0.1.0 (local development)

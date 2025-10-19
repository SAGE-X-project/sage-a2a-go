# sage-a2a-go Architecture

## Overview

sage-a2a-go provides **DID-authenticated A2A protocol implementation** by integrating:
- **A2A Protocol** (specification) - Protocol definitions
- **a2a-go** (SDK implementation) - Go SDK for A2A protocol
- **SAGE** (DID infrastructure) - Blockchain-anchored identity and cryptography

## Design Principles

### 1. Dependency-Based Architecture
```
sage-a2a-go (DID Auth Layer)
    ├─→ a2a-go (A2A SDK) - Direct dependency
    ├─→ SAGE (DID Core) - Direct dependency
    └─→ A2A Protocol - Indirect (via a2a-go)
```

**Benefits**:
- ✅ Automatic updates when a2a-go updates
- ✅ No code duplication
- ✅ Clean separation of concerns
- ✅ Easy to track A2A protocol versions

### 2. Wrapper Pattern
sage-a2a-go **wraps** a2a-go components and adds DID authentication:

```go
// Client: a2aclient.Client + DID signing
DIDAuthenticatedClient {
    underlying: *a2aclient.Client
    did: did.AgentDID
    keyPair: crypto.KeyPair
}

// Server: a2asrv.RequestHandler + DID verification
DIDAuthenticatedHandler {
    underlying: a2asrv.RequestHandler
    verifier: verifier.DIDVerifier
}
```

### 3. Interceptor-Based Integration
DID authentication is injected via **interceptors**, not by modifying core logic:

```go
// Client: CallInterceptor signs requests before sending
type DIDSigningInterceptor struct {
    did did.AgentDID
    keyPair crypto.KeyPair
    signer signer.A2ASigner
}

// Server: Middleware verifies requests before handling
type DIDVerificationMiddleware struct {
    verifier verifier.DIDVerifier
}
```

## Version Management Strategy

### A2A Protocol Version Tracking
```
sage-a2a-go/versions.go:
    const (
        A2AProtocolVersion = "0.3.0"  // Current supported version
        MinA2AVersion = "0.2.6"       // Minimum compatible version
    )
```

### Dependency Version Pinning
```
go.mod:
    require (
        github.com/a2aproject/a2a-go v0.x.y  // Pin to specific version
        github.com/sage-x-project/sage v1.1.0
    )
```

### Update Process
1. **A2A Protocol updates** (e.g., v0.3.0 → v0.4.0):
   - Wait for a2a-go to release compatible version
   - Update `go.mod` to new a2a-go version
   - Update `versions.go` constants
   - Add wrapper methods for new features
   - Update tests and documentation

2. **a2a-go updates** (bug fixes, improvements):
   - Run `go get -u github.com/a2aproject/a2a-go`
   - Run tests to ensure compatibility
   - Update if needed

## Package Structure

```
sage-a2a-go/
├── pkg/
│   ├── client/              # DID-authenticated A2A client
│   │   ├── did_client.go          # DIDAuthenticatedClient (wraps a2aclient.Client)
│   │   ├── did_interceptor.go     # CallInterceptor for DID signing
│   │   ├── factory.go             # Easy client creation
│   │   └── *_test.go
│   │
│   ├── server/              # DID-verified A2A server
│   │   ├── did_handler.go         # DIDAuthenticatedHandler (wraps RequestHandler)
│   │   ├── middleware.go          # DID verification middleware (existing)
│   │   ├── builder.go             # Server builder pattern
│   │   └── *_test.go
│   │
│   ├── protocol/            # Agent Card management (existing)
│   │   ├── agent_card.go
│   │   └── default_agent_card_signer.go
│   │
│   ├── signer/              # RFC9421 HTTP signing (existing)
│   │   ├── a2a_signer.go
│   │   └── default_a2a_signer.go
│   │
│   ├── verifier/            # DID verification (existing)
│   │   ├── did_verifier.go
│   │   ├── key_selector.go
│   │   └── rfc9421_verifier.go
│   │
│   └── types/               # Re-export a2a-go types
│       └── types.go               # Re-export for convenience
│
├── versions.go              # Version constants
├── go.mod                   # Dependencies
└── examples/
    ├── simple-client/       # Basic client usage
    ├── simple-server/       # Basic server usage
    └── full-communication/  # End-to-end example
```

## Component Responsibilities

### 1. Client Package (`pkg/client/`)
**Purpose**: Provide DID-authenticated A2A client

**Key Types**:
```go
// Main client wrapper
type DIDAuthenticatedClient struct {
    client      *a2aclient.Client
    agentDID    did.AgentDID
    keyPair     crypto.KeyPair
}

// Implements all a2aclient.Client methods:
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

**DID Integration**:
- Uses `CallInterceptor` to inject DID signatures
- Transparent to caller (same API as a2aclient.Client)

### 2. Server Package (`pkg/server/`)
**Purpose**: Provide DID-verified A2A server

**Key Types**:
```go
// Main server handler wrapper
type DIDAuthenticatedHandler struct {
    handler     a2asrv.RequestHandler
    verifier    verifier.DIDVerifier
    keySelector verifier.KeySelector
}

// Implements all a2asrv.RequestHandler methods:
- OnGetTask(ctx, query) (*a2a.Task, error)
- OnCancelTask(ctx, id) (*a2a.Task, error)
- OnSendMessage(ctx, msg) (a2a.SendMessageResult, error)
- OnSendMessageStream(ctx, msg) iter.Seq2[a2a.Event, error]
- OnResubscribeToTask(ctx, id) iter.Seq2[a2a.Event, error]
- OnGetTaskPushConfig(ctx, params) (*a2a.TaskPushConfig, error)
- OnListTaskPushConfig(ctx, params) ([]*a2a.TaskPushConfig, error)
- OnSetTaskPushConfig(ctx, config) (*a2a.TaskPushConfig, error)
- OnDeleteTaskPushConfig(ctx, params) error
```

**DID Integration**:
- Verifies DID signatures before delegating to underlying handler
- Extracts DID and injects into context
- Returns authentication errors if verification fails

### 3. Existing Packages (No Changes)
- `pkg/protocol/` - Agent Card management
- `pkg/signer/` - RFC9421 HTTP signing
- `pkg/verifier/` - DID verification

## Extension Points

### Adding New A2A Methods
When A2A protocol adds new methods (e.g., `ListTasks` in v0.4.0):

1. Update a2a-go dependency
2. Add wrapper method to `DIDAuthenticatedClient`:
   ```go
   func (c *DIDAuthenticatedClient) ListTasks(ctx context.Context, params *a2a.ListTasksParams) (*a2a.ListTasksResponse, error) {
       return c.client.ListTasks(ctx, params)
   }
   ```
3. Add wrapper method to `DIDAuthenticatedHandler`:
   ```go
   func (h *DIDAuthenticatedHandler) OnListTasks(ctx context.Context, params *a2a.ListTasksParams) (*a2a.ListTasksResponse, error) {
       // Verify DID from context
       if err := h.verifyDID(ctx); err != nil {
           return nil, err
       }
       return h.handler.OnListTasks(ctx, params)
   }
   ```

### Custom Authentication Schemes
DID verification can be extended or replaced:
```go
// Custom verifier
type CustomDIDVerifier struct { ... }

func (v *CustomDIDVerifier) VerifyHTTPSignature(...) error {
    // Custom logic
}

// Use in server
handler := NewDIDAuthenticatedHandler(baseHandler, customVerifier, keySelector)
```

## Testing Strategy

### Unit Tests
- Test wrappers independently
- Mock a2a-go components
- Verify DID integration logic

### Integration Tests
- Test with real a2aclient.Client
- Test with real a2asrv.RequestHandler
- End-to-end DID authentication

### Version Compatibility Tests
- Test against multiple A2A versions
- Verify graceful degradation
- Test version negotiation

## Migration Path

### From Current sage-a2a-go
1. Existing code in `pkg/client/a2a_client.go` → `pkg/client/did_client.go`
2. Existing `pkg/server/middleware.go` → Keep as-is (reuse in handler)
3. Add new wrapper classes
4. Update examples to use new API
5. Deprecate old API gradually

### From Direct a2a-go Usage
Users can migrate from a2a-go to sage-a2a-go easily:
```go
// Before (a2a-go)
client := a2aclient.NewClient(...)

// After (sage-a2a-go)
client := sagea2a.NewDIDAuthenticatedClient(agentDID, keyPair, ...)
```

## Performance Considerations

### Minimal Overhead
- DID signing: ~1-2ms per request (ECDSA/Ed25519)
- DID verification: ~2-3ms per request (includes blockchain resolution caching)
- Wrapper pattern: Zero overhead (just delegation)

### Optimization Opportunities
- Cache DID resolution results
- Batch signature verification
- Connection pooling (handled by a2a-go)

## Security Considerations

### DID Signature Coverage
All HTTP requests are signed with:
- HTTP method
- Target URI
- Headers (configurable)
- Body (for POST/PUT)
- Timestamp (replay protection)

### Key Management
- Private keys managed by SAGE crypto.KeyPair
- Public keys resolved from blockchain
- Multi-key support for key rotation

### Attack Mitigations
- Replay attacks: Timestamp validation
- Man-in-the-middle: TLS + HTTP signatures
- DID spoofing: Blockchain-anchored identity

## Maintenance Guidelines

### Regular Updates
- **Monthly**: Check for a2a-go updates
- **Quarterly**: Check for A2A protocol updates
- **After security fixes**: Immediate update

### Breaking Changes
When A2A introduces breaking changes:
1. Create feature branch
2. Update dependencies
3. Fix compatibility issues
4. Update major version (semver)
5. Provide migration guide

### Deprecation Policy
- Deprecated features: 6 months notice
- Removed features: Major version bump
- Security issues: Immediate fix, patch release

## Future Enhancements

### Short-term (v1.x)
- [ ] gRPC transport support
- [ ] WebSocket transport support
- [ ] Agent discovery integration
- [ ] Metrics and observability

### Long-term (v2.x)
- [ ] Multi-DID support
- [ ] Delegated authentication
- [ ] Plugin system for custom transports
- [ ] Advanced caching strategies

---

**Last Updated**: 2025-10-19
**A2A Protocol Version**: 0.3.0
**a2a-go Version**: TBD (to be added in go.mod)

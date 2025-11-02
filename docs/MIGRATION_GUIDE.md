# Migration Guide: sage-multi-agent → sage-a2a-go v1.5.2

This guide helps the sage-multi-agent team migrate from direct SAGE/A2A imports to the unified sage-a2a-go API.

## Overview

**Goal**: Remove all direct imports of `github.com/sage-x-project/sage` and `github.com/a2aproject/a2a-go` from sage-multi-agent. Use **only** `github.com/sage-x-project/sage-a2a-go` packages.

**Why**: Unified API ensures compatibility, reduces dependency conflicts, and provides a stable abstraction layer.

## Migration Status

### ✅ Phase 1: Unified API (Available in v1.5.2)

The following packages are now available for immediate migration:

- `pkg/crypto` - Cryptographic operations
- `pkg/identity` - DID management
- `pkg/agent` - High-level agent builder

### 🚧 Phase 2: HPKE & Sessions (In Development - Week 1)

- `pkg/hpke` - HPKE server and encryption
- `pkg/session` - Session management
- `pkg/transport` - HTTP transport adapter

### 📋 Phase 3: Advanced Features (Weeks 2-3)

- `pkg/crypto` - JWK import/export
- `pkg/registry` - Registry client

---

## Phase 1 Migration (Immediate)

### 1. Crypto Package Migration

#### Before (Direct SAGE Import):
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

keyPair, err := keys.GenerateSecp256k1KeyPair()
ed25519KeyPair, err := keys.GenerateEd25519KeyPair()
```

#### After (sage-a2a-go):
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"

keyPair, err := crypto.GenerateSecp256k1KeyPair()
ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
```

#### Affected Files:
- `tools/keygen/gen_agents_key.go`
- `agents/*/agent.go` (all agent implementations)
- `internal/transport/did_authenticated.go`

---

### 2. Identity/DID Package Migration

#### Before (Direct SAGE Import):
```go
import "github.com/sage-x-project/sage/pkg/agent/did"

type AgentDID = did.AgentDID
metadata, err := did.GetAgentMetadata(ctx, client, agentDID)
chain, address, err := did.ParseDID(agentDID)
```

#### After (sage-a2a-go):
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/identity"

type AgentDID = identity.AgentDID
metadata, err := identity.GetAgentMetadata(ctx, client, agentDID)
chain, address, err := identity.ParseDID(agentDID)
```

#### Affected Files:
- `internal/did/resolver.go`
- `internal/registry/client.go`
- `agents/*/agent.go`
- `cmd/cli/commands/agent.go`

---

### 3. Agent Builder Migration

#### Before (Manual A2A Client Setup):
```go
import (
    "github.com/a2aproject/a2a-go/a2a"
    a2aclient "github.com/a2aproject/a2a-go/client"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

keyPair, _ := keys.GenerateSecp256k1KeyPair()

client := a2aclient.NewClient(
    a2aclient.WithTransport(a2a.TransportProtocolJSONRPC),
    a2aclient.WithDIDAuthentication(myDID, keyPair),
)
```

#### After (Unified Agent Builder):
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/agent"
    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
)

keyPair, _ := crypto.GenerateSecp256k1KeyPair()
myDID := identity.AgentDID("did:sage:ethereum:0x...")

card := &a2a.AgentCard{
    Name:               "My Agent",
    URL:                "https://my-agent.example.com",
    PreferredTransport: a2a.TransportProtocolJSONRPC,
}

myAgent, err := agent.NewAgent(ctx, myDID, keyPair, card)
// myAgent.A2AClient is ready to use
```

#### Affected Files:
- `agents/*/agent.go` (initialization code)
- `cmd/cli/commands/agent.go`
- `internal/transport/client.go`

---

## Phase 2 Migration (Week 1 - In Development)

### 4. HPKE Server Migration

**Status**: 🚧 Implementation in progress

#### Target Usage (Coming Soon):
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/hpke"
    "github.com/sage-x-project/sage-a2a-go/pkg/session"
)

// Create session manager
sessionMgr := session.NewManager(session.DefaultOptions())

// Create HPKE server
hpkeServer, err := hpke.NewServer(
    signingKey,
    sessionMgr,
    "did:sage:ethereum:0x...",
    didResolver,
    nil, // default options
)

// Use as HTTP handler
http.Handle("/messages", hpkeServer.MessagesHandler())
```

#### Will Replace:
- Direct SAGE HPKE imports in `internal/transport/hpke_server.go`
- Custom session management in `internal/session/manager.go`

---

### 5. Session Manager Migration

**Status**: 🚧 Implementation in progress

#### Target Usage (Coming Soon):
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/session"

sessionMgr := session.NewManager(session.DefaultOptions())

// Create session
sess, err := sessionMgr.Create(ctx, remoteDID, sharedSecret)

// Encrypt message
encrypted, err := sessionMgr.Encrypt(ctx, sessionID, plaintext)

// Decrypt message
plaintext, err := sessionMgr.Decrypt(ctx, sessionID, encrypted)
```

#### Will Replace:
- Custom session code in `internal/session/manager.go`
- Session storage in `internal/session/store.go`

---

## File-by-File Migration Checklist

### High Priority (Phase 1 - Immediate)

- [ ] `tools/keygen/gen_agents_key.go`
  - Replace `github.com/sage-x-project/sage/pkg/agent/crypto/keys` → `pkg/crypto`

- [ ] `agents/auction_agent/agent.go`
  - Replace SAGE crypto imports → `pkg/crypto`
  - Replace SAGE DID imports → `pkg/identity`
  - Use `agent.NewAgent()` builder

- [ ] `agents/trading_agent/agent.go`
  - Same as auction_agent

- [ ] `agents/analytics_agent/agent.go`
  - Same as auction_agent

- [ ] `internal/did/resolver.go`
  - Replace `github.com/sage-x-project/sage/pkg/agent/did` → `pkg/identity`

- [ ] `internal/transport/did_authenticated.go`
  - Replace SAGE crypto imports → `pkg/crypto`
  - Replace SAGE DID imports → `pkg/identity`

### Medium Priority (Phase 2 - Week 1)

- [ ] `internal/transport/hpke_server.go`
  - Wait for `pkg/hpke` implementation
  - Replace SAGE HPKE imports → `pkg/hpke`

- [ ] `internal/session/manager.go`
  - Wait for `pkg/session` implementation
  - Replace custom session code → `pkg/session`

### Low Priority (Phase 3 - Weeks 2-3)

- [ ] `internal/registry/client.go`
  - Wait for `pkg/registry` implementation
  - Replace direct registry calls → `pkg/registry`

---

## Testing Migration

### Before:
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func TestAgentCreation(t *testing.T) {
    keyPair, _ := keys.GenerateSecp256k1KeyPair()
    testDID := did.AgentDID("did:sage:ethereum:0x...")
    // ...
}
```

### After:
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
)

func TestAgentCreation(t *testing.T) {
    keyPair, _ := crypto.GenerateSecp256k1KeyPair()
    testDID := identity.AgentDID("did:sage:ethereum:0x...")
    // ...
}
```

---

## Breaking Changes in v1.5.2

### 1. KME → KEM Naming (RFC 9180 Compliance)

**Before (SAGE v1.3.1):**
```go
client.ResolveKMEKey(ctx, agentDID)  // Old name
```

**After (SAGE v1.5.2):**
```go
client.ResolveKEMKey(ctx, agentDID)  // RFC 9180 compliant
```

### 2. AgentCard.PreferredTransport Required

**Before:**
```go
card := &a2a.AgentCard{
    Name: "Agent",
    URL:  "https://agent.com",
    // PreferredTransport optional
}
```

**After:**
```go
card := &a2a.AgentCard{
    Name:               "Agent",
    URL:                "https://agent.com",
    PreferredTransport: a2a.TransportProtocolJSONRPC, // REQUIRED
}
```

### 3. KeyType is `int` (not `string`)

**Before (SAGE v1.3.1):**
```go
keyType := "secp256k1"  // String-based
```

**After (SAGE v1.5.2):**
```go
keyType := identity.KeyTypeECDSA  // int constant (0)
// KeyTypeECDSA = 0
// KeyTypeEd25519 = 1
// KeyTypeX25519 = 2
```

---

## Benefits of Migration

1. **Single Dependency**: Only import sage-a2a-go, not SAGE + A2A + A2A-Go
2. **Version Safety**: sage-a2a-go ensures SAGE/A2A version compatibility
3. **Simpler Updates**: Upgrade sage-a2a-go version, not 3 separate packages
4. **Better Testing**: sage-a2a-go has 89% test coverage with integration tests
5. **Stable API**: Breaking changes handled in sage-a2a-go, not your code

---

## Migration Timeline

| Phase | Timeline | Features | Status |
|-------|----------|----------|--------|
| Phase 1 | Immediate | Crypto, Identity, Agent Builder | ✅ Available |
| Phase 2 | Week 1 | HPKE, Sessions, HTTP Transport | 🚧 In Progress |
| Phase 3 | Weeks 2-3 | JWK, Registry Client | 📋 Planned |

---

## Support

- **Issues**: https://github.com/sage-x-project/sage-a2a-go/issues
- **Documentation**: https://github.com/sage-x-project/sage-a2a-go/blob/main/README.md
- **Examples**: `cmd/examples/` directory

---

## Quick Migration Commands

```bash
# Update sage-a2a-go to v1.5.2
go get github.com/sage-x-project/sage-a2a-go@v1.5.2

# Find files importing SAGE directly (to migrate)
grep -r "github.com/sage-x-project/sage" --include="*.go" .

# Find files importing A2A directly (to migrate)
grep -r "github.com/a2aproject/a2a-go" --include="*.go" .

# Run tests after migration
go test ./...
```

---

## Example: Complete Agent Migration

### Before (23 files with direct imports):
```go
// agent.go
import (
    "github.com/a2aproject/a2a-go/a2a"
    a2aclient "github.com/a2aproject/a2a-go/client"
    "github.com/sage-x-project/sage/pkg/agent/crypto/keys"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/hpke"
)

type Agent struct {
    did       did.AgentDID
    keyPair   crypto.KeyPair
    client    *a2aclient.Client
    hpkeServer *hpke.Server
}
```

### After (Only sage-a2a-go imports):
```go
// agent.go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/agent"
    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
    "github.com/sage-x-project/sage-a2a-go/pkg/hpke"     // Coming in Week 1
    "github.com/sage-x-project/sage-a2a-go/pkg/session"  // Coming in Week 1
)

type Agent struct {
    *agent.Agent  // Embeds DID, KeyPair, A2AClient, Card
    hpkeServer *hpke.Server
}
```

---

**Last Updated**: 2025-11-02 (sage-a2a-go v1.5.2)

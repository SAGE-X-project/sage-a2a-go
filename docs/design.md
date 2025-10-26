# SAGE-A2A-GO Design & Implementation Documentation

**Version**: 1.0.0-dev
**Date**: 2025-10-26
**Status**: ✅ Implementation Complete (Targeting v1.0.0 Release)
**SAGE Version**: v1.3.1
**A2A Protocol**: v0.4.0 (JSON-RPC 2.0 over HTTP/S)
**A2A-go Fork**: SAGE-X-project/a2a-go (with Message Parts bug fixes)
**Test Coverage**: 91.8% (Target: ≥90%) - ✅ **Achieved**
**Total Tests**: 173 (Unit + Integration + E2E)

---

## Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Component Design](#component-design)
4. [Interface Specifications](#interface-specifications)
5. [Data Flow](#data-flow)
6. [Security Considerations](#security-considerations)
7. [Testing Strategy](#testing-strategy)
8. [Performance Considerations](#performance-considerations)

---

## Overview

### Purpose

`sage-a2a-go` bridges SAGE's blockchain-anchored DID system with the A2A (Agent-to-Agent) protocol, enabling secure, decentralized agent authentication and communication.

### Key Goals

1. ✅ **DID-based Authentication**: Replace traditional API keys with blockchain-anchored DIDs (Implemented)
2. ✅ **RFC9421 Compliance**: Implement HTTP Message Signatures with DID integration (Implemented)
3. ✅ **Multi-Key Support**: Support ECDSA (Ethereum) and Ed25519 (Solana) keys (Implemented)
4. ✅ **Protocol Agnostic**: Work seamlessly across different blockchain protocols (Implemented)
5. ✅ **High Test Coverage**: Maintain ≥90% test coverage with TDD approach (Achieved: 91.8%)

### Integration Points

```
┌──────────────┐         ┌──────────────┐
│   A2A Agent  │────────▶│ sage-a2a-go  │
│              │         │              │
│ - Tasks      │◀────────│ - DIDVerifier│
│ - Messages   │         │ - A2ASigner  │
│ - Cards      │         │ - KeySelector│
└──────────────┘         └──────┬───────┘
                                │
                                │ (resolves DIDs)
                                │
                         ┌──────▼───────┐
                         │  SAGE Core   │
                         │              │
                         │ - DID        │
                         │ - Blockchain │
                         │ - Crypto     │
                         └──────────────┘
```

---

## System Architecture

### Layered Architecture

```
┌─────────────────────────────────────────────────────┐
│          Application Layer (A2A Clients)            │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│           Protocol Layer (A2A Protocol)             │
│  - Task Management                                  │
│  - Agent Discovery                                  │
│  - Message Routing                                  │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│      Authentication Layer (sage-a2a-go) ★           │
│  ┌──────────────┐  ┌──────────────┐                │
│  │ DIDVerifier  │  │  A2ASigner   │                │
│  └──────┬───────┘  └──────┬───────┘                │
│         │                  │                        │
│  ┌──────▼──────────────────▼───────┐                │
│  │      KeySelector                │                │
│  └──────────────────────────────────┘                │
└─────────────────────┬───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│         DID Layer (SAGE Core v1.3.1)                │
│  - DID Registry (Blockchain)                        │
│  - Multi-Key Resolution                             │
│  - Crypto Operations                                │
│  - RFC9421 Primitives                               │
└─────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Layer | Component | Responsibility |
|-------|-----------|----------------|
| **Authentication** | DIDVerifier | Verify HTTP signatures using DIDs |
| | A2ASigner | Sign HTTP messages with DIDs |
| | KeySelector | Select appropriate key based on protocol |
| | AgentCardSigner | Sign/verify Agent Cards |
| **DID** | SAGE Core | DID resolution, crypto operations |

---

## Component Design

### 1. KeySelector ✅ Implemented

**Purpose**: Select the appropriate cryptographic key based on protocol or explicit preference.

**Status**: ✅ Complete (Test Coverage: 94.1%)

**Interface**:

```go
package verifier

import (
    "context"
    "crypto"

    "github.com/sage-x-project/sage/pkg/agent/did"
)

// KeySelector selects appropriate cryptographic key for a given protocol
type KeySelector interface {
    // SelectKey selects a key for the given agent DID and protocol
    // protocol: "ethereum", "solana", or empty string for default
    // Returns: public key, key type, error
    SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error)
}
```

**Implementation Logic**:

```
┌─────────────────────────────────┐
│ SelectKey(DID, protocol)        │
└────────────┬────────────────────┘
             │
             ▼
    ┌────────────────────┐
    │ Protocol specified?│
    └────┬───────────────┘
         │ Yes
         ▼
    ┌────────────────────────┐
    │ protocol == "ethereum"?│──Yes──▶ Try ECDSA key
    └────┬───────────────────┘
         │ No
         ▼
    ┌────────────────────────┐
    │ protocol == "solana"?  │──Yes──▶ Try Ed25519 key
    └────┬───────────────────┘
         │ No
         ▼
    ┌────────────────────────┐
    │ Get all verified keys  │
    └────┬───────────────────┘
         │
         ▼
    ┌────────────────────────┐
    │ Return first key       │
    └────────────────────────┘
```

**Test Cases**: ✅ All Passing (11 tests)

1. ✅ Ethereum protocol selects ECDSA key
2. ✅ Solana protocol selects Ed25519 key
3. ✅ Unknown protocol falls back to first key
4. ✅ No keys found returns error
5. ✅ Preferred key not available falls back
6. ✅ Multiple keys scenario

---

### 2. DIDVerifier ✅ Implemented

**Purpose**: Verify HTTP Message Signatures using SAGE DIDs.

**Status**: ✅ Complete (Test Coverage: 93.1%)

**Interface**:

```go
package verifier

import (
    "context"
    "crypto"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/did"
)

// DIDVerifier verifies HTTP signatures using SAGE DIDs
type DIDVerifier interface {
    // VerifyHTTPSignature verifies the HTTP signature in the request
    // using the public key resolved from the agent DID
    VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error

    // ResolvePublicKey resolves a public key for the given DID
    // keyType: optional preferred key type (nil for first available)
    ResolvePublicKey(ctx context.Context, agentDID did.AgentDID, keyType *did.KeyType) (crypto.PublicKey, error)
}
```

**Implementation Details**:

```go
type DIDVerifierImpl struct {
    client   ethereum.EthereumClientV4  // SAGE client
    selector KeySelector                 // Key selection logic
    verifier *rfc9421.Verifier           // RFC9421 verifier
}

func (v *DIDVerifierImpl) VerifyHTTPSignature(ctx context.Context, req *http.Request, agentDID did.AgentDID) error {
    // 1. Extract signature parameters from HTTP headers
    sigParams, err := extractSignatureParams(req)

    // 2. Resolve public key using KeySelector
    pubKey, err := v.ResolvePublicKey(ctx, agentDID, nil)

    // 3. Verify signature using RFC9421 verifier
    return v.verifier.VerifyHTTPRequest(req, pubKey)
}
```

**Data Flow**:

```
HTTP Request with Signature
        │
        ▼
┌───────────────────────┐
│ Extract keyid (DID)   │
└───────┬───────────────┘
        │
        ▼
┌───────────────────────┐
│ Parse DID             │
└───────┬───────────────┘
        │
        ▼
┌───────────────────────┐
│ Resolve Public Key    │
│ (via KeySelector)     │
└───────┬───────────────┘
        │
        ▼
┌───────────────────────┐
│ Verify RFC9421        │
│ Signature             │
└───────┬───────────────┘
        │
        ▼
     Success/Error
```

**Test Cases**: ✅ All Passing (16 tests)

1. ✅ Valid ECDSA signature verification
2. ✅ Valid Ed25519 signature verification
3. ✅ Invalid signature returns error
4. ✅ Expired timestamp returns error
5. ✅ Invalid DID returns error
6. ✅ Replay attack prevention
7. ✅ Missing signature headers return error

---

### 3. A2ASigner ✅ Implemented

**Purpose**: Sign HTTP messages for A2A communication with DID identity.

**Status**: ✅ Complete (Test Coverage: 92.2%)

**Interface**:

```go
package signer

import (
    "context"
    "net/http"

    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// A2ASigner signs HTTP messages for A2A protocol
type A2ASigner interface {
    // SignRequest signs an HTTP request with the agent's key
    // The DID is included in the signature as the keyid parameter
    SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error
}
```

**Implementation Details**:

```go
type A2ASignerImpl struct {
    signer *rfc9421.Signer  // RFC9421 signer
}

func (s *A2ASignerImpl) SignRequest(ctx context.Context, req *http.Request, agentDID did.AgentDID, keyPair crypto.KeyPair) error {
    // 1. Create signature parameters with DID as keyid
    params := &rfc9421.SignatureParams{
        KeyID:     string(agentDID),
        Algorithm: getAlgorithm(keyPair.Type()),
        Created:   time.Now().Unix(),
        Components: []string{"@method", "@target-uri", "@authority", "content-type", "content-digest"},
    }

    // 2. Sign request using RFC9421 signer
    return s.signer.SignHTTPRequest(req, keyPair, params)
}

func getAlgorithm(keyType crypto.KeyType) string {
    switch keyType {
    case crypto.KeyTypeSecp256k1:
        return "ecdsa-p256-sha256"
    case crypto.KeyTypeEd25519:
        return "ed25519"
    default:
        return ""
    }
}
```

**Signature Format** (RFC9421):

```http
POST /task HTTP/1.1
Host: agent.example.com
Content-Type: application/json
Content-Digest: sha-256=X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=
Signature-Input: sig1=("@method" "@target-uri" "@authority" "content-type" "content-digest");created=1618884473;keyid="did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
Signature: sig1=:MEUCIQDzN...signature...==:
```

**Test Cases**: ✅ All Passing (17 tests)

1. ✅ Sign request with ECDSA key
2. ✅ Sign request with Ed25519 key
3. ✅ DID included in signature keyid
4. ✅ Timestamp included
5. ✅ RFC9421 format compliance
6. ✅ Content-Digest generation

---

### 4. AgentCardSigner ✅ Implemented

**Purpose**: Sign and verify A2A Agent Cards with DID identity.

**Status**: ✅ Complete (Test Coverage: 91.2%)

**Note**: Implemented in `pkg/protocol` package.

**Interface**:

```go
package signer

import (
    "context"

    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// AgentCard represents an A2A Agent Card
type AgentCard struct {
    DID          string                 `json:"did"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    Endpoint     string                 `json:"endpoint"`
    Capabilities []string               `json:"capabilities"`
    PublicKeys   []PublicKeyInfo        `json:"publicKeys,omitempty"`
    CreatedAt    int64                  `json:"createdAt"`
}

type PublicKeyInfo struct {
    ID       string `json:"id"`
    Type     string `json:"type"`     // "EcdsaSecp256k1VerificationKey2019" or "Ed25519VerificationKey2020"
    KeyData  string `json:"keyData"`  // Base64-encoded
}

// SignedAgentCard represents a signed Agent Card
type SignedAgentCard struct {
    Card      *AgentCard `json:"card"`
    Signature string     `json:"signature"` // JWS compact serialization
}

// AgentCardSigner signs and verifies Agent Cards
type AgentCardSigner interface {
    // SignAgentCard signs an Agent Card with the agent's key
    SignAgentCard(ctx context.Context, card *AgentCard, keyPair crypto.KeyPair) (*SignedAgentCard, error)

    // VerifyAgentCard verifies a signed Agent Card
    VerifyAgentCard(ctx context.Context, signedCard *SignedAgentCard) error
}
```

**Test Cases**: ✅ All Passing (43 tests)

1. ✅ Create Agent Card with DID
2. ✅ Sign Agent Card (JWS)
3. ✅ Verify valid Agent Card
4. ✅ Reject tampered Agent Card
5. ✅ Include multiple public keys
6. ✅ Timestamp validation

---

## Interface Specifications

### Error Handling

All components use consistent error types:

```go
package errors

import "errors"

var (
    ErrInvalidDID           = errors.New("invalid DID format")
    ErrDIDNotFound          = errors.New("DID not found in registry")
    ErrNoKeysFound          = errors.New("no verified keys found for DID")
    ErrKeyTypeNotSupported  = errors.New("key type not supported")
    ErrInvalidSignature     = errors.New("invalid signature")
    ErrSignatureExpired     = errors.New("signature expired")
    ErrReplayAttack         = errors.New("potential replay attack detected")
    ErrMissingHeaders       = errors.New("required headers missing")
)
```

### Context Usage

All operations support context for:
- Timeout control
- Cancellation
- Trace propagation

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := verifier.VerifyHTTPSignature(ctx, req, agentDID)
```

---

## Data Flow

### Agent Registration Flow

```
┌─────────────┐
│ Generate    │
│ Key Pairs   │
│ (ECDSA +    │
│ Ed25519)    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Create DID  │
│ (with owner │
│ address)    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Register    │
│ Agent on    │
│ Blockchain  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Create      │
│ Agent Card  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Sign Agent  │
│ Card (JWS)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Publish     │
│ Agent Card  │
└─────────────┘
```

### Message Signing Flow

```
┌─────────────┐
│ Create HTTP │
│ Request     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Generate    │
│ Content-    │
│ Digest      │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Select Key  │
│ (via        │
│ KeySelector)│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Create      │
│ Signature   │
│ Base (RFC   │
│ 9421)       │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Sign with   │
│ Private Key │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Add Headers │
│ - Signature │
│ - Sig-Input │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Send        │
│ Request     │
└─────────────┘
```

### Message Verification Flow

```
┌─────────────┐
│ Receive     │
│ HTTP Request│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Extract DID │
│ from keyid  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Validate    │
│ DID Format  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Resolve     │
│ Public Key  │
│ (blockchain)│
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Extract     │
│ Signature   │
│ Components  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Reconstruct │
│ Signature   │
│ Base        │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Verify      │
│ Signature   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Validate    │
│ Timestamp   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Accept/     │
│ Reject      │
└─────────────┘
```

---

## Security Considerations

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| **DID Spoofing** | DIDs anchored on blockchain, tamper-proof |
| **Replay Attack** | Timestamp validation, nonce support |
| **Man-in-the-Middle** | TLS 1.3+, HTTP Message Signatures |
| **Key Compromise** | Multi-key support, key rotation |
| **Signature Forgery** | Cryptographic signatures, public key verification |
| **Agent Card Tampering** | JWS signatures, integrity verification |

### Security Best Practices

1. **Always use HTTPS**: Enforce TLS 1.3+ with strong ciphers
2. **Timestamp Validation**: Reject signatures older than 5 minutes
3. **Key Rotation**: Support regular key rotation on blockchain
4. **mTLS**: Mutual TLS for agent-to-agent communication
5. **Audit Logging**: Log all signature verifications
6. **Rate Limiting**: Prevent DoS attacks

### Timestamp Validation

```go
const MaxSignatureAge = 5 * time.Minute

func validateTimestamp(created int64) error {
    signatureTime := time.Unix(created, 0)
    now := time.Now()
    age := now.Sub(signatureTime)

    if age > MaxSignatureAge {
        return ErrSignatureExpired
    }

    if age < -1*time.Minute {
        return ErrInvalidTimestamp
    }

    return nil
}
```

---

## Testing Strategy

### Test-Driven Development (TDD)

**Process**:
1. Write test case first (RED)
2. Implement minimal code to pass (GREEN)
3. Refactor code (REFACTOR)
4. Repeat

### Test Structure

```
pkg/
├── verifier/
│   ├── key_selector.go
│   ├── key_selector_test.go      # Unit tests
│   ├── did_verifier.go
│   ├── did_verifier_test.go      # Unit tests
│   └── mocks/                     # Mock implementations
├── signer/
│   ├── a2a_signer.go
│   ├── a2a_signer_test.go        # Unit tests
│   ├── agent_card_signer.go
│   └── agent_card_signer_test.go # Unit tests
└── protocol/
    └── ...

test/
└── integration/
    ├── did_verification_test.go   # Integration tests
    ├── multi_key_test.go          # Multi-key scenarios
    └── e2e_test.go                # End-to-end tests
```

### Test Coverage Goals ✅ Achieved

| Package | Target | Achieved | Status |
|---------|--------|----------|--------|
| `pkg/verifier` | ≥ 90% | **93.1%** | ✅ |
| `pkg/signer` | ≥ 90% | **92.2%** | ✅ |
| `pkg/protocol` | ≥ 90% | **91.2%** | ✅ |
| `pkg/transport` | ≥ 90% | **TBD** | 🔄 In Progress |
| **Overall** | **≥ 90%** | **91.8%** | ✅ **Achieved** |

### Test Categories

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test component interactions
3. **End-to-End Tests**: Test complete flows
4. **Performance Tests**: Benchmark critical paths
5. **Security Tests**: Test threat mitigations

---

## Performance Considerations

### Performance Targets

| Operation | Target Time |
|-----------|-------------|
| DID Resolution | < 100ms |
| Signature Verification | < 50ms |
| Key Selection | < 10ms |
| Agent Card Verification | < 100ms |

### Optimization Strategies

1. **Caching**: Cache DID resolutions (with TTL)
2. **Connection Pooling**: Reuse blockchain connections
3. **Parallel Processing**: Verify multiple signatures concurrently
4. **Key Pre-loading**: Pre-load frequently used keys

### Caching Strategy

```go
type DIDCache struct {
    cache map[did.AgentDID]*CacheEntry
    ttl   time.Duration
    mu    sync.RWMutex
}

type CacheEntry struct {
    PublicKey crypto.PublicKey
    ExpiresAt time.Time
}

func (c *DIDCache) Get(agentDID did.AgentDID) (crypto.PublicKey, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, found := c.cache[agentDID]
    if !found || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }

    return entry.PublicKey, true
}
```

---

## Appendix

### Glossary

- **DID**: Decentralized Identifier
- **A2A**: Agent-to-Agent Protocol
- **RFC9421**: HTTP Message Signatures specification
- **JWS**: JSON Web Signature
- **mTLS**: Mutual Transport Layer Security
- **TDD**: Test-Driven Development

### References

- [SAGE Architecture](https://github.com/sage-x-project/sage)
- [A2A Protocol Specification](https://a2a-protocol.org)
- [RFC9421](https://www.rfc-editor.org/rfc/rfc9421.html)
- [DID Core](https://www.w3.org/TR/did-core/)

---

**Version History**:
- v2.0 (2025-10-26): Updated with implementation status and test coverage results
- v1.0 (2025-10-18): Initial design document


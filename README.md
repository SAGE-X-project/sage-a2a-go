# SAGE-A2A-GO

**DID-based Authentication for Agent-to-Agent (A2A) Protocol**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![SAGE Version](https://img.shields.io/badge/SAGE-v1.1.0-blue)](https://github.com/sage-x-project/sage)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)

## Overview

`sage-a2a-go` integrates the SAGE Decentralized Identity (DID) system with the A2A (Agent-to-Agent) protocol, providing:

- ✅ **DID-based Authentication**: Use SAGE DIDs for agent identity verification
- ✅ **RFC9421 HTTP Message Signatures**: Secure message signing with DIDs
- ✅ **Multi-Key Support**: ECDSA (Ethereum) and Ed25519 (Solana) cryptographic keys
- ✅ **Agent Card Signing**: Cryptographically signed Agent Cards with DIDs
- ✅ **Protocol Interoperability**: Seamless integration with A2A protocol

## Architecture

```
┌─────────────────────────────────────────┐
│         sage-a2a-go Project             │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   A2A Protocol Layer              │ │
│  │   - Agent Communication           │ │
│  │   - Task Management               │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
│  ┌───────────────▼───────────────────┐ │
│  │   DID Authentication Layer        │ │
│  │   - DIDVerifier                   │ │
│  │   - A2ASigner                     │ │
│  │   - AgentCardSigner               │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
│  ┌───────────────▼───────────────────┐ │
│  │   KeySelector                     │ │
│  │   - Multi-key selection           │ │
│  │   - Protocol-aware                │ │
│  └───────────────┬───────────────────┘ │
│                  │                      │
└──────────────────┼──────────────────────┘
                   │
                   │ (uses)
                   │
┌──────────────────▼──────────────────────┐
│         SAGE Core (v1.1.0)              │
│                                         │
│  ✅ DID Registry (Blockchain)          │
│  ✅ Multi-key Resolution               │
│  ✅ Crypto Primitives                  │
│  ✅ RFC9421 Support                    │
└─────────────────────────────────────────┘
```

## Features

### DIDVerifier

Verify HTTP signatures using SAGE DIDs:

- Resolve public keys from blockchain-anchored DID registry
- Support multi-key agents (ECDSA + Ed25519)
- Verify RFC9421 HTTP message signatures
- Prevent replay attacks with timestamp validation

### KeySelector

Intelligently select cryptographic keys based on protocol:

- Ethereum protocol → ECDSA (secp256k1) key
- Solana protocol → Ed25519 key
- Fallback to first available verified key
- Support for custom selection logic

### A2ASigner

Sign HTTP messages for A2A communication:

- Sign requests with agent's private key
- Include DID in signature parameters
- RFC9421 compliant signatures
- Support both ECDSA and Ed25519

### AgentCardSigner

Sign and verify A2A Agent Cards with comprehensive metadata:

- **Builder Pattern**: Fluent API for creating Agent Cards
- **JWS Signatures**: JSON Web Signature (JWS) compact serialization
- **DID Integration**: Agent Cards include blockchain-anchored DIDs
- **Metadata Support**: Capabilities, public keys, expiration, custom metadata
- **Verification**: Cryptographic verification with DID resolution
- **Validation**: Built-in methods for checking expiration and capabilities

## Installation

```bash
go get github.com/sage-x-project/sage-a2a-go
```

## Quick Start

### 1. Register Agent with DID

```go
package main

import (
    "context"
    "fmt"

    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func main() {
    // Setup Ethereum client
    config := &did.RegistryConfig{
        ContractAddress: "0x...",
        RPCEndpoint:     "https://ethereum-rpc-url",
        PrivateKey:      "your-private-key",
    }

    client, _ := ethereum.NewEthereumClientV4(config)

    // Generate keys
    ecdsaKey, _ := crypto.GenerateSecp256k1KeyPair()
    ed25519Key, _ := crypto.GenerateEd25519KeyPair()

    // Register agent with multi-key
    agentDID := did.AgentDID("did:sage:ethereum:0x...")
    // ... registration logic ...
}
```

### 2. Sign HTTP Request with DID

```go
import (
    "net/http"
    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
)

func signRequest() {
    // Create signer
    a2aSigner := signer.NewA2ASigner()

    // Create request
    req, _ := http.NewRequest("POST", "https://agent.example.com/task", nil)

    // Sign with DID
    ctx := context.Background()
    agentDID := did.AgentDID("did:sage:ethereum:0x...")
    err := a2aSigner.SignRequest(ctx, req, agentDID, keyPair)
}
```

### 3. Verify HTTP Signature

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
)

func verifyRequest(req *http.Request) error {
    // Create verifier
    didVerifier := verifier.NewDIDVerifier(client)

    // Extract DID from request
    agentDID := extractDIDFromRequest(req)

    // Verify signature
    ctx := context.Background()
    return didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
}
```

## Usage

### KeySelector

Select appropriate key based on protocol:

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/verifier"

// Create selector
selector := verifier.NewDefaultKeySelector(client)

// Select key for Ethereum
pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "ethereum")
// Returns ECDSA key

// Select key for Solana
pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "solana")
// Returns Ed25519 key
```

### Agent Card

Create and sign Agent Cards using the fluent builder API:

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/protocol"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

// Create Agent Card with builder pattern
agentDID := did.AgentDID("did:sage:ethereum:0x...")
card := protocol.NewAgentCardBuilder(agentDID, "MyAgent", "https://agent.example.com").
    WithDescription("AI Agent with DID authentication").
    WithCapabilities("task.create", "task.execute", "messaging.send").
    WithMetadata("region", "us-west-2").
    WithExpiresAt(time.Now().Add(365 * 24 * time.Hour)).
    Build()

// Validate Agent Card
if err := card.Validate(); err != nil {
    log.Fatal(err)
}

// Create signer with DID resolution client
cardSigner := protocol.NewDefaultAgentCardSigner(client)

// Sign Agent Card with JWS
ctx := context.Background()
signedCard, err := cardSigner.SignAgentCard(ctx, card, keyPair)
if err != nil {
    log.Fatal(err)
}

// Verify Agent Card signature (resolves public key from DID)
err = cardSigner.VerifyAgentCard(ctx, signedCard)
if err != nil {
    log.Fatal(err)
}

// Check capabilities
if card.HasCapability("task.execute") {
    fmt.Println("Agent can execute tasks")
}

// Check expiration
if card.IsExpired() {
    fmt.Println("Agent Card has expired")
}
```

## Development

### Prerequisites

- Go 1.23 or higher
- SAGE v1.1.0
- Local Ethereum testnet (for testing)

### Building

```bash
go build ./...
```

### Testing

Run unit tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

Run integration tests:

```bash
go test ./test/integration/... -tags=integration
```

### Test Coverage

This project maintains **90%+ test coverage**. All code is developed using TDD (Test-Driven Development).

## Documentation

- [Design Documentation](docs/design.md)
- [API Reference](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go)
- [Integration Guide](SAGE_A2A_INTEGRATION_GUIDE.md)
- [A2A Migration Guide](A2A_MIGRATION_GUIDE.md)

## Project Structure

```
sage-a2a-go/
├── cmd/
│   └── example/          # Example applications
├── docs/                 # Documentation
│   ├── Todo.md          # Development tasks
│   └── design.md        # Design documentation
├── pkg/
│   ├── verifier/        # DIDVerifier and KeySelector
│   ├── signer/          # A2ASigner and AgentCardSigner
│   └── protocol/        # A2A protocol integration
├── test/
│   └── integration/     # Integration tests
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`feature/your-feature`)
3. Write tests first (TDD approach)
4. Implement the feature
5. Ensure test coverage ≥ 90%
6. Commit with conventional commit messages
7. Create a Pull Request

### Commit Message Format

```
<type>: <subject>

Types: feat, fix, test, refactor, docs, chore
Example: feat: implement DIDVerifier with RFC9421 support
```

## Security

### Threat Model

- **DID Spoofing**: Prevented by blockchain-anchored DIDs
- **Replay Attacks**: Prevented by timestamp validation
- **Man-in-the-Middle**: Prevented by TLS 1.3+ and HTTP signatures
- **Key Compromise**: Mitigated by multi-key support and key rotation

### Best Practices

- Always use HTTPS/TLS 1.3+
- Validate timestamps in signatures
- Rotate keys regularly
- Use mTLS for agent-to-agent communication
- Verify Agent Card signatures before trusting

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## References

- [SAGE Project](https://github.com/sage-x-project/sage)
- [A2A Protocol](https://a2a-protocol.org)
- [A2A Go SDK](https://github.com/a2aproject/a2a-go)
- [RFC9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [DID Core Specification](https://www.w3.org/TR/did-core/)

## Contact

- **GitHub**: [sage-x-project/sage-a2a-go](https://github.com/sage-x-project/sage-a2a-go)
- **Issues**: [Report a bug](https://github.com/sage-x-project/sage-a2a-go/issues)
- **SAGE Project**: [sage-x-project](https://github.com/sage-x-project)

---

**Built with ❤️ by the SAGE Development Team**

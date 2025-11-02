# SAGE Agent Framework

High-level framework for building SAGE protocol agents without directly importing sage.

## Features

- **Zero Sage Imports**: No direct sage package imports in agent code
- **83% Code Reduction**: 165 lines → 10 lines for initialization
- **Business Logic Focus**: Crypto/DID/HPKE handled by framework
- **Environment-based Configuration**: Easy deployment with env vars
- **Full HPKE Support**: Built-in encrypted communication
- **DID Authentication**: Automatic signature verification

## Quick Start

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework"

// Create agent from environment variables
agent, err := framework.NewAgentFromEnv(
    "payment",  // name
    "PAYMENT",  // env prefix
    true,       // HPKE enabled
    true,       // require signature
)
if err != nil {
    log.Fatal(err)
}

// Use agent's HTTP server
server := agent.GetHTTPServer()

// Create HPKE client for sending messages
transport := prototx.NewA2ATransport(...)
client, err := agent.CreateHPKEClient(transport)
```

## Environment Variables

The framework uses environment variables for configuration:

### Agent-specific Variables

- `{PREFIX}_DID`: Agent's decentralized identifier (required)
- `{PREFIX}_JWK_FILE`: Path to signing key JWK file (required)
- `{PREFIX}_KEM_JWK_FILE`: Path to KEM key JWK file (required if HPKE enabled)

### Shared Variables

- `ETH_RPC_URL`: Ethereum RPC endpoint (default: http://127.0.0.1:8545)
- `SAGE_REGISTRY_ADDRESS`: Registry contract address (default: 0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512)
- `SAGE_EXTERNAL_KEY`: Operator private key (default: test key)

## Example: Payment Agent

```go
package main

import (
    "log"
    "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework"
)

type PaymentAgent struct {
    agent  *framework.Agent
    logger *log.Logger
}

func NewPaymentAgent() (*PaymentAgent, error) {
    // This replaces ~165 lines of initialization code
    agent, err := framework.NewAgentFromEnv(
        "payment", "PAYMENT", true, true,
    )
    if err != nil {
        return nil, err
    }

    return &PaymentAgent{
        agent:  agent,
        logger: log.New(os.Stdout, "[payment] ", log.LstdFlags),
    }, nil
}

func (p *PaymentAgent) Start() error {
    // Get HTTP server and mount it
    server := p.agent.GetHTTPServer()
    // ... mount server on your HTTP router
}
```

## Architecture

The framework consists of five core packages:

### 1. keys

Manages cryptographic key loading and storage.

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/keys"

// Load signing key
signKey, err := keys.LoadFromJWKFile("/path/to/signing.jwk")

// Load complete key set
keySet, err := keys.LoadKeySet(keys.KeyConfig{
    SigningKeyFile: "/path/to/signing.jwk",
    KEMKeyFile:     "/path/to/kem.jwk",
})
```

### 2. session

Manages HPKE encryption sessions for stateful communication.

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/session"

sessionMgr := session.NewManager()
```

### 3. did

Handles DID resolution and public key retrieval.

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/did"

resolver, err := did.NewResolver(did.Config{
    RPCEndpoint:     "http://127.0.0.1:8545",
    ContractAddress: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
    PrivateKey:      "0x...",
})
```

### 4. middleware

Provides HTTP middleware for DID authentication.

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/middleware"

mw, err := middleware.NewDIDAuth(middleware.Config{
    Resolver: resolver,
    Optional: false, // require signature
})
```

### 5. hpke

Manages HPKE encryption for secure agent-to-agent communication.

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent/framework/hpke"

// Server side
hpkeServer, err := hpke.NewServer(hpke.ServerConfig{
    SigningKey:     keySet.SigningKey,
    KEMKey:         keySet.KEMKey,
    DID:            agentDID,
    Resolver:       resolver,
    SessionManager: sessionMgr,
})

// Client side
hpkeClient, err := hpke.NewClient(hpke.ClientConfig{
    Transport:      transport,
    Resolver:       resolver,
    SigningKey:     keySet.SigningKey,
    ClientDID:      agentDID,
    SessionManager: sessionMgr,
})
```

## Code Reduction Example

### Before (without framework)

```go
// agents/payment/agent.go - Current implementation
func (e *PaymentAgent) ensureHPKE() error {
    // 165 lines of initialization code
    sigPath := os.Getenv("PAYMENT_JWK_FILE")
    raw, err := os.ReadFile(sigPath)
    if err != nil {
        return fmt.Errorf("read signing key: %w", err)
    }

    signKP, err := formats.NewJWKImporter().Import(raw, crypto.KeyFormatJWK)
    if err != nil {
        return fmt.Errorf("import signing key: %w", err)
    }

    // ... 160 more lines ...
}
```

### After (with framework)

```go
func NewPaymentAgent() (*PaymentAgent, error) {
    agent, err := framework.NewAgentFromEnv(
        "payment", "PAYMENT", true, true,
    )
    if err != nil {
        return nil, err
    }

    return &PaymentAgent{agent: agent}, nil
}
```

## Migration from sage-multi-agent

This framework was migrated from `sage-multi-agent/internal/agent` to provide
a stable, production-ready API for building SAGE protocol agents.

Key differences:
- Package path: `sage-multi-agent/internal/agent` → `sage-a2a-go/pkg/agent/framework`
- All imports updated to sage-a2a-go
- Ready for production use

## Documentation

For more detailed documentation, see:
- [AGENT_FRAMEWORK_DESIGN.md](../../../docs/AGENT_FRAMEWORK_DESIGN.md) - Design principles
- [AGENT_FRAMEWORK_MIGRATION_GUIDE.md](../../../AGENT_FRAMEWORK_MIGRATION_GUIDE.md) - Migration guide
- [API Documentation](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go/pkg/agent/framework)

## License

Copyright (C) 2025 SAGE-X Project

This package is part of sage-a2a-go and is licensed under the GNU Lesser General Public License v3.0 (LGPL-3.0).

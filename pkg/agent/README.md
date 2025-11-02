# SAGE Agent Framework

High-level framework for building SAGE protocol agents without directly importing sage packages.

## Features

- **Zero Direct Sage Imports**: Agent code doesn't import sage packages directly
- **83% Code Reduction**: Initialize agents in 10 lines instead of 165 lines
- **Production-Ready Patterns**: Eager HPKE, Lazy HPKE, Framework Helpers
- **Consistent Error Handling**: All errors include context
- **Comprehensive Abstractions**: Keys, DID, Session, Middleware, HPKE

## Quick Start

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/agent"

// Create agent from environment variables
fwAgent, err := agent.NewAgentFromEnv(
    "payment",  // agent name
    "PAYMENT",  // env var prefix
    true,       // HPKE enabled
    true,       // require signature verification
)
if err != nil {
    return fmt.Errorf("create agent: %w", err)
}

// Use the agent
if fwAgent.GetHTTPServer() != nil {
    http.Handle("/process", fwAgent.GetHTTPServer().MessagesHandler())
}
```

## Environment Variables

Required environment variables (with `PAYMENT` prefix example):

- `PAYMENT_JWK_FILE`: Path to signing key (secp256k1 JWK)
- `PAYMENT_KEM_JWK_FILE`: Path to KEM key (X25519 JWK, for HPKE)
- `PAYMENT_DID`: Agent DID (optional, auto-derived from key)
- `ETH_RPC_URL`: Ethereum RPC endpoint
- `SAGE_REGISTRY_ADDRESS`: SAGE registry contract address
- `SAGE_EXTERNAL_KEY`: Operator private key for registry

## Usage Patterns

### Eager HPKE Pattern (Production)

Best for production servers that always use HPKE:

```go
func NewPaymentAgent() (*PaymentAgent, error) {
    // Framework agent with HPKE initialized at startup
    fwAgent, err := agent.NewAgentFromEnv("payment", "PAYMENT", true, true)
    if err != nil {
        return nil, err
    }

    return &PaymentAgent{agent: fwAgent}, nil
}

func (pa *PaymentAgent) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // HPKE is ready to use
    pa.agent.GetHTTPServer().MessagesHandler().ServeHTTP(w, r)
}
```

### Framework Helpers Pattern

Use individual framework components:

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/agent/keys"
    "github.com/sage-x-project/sage-a2a-go/pkg/agent/did"
)

// Load keys
kp, err := keys.LoadFromJWKFile("/path/to/key.jwk")

// Create resolver
resolver, err := did.NewResolverFromEnv()
```

## Architecture

```
pkg/agent/
├── agent.go          - Main Agent type and constructors
├── keys/             - Key loading and management
├── did/              - DID resolver and registry
├── session/          - Session management
├── middleware/       - HTTP DID authentication
└── hpke/             - HPKE server/client wrappers
```

## Documentation

- See `examples/agent_framework_payment.go` for complete example
- See sage-multi-agent repository for real-world usage in 4 agents

## Benefits

Compared to direct sage imports:

- **18 fewer imports** across 4 agents
- **350+ fewer lines** of boilerplate
- **Consistent patterns** across all agents
- **Easier testing** with clear abstractions
- **Better maintainability** with centralized crypto logic

## License

Same as sage-a2a-go parent project.

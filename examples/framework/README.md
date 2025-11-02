# SAGE Agent Framework Examples

This directory contains example code demonstrating how to use the SAGE Agent Framework.

## Examples

### payment_agent.go

A complete example of a payment processing agent that:
- Uses the framework for initialization (10 lines vs 165 lines without framework)
- Handles encrypted HPKE messages automatically
- Focuses purely on business logic (payment processing)
- Demonstrates proper error handling and logging

## Running the Examples

### Prerequisites

1. Set up environment variables:

```bash
# Agent-specific
export PAYMENT_DID="did:sage:ethereum:payment"
export PAYMENT_JWK_FILE="/path/to/payment_signing.jwk"
export PAYMENT_KEM_JWK_FILE="/path/to/payment_kem.jwk"

# Shared (optional - defaults provided)
export ETH_RPC_URL="http://127.0.0.1:8545"
export SAGE_REGISTRY_ADDRESS="0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512"
export SAGE_EXTERNAL_KEY="0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
```

2. Run the example:

```bash
cd examples/framework
go run payment_agent.go
```

## Key Learning Points

### 1. Simplified Initialization

**Without framework (165 lines):**
```go
func (e *PaymentAgent) ensureHPKE() error {
    sigPath := os.Getenv("PAYMENT_JWK_FILE")
    raw, err := os.ReadFile(sigPath)
    // ... 160 more lines of crypto/DID/HPKE setup
}
```

**With framework (10 lines):**
```go
agent, err := framework.NewAgentFromEnv(
    "payment", "PAYMENT", true, true,
)
```

### 2. Zero Direct Sage Imports

The framework handles all sage imports internally. Your agent code only imports:
- `github.com/sage-x-project/sage-a2a-go/pkg/agent/framework`
- `github.com/sage-x-project/sage/pkg/agent/transport` (only for types)

### 3. Pure Business Logic

Message handlers receive pre-decrypted, pre-verified messages:
```go
func (p *PaymentAgent) HandleMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // msg.Payload is already decrypted
    // msg.SenderDID is already verified

    // Focus on business logic only
    var payment PaymentRequest
    json.Unmarshal(msg.Payload, &payment)

    // Process payment...
    return &transport.Response{Success: true, Data: result}, nil
}
```

### 4. Environment-based Configuration

All configuration comes from environment variables, making deployment easy:
- Agent-specific vars use prefix (e.g., `PAYMENT_DID`, `MEDICAL_DID`)
- Shared vars are reused (e.g., `ETH_RPC_URL`)
- Defaults provided for development

## Code Metrics

### Before (without framework)
- **Total lines**: 686
- **Initialization**: 165 lines
- **Sage imports**: 7 direct imports
- **Maintenance burden**: High (crypto details exposed)

### After (with framework)
- **Total lines**: ~150 (78% reduction)
- **Initialization**: 10 lines (94% reduction)
- **Sage imports**: 0 direct imports (100% elimination)
- **Maintenance burden**: Low (framework handles crypto)

## Next Steps

1. **Customize the example**: Modify `payment_agent.go` to implement your business logic
2. **Add more agents**: Create medical, legal, or other specialized agents
3. **Deploy to production**: Use Docker/Kubernetes with environment variables
4. **Monitor and scale**: Framework provides consistent patterns across agents

## Documentation

- [Framework README](../../pkg/agent/framework/README.md) - Framework overview
- [API Documentation](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go/pkg/agent/framework) - Complete API reference
- [Design Document](../../docs/AGENT_FRAMEWORK_DESIGN.md) - Architecture and design decisions
- [Migration Guide](../../AGENT_FRAMEWORK_MIGRATION_GUIDE.md) - Migrating existing agents

## Support

For questions or issues with the examples:
1. Check the framework documentation
2. Review the sage-multi-agent prototype at `github.com/sage-x-project/sage-multi-agent/internal/agent`
3. Open an issue on GitHub

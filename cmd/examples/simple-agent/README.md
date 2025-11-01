# Simple Agent

Basic example demonstrating how to create an agent with DID authentication.

## Features

- ✅ **DID Identity** - Create agent with blockchain-anchored identity
- ✅ **Agent Card** - Generate and sign agent card with JWS
- ✅ **Key Management** - Protocol-aware key selection
- ✅ **Card Validation** - Verify agent card signatures

## Usage

```bash
go run ./cmd/examples/simple-agent/main.go
```

## What It Shows

1. Create DID for the agent
2. Generate cryptographic key pairs
3. Create and sign agent card
4. Verify agent card signature
5. Display agent card with expiry information

## See Also

- [Multi-Key Agent](../multi-key-agent/) - Multiple protocol keys
- [Agent Communication](../agent-communication/) - Agent-to-agent communication
- [API Reference](../../../docs/API_REFERENCE.md)

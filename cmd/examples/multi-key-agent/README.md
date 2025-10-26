# Multi-Key Agent

Example demonstrating multi-protocol key management for agents.

## Features

- ✅ **Multiple Keys** - Support for Ed25519, ECDSA, and P-256 keys
- ✅ **Protocol-Aware** - Different keys for different blockchain protocols
- ✅ **Key Rotation** - Up to 10 keys per agent
- ✅ **Automatic Selection** - Protocol-based key selection

## Usage

```bash
go run ./cmd/examples/multi-key-agent/main.go
```

## What It Shows

1. Register multiple keys for different protocols
2. Select appropriate key based on protocol
3. Manage key lifecycle
4. Demonstrate cross-protocol compatibility

## Supported Protocols

- **Ethereum**: ECDSA (secp256k1)
- **Solana**: Ed25519
- **General**: P-256 (ECDSA)

## See Also

- [Simple Agent](../simple-agent/) - Basic agent example
- [Agent Communication](../agent-communication/) - Multi-agent communication
- [Integration Guide](../../../docs/INTEGRATION_GUIDE.md)

# Agent Communication

Example demonstrating agent-to-agent communication with DID authentication.

## Features

- ✅ **Mock Blockchain** - In-memory blockchain for testing
- ✅ **Agent-to-Agent** - Direct agent communication
- ✅ **DID Resolution** - Resolve DIDs from mock blockchain
- ✅ **Request/Response** - Full request-response cycle

## Usage

```bash
go run ./cmd/examples/agent-communication/main.go
```

## What It Shows

1. Create two agents with different DIDs
2. Register agents on mock blockchain
3. Agent A sends request to Agent B
4. Agent B processes and responds
5. Full DID verification flow

## Components

- **Agent**: DID-authenticated agent
- **MockBlockchain**: In-memory DID registry
- **HTTP Server**: Simple A2A agent server
- **Client**: DID-authenticated client

## See Also

- [Simple Agent](../simple-agent/) - Basic agent setup
- [Simple Client](../simple-client/) - Basic client usage
- [Architecture](../../../docs/ARCHITECTURE.md)

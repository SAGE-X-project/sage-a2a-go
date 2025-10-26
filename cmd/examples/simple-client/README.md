# Simple Client

Basic example demonstrating how to create a DID-authenticated A2A client.

## Features

-  **DID Authentication** - Create client with blockchain-anchored identity
-  **Agent Card** - Configure target agent connection
-  **Message Sending** - Send messages to A2A agents
-  **Simple API** - Minimal code to get started

## Usage

```bash
go run ./cmd/examples/simple-client/main.go
```

## Example Output

```
SAGE A2A Go - Simple Client Example
=====================================

1. Generating client DID and key pair...
   Client DID: did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678

2. Creating agent card for target agent...
   Target Agent: Example Agent
   Target URL: https://agent.example.com

3. Creating DID-authenticated A2A client...
   Client created successfully!

4. Creating a message...
   Message: Hello from SAGE A2A Go!

5. Sending message to agent...
   (Note: This will fail without a real A2A server running)
      Expected error (no server running): ...

 Example completed successfully!

To test with a real server:
  1. Start an A2A-compliant agent server
  2. Update targetCard.URL to point to your server
  3. Run this example again
```

## Code Walkthrough

### 1. Generate Client Identity

```go
// Generate key pair for signing requests
keyPair, err := crypto.GenerateSecp256k1KeyPair()
if err != nil {
    log.Fatalf("Failed to generate key pair: %v", err)
}

// Create DID for the client
clientDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef...")
```

### 2. Configure Target Agent

```go
// Create agent card for the target agent
targetCard := &a2a.AgentCard{
    Name:               "Example Agent",
    Description:        "An example A2A agent",
    URL:                "https://agent.example.com",
    PreferredTransport: a2a.TransportProtocolJSONRPC,
    AdditionalInterfaces: []a2a.AgentInterface{
        {
            Transport: a2a.TransportProtocolJSONRPC,
            URL:       "https://agent.example.com/rpc",
        },
    },
}
```

### 3. Create Client

```go
// Create DID-authenticated A2A client
client, err := transport.NewDIDAuthenticatedClient(
    ctx,
    clientDID,
    keyPair,
    targetCard,
)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Destroy()
```

### 4. Send Message

```go
// Create a message
message := a2a.NewMessage(
    a2a.MessageRoleUser,
    &a2a.TextPart{Text: "Hello from SAGE A2A Go!"},
)

// Send message to agent
params := &a2a.MessageSendParams{
    Message: message,
}

result, err := client.SendMessage(ctx, params)
```

## Production Usage

For production use, you would:

1. **Load DID and Keys Securely**
   ```go
   // Load from secure storage, not hardcoded
   clientDID := loadDIDFromVault()
   keyPair := loadKeyFromSecureStorage()
   ```

2. **Fetch Agent Card from Registry**
   ```go
   // Fetch from a registry service
   targetCard, err := fetchAgentCard("agent-name")
   ```

3. **Handle Errors Properly**
   ```go
   result, err := client.SendMessage(ctx, params)
   if err != nil {
       // Implement retry logic, logging, etc.
       return handleError(err)
   }
   ```

4. **Set Timeouts**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

## Next Steps

- See [chat-demo](../chat-demo/) for interactive chat example
- See [sse-streaming](../sse-streaming/) for real-time streaming
- See [agent-communication](../agent-communication/) for agent-to-agent communication

## See Also

- [API Reference](../../../docs/API_REFERENCE.md)
- [Integration Guide](../../../docs/INTEGRATION_GUIDE.md)
- [A2A Protocol](https://github.com/a2aproject/a2a)

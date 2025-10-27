# Chat Demo

Simple chat application demonstrating real-time SSE streaming.

## Features

- ✅ **Real-time streaming** - Messages stream as they're generated
- ✅ **Context preservation** - Maintains conversation context
- ✅ **Simple CLI interface** - Easy to use
- ✅ **Demo mode** - Preset messages for testing

## Usage

```bash
go run ./cmd/examples/chat-demo/main.go
```

## Demo Output

```
╔═══════════════════════════════════════════════════════════╗
║         A2A Chat Demo - sage-a2a-go                     ║
╚═══════════════════════════════════════════════════════════╝

Real-time streaming chat powered by SSE

Commands:
  /quit     - Exit the chat
  /help     - Show this help
  <message> - Chat with the assistant

────────────────────────────────────────────────────────────

🎬 Demo Mode: Sending preset messages...

👤 You: Hello! Can you help me?

🤖 Assistant: Hello! I'm here to help. What can I do for you today?

👤 You: What can you do?

🤖 Assistant: I can assist with various tasks like answering questions,
providing information, helping with code, and having conversations!

👤 You: Tell me a joke

🤖 Assistant: Why do programmers prefer dark mode? Because light
attracts bugs! 😄

────────────────────────────────────────────────────────────

💡 Demo completed!

To enable interactive mode:
1. Uncomment the interactive loop below
2. Provide real agent credentials
3. Run the program
```

## Interactive Mode

To enable interactive chat:

1. Open `main.go`
2. Uncomment the interactive loop at the bottom
3. Provide real agent credentials:
   ```go
   myDID := did.AgentDID(os.Getenv("AGENT_DID"))
   keyPair, _ := loadKeyFromFile("agent-key.pem")

   targetCard, _ := fetchAgentCard("https://agent.example.com")
   ```
4. Run the program

## Code Structure

### ChatSession
Manages conversation state:
```go
type ChatSession struct {
    transport *transport.DIDHTTPTransport
    contextID string  // Maintains conversation context
    taskID    a2a.TaskID  // Tracks current task
}
```

### Streaming Messages
Real-time response handling:
```go
for event, err := range s.transport.SendStreamingMessage(ctx, params) {
    switch e := event.(type) {
    case *a2a.Message:
        // Stream text as it arrives
        for _, part := range e.Parts {
            if textPart, ok := part.(*a2a.TextPart); ok {
                fmt.Print(textPart.Text)
            }
        }
    case *a2a.Task:
        // Save task for context
        s.taskID = e.ID
    case *a2a.TaskStatusUpdateEvent:
        // Check if done
        if e.Status.State.Terminal() {
            return nil
        }
    }
}
```

## Commands

| Command | Description |
|---------|-------------|
| `/quit` | Exit the chat |
| `/help` | Show help |
| Any text | Send message to assistant |

## Features Demonstrated

1. **SSE Streaming** - Real-time message delivery
2. **Context Management** - Conversation continuity
3. **Task Tracking** - Maintain task state
4. **Error Handling** - Graceful error recovery
5. **Simple UI** - Clean CLI interface

## Production Considerations

For production use:

1. **Authentication**
   ```go
   // Load credentials securely
   myDID := loadDIDFromVault()
   keyPair := loadKeyFromSecureStorage()
   ```

2. **Agent Discovery**
   ```go
   // Fetch from registry
   targetCard := fetchFromRegistry(agentName)
   ```

3. **Error Recovery**
   ```go
   // Implement retry logic
   for attempt := 0; attempt < maxRetries; attempt++ {
       err := session.SendMessage(input)
       if err == nil {
           break
       }
   }
   ```

4. **Rate Limiting**
   ```go
   // Prevent spam
   limiter := rate.NewLimiter(rate.Every(time.Second), 5)
   limiter.Wait(ctx)
   ```

5. **Logging**
   ```go
   // Structured logging
   logger.Info("message sent", "id", msgID, "context", contextID)
   ```

## See Also

- [SSE Streaming Guide](../../../docs/SSE_STREAMING_GUIDE.md)
- [SSE Example](../sse-streaming/) - More SSE features
- [API Reference](../../../docs/API_REFERENCE.md)

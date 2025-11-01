# SSE Streaming Example

Demonstrates real-time Server-Sent Events (SSE) streaming with sage-a2a-go.

## Features

This example shows:
- ✅ **Real-time message streaming** with `SendStreamingMessage`
- ✅ **Handling all 4 A2A event types** (Message, Task, StatusUpdate, ArtifactUpdate)
- ✅ **Progress tracking** during streaming
- ✅ **Graceful cancellation** with Ctrl+C
- ✅ **Error recovery** patterns
- ✅ **Task reconnection** with `ResubscribeToTask`

## Usage

```bash
go run ./cmd/examples/sse-streaming/main.go
```

## What It Does

### Example 1: Streaming Conversation
Sends a message and streams the response in real-time:
```
Example 1: Streaming Conversation
============================================================
Streaming conversation...
============================================================

[TASK] task-abc-123
  Status: working
  Context: ctx-456

[MESSAGE] msg-1
  Once upon a time, there was a little robot...

[STATUS] completed

============================================================
Task completed in 12s
Messages received: 3
Artifacts created: 0
============================================================
```

### Example 2: Error Handling
Demonstrates timeout handling:
```
Example 2: Error Handling
============================================================

Event: *a2a.Task
Event: *a2a.Message
✓ Timeout detected (expected)
```

### Example 3: Task Reconnection (Optional)
Reconnect to an existing task to receive missed events:
```
Example 3: Task Reconnection
============================================================
Reconnecting to task: task-abc-123
============================================================

Receiving backfill events...
[MESSAGE] msg-backfill-1
  (Message received while disconnected)
```

## Code Highlights

### Progress Tracking
```go
type ProgressTracker struct {
    startTime time.Time
    messages  int
    artifacts int
    status    a2a.TaskState
}

func (p *ProgressTracker) HandleEvent(event a2a.Event) {
    switch e := event.(type) {
    case *a2a.Message:
        p.messages++
        p.printMessage(e)
    case *a2a.Task:
        p.status = e.Status.State
        p.printTask(e)
    case *a2a.TaskStatusUpdateEvent:
        p.status = e.Status.State
        p.printStatusUpdate(e)
    case *a2a.TaskArtifactUpdateEvent:
        p.artifacts++
        p.printArtifact(e)
    }
}
```

### Streaming Loop
```go
for event, err := range httpTransport.SendStreamingMessage(ctx, params) {
    if err != nil {
        if errors.Is(err, context.Canceled) {
            fmt.Println("Stream cancelled by user")
            return nil
        }
        return fmt.Errorf("stream error: %w", err)
    }

    tracker.HandleEvent(event)

    // Exit if task completed
    if tracker.status.Terminal() {
        break
    }
}
```

### Graceful Cancellation
```go
// Setup context with cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// Handle interrupt signal
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt)
go func() {
    <-sigChan
    fmt.Println("\n\nCancelling stream...")
    cancel()
}()
```

## Event Types

The example handles all 4 A2A event types:

| Event Type | Purpose | Example Output |
|-----------|---------|----------------|
| `*a2a.Message` | Message from agent | `[MESSAGE] msg-1` |
| `*a2a.Task` | Task created/updated | `[TASK] task-123` |
| `*a2a.TaskStatusUpdateEvent` | Status change | `[STATUS] completed` |
| `*a2a.TaskArtifactUpdateEvent` | Artifact created | `[ARTIFACT] output.txt` |

## Customization

### Change the Prompt
```go
prompt := "Your custom prompt here"
```

### Adjust Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
```

### Test Reconnection
Uncomment the reconnection example:
```go
taskID := a2a.TaskID("task-123-from-previous-run")
if err := demonstrateReconnection(httpTransport, taskID); err != nil {
    log.Printf("Reconnection error: %v", err)
}
```

## Production Usage

This is a demonstration. In production:

1. **Load Credentials Securely**
```go
// Don't generate keys - load from secure storage
keyPair, err := loadKeyFromVault()
myDID := did.AgentDID(os.Getenv("AGENT_DID"))
```

2. **Fetch Agent Card from Registry**
```go
targetCard, err := fetchAgentCard("agent-name")
```

3. **Validate Agent Card Signature**
```go
if err := verifier.VerifyAgentCard(targetCard); err != nil {
    return err
}
```

4. **Implement Retry Logic**
```go
for attempt := 0; attempt < maxRetries; attempt++ {
    err := streamConversation(client, prompt)
    if err == nil {
        break
    }
    time.Sleep(backoff)
}
```

## See Also

- [SSE Streaming Guide](../../../docs/SSE_STREAMING_GUIDE.md) - Complete streaming guide
- [API Reference](../../../docs/API_REFERENCE.md) - Full API documentation
- [Integration Guide](../../../docs/INTEGRATION_GUIDE.md) - Integration tutorial

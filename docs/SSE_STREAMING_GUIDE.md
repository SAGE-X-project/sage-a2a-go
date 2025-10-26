# SSE Streaming Guide

Complete guide to using Server-Sent Events (SSE) streaming with sage-a2a-go.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [SendStreamingMessage](#sendstreamingmessage)
- [ResubscribeToTask](#resubscribetotask)
- [Event Types](#event-types)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Advanced Usage](#advanced-usage)

## Overview

SSE (Server-Sent Events) streaming enables real-time, one-way communication from the A2A server to your client. This is essential for:

- **Real-time message responses** - Stream assistant responses as they're generated
- **Task progress updates** - Monitor task status changes in real-time
- **Artifact updates** - Receive incremental file/data updates
- **Reconnection** - Resume streams after network interruptions

### Why SSE?

- ✅ **W3C Standard** - Browser and HTTP library support
- ✅ **Simple Protocol** - Text-based, easy to debug
- ✅ **Automatic Reconnection** - Built-in resilience
- ✅ **Firewall Friendly** - Standard HTTP
- ✅ **DID Authenticated** - Every request signed with your agent's DID

## Quick Start

### Basic Streaming Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/a2aproject/a2a-go/a2a"
    "github.com/sage-x-project/sage-a2a-go/pkg/transport"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
)

func main() {
    ctx := context.Background()

    // Setup your agent identity
    myDID := did.AgentDID("did:sage:ethereum:0x...")
    myKeyPair, _ := crypto.GenerateSecp256k1KeyPair()

    // Create client
    client, err := transport.NewDIDAuthenticatedClient(
        ctx,
        myDID,
        myKeyPair,
        targetAgentCard,
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Destroy()

    // Send streaming message
    params := &a2a.MessageSendParams{
        Message: &a2a.Message{
            Role:  a2a.MessageRoleUser,
            Parts: []a2a.Part{&a2a.TextPart{Text: "Tell me a story"}},
        },
    }

    // Process streaming events
    for event, err := range client.SendStreamingMessage(ctx, params) {
        if err != nil {
            log.Printf("Stream error: %v", err)
            break
        }

        switch e := event.(type) {
        case *a2a.Message:
            fmt.Printf("Message: %s\n", e.ID)
        case *a2a.Task:
            fmt.Printf("Task: %s (status: %s)\n", e.ID, e.Status.State)
        case *a2a.TaskStatusUpdateEvent:
            fmt.Printf("Status update: %s\n", e.Status.State)
        case *a2a.TaskArtifactUpdateEvent:
            fmt.Printf("Artifact: %s\n", e.Artifact.Name)
        }
    }
}
```

## SendStreamingMessage

Stream messages with real-time responses from the agent.

### Method Signature

```go
func SendStreamingMessage(
    ctx context.Context,
    message *a2a.MessageSendParams
) iter.Seq2[a2a.Event, error]
```

### Parameters

- `ctx` - Context for cancellation and timeouts
- `message` - Message parameters with your prompt

### Returns

Iterator that yields `(a2a.Event, error)` pairs:
- `a2a.Event` - One of 4 event types (Message, Task, StatusUpdate, ArtifactUpdate)
- `error` - Non-nil if stream encounters an error

### Complete Example

```go
func streamConversation(client a2aclient.Client, prompt string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    params := &a2a.MessageSendParams{
        Message: &a2a.Message{
            Role:  a2a.MessageRoleUser,
            Parts: []a2a.Part{&a2a.TextPart{Text: prompt}},
        },
    }

    var currentTask *a2a.Task
    var messages []*a2a.Message

    for event, err := range client.SendStreamingMessage(ctx, params) {
        if err != nil {
            return fmt.Errorf("stream error: %w", err)
        }

        switch e := event.(type) {
        case *a2a.Message:
            // Store message
            messages = append(messages, e)

            // Extract text content
            for _, part := range e.Parts {
                if textPart, ok := part.(*a2a.TextPart); ok {
                    fmt.Printf("Assistant: %s\n", textPart.Text)
                }
            }

        case *a2a.Task:
            // Task created or updated
            currentTask = e
            fmt.Printf("Task %s: %s\n", e.ID, e.Status.State)

        case *a2a.TaskStatusUpdateEvent:
            // Status changed
            fmt.Printf("Status: %s → %s\n",
                currentTask.Status.State, e.Status.State)
            currentTask.Status = e.Status

            // Check if task completed
            if e.Status.State.Terminal() {
                fmt.Println("Task completed!")
                return nil
            }

        case *a2a.TaskArtifactUpdateEvent:
            // Artifact created/updated
            fmt.Printf("Artifact: %s (%s)\n",
                e.Artifact.Name, e.Artifact.MimeType)
        }
    }

    return nil
}
```

## ResubscribeToTask

Reconnect to an existing task's event stream after disconnection.

### Method Signature

```go
func ResubscribeToTask(
    ctx context.Context,
    id *a2a.TaskIDParams
) iter.Seq2[a2a.Event, error]
```

### Use Cases

1. **Network Interruption** - Resume after connection lost
2. **Process Restart** - Continue monitoring after restart
3. **Long-running Tasks** - Check back on tasks started earlier
4. **Backfill Events** - Receive events that occurred while disconnected

### Example: Reconnection Handler

```go
func monitorTaskWithReconnection(client a2aclient.Client, taskID a2a.TaskID) {
    ctx := context.Background()
    maxRetries := 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        fmt.Printf("Connecting to task %s (attempt %d/%d)\n",
            taskID, attempt+1, maxRetries)

        err := subscribeToTask(ctx, client, taskID)

        if err == nil {
            // Task completed successfully
            return
        }

        // Check if error is recoverable
        if !isRecoverableError(err) {
            log.Fatalf("Unrecoverable error: %v", err)
        }

        // Wait before retry
        time.Sleep(time.Second * time.Duration(attempt+1))
    }

    log.Fatal("Max retries exceeded")
}

func subscribeToTask(ctx context.Context, client a2aclient.Client, taskID a2a.TaskID) error {
    params := &a2a.TaskIDParams{ID: taskID}

    for event, err := range client.ResubscribeToTask(ctx, params) {
        if err != nil {
            return err
        }

        switch e := event.(type) {
        case *a2a.TaskStatusUpdateEvent:
            fmt.Printf("Status: %s\n", e.Status.State)

            if e.Status.State.Terminal() {
                fmt.Println("Task finished!")
                return nil
            }

        case *a2a.Message:
            fmt.Printf("Backfill message: %s\n", e.ID)

        case *a2a.TaskArtifactUpdateEvent:
            fmt.Printf("Artifact update: %s\n", e.Artifact.Name)
        }
    }

    return nil
}

func isRecoverableError(err error) bool {
    // Network errors are recoverable
    if strings.Contains(err.Error(), "connection") {
        return true
    }
    if strings.Contains(err.Error(), "timeout") {
        return true
    }
    // Authorization errors are not
    if strings.Contains(err.Error(), "401") ||
       strings.Contains(err.Error(), "403") {
        return false
    }
    return true
}
```

## Event Types

SSE streams can yield 4 different event types:

### 1. Message

Complete message from the agent with content.

```go
type Message struct {
    ID        string      // Unique message ID
    Role      MessageRole // "agent" or "user"
    Parts     []Part      // Content parts (text, files, etc.)
    ContextID string      // Conversation context
    TaskID    TaskID      // Associated task
}
```

**Example:**
```go
case *a2a.Message:
    for _, part := range e.Parts {
        if textPart, ok := part.(*a2a.TextPart); ok {
            fmt.Println(textPart.Text)
        }
    }
```

### 2. Task

Task creation or full task state.

```go
type Task struct {
    ID        TaskID     // Unique task ID
    ContextID string     // Conversation context
    Status    TaskStatus // Current status
    Messages  []*Message // Message history
    Artifacts []*Artifact // Generated artifacts
}
```

**Example:**
```go
case *a2a.Task:
    fmt.Printf("Task %s created\n", e.ID)
    fmt.Printf("Status: %s\n", e.Status.State)
```

### 3. TaskStatusUpdateEvent

Incremental status change notification.

```go
type TaskStatusUpdateEvent struct {
    TaskID TaskID     // Which task changed
    Status TaskStatus // New status
}
```

**Example:**
```go
case *a2a.TaskStatusUpdateEvent:
    if e.Status.State == a2a.TaskStateCompleted {
        fmt.Println("Task completed!")
    }
```

### 4. TaskArtifactUpdateEvent

Artifact creation or modification.

```go
type TaskArtifactUpdateEvent struct {
    TaskID   TaskID    // Which task
    Artifact *Artifact // Artifact details
}
```

**Example:**
```go
case *a2a.TaskArtifactUpdateEvent:
    fmt.Printf("New file: %s (%d bytes)\n",
        e.Artifact.Name, e.Artifact.Size)

    // Download artifact if needed
    if e.Artifact.URL != "" {
        downloadArtifact(e.Artifact.URL)
    }
```

## Error Handling

### Graceful Error Handling

```go
for event, err := range client.SendStreamingMessage(ctx, params) {
    if err != nil {
        // Check error type
        if errors.Is(err, context.Canceled) {
            fmt.Println("Stream cancelled by user")
            return nil
        }

        if errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("Stream timeout")
            return fmt.Errorf("timeout: %w", err)
        }

        // Network or server errors
        fmt.Printf("Stream error: %v\n", err)
        return err
    }

    // Process event
    handleEvent(event)
}
```

### Context Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

// Cancel on interrupt
go func() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    fmt.Println("\nCancelling stream...")
    cancel()
}()

for event, err := range client.SendStreamingMessage(ctx, params) {
    if err != nil {
        if errors.Is(err, context.Canceled) {
            fmt.Println("Stream cancelled successfully")
            return nil
        }
        return err
    }
    // Process event
}
```

### Timeout Management

```go
// Set timeout for entire conversation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// Or use deadlines
deadline := time.Now().Add(10 * time.Minute)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()
```

## Best Practices

### 1. Always Use Context

```go
// ✅ Good: Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

for event, err := range client.SendStreamingMessage(ctx, params) {
    // ...
}

// ❌ Bad: Using background context indefinitely
for event, err := range client.SendStreamingMessage(context.Background(), params) {
    // Could hang forever
}
```

### 2. Handle All Event Types

```go
// ✅ Good: Comprehensive event handling
for event, err := range stream {
    if err != nil {
        return err
    }

    switch e := event.(type) {
    case *a2a.Message:
        handleMessage(e)
    case *a2a.Task:
        handleTask(e)
    case *a2a.TaskStatusUpdateEvent:
        handleStatusUpdate(e)
    case *a2a.TaskArtifactUpdateEvent:
        handleArtifact(e)
    default:
        log.Printf("Unknown event type: %T", event)
    }
}

// ❌ Bad: Only handling some types
for event, err := range stream {
    if msg, ok := event.(*a2a.Message); ok {
        handleMessage(msg)
    }
    // Missing other event types!
}
```

### 3. Check Terminal States

```go
// ✅ Good: Exit when task completes
for event, err := range stream {
    if err != nil {
        return err
    }

    if statusUpdate, ok := event.(*a2a.TaskStatusUpdateEvent); ok {
        if statusUpdate.Status.State.Terminal() {
            fmt.Printf("Task finished: %s\n", statusUpdate.Status.State)
            break // Exit loop
        }
    }
}

// ❌ Bad: Loop continues after completion
for event, err := range stream {
    // Never exits, even when task is done
}
```

### 4. Resource Cleanup

```go
// ✅ Good: Always cleanup
client, err := transport.NewDIDAuthenticatedClient(ctx, did, key, card)
if err != nil {
    return err
}
defer client.Destroy() // Cleanup connections

// ❌ Bad: No cleanup
client, _ := transport.NewDIDAuthenticatedClient(ctx, did, key, card)
// Resource leak!
```

### 5. Error Recovery

```go
// ✅ Good: Implement retry logic
func streamWithRetry(client a2aclient.Client, params *a2a.MessageSendParams) error {
    maxRetries := 3
    backoff := time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

        err := processStream(ctx, client, params)
        cancel()

        if err == nil {
            return nil // Success
        }

        if !isRetriable(err) {
            return err // Don't retry
        }

        if attempt < maxRetries-1 {
            time.Sleep(backoff)
            backoff *= 2 // Exponential backoff
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

## Advanced Usage

### Progress Tracking

```go
type ProgressTracker struct {
    taskID      a2a.TaskID
    status      a2a.TaskState
    messages    int
    artifacts   int
    startTime   time.Time
}

func (p *ProgressTracker) HandleEvent(event a2a.Event) {
    switch e := event.(type) {
    case *a2a.Task:
        p.taskID = e.ID
        p.status = e.Status.State
        p.startTime = time.Now()

    case *a2a.Message:
        p.messages++
        fmt.Printf("\rMessages: %d | Artifacts: %d | Duration: %s",
            p.messages, p.artifacts, time.Since(p.startTime))

    case *a2a.TaskStatusUpdateEvent:
        p.status = e.Status.State
        if e.Status.State.Terminal() {
            fmt.Printf("\nTask %s: %s\n", p.taskID, e.Status.State)
        }

    case *a2a.TaskArtifactUpdateEvent:
        p.artifacts++
    }
}
```

### Stream Buffering

```go
// Buffer events for batch processing
func bufferStream(stream iter.Seq2[a2a.Event, error], bufferSize int) <-chan []a2a.Event {
    out := make(chan []a2a.Event)

    go func() {
        defer close(out)

        buffer := make([]a2a.Event, 0, bufferSize)

        for event, err := range stream {
            if err != nil {
                if len(buffer) > 0 {
                    out <- buffer
                }
                return
            }

            buffer = append(buffer, event)

            if len(buffer) >= bufferSize {
                out <- buffer
                buffer = make([]a2a.Event, 0, bufferSize)
            }
        }

        if len(buffer) > 0 {
            out <- buffer
        }
    }()

    return out
}
```

### Multi-Task Monitoring

```go
func monitorMultipleTasks(client a2aclient.Client, taskIDs []a2a.TaskID) {
    ctx := context.Background()

    // Create channels for each task
    var wg sync.WaitGroup

    for _, taskID := range taskIDs {
        wg.Add(1)

        go func(id a2a.TaskID) {
            defer wg.Done()

            params := &a2a.TaskIDParams{ID: id}

            for event, err := range client.ResubscribeToTask(ctx, params) {
                if err != nil {
                    log.Printf("Task %s error: %v", id, err)
                    return
                }

                handleTaskEvent(id, event)
            }
        }(taskID)
    }

    wg.Wait()
}
```

## Troubleshooting

### Stream Doesn't Start

**Problem:** `SendStreamingMessage` returns immediately with error.

**Solutions:**
- Check server URL is correct
- Verify DID signature is working
- Ensure agent card is valid
- Check network connectivity

### Stream Disconnects Randomly

**Problem:** Stream stops with connection error.

**Solutions:**
- Use `ResubscribeToTask` to reconnect
- Implement retry logic with exponential backoff
- Check for firewall/proxy issues
- Monitor network stability

### Events Missing

**Problem:** Not receiving expected events.

**Solutions:**
- Check all event type handlers exist
- Verify context isn't being cancelled early
- Look for errors in the error channel
- Check server is sending events correctly

### Memory Leak

**Problem:** Memory usage grows over time.

**Solutions:**
- Call `client.Destroy()` when done
- Don't accumulate all events in memory
- Process events as they arrive
- Use context cancellation properly

## See Also

- [API Reference](API_REFERENCE.md)
- [Architecture Guide](ARCHITECTURE.md)
- [Integration Guide](INTEGRATION_GUIDE.md)
- [A2A Protocol Specification](https://a2a-protocol.org)
- [W3C SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)

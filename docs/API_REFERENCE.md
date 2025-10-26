# API Reference

Complete API reference for sage-a2a-go v2.0.0.

## Table of Contents

- [Transport Package](#transport-package)
  - [DIDHTTPTransport](#didhttptransport)
  - [Factory Functions](#factory-functions)
- [Protocol Package](#protocol-package)
- [Signer Package](#signer-package)
- [Verifier Package](#verifier-package)
- [Server Package](#server-package)

---

## Transport Package

`import "github.com/sage-x-project/sage-a2a-go/pkg/transport"`

Provides DID-authenticated HTTP/JSON-RPC 2.0 transport for A2A protocol.

### DIDHTTPTransport

HTTP transport with automatic DID signatures on all requests.

#### Type Definition

```go
type DIDHTTPTransport struct {
    // private fields
}
```

#### Constructor

```go
func NewDIDHTTPTransport(
    baseURL string,
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    httpClient *http.Client,
) a2aclient.Transport
```

**Parameters:**
- `baseURL` - Base URL of the A2A agent (e.g., "https://agent.example.com")
- `agentDID` - Your agent's DID for signing requests
- `keyPair` - Your agent's private key for signing
- `httpClient` - Optional HTTP client (nil for default)

**Returns:** Transport implementing `a2aclient.Transport` interface

**Example:**
```go
transport := NewDIDHTTPTransport(
    "https://agent.example.com",
    did.AgentDID("did:sage:ethereum:0x..."),
    keyPair,
    nil, // use http.DefaultClient
)
```

---

### Methods

#### GetTask

Retrieve a specific task by ID.

```go
func (t *DIDHTTPTransport) GetTask(
    ctx context.Context,
    query *a2a.TaskQueryParams,
) (*a2a.Task, error)
```

**Parameters:**
- `ctx` - Context for cancellation
- `query` - Task query parameters with ID

**Returns:** Task object or error

**Example:**
```go
task, err := transport.GetTask(ctx, &a2a.TaskQueryParams{
    ID: "task-123",
})
```

---

#### CancelTask

Cancel an in-progress task.

```go
func (t *DIDHTTPTransport) CancelTask(
    ctx context.Context,
    id *a2a.TaskIDParams,
) (*a2a.Task, error)
```

**Parameters:**
- `ctx` - Context for cancellation
- `id` - Task ID to cancel

**Returns:** Updated task object or error

**Example:**
```go
task, err := transport.CancelTask(ctx, &a2a.TaskIDParams{
    ID: "task-123",
})
```

---

#### SendMessage

Send a message and receive response (non-streaming).

```go
func (t *DIDHTTPTransport) SendMessage(
    ctx context.Context,
    message *a2a.MessageSendParams,
) (a2a.SendMessageResult, error)
```

**Parameters:**
- `ctx` - Context for cancellation
- `message` - Message parameters

**Returns:** Either `*a2a.Task` or `*a2a.Message`

**Example:**
```go
result, err := transport.SendMessage(ctx, &a2a.MessageSendParams{
    Message: &a2a.Message{
        Role:  a2a.MessageRoleUser,
        Parts: []a2a.Part{&a2a.TextPart{Text: "Hello"}},
    },
})

// Type assertion to check result type
if task, ok := result.(*a2a.Task); ok {
    fmt.Printf("Task created: %s\n", task.ID)
} else if msg, ok := result.(*a2a.Message); ok {
    fmt.Printf("Message received: %s\n", msg.ID)
}
```

---

#### SendStreamingMessage

Send a message and stream responses in real-time.

```go
func (t *DIDHTTPTransport) SendStreamingMessage(
    ctx context.Context,
    message *a2a.MessageSendParams,
) iter.Seq2[a2a.Event, error]
```

**Parameters:**
- `ctx` - Context for cancellation
- `message` - Message parameters

**Returns:** Iterator yielding `(Event, error)` pairs

**Events:** Can yield 4 types:
- `*a2a.Message` - Message from agent
- `*a2a.Task` - Task created/updated
- `*a2a.TaskStatusUpdateEvent` - Status change
- `*a2a.TaskArtifactUpdateEvent` - Artifact update

**Example:**
```go
for event, err := range transport.SendStreamingMessage(ctx, params) {
    if err != nil {
        return err
    }

    switch e := event.(type) {
    case *a2a.Message:
        fmt.Println("Message:", e.ID)
    case *a2a.Task:
        fmt.Println("Task:", e.ID)
    case *a2a.TaskStatusUpdateEvent:
        fmt.Println("Status:", e.Status.State)
    case *a2a.TaskArtifactUpdateEvent:
        fmt.Println("Artifact:", e.Artifact.Name)
    }
}
```

**See:** [SSE Streaming Guide](SSE_STREAMING_GUIDE.md)

---

#### ResubscribeToTask

Reconnect to an existing task's event stream.

```go
func (t *DIDHTTPTransport) ResubscribeToTask(
    ctx context.Context,
    id *a2a.TaskIDParams,
) iter.Seq2[a2a.Event, error]
```

**Parameters:**
- `ctx` - Context for cancellation
- `id` - Task ID to resubscribe to

**Returns:** Iterator yielding `(Event, error)` pairs (same as SendStreamingMessage)

**Example:**
```go
for event, err := range transport.ResubscribeToTask(ctx, &a2a.TaskIDParams{
    ID: "task-123",
}) {
    if err != nil {
        return err
    }
    handleEvent(event)
}
```

**See:** [SSE Streaming Guide](SSE_STREAMING_GUIDE.md)

---

#### GetTaskPushConfig

Get push notification configuration for a task.

```go
func (t *DIDHTTPTransport) GetTaskPushConfig(
    ctx context.Context,
    params *a2a.GetTaskPushConfigParams,
) (*a2a.TaskPushConfig, error)
```

---

#### ListTaskPushConfig

List all push notification configurations.

```go
func (t *DIDHTTPTransport) ListTaskPushConfig(
    ctx context.Context,
    params *a2a.ListTaskPushConfigParams,
) ([]*a2a.TaskPushConfig, error)
```

---

#### SetTaskPushConfig

Configure push notifications for a task.

```go
func (t *DIDHTTPTransport) SetTaskPushConfig(
    ctx context.Context,
    config *a2a.TaskPushConfig,
) (*a2a.TaskPushConfig, error)
```

---

#### DeleteTaskPushConfig

Delete push notification configuration.

```go
func (t *DIDHTTPTransport) DeleteTaskPushConfig(
    ctx context.Context,
    params *a2a.DeleteTaskPushConfigParams,
) error
```

---

#### ListTasks

List and filter tasks with pagination (A2A v0.4.0).

```go
func (t *DIDHTTPTransport) ListTasks(
    ctx context.Context,
    params *protocol.ListTasksParams,
) (*protocol.ListTasksResult, error)
```

**Parameters:**
```go
type ListTasksParams struct {
    ContextID        string              // Filter by context
    Status           a2a.TaskState       // Filter by status
    PageSize         int                 // Results per page (1-100)
    PageToken        string              // Pagination token
    HistoryLength    int                 // Max history items
    LastUpdatedAfter int64               // Unix timestamp
    IncludeArtifacts bool                // Include artifacts
    Metadata         map[string]interface{} // Custom filters
}
```

**Returns:**
```go
type ListTasksResult struct {
    Tasks         []*a2a.Task // Matching tasks
    TotalSize     int         // Total count
    PageSize      int         // Results per page
    NextPageToken string      // For next page
}
```

**Example:**
```go
// First page
result, err := transport.ListTasks(ctx, &protocol.ListTasksParams{
    Status:   a2a.TaskStateWorking,
    PageSize: 50,
})

// Next page
if result.NextPageToken != "" {
    nextPage, err := transport.ListTasks(ctx, &protocol.ListTasksParams{
        PageSize:  50,
        PageToken: result.NextPageToken,
    })
}
```

---

#### GetAgentCard

Retrieve agent card from well-known URL.

```go
func (t *DIDHTTPTransport) GetAgentCard(
    ctx context.Context,
) (*a2a.AgentCard, error)
```

**Example:**
```go
card, err := transport.GetAgentCard(ctx)
fmt.Printf("Agent: %s\n", card.Name)
```

---

#### Destroy

Clean up transport resources.

```go
func (t *DIDHTTPTransport) Destroy() error
```

**Example:**
```go
defer transport.Destroy()
```

---

### Factory Functions

Convenience functions for creating clients with DID authentication.

#### NewDIDAuthenticatedClient

Create a client with DID HTTP transport.

```go
func NewDIDAuthenticatedClient(
    ctx context.Context,
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    targetCard *a2a.AgentCard,
) (a2aclient.Client, error)
```

**Parameters:**
- `ctx` - Context for initialization
- `agentDID` - Your agent's DID
- `keyPair` - Your signing key
- `targetCard` - Target agent's card

**Returns:** Configured `a2aclient.Client`

**Example:**
```go
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

// Use all a2aclient.Client methods
task, err := client.SendMessage(ctx, message)
```

---

#### NewDIDAuthenticatedClientWithConfig

Create client with custom configuration.

```go
func NewDIDAuthenticatedClientWithConfig(
    ctx context.Context,
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    targetCard *a2a.AgentCard,
    config a2aclient.Config,
) (a2aclient.Client, error)
```

**Example:**
```go
client, err := transport.NewDIDAuthenticatedClientWithConfig(
    ctx, myDID, myKeyPair, targetCard,
    a2aclient.Config{
        AcceptedOutputModes: []string{"application/json"},
        MaxRetries:          3,
    },
)
```

---

#### NewDIDAuthenticatedClientWithInterceptors

Create client with request/response interceptors.

```go
func NewDIDAuthenticatedClientWithInterceptors(
    ctx context.Context,
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    targetCard *a2a.AgentCard,
    interceptors ...a2aclient.Interceptor,
) (a2aclient.Client, error)
```

**Example:**
```go
loggingInterceptor := func(ctx context.Context, req, resp interface{}) error {
    log.Printf("Request: %+v", req)
    return nil
}

client, err := transport.NewDIDAuthenticatedClientWithInterceptors(
    ctx, myDID, myKeyPair, targetCard,
    loggingInterceptor,
)
```

---

#### WithDIDHTTPTransport

Factory option for use with `a2aclient.NewFromCard`.

```go
func WithDIDHTTPTransport(
    agentDID did.AgentDID,
    keyPair crypto.KeyPair,
    httpClient *http.Client,
) a2aclient.FactoryOption
```

**Example:**
```go
client, err := a2aclient.NewFromCard(
    ctx,
    targetCard,
    transport.WithDIDHTTPTransport(myDID, myKeyPair, nil),
    a2aclient.WithConfig(a2aclient.Config{...}),
)
```

---

## Protocol Package

`import "github.com/sage-x-project/sage-a2a-go/pkg/protocol"`

Provides A2A protocol type definitions and utilities.

### ListTasksParams

Parameters for listing tasks (A2A v0.4.0).

```go
type ListTasksParams struct {
    ContextID        string                 `json:"contextId,omitempty"`
    Status           a2a.TaskState          `json:"status,omitempty"`
    PageSize         int                    `json:"pageSize,omitempty"`
    PageToken        string                 `json:"pageToken,omitempty"`
    HistoryLength    int                    `json:"historyLength,omitempty"`
    LastUpdatedAfter int64                  `json:"lastUpdatedAfter,omitempty"`
    IncludeArtifacts bool                   `json:"includeArtifacts,omitempty"`
    Metadata         map[string]interface{} `json:"metadata,omitempty"`
}
```

### ListTasksResult

Result of listing tasks.

```go
type ListTasksResult struct {
    Tasks         []*a2a.Task `json:"tasks"`
    TotalSize     int         `json:"totalSize"`
    PageSize      int         `json:"pageSize"`
    NextPageToken string      `json:"nextPageToken"`
}
```

---

## Signer Package

`import "github.com/sage-x-project/sage-a2a-go/pkg/signer"`

Provides HTTP request signing with DID authentication.

### A2ASigner

Interface for signing HTTP requests.

```go
type A2ASigner interface {
    SignRequest(
        ctx context.Context,
        req *http.Request,
        agentDID did.AgentDID,
        keyPair crypto.KeyPair,
    ) error

    SignRequestWithOptions(
        ctx context.Context,
        req *http.Request,
        agentDID did.AgentDID,
        keyPair crypto.KeyPair,
        opts *SigningOptions,
    ) error
}
```

### DefaultA2ASigner

RFC 9421 compliant signer implementation.

```go
func NewDefaultA2ASigner() A2ASigner
```

**Example:**
```go
signer := signer.NewDefaultA2ASigner()

req, _ := http.NewRequest("POST", "https://api.example.com", body)
err := signer.SignRequest(ctx, req, myDID, myKeyPair)

// Request now has Signature and Signature-Input headers
```

### SigningOptions

Optional signing parameters.

```go
type SigningOptions struct {
    Components []string  // Components to sign
    Timestamp  int64     // Custom timestamp
    Expires    int64     // Signature expiration
    Nonce      string    // Replay prevention
}
```

---

## Verifier Package

`import "github.com/sage-x-project/sage-a2a-go/pkg/verifier"`

Provides HTTP signature verification using DIDs.

### DIDVerifier

Interface for verifying DID signatures.

```go
type DIDVerifier interface {
    VerifyHTTPSignature(
        ctx context.Context,
        req *http.Request,
        expectedDID did.AgentDID,
    ) error

    VerifyHTTPSignatureWithKeyID(
        ctx context.Context,
        req *http.Request,
    ) (did.AgentDID, error)
}
```

### DefaultDIDVerifier

RFC 9421 compliant verifier implementation.

```go
func NewDefaultDIDVerifier(
    didClient *did.Client,
    keySelector KeySelector,
) DIDVerifier
```

**Example:**
```go
verifier := verifier.NewDefaultDIDVerifier(didClient, keySelector)

// Verify with expected DID
err := verifier.VerifyHTTPSignature(ctx, req, expectedDID)

// Or extract DID from signature
agentDID, err := verifier.VerifyHTTPSignatureWithKeyID(ctx, req)
```

---

## Server Package

`import "github.com/sage-x-project/sage-a2a-go/pkg/server"`

Provides HTTP middleware for DID authentication.

### DIDAuthMiddleware

HTTP middleware that verifies DID signatures on incoming requests.

```go
func NewDIDAuthMiddleware(
    verifier verifier.DIDVerifier,
    options ...MiddlewareOption,
) func(http.Handler) http.Handler
```

**Example:**
```go
middleware := server.NewDIDAuthMiddleware(
    verifier,
    server.WithRequiredDID(true),
    server.WithErrorHandler(customErrorHandler),
)

http.Handle("/api/", middleware(apiHandler))
```

### Middleware Options

```go
func WithRequiredDID(required bool) MiddlewareOption
func WithErrorHandler(handler ErrorHandler) MiddlewareOption
```

### Context Helpers

```go
func GetAgentDIDFromContext(ctx context.Context) (did.AgentDID, error)
```

**Example:**
```go
func handler(w http.ResponseWriter, r *http.Request) {
    agentDID, err := server.GetAgentDIDFromContext(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", 401)
        return
    }

    fmt.Fprintf(w, "Authenticated as: %s", agentDID)
}
```

---

## Type Reference

### Common Types from a2a-go

These types are from the `github.com/a2aproject/a2a-go/a2a` package:

```go
// Message roles
const (
    MessageRoleUser  MessageRole = "user"
    MessageRoleAgent MessageRole = "agent"
)

// Task states
const (
    TaskStateSubmitted     TaskState = "submitted"
    TaskStateWorking       TaskState = "working"
    TaskStateInputRequired TaskState = "input-required"
    TaskStateCompleted     TaskState = "completed"
    TaskStateFailed        TaskState = "failed"
    TaskStateCanceled      TaskState = "canceled"
)

// Check if state is terminal
func (ts TaskState) Terminal() bool
```

---

## Error Handling

All methods return standard Go errors. Common error types:

```go
// HTTP errors
fmt.Errorf("HTTP error: %d %s", statusCode, status)

// JSON-RPC errors
fmt.Errorf("JSON-RPC error %d: %s", code, message)

// Signing errors
fmt.Errorf("failed to sign request: %w", err)

// Verification errors
fmt.Errorf("signature verification failed: %w", err)

// Context errors
context.Canceled
context.DeadlineExceeded
```

**Example:**
```go
task, err := client.GetTask(ctx, params)
if err != nil {
    if errors.Is(err, context.Canceled) {
        // User cancelled
    } else if errors.Is(err, context.DeadlineExceeded) {
        // Timeout
    } else {
        // Other error
        log.Printf("Error: %v", err)
    }
}
```

---

## See Also

- [SSE Streaming Guide](SSE_STREAMING_GUIDE.md) - Complete streaming guide
- [Integration Guide](INTEGRATION_GUIDE.md) - Integration tutorial
- [Architecture](ARCHITECTURE.md) - System architecture
- [A2A Protocol Spec](https://a2a-protocol.org) - Protocol specification
- [GoDoc](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go) - Generated documentation

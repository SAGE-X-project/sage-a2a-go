# A2A Integration Migration Guide

## 개요

이 문서는 SAGE core에서 제거된 A2A (Agent-to-Agent) 통신 프로토콜 구현을 `sage-a2a-go` 프로젝트로 마이그레이션하기 위한 가이드입니다.

## 목차

- [배경 및 목적](#배경-및-목적)
- [제거된 코드 분석](#제거된-코드-분석)
- [아키텍처 설계](#아키텍처-설계)
- [구현 가이드](#구현-가이드)
- [테스트 전략](#테스트-전략)
- [통합 방법](#통합-방법)

---

## 배경 및 목적

### 왜 분리했는가?

**문제점:**
1. **잘못된 패키지 임포트**
   - 기존 코드: `github.com/a2aproject/a2a/grpc` 사용
   - 실제: 해당 패키지는 존재하지 않음 (`.proto` 파일만 존재)
   - 올바른 경로: `github.com/a2aproject/a2a-go/a2apb`

2. **`go mod tidy` 실패**
   ```
   github.com/a2aproject/a2a/grpc: package not found
   (a2a repo only contains .proto specs, not Go code)
   ```

3. **아키텍처 원칙 위반**
   - SAGE core는 인터페이스만 정의해야 함
   - 구체적인 프로토콜 구현은 별도 프로젝트로 분리

**해결 방안:**
- SAGE core: `transport.MessageTransport` 인터페이스만 정의
- sage-a2a-go: A2A 프로토콜 구현체 제공
- 장점: 관심사의 분리, 독립적인 버전 관리, 명확한 의존성

### 관련 PR 및 커밋

- **PR #90**: Remove A2A implementation from SAGE core
- **커밋**: `b3ace36` - refactor: Remove A2A implementation from SAGE core
- **브랜치**: `refactor/remove-a2a-implementation`

---

## 제거된 코드 분석

### 삭제된 파일 목록 (총 1,656줄)

#### 1. A2A Transport 구현체

**pkg/agent/transport/a2a/client.go** (약 200줄)
```go
//go:build a2a

package a2a

import (
    "context"
    "time"

    a2apb "github.com/a2aproject/a2a/grpc"  // ❌ 잘못된 경로
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// A2ATransport implements transport.MessageTransport using A2A protocol
type A2ATransport struct {
    client     a2apb.A2AServiceClient
    conn       *grpc.ClientConn
    serverAddr string
    timeout    time.Duration
}

// NewA2ATransport creates a new A2A transport client
func NewA2ATransport(serverAddr string, timeout time.Duration) (*A2ATransport, error) {
    conn, err := grpc.Dial(serverAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        return nil, err
    }

    return &A2ATransport{
        client:     a2apb.NewA2AServiceClient(conn),
        conn:       conn,
        serverAddr: serverAddr,
        timeout:    timeout,
    }, nil
}

// Send implements transport.MessageTransport
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    ctx, cancel := context.WithTimeout(ctx, t.timeout)
    defer cancel()

    req := &a2apb.SendMessageRequest{
        Message: &a2apb.Message{
            Id:        msg.ID,
            ContextId: msg.ContextID,
            TaskId:    msg.TaskID,
            Payload:   msg.Payload,
            Did:       msg.DID,
            Signature: msg.Signature,
            Metadata:  msg.Metadata,
            Role:      msg.Role,
        },
    }

    resp, err := t.client.SendMessage(ctx, req)
    if err != nil {
        return nil, err
    }

    return &transport.Response{
        ID:      resp.Message.Id,
        Payload: resp.Message.Payload,
        Status:  "success",
    }, nil
}

// Close closes the gRPC connection
func (t *A2ATransport) Close() error {
    return t.conn.Close()
}
```

**pkg/agent/transport/a2a/server.go** (약 250줄)
```go
//go:build a2a

package a2a

import (
    "context"

    a2apb "github.com/a2aproject/a2a/grpc"  // ❌ 잘못된 경로
    "google.golang.org/grpc"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// MessageHandler is the callback function for handling incoming messages
type MessageHandler func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)

// A2AServerAdapter adapts SAGE's message handler to A2A gRPC service
type A2AServerAdapter struct {
    a2apb.UnimplementedA2AServiceServer
    handler MessageHandler
}

// NewA2AServerAdapter creates a new A2A server adapter
func NewA2AServerAdapter(handler MessageHandler) *A2AServerAdapter {
    return &A2AServerAdapter{
        handler: handler,
    }
}

// SendMessage implements A2AServiceServer.SendMessage
func (s *A2AServerAdapter) SendMessage(ctx context.Context, req *a2apb.SendMessageRequest) (*a2apb.SendMessageResponse, error) {
    // Convert A2A message to SAGE message
    sageMsg := &transport.SecureMessage{
        ID:        req.Message.Id,
        ContextID: req.Message.ContextId,
        TaskID:    req.Message.TaskId,
        Payload:   req.Message.Payload,
        DID:       req.Message.Did,
        Signature: req.Message.Signature,
        Metadata:  req.Message.Metadata,
        Role:      req.Message.Role,
    }

    // Handle the message
    resp, err := s.handler(ctx, sageMsg)
    if err != nil {
        return nil, err
    }

    // Convert SAGE response to A2A response
    return &a2apb.SendMessageResponse{
        Message: &a2apb.Message{
            Id:      resp.ID,
            Payload: resp.Payload,
        },
    }, nil
}

// RegisterWithServer registers the adapter with a gRPC server
func (s *A2AServerAdapter) RegisterWithServer(server *grpc.Server) {
    a2apb.RegisterA2AServiceServer(server, s)
}
```

**pkg/agent/transport/a2a/adapter_test.go** (약 150줄)
```go
//go:build a2a

package a2a

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestA2ATransportIntegration(t *testing.T) {
    // Test implementation
}

func TestMessageConversion(t *testing.T) {
    // Test A2A <-> SAGE message conversion
}
```

#### 2. A2A Test Servers

**tests/handshake/server/main.go** (약 250줄)
- A2A 프로토콜을 사용한 핸드셰이크 테스트 서버
- gRPC 서버 설정 및 A2A 서비스 등록
- 빌드 태그: `//go:build ignore`

**tests/integration/session/handshake/server/main.go** (약 300줄)
- HPKE + A2A 통합 테스트 서버
- 세션 관리 및 암호화 통신 테스트
- 빌드 태그: `//go:build integration && a2a`

**tests/integration/session/hpke/server/main.go** (약 350줄)
- HPKE 암호화와 A2A 전송 통합 테스트
- 완전한 end-to-end 통신 시나리오
- 빌드 태그: `//go:build integration && a2a`

### 유지된 코드

#### SAGE Core Transport Interface

**pkg/agent/transport/interface.go**
```go
package transport

import "context"

// MessageTransport is the transport layer abstraction interface.
// This interface allows different transport protocols to be used
// interchangeably for agent communication.
type MessageTransport interface {
    // Send sends a secure message and returns a response
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}

// SecureMessage represents a message to be sent
type SecureMessage struct {
    ID        string            // Message ID
    ContextID string            // Context/Session ID
    TaskID    string            // Task ID
    Payload   []byte            // Encrypted payload
    DID       string            // Sender's DID
    Signature []byte            // Message signature
    Metadata  map[string]string // Additional metadata
    Role      string            // Sender role (client/server)
}

// Response represents a response message
type Response struct {
    ID      string // Response message ID
    Payload []byte // Response payload
    Status  string // Response status
}
```

**유지된 구현체:**
- `pkg/agent/transport/http/` - HTTP transport 구현
- `pkg/agent/transport/websocket/` - WebSocket transport 구현
- `pkg/agent/transport/selector.go` - Transport 선택 로직
- `pkg/agent/transport/mock.go` - 테스트용 Mock transport

---

## 아키텍처 설계

### 전체 구조

```
┌─────────────────────────────────────────────────────────────┐
│                     SAGE Application                        │
│                                                             │
│  ┌───────────────────────────────────────────────────┐    │
│  │         pkg/agent/transport/interface.go          │    │
│  │                                                   │    │
│  │  type MessageTransport interface {               │    │
│  │      Send(ctx, msg) (*Response, error)          │    │
│  │  }                                               │    │
│  └───────────────────────────────────────────────────┘    │
│                          ↑                                 │
│                          │ implements                     │
│                          │                                 │
└──────────────────────────┼─────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ↓                  ↓                  ↓
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ HTTP         │   │ WebSocket    │   │ A2A          │
│ Transport    │   │ Transport    │   │ Transport    │
│              │   │              │   │              │
│ (in core)    │   │ (in core)    │   │ (separate)   │
└──────────────┘   └──────────────┘   └──────────────┘
                                              │
                                              │
                                              ↓
                                    ┌──────────────────┐
                                    │  sage-a2a-go     │
                                    │                  │
                                    │  - A2ATransport  │
                                    │  - A2AAdapter    │
                                    │  - Tests         │
                                    └──────────────────┘
                                              │
                                              │ uses
                                              ↓
                                    ┌──────────────────┐
                                    │ a2aproject/      │
                                    │ a2a-go           │
                                    │                  │
                                    │ (official SDK)   │
                                    └──────────────────┘
```

### 패키지 구조

**sage-a2a-go 프로젝트 구조:**
```
sage-a2a-go/
├── README.md
├── LICENSE
├── go.mod
├── go.sum
│
├── transport/
│   ├── client.go          # A2ATransport 구현
│   ├── server.go          # A2AServerAdapter 구현
│   ├── converter.go       # Message 변환 로직
│   └── options.go         # 설정 옵션
│
├── internal/
│   └── validation/        # 내부 검증 로직
│
├── examples/
│   ├── basic/             # 기본 사용 예제
│   ├── secure/            # HPKE 암호화 예제
│   └── integration/       # 통합 예제
│
└── tests/
    ├── unit/              # 단위 테스트
    ├── integration/       # 통합 테스트
    └── testdata/          # 테스트 데이터
```

### 의존성 관계

```
sage-a2a-go
├── github.com/sage-x-project/sage (core transport interface만)
├── github.com/a2aproject/a2a-go/a2apb (올바른 경로!)
├── google.golang.org/grpc
└── google.golang.org/protobuf
```

---

## 구현 가이드

### 1. 프로젝트 초기 설정

```bash
# 저장소 생성 및 초기화
mkdir sage-a2a-go
cd sage-a2a-go
git init

# Go 모듈 초기화
go mod init github.com/sage-x-project/sage-a2a-go

# 기본 디렉토리 구조 생성
mkdir -p transport internal/validation examples/{basic,secure,integration} tests/{unit,integration,testdata}
```

### 2. go.mod 설정

```go
module github.com/sage-x-project/sage-a2a-go

go 1.23

require (
    github.com/sage-x-project/sage v0.x.x
    github.com/a2aproject/a2a-go v0.3.0  // ✅ 올바른 패키지!
    google.golang.org/grpc v1.73.0
    google.golang.org/protobuf v1.36.10
)

require (
    // 테스트 의존성
    github.com/stretchr/testify v1.11.1
)
```

### 3. Client 구현 (transport/client.go)

```go
package transport

import (
    "context"
    "fmt"
    "time"

    a2apb "github.com/a2aproject/a2a-go/a2apb"  // ✅ 올바른 경로
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// A2ATransport implements transport.MessageTransport using A2A protocol
type A2ATransport struct {
    client     a2apb.A2AServiceClient
    conn       *grpc.ClientConn
    serverAddr string
    timeout    time.Duration
    opts       *Options
}

// Options contains configuration for A2A transport
type Options struct {
    ServerAddr string
    Timeout    time.Duration
    TLSConfig  *tls.Config // optional TLS
    // 추가 설정 옵션들
}

// NewA2ATransport creates a new A2A transport client
func NewA2ATransport(opts *Options) (*A2ATransport, error) {
    if opts == nil {
        return nil, fmt.Errorf("options cannot be nil")
    }

    // gRPC dial options 설정
    dialOpts := []grpc.DialOption{
        grpc.WithBlock(),
    }

    // TLS 설정
    if opts.TLSConfig != nil {
        creds := credentials.NewTLS(opts.TLSConfig)
        dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
    } else {
        dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
    }

    // gRPC 연결 생성
    conn, err := grpc.Dial(opts.ServerAddr, dialOpts...)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to A2A server: %w", err)
    }

    return &A2ATransport{
        client:     a2apb.NewA2AServiceClient(conn),
        conn:       conn,
        serverAddr: opts.ServerAddr,
        timeout:    opts.Timeout,
        opts:       opts,
    }, nil
}

// Send implements transport.MessageTransport
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Timeout 적용
    if t.timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, t.timeout)
        defer cancel()
    }

    // SAGE message -> A2A message 변환
    req := &a2apb.SendMessageRequest{
        Message: &a2apb.Message{
            Id:        msg.ID,
            ContextId: msg.ContextID,
            TaskId:    msg.TaskID,
            Payload:   msg.Payload,
            Did:       msg.DID,
            Signature: msg.Signature,
            Metadata:  msg.Metadata,
            Role:      msg.Role,
        },
    }

    // A2A 서비스 호출
    resp, err := t.client.SendMessage(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("A2A SendMessage failed: %w", err)
    }

    // A2A response -> SAGE response 변환
    return &transport.Response{
        ID:      resp.Message.Id,
        Payload: resp.Message.Payload,
        Status:  "success",
    }, nil
}

// Close closes the gRPC connection
func (t *A2ATransport) Close() error {
    if t.conn != nil {
        return t.conn.Close()
    }
    return nil
}
```

### 4. Server 구현 (transport/server.go)

```go
package transport

import (
    "context"
    "fmt"

    a2apb "github.com/a2aproject/a2a-go/a2apb"  // ✅ 올바른 경로
    "google.golang.org/grpc"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// MessageHandler is the callback function for handling incoming messages
type MessageHandler func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error)

// A2AServerAdapter adapts SAGE's message handler to A2A gRPC service
type A2AServerAdapter struct {
    a2apb.UnimplementedA2AServiceServer
    handler MessageHandler
}

// NewA2AServerAdapter creates a new A2A server adapter
func NewA2AServerAdapter(handler MessageHandler) *A2AServerAdapter {
    if handler == nil {
        panic("handler cannot be nil")
    }
    return &A2AServerAdapter{
        handler: handler,
    }
}

// SendMessage implements A2AServiceServer.SendMessage
func (s *A2AServerAdapter) SendMessage(ctx context.Context, req *a2apb.SendMessageRequest) (*a2apb.SendMessageResponse, error) {
    // 입력 검증
    if req.Message == nil {
        return nil, fmt.Errorf("message cannot be nil")
    }

    // A2A message -> SAGE message 변환
    sageMsg := &transport.SecureMessage{
        ID:        req.Message.Id,
        ContextID: req.Message.ContextId,
        TaskID:    req.Message.TaskId,
        Payload:   req.Message.Payload,
        DID:       req.Message.Did,
        Signature: req.Message.Signature,
        Metadata:  req.Message.Metadata,
        Role:      req.Message.Role,
    }

    // SAGE handler 호출
    resp, err := s.handler(ctx, sageMsg)
    if err != nil {
        return nil, fmt.Errorf("handler error: %w", err)
    }

    // SAGE response -> A2A response 변환
    return &a2apb.SendMessageResponse{
        Message: &a2apb.Message{
            Id:      resp.ID,
            Payload: resp.Payload,
        },
    }, nil
}

// RegisterWithServer registers the adapter with a gRPC server
func (s *A2AServerAdapter) RegisterWithServer(server *grpc.Server) {
    a2apb.RegisterA2AServiceServer(server, s)
}
```

### 5. Message Converter (transport/converter.go)

```go
package transport

import (
    a2apb "github.com/a2aproject/a2a-go/a2apb"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// ToA2AMessage converts SAGE SecureMessage to A2A Message
func ToA2AMessage(msg *transport.SecureMessage) *a2apb.Message {
    if msg == nil {
        return nil
    }

    return &a2apb.Message{
        Id:        msg.ID,
        ContextId: msg.ContextID,
        TaskId:    msg.TaskID,
        Payload:   msg.Payload,
        Did:       msg.DID,
        Signature: msg.Signature,
        Metadata:  msg.Metadata,
        Role:      msg.Role,
    }
}

// FromA2AMessage converts A2A Message to SAGE SecureMessage
func FromA2AMessage(msg *a2apb.Message) *transport.SecureMessage {
    if msg == nil {
        return nil
    }

    return &transport.SecureMessage{
        ID:        msg.Id,
        ContextID: msg.ContextId,
        TaskID:    msg.TaskId,
        Payload:   msg.Payload,
        DID:       msg.Did,
        Signature: msg.Signature,
        Metadata:  msg.Metadata,
        Role:      msg.Role,
    }
}
```

### 6. Options 설정 (transport/options.go)

```go
package transport

import (
    "crypto/tls"
    "time"
)

// Options contains configuration for A2A transport
type Options struct {
    ServerAddr string
    Timeout    time.Duration
    TLSConfig  *tls.Config

    // gRPC 옵션
    MaxRecvMsgSize int
    MaxSendMsgSize int

    // 재시도 설정
    MaxRetries    int
    RetryInterval time.Duration
}

// DefaultOptions returns default options
func DefaultOptions(serverAddr string) *Options {
    return &Options{
        ServerAddr:     serverAddr,
        Timeout:        30 * time.Second,
        TLSConfig:      nil,
        MaxRecvMsgSize: 4 * 1024 * 1024, // 4MB
        MaxSendMsgSize: 4 * 1024 * 1024, // 4MB
        MaxRetries:     3,
        RetryInterval:  1 * time.Second,
    }
}

// WithTLS sets TLS configuration
func (o *Options) WithTLS(config *tls.Config) *Options {
    o.TLSConfig = config
    return o
}

// WithTimeout sets timeout
func (o *Options) WithTimeout(timeout time.Duration) *Options {
    o.Timeout = timeout
    return o
}
```

---

## 테스트 전략

### 1. 단위 테스트 (tests/unit/)

**tests/unit/converter_test.go**
```go
package unit

import (
    "testing"

    "github.com/stretchr/testify/assert"

    "github.com/sage-x-project/sage-a2a-go/transport"
    sagetransport "github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestMessageConversion(t *testing.T) {
    t.Run("SAGE to A2A conversion", func(t *testing.T) {
        sageMsg := &sagetransport.SecureMessage{
            ID:        "test-123",
            ContextID: "ctx-456",
            TaskID:    "task-789",
            Payload:   []byte("test payload"),
            DID:       "did:sage:test",
            Signature: []byte("signature"),
            Metadata: map[string]string{
                "key": "value",
            },
            Role: "client",
        }

        a2aMsg := transport.ToA2AMessage(sageMsg)

        assert.Equal(t, sageMsg.ID, a2aMsg.Id)
        assert.Equal(t, sageMsg.ContextID, a2aMsg.ContextId)
        assert.Equal(t, sageMsg.TaskID, a2aMsg.TaskId)
        assert.Equal(t, sageMsg.Payload, a2aMsg.Payload)
        assert.Equal(t, sageMsg.DID, a2aMsg.Did)
        assert.Equal(t, sageMsg.Signature, a2aMsg.Signature)
        assert.Equal(t, sageMsg.Metadata, a2aMsg.Metadata)
        assert.Equal(t, sageMsg.Role, a2aMsg.Role)
    })

    t.Run("A2A to SAGE conversion", func(t *testing.T) {
        // 반대 방향 테스트
    })

    t.Run("Nil message handling", func(t *testing.T) {
        assert.Nil(t, transport.ToA2AMessage(nil))
        assert.Nil(t, transport.FromA2AMessage(nil))
    })
}
```

### 2. 통합 테스트 (tests/integration/)

**tests/integration/roundtrip_test.go**
```go
package integration

import (
    "context"
    "net"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"

    "github.com/sage-x-project/sage-a2a-go/transport"
    sagetransport "github.com/sage-x-project/sage/pkg/agent/transport"
)

func TestA2ARoundTrip(t *testing.T) {
    // 테스트 서버 시작
    lis, err := net.Listen("tcp", "127.0.0.1:0")
    require.NoError(t, err)
    defer lis.Close()

    grpcServer := grpc.NewServer()
    defer grpcServer.Stop()

    // 테스트 핸들러 등록
    handler := func(ctx context.Context, msg *sagetransport.SecureMessage) (*sagetransport.Response, error) {
        return &sagetransport.Response{
            ID:      msg.ID + "-response",
            Payload: append([]byte("echo: "), msg.Payload...),
            Status:  "success",
        }, nil
    }

    adapter := transport.NewA2AServerAdapter(handler)
    adapter.RegisterWithServer(grpcServer)

    go grpcServer.Serve(lis)

    // 클라이언트 생성
    opts := transport.DefaultOptions(lis.Addr().String())
    client, err := transport.NewA2ATransport(opts)
    require.NoError(t, err)
    defer client.Close()

    // 메시지 전송
    ctx := context.Background()
    msg := &sagetransport.SecureMessage{
        ID:      "test-msg-1",
        Payload: []byte("Hello A2A"),
        DID:     "did:sage:test",
    }

    resp, err := client.Send(ctx, msg)
    require.NoError(t, err)

    assert.Equal(t, "test-msg-1-response", resp.ID)
    assert.Equal(t, []byte("echo: Hello A2A"), resp.Payload)
    assert.Equal(t, "success", resp.Status)
}
```

### 3. 예제 코드 (examples/)

**examples/basic/main.go**
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage-a2a-go/transport"
    sagetransport "github.com/sage-x-project/sage/pkg/agent/transport"
)

func main() {
    // A2A 클라이언트 생성
    opts := transport.DefaultOptions("localhost:50051")
    client, err := transport.NewA2ATransport(opts)
    if err != nil {
        log.Fatalf("Failed to create A2A transport: %v", err)
    }
    defer client.Close()

    // 메시지 전송
    ctx := context.Background()
    msg := &sagetransport.SecureMessage{
        ID:      "example-msg-1",
        Payload: []byte("Hello from SAGE"),
        DID:     "did:sage:example",
    }

    resp, err := client.Send(ctx, msg)
    if err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }

    fmt.Printf("Response: %s\n", string(resp.Payload))
}
```

---

## 통합 방법

### SAGE 애플리케이션에서 사용하기

**1. 의존성 추가**

```bash
go get github.com/sage-x-project/sage-a2a-go@latest
```

**2. Transport 선택 로직에 A2A 추가**

```go
package main

import (
    "context"

    "github.com/sage-x-project/sage/pkg/agent/transport"
    "github.com/sage-x-project/sage/pkg/agent/transport/http"
    "github.com/sage-x-project/sage/pkg/agent/transport/websocket"
    a2a "github.com/sage-x-project/sage-a2a-go/transport"
)

func selectTransport(protocol string, endpoint string) (transport.MessageTransport, error) {
    switch protocol {
    case "http", "https":
        return http.NewHTTPTransport(endpoint)
    case "ws", "wss":
        return websocket.NewWebSocketTransport(endpoint)
    case "grpc", "a2a":
        opts := a2a.DefaultOptions(endpoint)
        return a2a.NewA2ATransport(opts)
    default:
        return nil, fmt.Errorf("unsupported protocol: %s", protocol)
    }
}

func main() {
    // A2A transport 사용
    transport, err := selectTransport("a2a", "localhost:50051")
    if err != nil {
        panic(err)
    }
    defer transport.Close()

    // 메시지 전송
    ctx := context.Background()
    msg := &transport.SecureMessage{
        ID:      "msg-1",
        Payload: []byte("Hello"),
    }

    resp, err := transport.Send(ctx, msg)
    // ...
}
```

**3. 서버 측 통합**

```go
package main

import (
    "context"
    "net"
    "log"

    "google.golang.org/grpc"

    "github.com/sage-x-project/sage/pkg/agent/transport"
    "github.com/sage-x-project/sage/pkg/agent/session"
    a2a "github.com/sage-x-project/sage-a2a-go/transport"
)

func main() {
    // gRPC 서버 생성
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()

    // SAGE 메시지 핸들러
    handler := func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
        // 실제 SAGE 로직 처리
        // session manager, HPKE 암호화 등
        return processMessage(ctx, msg)
    }

    // A2A adapter 등록
    adapter := a2a.NewA2AServerAdapter(handler)
    adapter.RegisterWithServer(grpcServer)

    log.Println("A2A server listening on :50051")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}

func processMessage(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // SAGE 메시지 처리 로직
    return &transport.Response{
        ID:      msg.ID + "-response",
        Payload: []byte("processed"),
        Status:  "success",
    }, nil
}
```

---

## 추가 고려사항

### 1. 보안

- **TLS 설정**: 프로덕션 환경에서는 반드시 TLS 사용
- **인증/인가**: DID 기반 인증 메커니즘 통합
- **Rate Limiting**: gRPC interceptor로 rate limiting 구현

### 2. 성능

- **Connection Pooling**: gRPC의 기본 연결 풀 활용
- **Timeout 설정**: 적절한 timeout 값 설정
- **Retry 로직**: 실패 시 재시도 메커니즘

### 3. 모니터링

- **Metrics**: Prometheus metrics 추가
- **Logging**: 구조화된 로깅 (zerolog, zap 등)
- **Tracing**: OpenTelemetry 통합

### 4. 문서화

- README.md: 프로젝트 소개 및 Quick Start
- API 문서: GoDoc 주석 작성
- 예제 코드: 다양한 사용 사례

---

## 체크리스트

### 구현 단계
- [ ] 프로젝트 초기 설정 완료
- [ ] go.mod 및 의존성 설정
- [ ] Client 구현 (A2ATransport)
- [ ] Server 구현 (A2AServerAdapter)
- [ ] Message converter 구현
- [ ] Options 및 설정 구현
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 작성
- [ ] 예제 코드 작성
- [ ] README 및 문서 작성
- [ ] CI/CD 파이프라인 설정

### 테스트 항목
- [ ] Message 변환 테스트
- [ ] Client-Server roundtrip 테스트
- [ ] Timeout 및 에러 처리 테스트
- [ ] TLS 연결 테스트
- [ ] 동시성 테스트
- [ ] Performance 벤치마크

### 문서 작성
- [ ] README.md
- [ ] API 문서 (GoDoc)
- [ ] 사용 가이드
- [ ] 예제 코드
- [ ] 통합 가이드

---

## 참고 자료

### A2A Protocol
- **공식 저장소**: https://github.com/a2aproject/a2a
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **스펙 문서**: A2A protocol specification (JSON-RPC 2.0 over gRPC)

### SAGE Core
- **Transport Interface**: `pkg/agent/transport/interface.go`
- **HTTP Transport**: `pkg/agent/transport/http/`
- **WebSocket Transport**: `pkg/agent/transport/websocket/`

### 관련 PR
- **PR #90**: Remove A2A implementation from SAGE core
  - https://github.com/SAGE-X-project/sage/pull/90

---

## 문의 및 지원

이 문서에 대한 질문이나 개선 사항이 있다면:
- GitHub Issues: https://github.com/SAGE-X-project/sage-a2a-go/issues
- 또는 SAGE 프로젝트 Discussion 참여

---

**작성일**: 2025-10-17
**작성자**: SAGE Development Team
**문서 버전**: 1.0

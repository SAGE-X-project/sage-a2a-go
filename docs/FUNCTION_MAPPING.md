// sage-multi-agent를 위한 함수 매핑 가이드
# sage-a2a-go v1.5.2 함수 매핑 가이드

**대상**: sage-multi-agent 개발팀
**목적**: SAGE 직접 import를 sage-a2a-go로 완전히 대체하기 위한 함수별 매핑
**날짜**: 2025-11-02

---

## 📋 목차

1. [Crypto/Key Management](#1-cryptokey-management)
2. [DID Resolution](#2-did-resolution)
3. [HPKE Client](#3-hpke-client)
4. [HPKE Server](#4-hpke-server)
5. [Session Management](#5-session-management)
6. [HTTP Transport](#6-http-transport)
7. [A2A Client/Server](#7-a2a-clientserver)
8. [Registry Operations](#8-registry-operations)

---

## 1. Crypto/Key Management

### 1.1 키 생성 (Key Generation)

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import "github.com/sage-x-project/sage/pkg/agent/crypto/keys"

// Secp256k1 키 생성
keyPair, err := keys.GenerateSecp256k1KeyPair()

// Ed25519 키 생성
ed25519Key, err := keys.GenerateEd25519KeyPair()

// X25519 키 생성 (HPKE용)
x25519Key, err := keys.GenerateX25519KeyPair()
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"

// Secp256k1 키 생성
keyPair, err := crypto.GenerateSecp256k1KeyPair()

// Ed25519 키 생성
ed25519Key, err := crypto.GenerateEd25519KeyPair()

// X25519 키 생성 (HPKE용)
x25519Key, err := crypto.GenerateX25519KeyPair()
```

**영향받는 파일**:
- `tools/keygen/gen_agents_key.go` (Line 108)
- `agents/root/agent.go`
- `agents/payment/agent.go`
- `agents/medical/agent.go`

---

### 1.2 JWK Import/Export

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import (
    agentcrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/crypto/formats"
)

// JWK Importer 생성
jwkImp := formats.NewJWKImporter()

// JWK에서 키 import
keyPair, err := jwkImp.Import(jwkData, agentcrypto.KeyFormatJWK)

// JWK Exporter 생성
jwkExp := formats.NewJWKExporter()

// 키를 JWK로 export
jwkBytes, err := jwkExp.Export(keyPair, agentcrypto.KeyFormatJWK)
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"

// JWK에서 공개키 import
jwk, err := crypto.UnmarshalJWK(jwkData)
pubKey, err := crypto.ImportPublicKeyFromJWK(jwk)

// JWK에서 개인키 import
jwk, err := crypto.UnmarshalJWK(jwkData)
privKey, err := crypto.ImportPrivateKeyFromJWK(jwk)

// 공개키를 JWK로 export
jwk, err := crypto.ExportPublicKeyToJWK(keyPair.PublicKey(), "key-id-1")
jwkBytes, err := crypto.MarshalJWK(jwk)

// 개인키를 JWK로 export
jwk, err := crypto.ExportPrivateKeyToJWK(keyPair.PrivateKey(), "key-id-1")
jwkBytes, err := crypto.MarshalJWK(jwk)
```

**영향받는 파일**:
- `tools/keygen/gen_agents_key.go` (Lines 89, 114)
- `agents/payment/agent.go` (Lines 519, 522-536)
- `agents/medical/agent.go` (Lines 507, 510-524)

---

## 2. DID Resolution

### 2.1 DID 타입 및 검증

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import sagedid "github.com/sage-x-project/sage/pkg/agent/did"

// DID 타입
type AgentDID = sagedid.AgentDID

// DID 문자열
myDID := sagedid.AgentDID("did:sage:ethereum:0x...")
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/identity"

// DID 타입
type AgentDID = identity.AgentDID

// DID 문자열
myDID := identity.AgentDID("did:sage:ethereum:0x...")

// DID 검증
err := identity.ValidateDID(string(myDID))

// DID 파싱
chain, address, err := identity.ParseDID(myDID)
```

**영향받는 파일**:
- `agents/root/agent.go` (Line 67)
- `agents/payment/agent.go` (Line 295)
- `agents/medical/agent.go`
- `protocol/a2a_transport.go`

---

### 2.2 Ethereum Resolver 초기화

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/did"
    dideth "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// Ethereum 클라이언트 생성 (공개키 resolution용)
ethClient, err := dideth.NewEthereumClient(rpcURL, registryAddr)

// AgentCard v4 클라이언트 생성 (메타데이터 resolution용)
agentCardClient, err := dideth.NewAgentCardClient(rpcURL, registryAddr)

// Resolver 인터페이스로 사용
var resolver did.Resolver = agentCardClient
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/registry"

// Resolver 설정
config := &registry.ResolverConfig{
    RPCURL:          "https://ethereum-rpc.example.com",
    RegistryAddress: "0x...",
}

// Resolver 생성
resolver, err := registry.NewResolver(config)

// 에이전트 메타데이터 조회
metadata, err := resolver.GetAgentMetadata(ctx, agentDID)

// 공개키 조회
pubKey, err := resolver.ResolvePublicKey(ctx, agentDID, "secp256k1")

// KEM 키 조회
kemKey, err := resolver.ResolveKEMKey(ctx, agentDID)

// 활성 상태 확인
isActive, err := resolver.IsActive(ctx, agentDID)

// SAGE Resolver 인터페이스 필요시 (마이그레이션 중)
sageResolver := resolver.GetSAGEResolver()
```

**영향받는 파일**:
- `internal/a2autil/middleware.go` (Lines 51, 57)
- `agents/root/agent.go` (Lines 196-214)
- `agents/payment/agent.go` (Lines 554-572)
- `agents/medical/agent.go` (Lines 542-560)

---

## 3. HPKE Client

### 3.1 HPKE 클라이언트 초기화 및 사용

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/hpke"
    "github.com/sage-x-project/sage/pkg/agent/session"
)

// Session Manager 생성
sessionMgr := session.NewManager()

// HPKE Client 생성
hpkeClient := hpke.NewClient(signingKey, sessionMgr, clientDID, resolver, opts)

// Handshake 수행
kid, err := hpkeClient.Initialize(ctx, contextID, clientDID, serverDID, serverKEMKey)

// 세션으로 암호화
sess, _ := sessionMgr.GetByKeyID(kid)
ciphertext, err := sess.Encrypt(plaintext)

// 세션으로 복호화
plaintext, err := sess.Decrypt(ciphertext)
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/hpke"
    "github.com/sage-x-project/sage/pkg/agent/session"  // 아직 SAGE 사용
)

// Session Manager 생성 (SAGE 세션 매니저 사용)
sessionMgr := session.NewManager()

// HPKE Client 생성
hpkeClient, err := hpke.NewClient(
    clientDID,
    signingKey,
    sessionMgr,
    resolver.GetSAGEResolver(),  // registry.Resolver에서 가져옴
    hpke.DefaultClientOptions(),
)

// Handshake 수행
kid, err := hpkeClient.Initialize(ctx, contextID, serverDID, serverKEMKey)

// 암호화 (간소화된 API)
ciphertext, err := hpkeClient.Encrypt(ctx, plaintext)

// 복호화 (간소화된 API)
plaintext, err := hpkeClient.Decrypt(ctx, ciphertext)

// Session ID 확인
sessionID := hpkeClient.GetSessionID()
```

**영향받는 파일**:
- `agents/root/agent.go` (Lines 82-87, 282, 299-336)

**개선점**:
- ✅ 암호화/복호화가 클라이언트 메서드로 간소화
- ✅ 세션 관리를 클라이언트가 내부적으로 처리
- ✅ 에러 처리 개선

---

## 4. HPKE Server

### 4.1 HPKE 서버 초기화 및 사용

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/hpke"
    "github.com/sage-x-project/sage/pkg/agent/session"
    sagehttp "github.com/sage-x-project/sage/pkg/agent/transport/http"
)

// Session Manager 생성
sessionMgr := session.NewManager()

// HPKE Server 생성
hpkeServer := hpke.NewServer(
    signingKey,
    sessionMgr,
    serverDID,
    resolver,
    &hpke.ServerOpts{
        KEM: kemKey,
    },
)

// HTTP 핸들러로 변환
httpAdapter := sagehttp.NewHTTPServer(hpkeServer.HandleMessage)
handler := httpAdapter.MessagesHandler()

// HTTP 서버에 등록
mux.Handle("/messages", handler)

// 데이터 모드 처리 (세션 기반)
sess, found := sessionMgr.GetByKeyID(kid)
if found {
    plaintext, err := sess.Decrypt(ciphertext)
    // ... 처리 ...
    response, err := sess.Encrypt(responseData)
}
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/hpke"
    "github.com/sage-x-project/sage/pkg/agent/session"  // 아직 SAGE 사용
)

// Session Manager 생성 (SAGE 세션 매니저 사용)
sessionMgr := session.NewManager()

// HPKE Server 생성
hpkeServer, err := hpke.NewServer(
    signingKey,
    sessionMgr,
    serverDID,
    resolver.GetSAGEResolver(),
    &hpke.ServerOptions{
        KEM: kemKey,  // 선택사항: 지정 안하면 자동 생성
    },
)

// HTTP 핸들러 직접 제공 (간소화!)
handler := hpkeServer.MessagesHandler()

// HTTP 서버에 등록
mux.Handle("/messages", handler)

// KEM 공개키 가져오기
kemPubKey := hpkeServer.GetKEMPublicKey()

// 데이터 모드 처리는 동일
sess, found := sessionMgr.GetByKeyID(kid)
if found {
    plaintext, err := sess.Decrypt(ciphertext)
    // ... 처리 ...
    response, err := sess.Encrypt(responseData)
}
```

**영향받는 파일**:
- `agents/payment/agent.go` (Lines 54-57, 300-309)
- `agents/medical/agent.go` (Lines 260-313)

**개선점**:
- ✅ HTTP 어댑터 불필요 (내장됨)
- ✅ KEM 키 자동 생성 옵션
- ✅ 간소화된 초기화

---

## 5. Session Management

### 5.1 세션 관리 (현재는 SAGE 세션 매니저 사용)

> **참고**: v1.5.2에서 `pkg/session` 패키지가 추가되었으나,
> SAGE의 세션 매니저와 호환성을 위해 현재는 SAGE 세션을 계속 사용합니다.
> 향후 v1.6.0에서 완전한 sage-a2a-go 세션으로 마이그레이션 예정입니다.

#### 현재 사용법 (SAGE 세션 - 유지)
```go
import "github.com/sage-x-project/sage/pkg/agent/session"

// Session Manager 생성
sessionMgr := session.NewManager()

// 세션 생성 (HPKE 핸드셰이크 후)
sess, err := sessionMgr.CreateSession(kid, clientDID, serverDID, sharedSecret)

// 세션 조회
sess, found := sessionMgr.GetByKeyID(kid)

// 암호화/복호화
ciphertext, err := sess.Encrypt(plaintext)
plaintext, err := sess.Decrypt(ciphertext)
```

#### 향후 (v1.6.0 - sage-a2a-go 세션)
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/session"

// Session Manager 생성
sessionMgr := session.NewManager(session.DefaultOptions())

// 세션 생성
sess, err := sessionMgr.Create(ctx, remoteDID, sharedSecret)

// 암호화/복호화
ciphertext, err := sessionMgr.Encrypt(ctx, sess.ID, plaintext)
plaintext, err := sessionMgr.Decrypt(ctx, sess.ID, ciphertext)
```

**영향받는 파일**:
- `agents/root/agent.go` (Line 85)
- `agents/payment/agent.go` (Lines 54-57, 139-152)
- `agents/medical/agent.go` (Lines 139-180)

---

## 6. HTTP Transport

### 6.1 DID 인증 HTTP 클라이언트

#### ❌ 기존 코드 (sage-a2a-go client 사용)
```go
import a2aclient "github.com/sage-x-project/sage-a2a-go/pkg/client"

// A2A Client 생성 (HTTP 서명 포함)
a2a := a2aclient.NewA2AClient(myDID, myKey, httpClient)

// 서명된 HTTP 요청 실행
resp, err := a2a.Do(ctx, req)
```

#### ✅ 새 코드 (sage-a2a-go - 변경 없음, 계속 사용)
```go
import a2aclient "github.com/sage-x-project/sage-a2a-go/pkg/client"

// 동일하게 사용
a2a := a2aclient.NewA2AClient(myDID, myKey, httpClient)
resp, err := a2a.Do(ctx, req)
```

**또는 새 transport 패키지 사용**:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/transport"

// HTTP Transport 생성
httpTransport, err := transport.NewHTTPTransport(
    myDID,
    myKeyPair,
    transport.DefaultHTTPTransportOptions(),
)

// 일반 메시지 전송 (자동으로 DID 서명)
result, err := httpTransport.SendMessage(ctx, targetURL, payload)

// HPKE 암호화 메시지 전송
resp, err := httpTransport.SendSecureMessage(ctx, targetURL, secureMsg)

// GET 요청 (자동으로 DID 서명)
data, err := httpTransport.Get(ctx, targetURL)
```

**영향받는 파일**:
- `api/api.go` (Lines 33, 147, 203)
- `agents/root/agent.go` (Lines 69, 154, 188)

**개선점**:
- ✅ 더 다양한 HTTP 메서드 지원
- ✅ 자동 DID 서명
- ✅ HPKE 메시지 지원

---

## 7. A2A Client/Server

### 7.1 DID 인증 미들웨어

#### ❌ 기존 코드 (sage-a2a-go + SAGE 혼합)
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/server"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// Ethereum 클라이언트 생성
agentCardClient, _ := ethereum.NewAgentCardClient(rpcURL, registryAddr)
ethClient, _ := ethereum.NewEthereumClient(rpcURL, registryAddr)

// 미들웨어 생성
middleware := server.NewDIDAuthMiddleware(agentCardClient, ethClient)

// HTTP 핸들러에 적용
protectedHandler := middleware.Wrap(handler)
```

#### ✅ 새 코드 (sage-a2a-go)
```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/server"
    "github.com/sage-x-project/sage-a2a-go/pkg/registry"
)

// Resolver 생성
resolver, err := registry.NewResolver(&registry.ResolverConfig{
    RPCURL:          rpcURL,
    RegistryAddress: registryAddr,
})

// 미들웨어 생성
middleware := server.NewDIDAuthMiddleware(
    resolver.GetSAGEResolver(),
    resolver.GetEthereumClient(),
)

// HTTP 핸들러에 적용
protectedHandler := middleware.Wrap(handler)
```

**영향받는 파일**:
- `internal/a2autil/middleware.go` (Lines 41-63)
- `agents/payment/agent.go` (Lines 60, 79)
- `agents/medical/agent.go` (Lines 77-89)
- `internal/agentmux/handler.go` (Lines 13, 19)

---

## 8. Registry Operations

### 8.1 에이전트 등록 (Commit-Reveal-Activate)

#### ❌ 기존 코드 (SAGE 직접 사용)
```go
import (
    "github.com/sage-x-project/sage/pkg/agent/did"
    agentcard "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// AgentCard 클라이언트 생성
client, err := agentcard.NewAgentCardClient(rpcURL, registryAddr)

// 등록 파라미터 준비
params := &did.RegistrationParams{
    ChainType:    did.ChainTypeEthereum,
    Address:      walletAddress,
    SigningKey:   signingPubKey,
    KemKey:       kemPubKey,
    Name:         "My Agent",
    URL:          "https://my-agent.com",
    Description:  "Agent description",
    Metadata:     metadataJSON,
}

// Commit 단계
status, err := client.CommitRegistration(ctx, params)

// Reveal 단계
status, err = client.RegisterAgent(ctx, status)

// Activate 단계
err = client.ActivateAgent(ctx, status)
```

#### ✅ 새 코드 (sage-a2a-go - TBD)

> **참고**: Registry 클라이언트는 v1.6.0에서 완전히 구현 예정입니다.
> 현재는 SAGE의 AgentCard 클라이언트를 계속 사용하세요.

**임시 해결방법 (v1.5.2)**:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/registry"

// Resolver 생성
resolver, err := registry.NewResolver(config)

// SAGE 클라이언트 가져오기 (마이그레이션 중)
ethClient := resolver.GetEthereumClient()

// SAGE 메서드 그대로 사용
// (등록 로직은 동일)
```

**향후 계획 (v1.6.0)**:
```go
import "github.com/sage-x-project/sage-a2a-go/pkg/registry"

// Registry 클라이언트 생성
registryClient, err := registry.NewClient(config)

// 등록 (간소화된 API)
err = registryClient.RegisterAgent(ctx, &registry.AgentParams{
    DID:         myDID,
    SigningKey:  signingKey,
    KemKey:      kemKey,
    Name:        "My Agent",
    URL:         "https://my-agent.com",
    Description: "Description",
})
```

**영향받는 파일**:
- `tools/registration/register_agents.go` (Lines 119, 188, 197, 207)
- `tools/registration/register_kem_agents.go` (Lines 108-111, 207-230)

---

## 📊 파일별 마이그레이션 체크리스트

### 우선순위 1: 핵심 인프라 (즉시 마이그레이션)

#### `tools/keygen/gen_agents_key.go`
- [ ] Line 33: `agentcrypto` import 제거
- [ ] Line 34: `formats` import 제거
- [ ] Line 35: `keys` import 제거
- [ ] Line 89: JWK exporter → `crypto.ExportPrivateKeyToJWK()`
- [ ] Line 108: `keys.GenerateSecp256k1KeyPair()` → `crypto.GenerateSecp256k1KeyPair()`
- [ ] Line 114: JWK export → `crypto.MarshalJWK()`

#### `internal/a2autil/middleware.go`
- [ ] Line 12: `did` import 제거
- [ ] Line 13: `dideth` import 제거
- [ ] Lines 41-57: Resolver 초기화 → `registry.NewResolver()`
- [ ] Line 62: 미들웨어 생성 코드 업데이트

### 우선순위 2: 에이전트 (Week 3-4)

#### `agents/root/agent.go`
- [ ] Line 34: `sagecrypto` import → `crypto`
- [ ] Line 35: `formats` import 제거
- [ ] Line 36: `sagedid` import → `identity`
- [ ] Line 37: `dideth` import 제거 (registry 사용)
- [ ] Line 38: `hpke` import → sage-a2a-go `hpke`
- [ ] Lines 82-87: HPKE 클라이언트 → `hpke.NewClient()`
- [ ] Lines 196-214: Resolver → `registry.NewResolver()`
- [ ] Line 282: HPKE 초기화 업데이트
- [ ] Lines 299-336: 암호화/복호화 → 클라이언트 메서드 사용

#### `agents/payment/agent.go`
- [ ] Line 31: `sagedid` import → `identity`
- [ ] Line 32: `dideth` import 제거
- [ ] Line 33: `sagehttp` import 제거 (불필요)
- [ ] Line 36: `hpke` import → sage-a2a-go `hpke`
- [ ] Lines 54-57: HPKE 서버 → `hpke.NewServer()`
- [ ] Lines 300-309: HTTP 핸들러 초기화 간소화
- [ ] Lines 522-536: 키 로딩 → `crypto.ImportPrivateKeyFromJWK()`
- [ ] Lines 554-572: Resolver → `registry.NewResolver()`

#### `agents/medical/agent.go`
- [ ] Payment agent와 동일한 변경사항 적용

### 우선순위 3: 도구 (Week 5)

#### `tools/registration/register_agents.go`
- [ ] SAGE registry 클라이언트 계속 사용 (v1.6.0까지)
- [ ] v1.6.0에서 마이그레이션 예정

#### `tools/registration/register_kem_agents.go`
- [ ] SAGE registry 클라이언트 계속 사용 (v1.6.0까지)
- [ ] v1.6.0에서 마이그레이션 예정

---

## 🚀 마이그레이션 단계별 가이드

### Step 1: 테스트 환경 준비
```bash
# sage-a2a-go v1.5.2로 업데이트
cd sage-multi-agent
go get github.com/sage-x-project/sage-a2a-go@v1.5.2
go mod tidy
```

### Step 2: Import 문 업데이트 (파일별)
```bash
# 예: tools/keygen/gen_agents_key.go
# Before:
# import "github.com/sage-x-project/sage/pkg/agent/crypto/keys"

# After:
# import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
```

### Step 3: 함수 호출 업데이트
이 문서의 매핑을 참조해서 각 함수 호출을 변경하세요.

### Step 4: 컴파일 및 테스트
```bash
# 빌드
go build ./...

# 테스트
go test ./...

# 특정 에이전트 테스트
go run cmd/payment/main.go
```

### Step 5: 통합 테스트
End-to-End 흐름 테스트:
1. Root agent 시작
2. Payment/Medical agent 시작
3. Client로 요청 테스트
4. HPKE 암호화 통신 확인

---

## ❓ FAQ

### Q1: 모든 SAGE import를 한번에 제거해야 하나요?
**A**: 아니요. 단계적으로 마이그레이션하세요:
1. Week 1-2: Crypto, Identity
2. Week 3-4: HPKE, Transport
3. Week 5: 정리 및 테스트

### Q2: Session Manager는 언제 마이그레이션하나요?
**A**: v1.6.0에서 진행 예정입니다. 현재는 SAGE 세션 계속 사용하세요.

### Q3: Registry 작업은 어떻게 하나요?
**A**: v1.5.2에서는 SAGE registry 클라이언트를 계속 사용하고,
v1.6.0에서 sage-a2a-go registry로 마이그레이션합니다.

### Q4: 기존 키 파일과 호환되나요?
**A**: 네! JWK 포맷은 동일하므로 기존 키 파일을 그대로 사용할 수 있습니다.

### Q5: 성능 차이가 있나요?
**A**: 없습니다. sage-a2a-go는 SAGE를 래핑하므로 성능은 동일합니다.

---

## 📞 지원

**문제 발생시**:
1. 이 문서의 예제 코드 확인
2. MIGRATION_GUIDE.md 참조
3. GitHub Issues: https://github.com/sage-x-project/sage-a2a-go/issues

**추가 문의**:
- sage-a2a-go 팀에게 Slack/Discord로 문의

---

**마지막 업데이트**: 2025-11-02 (v1.5.2)
**다음 업데이트**: v1.6.0 (Session & Registry 완성)

# SAGE-A2A-GO Integration Guide

This guide walks you through integrating sage-a2a-go into your AI agent project.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Agent Registration](#agent-registration)
5. [HTTP Message Signing](#http-message-signing)
6. [Signature Verification](#signature-verification)
7. [Agent Cards](#agent-cards)
8. [Multi-Key Management](#multi-key-management)
9. [Production Deployment](#production-deployment)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Dependencies

- **Go 1.23+**: Modern Go version with generics support
- **SAGE v1.1.0**: Blockchain-anchored DID system
- **Ethereum Node**: For DID resolution (Infura, Alchemy, or self-hosted)

### Optional Dependencies

- **Solana RPC**: For Solana-based agents (optional)
- **PostgreSQL**: For agent metadata storage (optional)

### Knowledge Requirements

- Basic understanding of HTTP and REST APIs
- Familiarity with cryptographic signatures
- Understanding of Decentralized Identifiers (DIDs)
- Knowledge of blockchain basics (Ethereum or Solana)

---

## Installation

### Step 1: Install sage-a2a-go

```bash
go get github.com/sage-x-project/sage-a2a-go
```

### Step 2: Install SAGE Core

```bash
go get github.com/sage-x-project/sage@v1.1.0
```

### Step 3: Verify Installation

```bash
go list -m github.com/sage-x-project/sage-a2a-go
go list -m github.com/sage-x-project/sage
```

---

## Quick Start

### Complete Example: Agent-to-Agent Communication

```go
package main

import (
    "context"
    "net/http"

    "github.com/sage-x-project/sage-a2a-go/pkg/signer"
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

func main() {
    ctx := context.Background()

    // Setup DID client
    config := &did.RegistryConfig{
        ContractAddress: "0x...",  // SAGE Registry V4 address
        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/YOUR-KEY",
        PrivateKey:      "your-private-key",
    }

    client, _ := ethereum.NewEthereumClientV4(config)

    // Create key pair
    keyPair, _ := crypto.GenerateSecp256k1KeyPair()

    // Your agent's DID
    agentDID := did.AgentDID("did:sage:ethereum:0x...")

    // Sign outgoing request
    req, _ := http.NewRequest("POST", "https://other-agent.com/task", body)
    a2aSigner := signer.NewDefaultA2ASigner()
    a2aSigner.SignRequest(ctx, req, agentDID, keyPair)

    // Verify incoming request (server-side)
    selector := verifier.NewDefaultKeySelector(client)
    sigVerifier := verifier.NewRFC9421Verifier()
    didVerifier := verifier.NewDefaultDIDVerifier(client, selector, sigVerifier)

    err := didVerifier.VerifyHTTPSignature(ctx, incomingReq, senderDID)
    if err != nil {
        // Signature verification failed
        http.Error(w, "Unauthorized", 401)
        return
    }

    // Request authenticated!
}
```

---

## Agent Registration

### Ethereum-based Agent

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// Generate ECDSA key pair
keyPair, err := crypto.GenerateSecp256k1KeyPair()
if err != nil {
    log.Fatal(err)
}

// Setup Ethereum client
config := &did.RegistryConfig{
    ContractAddress: "0x...",  // SAGE Registry V4 contract
    RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/YOUR-KEY",
    PrivateKey:      "your-wallet-private-key",
}

client, err := ethereum.NewEthereumClientV4(config)
if err != nil {
    log.Fatal(err)
}

// Register agent
ctx := context.Background()
txHash, err := client.RegisterAgent(ctx, keyPair)
if err != nil {
    log.Fatal(err)
}

log.Printf("Agent registered: tx=%s", txHash)
```

### Solana-based Agent

```go
import (
    "github.com/sage-x-project/sage/pkg/agent/crypto"
    "github.com/sage-x-project/sage/pkg/agent/did/solana"
)

// Generate Ed25519 key pair
keyPair, err := crypto.GenerateEd25519KeyPair()
if err != nil {
    log.Fatal(err)
}

// Setup Solana client
config := &did.SolanaConfig{
    RPCEndpoint: "https://api.mainnet-beta.solana.com",
    PrivateKey:  "your-solana-private-key",
}

client, err := solana.NewSolanaClient(config)
if err != nil {
    log.Fatal(err)
}

// Register agent
ctx := context.Background()
signature, err := client.RegisterAgent(ctx, keyPair)
if err != nil {
    log.Fatal(err)
}

log.Printf("Agent registered: signature=%s", signature)
```

### Multi-Key Registration

```go
// Generate both key types
ecdsaKey, _ := crypto.GenerateSecp256k1KeyPair()
ed25519Key, _ := crypto.GenerateEd25519KeyPair()

// Register ECDSA key on Ethereum
ethClient, _ := ethereum.NewEthereumClientV4(ethConfig)
ethClient.RegisterAgent(ctx, ecdsaKey)

// Register Ed25519 key on Solana
solClient, _ := solana.NewSolanaClient(solConfig)
solClient.RegisterAgent(ctx, ed25519Key)

// Agent now has multi-chain identity!
```

---

## HTTP Message Signing

### Basic Request Signing

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/signer"

// Create signer
a2aSigner := signer.NewDefaultA2ASigner()

// Create HTTP request
req, _ := http.NewRequest("POST", targetURL, body)
req.Header.Set("Content-Type", "application/json")

// Sign request
ctx := context.Background()
err := a2aSigner.SignRequest(ctx, req, agentDID, keyPair)
if err != nil {
    log.Fatal(err)
}

// Request now has Signature and Signature-Input headers
```

### Advanced Signing Options

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/signer"

opts := &signer.SigningOptions{
    // Components to sign
    Components: []string{
        "@method",
        "@target-uri",
        "@authority",
        "content-type",
        "content-digest",
    },

    // Timestamp for replay protection
    Created: time.Now().Unix(),

    // Signature expiration (5 minutes)
    Expires: time.Now().Add(5 * time.Minute).Unix(),

    // Random nonce for additional security
    Nonce: generateNonce(),

    // Explicit algorithm (auto-detected from key if omitted)
    Algorithm: "ES256K",
}

err := a2aSigner.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
```

### Signing with Content Digest

```go
import (
    "crypto/sha256"
    "encoding/base64"
)

// Calculate content digest
bodyBytes, _ := io.ReadAll(req.Body)
hash := sha256.Sum256(bodyBytes)
digest := "sha-256=:" + base64.StdEncoding.EncodeToString(hash[:]) + ":"

// Reset body for signing
req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
req.Header.Set("Content-Digest", digest)

// Include content-digest in signature
opts := &signer.SigningOptions{
    Components: []string{"@method", "@target-uri", "content-digest"},
}

a2aSigner.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
```

---

## Signature Verification

### Server-Side Verification

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/verifier"
    "net/http"
)

// Setup verifier components
client, _ := ethereum.NewEthereumClientV4(config)
selector := verifier.NewDefaultKeySelector(client)
sigVerifier := verifier.NewRFC9421Verifier()
didVerifier := verifier.NewDefaultDIDVerifier(client, selector, sigVerifier)

// HTTP handler
http.HandleFunc("/task", func(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract DID from signature and verify
    agentDID, err := didVerifier.VerifyHTTPSignatureWithKeyID(ctx, r)
    if err != nil {
        http.Error(w, "Unauthorized: "+err.Error(), 401)
        return
    }

    log.Printf("Authenticated as: %s", agentDID)

    // Process request...
})
```

### Verify with Known DID

```go
// If you already know the sender's DID
knownDID := did.AgentDID("did:sage:ethereum:0x...")

err := didVerifier.VerifyHTTPSignature(ctx, r, knownDID)
if err != nil {
    http.Error(w, "Signature verification failed", 401)
    return
}

// Signature valid and matches knownDID
```

### Custom Verification Logic

```go
// Resolve public key
pubKey, err := didVerifier.ResolvePublicKey(ctx, agentDID, nil)
if err != nil {
    log.Fatal(err)
}

// Custom verification
sigVerifier := verifier.NewRFC9421Verifier()
err = sigVerifier.VerifyHTTPRequest(req, pubKey)
if err != nil {
    log.Fatal("Verification failed:", err)
}
```

---

## Agent Cards

### Creating Agent Cards

```go
import (
    "github.com/sage-x-project/sage-a2a-go/pkg/protocol"
    "time"
)

// Build Agent Card
card := protocol.NewAgentCardBuilder(
    agentDID,
    "MyAIAgent",
    "https://my-agent.example.com",
).
    WithDescription("AI assistant with specialized capabilities").
    WithCapabilities(
        "task.create",
        "task.execute",
        "messaging.send",
        "messaging.receive",
        "data.analysis",
    ).
    WithMetadata("version", "2.0.0").
    WithMetadata("region", "us-west-2").
    WithMetadata("tier", "premium").
    WithExpiresAt(time.Now().Add(365 * 24 * time.Hour)).
    Build()

// Validate
if err := card.Validate(); err != nil {
    log.Fatal("Invalid card:", err)
}
```

### Signing Agent Cards

```go
// Create signer
cardSigner := protocol.NewDefaultAgentCardSigner(client)

// Sign card
ctx := context.Background()
signedCard, err := cardSigner.SignAgentCard(ctx, card, keyPair)
if err != nil {
    log.Fatal(err)
}

// Signed card contains JWS signature
log.Printf("Signature: %s", signedCard.Signature)
```

### Verifying Agent Cards

```go
// Verify with DID resolution
err := cardSigner.VerifyAgentCard(ctx, signedCard)
if err != nil {
    log.Fatal("Verification failed:", err)
}

// Or verify with known public key
err = cardSigner.VerifyAgentCardWithKey(ctx, signedCard, publicKey)
```

### Using Agent Cards

```go
// Check capabilities
if card.HasCapability("task.execute") {
    // Execute task
    processTask(task)
}

// Check expiration
if card.IsExpired() {
    log.Println("Card expired, request renewal")
    requestCardRenewal(card.DID)
}

// Access metadata
version := card.Metadata["version"].(string)
region := card.Metadata["region"].(string)
```

---

## Multi-Key Management

### Protocol-Based Key Selection

```go
import "github.com/sage-x-project/sage-a2a-go/pkg/verifier"

selector := verifier.NewDefaultKeySelector(client)

// Select ECDSA key for Ethereum operations
ethKey, keyType, err := selector.SelectKey(ctx, agentDID, "ethereum")
if err != nil {
    log.Fatal(err)
}
log.Printf("Selected %s key for Ethereum", keyType)

// Select Ed25519 key for Solana operations
solKey, keyType, err := selector.SelectKey(ctx, agentDID, "solana")
if err != nil {
    log.Fatal(err)
}
log.Printf("Selected %s key for Solana", keyType)

// Automatic selection (first available)
autoKey, keyType, err := selector.SelectKey(ctx, agentDID, "")
```

### Adding Keys to Agent Card

```go
import "github.com/sage-x-project/sage/pkg/agent/did"

// Marshal public keys
ecdsaKeyData, _ := did.MarshalPublicKey(ecdsaPublicKey)
ed25519KeyData, _ := did.MarshalPublicKey(ed25519PublicKey)

// Create Agent Card with multiple keys
card := protocol.NewAgentCardBuilder(agentDID, "MultiKeyAgent", endpoint).
    WithPublicKey(protocol.PublicKeyInfo{
        ID:      "eth-key-1",
        Type:    "EcdsaSecp256k1VerificationKey2019",
        KeyData: string(ecdsaKeyData),
        Purpose: []string{"authentication", "signing"},
    }).
    WithPublicKey(protocol.PublicKeyInfo{
        ID:      "sol-key-1",
        Type:    "Ed25519VerificationKey2020",
        KeyData: string(ed25519KeyData),
        Purpose: []string{"authentication", "signing"},
    }).
    Build()
```

---

## Production Deployment

### Configuration Management

```go
import "github.com/spf13/viper"

// Load configuration
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.AddConfigPath("/etc/agent/")
viper.AddConfigPath("$HOME/.agent")
viper.AddConfigPath(".")

err := viper.ReadInConfig()
if err != nil {
    log.Fatal(err)
}

// Create config from environment/file
config := &did.RegistryConfig{
    ContractAddress: viper.GetString("sage.contract_address"),
    RPCEndpoint:     viper.GetString("ethereum.rpc_endpoint"),
    PrivateKey:      viper.GetString("ethereum.private_key"),
}
```

### Environment Variables

```bash
# .env file
SAGE_CONTRACT_ADDRESS=0x...
ETHEREUM_RPC_ENDPOINT=https://eth-mainnet.g.alchemy.com/v2/YOUR-KEY
ETHEREUM_PRIVATE_KEY=0x...
AGENT_DID=did:sage:ethereum:0x...
AGENT_ENDPOINT=https://my-agent.example.com
```

### Error Handling

```go
// Robust error handling
err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
if err != nil {
    switch {
    case errors.Is(err, verifier.ErrInvalidSignature):
        log.Error("Invalid signature")
        http.Error(w, "Unauthorized", 401)
    case errors.Is(err, verifier.ErrDIDNotFound):
        log.Error("DID not found in registry")
        http.Error(w, "Unknown agent", 404)
    case errors.Is(err, verifier.ErrKeyNotFound):
        log.Error("No suitable key found")
        http.Error(w, "Key not available", 500)
    default:
        log.Error("Verification failed:", err)
        http.Error(w, "Internal error", 500)
    }
    return
}
```

### Logging

```go
import "log/slog"

// Structured logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

logger.Info("Verifying signature",
    "agent_did", agentDID,
    "method", req.Method,
    "path", req.URL.Path,
)

err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
if err != nil {
    logger.Error("Verification failed",
        "agent_did", agentDID,
        "error", err,
    )
    return
}

logger.Info("Signature verified successfully",
    "agent_did", agentDID,
)
```

### Monitoring

```go
import "github.com/prometheus/client_golang/prometheus"

// Metrics
var (
    signatureVerifications = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "agent_signature_verifications_total",
            Help: "Total number of signature verifications",
        },
        []string{"status"},
    )
)

func init() {
    prometheus.MustRegister(signatureVerifications)
}

// Track verifications
err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
if err != nil {
    signatureVerifications.WithLabelValues("failed").Inc()
} else {
    signatureVerifications.WithLabelValues("success").Inc()
}
```

---

## Troubleshooting

### Common Issues

#### 1. "DID not found"

**Cause**: Agent not registered on blockchain

**Solution**:
```go
// Verify registration
keys, err := client.ResolveAllPublicKeys(ctx, agentDID)
if err != nil {
    log.Fatal("Agent not registered:", err)
}
if len(keys) == 0 {
    log.Fatal("No keys found for agent")
}
```

#### 2. "Invalid signature"

**Cause**: Signature verification failed

**Solutions**:
- Check key pair matches registered key
- Verify DID is correct
- Ensure signature headers are present
- Check for clock skew (timestamp)

#### 3. "Key type not found"

**Cause**: Requested key algorithm not available

**Solution**:
```go
// Check available keys
keys, _ := client.ResolveAllPublicKeys(ctx, agentDID)
for _, key := range keys {
    log.Printf("Available key: %s", key.Type)
}

// Use fallback
pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "")
```

#### 4. "Context deadline exceeded"

**Cause**: Blockchain RPC timeout

**Solution**:
```go
// Increase timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
```

### Debug Mode

```go
import "log"

// Enable debug logging
log.SetFlags(log.LstdFlags | log.Lshortfile)

// Log signature details
log.Printf("Signature-Input: %s", req.Header.Get("Signature-Input"))
log.Printf("Signature: %s", req.Header.Get("Signature"))

// Verify step by step
pubKey, err := didVerifier.ResolvePublicKey(ctx, agentDID, nil)
log.Printf("Resolved public key: %T", pubKey)

err = sigVerifier.VerifyHTTPRequest(req, pubKey)
log.Printf("Signature verification: %v", err)
```

---

## Next Steps

1. Review the [examples](../cmd/examples/) for complete working code
2. Read the [API documentation](https://pkg.go.dev/github.com/sage-x-project/sage-a2a-go)
3. Check the [SAGE documentation](https://github.com/sage-x-project/sage) for blockchain details
4. Join the community for support and updates

---

## Additional Resources

- [RFC9421 Specification](https://www.rfc-editor.org/rfc/rfc9421.html)
- [DID Specification](https://www.w3.org/TR/did-core/)
- [SAGE GitHub Repository](https://github.com/sage-x-project/sage)
- [A2A Protocol](https://github.com/a2aproject/A2A)

---

**Last Updated**: 2025-10-18
**Version**: 1.0.0

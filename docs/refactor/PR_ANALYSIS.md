# PR Analysis: RFC 9421 HTTP Message Signatures Refactoring

## Overview

This document analyzes the code changes in the `feature/multi-agent-demo` branch, which refactors the HTTP message signing implementation to align with RFC 9421 standards.

## Changed Files

- `pkg/signer/default_a2a_signer.go`

## Summary of Changes

### Purpose
**RFC 9421 HTTP Message Signatures standard compliance refactoring**

The PR modernizes the A2A (Agent-to-Agent) HTTP signing mechanism by:
1. Adopting RFC 9421 standard library from the `sage` project
2. Enhancing message integrity with Content-Digest
3. Using standardized algorithm identifiers
4. Simplifying code by removing manual signature base construction

---

## Detailed Changes

### 1. Signature Component Modification

**Before:**
```go
Components: []string{"@method", "@target-uri"}
```

**After:**
```go
Components: []string{"@method", "@path", "@query", "content-digest"}
```

**Impact:**
- Replaced `@target-uri` with separate `@path` and `@query` components
- Added mandatory `content-digest` for message integrity verification
- Better alignment with RFC 9421 derived component identifiers

---

### 2. RFC 9421 Standard Module Integration

**Key Changes:**
- Imported `github.com/sage-x-project/sage/pkg/agent/core/rfc9421`
- Delegated signature generation to `rfc9421.HTTPVerifier.SignRequest()`
- Removed manual signature base string construction
- Adopted `crypto.Signer` interface for standardized signing

**Before (Manual Implementation):**
```go
func (s *DefaultA2ASigner) buildSignatureBase(req *http.Request, components []string) (string, error) {
    var lines []string
    for _, component := range components {
        // Manual construction of signature base
        ...
    }
    return strings.Join(lines, "\n"), nil
}
```

**After (Standard Library):**
```go
httpv := rfc9421.NewHTTPVerifier()
if err := httpv.SignRequest(req, "sig1", params, signer); err != nil {
    return fmt.Errorf("rfc9421 signing failed: %w", err)
}
```

---

### 3. Content-Digest Header Auto-Generation

**New Function:**
```go
func ensureContentDigestHeader(req *http.Request) error {
    var body []byte
    if req.Body != nil {
        body, err = io.ReadAll(req.Body)
        if err != nil {
            return err
        }
    }
    req.Body = io.NopCloser(bytes.NewReader(body))
    req.ContentLength = int64(len(body))
    req.GetBody = func() (io.ReadCloser, error) {
        return io.NopCloser(bytes.NewReader(body)), nil
    }

    h := sha256.Sum256(body)
    d := base64.StdEncoding.EncodeToString(h[:])
    req.Header.Set("Content-Digest", "sha-256=:"+d+":")
    return nil
}
```

**Features:**
- Automatically computes SHA-256 hash of request body
- Formats according to RFC 9421 syntax: `sha-256=:<base64>:`
- Buffers body in memory for reusability
- Sets `GetBody` to allow HTTP retry mechanisms

---

### 4. Algorithm Identifier Standardization

**Before:**
- Secp256k1: `"ecdsa-p256-sha256"` (custom identifier)
- Ed25519: `"ed25519"`

**After:**
- Secp256k1: `"es256k"` (standard identifier)
- Ed25519: `"ed25519"` (unchanged)

**Code:**
```go
func (s *DefaultA2ASigner) getAlgorithm(k sagecrypto.KeyType) string {
    switch k {
    case sagecrypto.KeyTypeSecp256k1:
        return "es256k"  // Changed from "ecdsa-p256-sha256"
    case sagecrypto.KeyTypeEd25519:
        return "ed25519"
    default:
        return ""
    }
}
```

---

### 5. Code Simplification

**Removed Functions:**
- `buildSignatureBase()` - Manual signature base construction
- `buildSignatureInput()` - Manual Signature-Input header formatting
- `buildSignatureHeader()` - Manual Signature header formatting

**Consolidated Logic:**
All signature generation now handled by `rfc9421.HTTPVerifier`:
```go
params := &rfc9421.SignatureInputParams{
    CoveredComponents: quoteComponents(opts.Components),
    KeyID:             string(agentDID),
    Algorithm:         alg,
    Created:           created,
    Expires:           opts.Expires,
    Nonce:             opts.Nonce,
}

priv := keyPair.PrivateKey()
signer, ok := priv.(gocrypto.Signer)
if !ok {
    return fmt.Errorf("private key does not implement crypto.Signer: %T", priv)
}

httpv := rfc9421.NewHTTPVerifier()
if err := httpv.SignRequest(req, "sig1", params, signer); err != nil {
    return fmt.Errorf("rfc9421 signing failed: %w", err)
}
```

---

### 6. HTTP Body Handling Improvements

**Before:**
- Body was read once and consumed
- No mechanism for retry or re-reading

**After:**
- Body buffered in memory
- `GetBody` function set for HTTP client retries
- `ContentLength` explicitly set
- Body remains readable after digest computation

**Implications:**
- Enables HTTP client automatic retries
- Allows middleware to inspect body multiple times
- Changes chunked transfer encoding to content-length

---

## Benefits of This Refactoring

### ✅ Standards Compliance
- Full RFC 9421 HTTP Message Signatures conformance
- Interoperability with other RFC 9421 implementations
- Future-proof against protocol updates

### ✅ Enhanced Security
- Content-Digest ensures message integrity
- Prevents body tampering in transit
- Cryptographic binding between headers and body

### ✅ Improved Maintainability
- Reduced code complexity (~70 lines removed)
- Leverages battle-tested standard library
- Easier to understand and audit

### ✅ Better Interoperability
- Standard algorithm names (`es256k` vs custom identifiers)
- Compatible with other HTTP signature implementations
- Follows IANA registry conventions

---

## Migration Notes

### Breaking Changes
1. **Signature Components Changed:**
   - Old: `@target-uri`
   - New: `@path`, `@query`, `content-digest`

   **Impact:** Existing signatures won't verify with new implementation

2. **Algorithm Identifier:**
   - Secp256k1 now uses `es256k` instead of `ecdsa-p256-sha256`

   **Impact:** Verifiers must support new identifier

3. **Content-Digest Required:**
   - All signed requests now include `Content-Digest` header

   **Impact:** Verifiers must handle this component

### Compatibility
- **Not backward compatible** with old signing format
- Both client and server must upgrade simultaneously
- Consider versioning mechanism for gradual rollout

---

## Testing Coverage

The existing test suite (`a2a_signer_test.go`) covers:
- ✅ ECDSA key signing
- ✅ Ed25519 key signing
- ✅ KeyID inclusion
- ✅ Timestamp inclusion
- ✅ Standard components verification
- ✅ Context cancellation
- ✅ Custom components
- ✅ Custom timestamps
- ✅ Expiration handling
- ✅ Nonce support
- ✅ Error cases (nil request, nil keypair, empty DID)

**Note:** Tests may need updates to verify:
- Content-Digest header presence
- New component names (`@path`, `@query`)
- New algorithm identifier (`es256k`)

---

## Related Documentation

- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [Sage RFC 9421 Implementation](https://github.com/sage-x-project/sage/tree/main/pkg/agent/core/rfc9421)
- [IANA HTTP Signature Algorithms](https://www.iana.org/assignments/http-message-signatures/http-message-signatures.xhtml)

---

## Next Steps

1. **Update Integration Tests:** Verify end-to-end A2A communication
2. **Update Documentation:** API reference and integration guides
3. **Performance Testing:** Measure impact of body buffering
4. **Migration Guide:** Create guide for existing deployments
5. **Monitoring:** Add metrics for signature verification failures

---

*Last Updated: 2025-11-01*
*Branch: feature/multi-agent-demo*
*Base: main*

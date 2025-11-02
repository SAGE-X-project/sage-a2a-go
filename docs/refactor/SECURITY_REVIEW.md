# Security Review: RFC 9421 Refactoring

## Executive Summary

This document outlines security vulnerabilities and potential issues discovered in the refactored HTTP message signing implementation (`feature/multi-agent-demo` branch).

**Overall Risk Level:** 🟡 **MEDIUM-HIGH**

Critical issues require immediate attention before production deployment.

---

## 🔴 High Severity Issues

### 1. Memory Exhaustion (DoS) Vulnerability

**Location:** `default_a2a_signer.go:146-163`

**Code:**
```go
func ensureContentDigestHeader(req *http.Request) error {
    var body []byte
    if req.Body != nil {
        var err error
        body, err = io.ReadAll(req.Body)  // ⚠️ VULNERABILITY: No size limit
```

**Description:**
- Reads entire HTTP request body into memory without size constraints
- Attacker can send multi-gigabyte requests to exhaust server memory
- `io.ReadAll()` will attempt to allocate memory for entire payload

**Attack Scenario:**
```bash
# Attacker sends 5GB payload
curl -X POST https://api.example.com/agent/task \
  --data-binary @5gb_file.bin \
  -H "Content-Type: application/octet-stream"
```

**Impact:**
- **Availability:** Server out-of-memory (OOM) crash
- **DoS:** Multiple concurrent large requests = complete service disruption
- **Resource Exhaustion:** Affects all services on same host

**CVSS Score:** 7.5 (High) - AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H

**Recommended Fix:**
```go
const maxBodySize = 10 * 1024 * 1024 // 10MB limit

func ensureContentDigestHeader(req *http.Request) error {
    var body []byte
    if req.Body != nil {
        var err error
        // Use LimitReader to enforce maximum size
        limitedReader := io.LimitReader(req.Body, maxBodySize+1)
        body, err = io.ReadAll(limitedReader)
        if err != nil {
            return err
        }
        if len(body) > maxBodySize {
            return fmt.Errorf("request body exceeds maximum size of %d bytes", maxBodySize)
        }
    }
    // ... rest of function
}
```

**Additional Mitigations:**
- Configure reverse proxy (nginx/traefik) with `client_max_body_size`
- Implement rate limiting per IP/DID
- Monitor memory usage and alert on spikes

---

### 2. Algorithm Injection Vulnerability

**Location:** `default_a2a_signer.go:92-95`

**Code:**
```go
alg := s.getAlgorithm(keyPair.Type())
if opts.Algorithm != "" {
    alg = opts.Algorithm  // ⚠️ VULNERABILITY: No validation
}
```

**Description:**
- User-supplied algorithm accepted without validation
- No verification that algorithm matches key type
- Potential for "algorithm confusion" attacks

**Attack Scenarios:**

1. **"none" Algorithm Attack:**
```go
opts := &SigningOptions{
    Algorithm: "none",  // Disable signature verification
}
```

2. **Key Type Mismatch:**
```go
// Using Ed25519 key but claiming ES256K algorithm
opts := &SigningOptions{
    Algorithm: "es256k",  // Wrong algorithm for Ed25519 key
}
```

3. **Weak Algorithm Downgrade:**
```go
opts := &SigningOptions{
    Algorithm: "hs256",  // Symmetric HMAC instead of asymmetric signature
}
```

**Impact:**
- **Authentication Bypass:** "none" algorithm may disable verification
- **Signature Forgery:** Algorithm confusion can lead to key reuse attacks
- **Cryptographic Weakness:** Downgrade to weaker algorithms

**CVSS Score:** 8.1 (High) - AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H

**Recommended Fix:**
```go
// Whitelist of allowed algorithms per key type
var allowedAlgorithms = map[sagecrypto.KeyType][]string{
    sagecrypto.KeyTypeSecp256k1: {"es256k"},
    sagecrypto.KeyTypeEd25519:   {"ed25519"},
}

func (s *DefaultA2ASigner) validateAlgorithm(alg string, keyType sagecrypto.KeyType) error {
    allowed, ok := allowedAlgorithms[keyType]
    if !ok {
        return fmt.Errorf("unsupported key type: %v", keyType)
    }

    for _, a := range allowed {
        if a == alg {
            return nil
        }
    }

    return fmt.Errorf("algorithm %s not allowed for key type %v (allowed: %v)",
        alg, keyType, allowed)
}

// In SignRequestWithOptions:
alg := s.getAlgorithm(keyPair.Type())
if opts.Algorithm != "" {
    if err := s.validateAlgorithm(opts.Algorithm, keyPair.Type()); err != nil {
        return fmt.Errorf("invalid algorithm: %w", err)
    }
    alg = opts.Algorithm
}
```

---

### 3. Timestamp Manipulation (Replay Attack)

**Location:** `default_a2a_signer.go:88-91`

**Code:**
```go
created := opts.Created
if created == 0 {
    created = time.Now().Unix()
}
// ⚠️ VULNERABILITY: No validation of timestamp range
```

**Description:**
- Accepts arbitrary past or future timestamps
- No freshness validation
- Enables signature replay attacks

**Attack Scenarios:**

1. **Signature Replay:**
```go
// Attacker captures legitimate request from 1 hour ago
opts := &SigningOptions{
    Created: time.Now().Add(-1 * time.Hour).Unix(),
}
// Replays old signature as new request
```

2. **Time Travel Attack:**
```go
// Future-dated signature
opts := &SigningOptions{
    Created: time.Now().Add(365 * 24 * time.Hour).Unix(), // 1 year future
    Expires: time.Now().Add(366 * 24 * time.Hour).Unix(), // Valid for 1 year
}
```

3. **Clock Skew Exploitation:**
```go
// Exploits servers with incorrect clocks
opts := &SigningOptions{
    Created: 0,  // Unix epoch (1970)
}
```

**Impact:**
- **Replay Attacks:** Reuse captured signatures indefinitely
- **Authorization Bypass:** Old permissions may still be valid
- **Audit Trail Corruption:** Incorrect timestamps in logs

**CVSS Score:** 7.4 (High) - AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N

**Recommended Fix:**
```go
const (
    maxClockSkew  = 60  // 60 seconds tolerance for clock differences
    maxValidAge   = 300 // 5 minutes maximum age for signatures
)

func validateTimestamp(created int64) error {
    now := time.Now().Unix()

    // Check if timestamp is too far in the past
    if created < now - maxValidAge {
        return fmt.Errorf("signature timestamp too old (max age: %d seconds)", maxValidAge)
    }

    // Check if timestamp is in the future (allowing for clock skew)
    if created > now + maxClockSkew {
        return fmt.Errorf("signature timestamp in the future (max skew: %d seconds)", maxClockSkew)
    }

    return nil
}

// In SignRequestWithOptions:
created := opts.Created
if created == 0 {
    created = time.Now().Unix()
} else {
    if err := validateTimestamp(created); err != nil {
        return fmt.Errorf("invalid timestamp: %w", err)
    }
}
```

**Additional Recommendations:**
- Always use `Nonce` field for critical operations
- Implement server-side nonce tracking to prevent replay
- Consider shorter validity windows for sensitive operations

---

## 🟡 Medium Severity Issues

### 4. Slice Mutation Side Effect

**Location:** `default_a2a_signer.go:79-81`

**Code:**
```go
if !includes(opts.Components, "content-digest") {
    opts.Components = append(opts.Components, "content-digest")  // ⚠️ Mutates caller's slice
}
```

**Description:**
- Directly modifies the caller's `Components` slice
- If slice capacity allows, underlying array is mutated
- Violates principle of least surprise

**Attack/Bug Scenario:**
```go
opts := &SigningOptions{
    Components: []string{"@method", "@path"},
}

// First call
signer.SignRequestWithOptions(ctx, req1, did, kp, opts)
// opts.Components now: ["@method", "@path", "content-digest"]

// Second call - accidentally includes duplicate
signer.SignRequestWithOptions(ctx, req2, did, kp, opts)
// opts.Components now: ["@method", "@path", "content-digest", "content-digest"]

// Third call - more duplicates
signer.SignRequestWithOptions(ctx, req3, did, kp, opts)
// opts.Components now: ["@method", "@path", "content-digest", "content-digest", "content-digest"]
```

**Impact:**
- **Signature Failures:** Duplicate components may cause verification errors
- **Unexpected Behavior:** Caller's data structure modified
- **Memory Leak:** Repeated calls grow slice indefinitely

**Recommended Fix:**
```go
// Create defensive copy
components := make([]string, len(opts.Components))
copy(components, opts.Components)

if !includes(components, "content-digest") {
    components = append(components, "content-digest")
}

// Use components instead of opts.Components for rest of function
params := &rfc9421.SignatureInputParams{
    CoveredComponents: quoteComponents(components),  // Use copy, not original
    // ...
}
```

---

### 5. Content-Digest Validation Bypass

**Location:** `default_a2a_signer.go:82-86`

**Code:**
```go
if strings.TrimSpace(req.Header.Get("Content-Digest")) == "" {
    if err := ensureContentDigestHeader(req); err != nil {
        return fmt.Errorf("compute content-digest: %w", err)
    }
}
```

**Description:**
- Only computes digest if header is missing
- Attacker can provide malicious pre-computed digest
- No validation of existing digest correctness

**Attack Scenario:**
```go
// Attacker sends request with mismatched body and digest
req := httptest.NewRequest("POST", "/api/task",
    strings.NewReader(`{"amount": 1000}`))

// Malicious digest for different body: {"amount": 1000000}
req.Header.Set("Content-Digest",
    "sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:")

// Signer accepts pre-existing digest without verification
signer.SignRequest(ctx, req, did, kp)
```

**Impact:**
- **Integrity Violation:** Body and digest mismatch
- **MITM Amplification:** Attacker can modify body but keep old digest
- **Verification Bypass:** Receiver may not detect tampering

**CVSS Score:** 6.5 (Medium) - AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N

**Recommended Fix:**

**Option 1: Always Recompute (Recommended)**
```go
// Always compute fresh digest, ignore existing
if err := ensureContentDigestHeader(req); err != nil {
    return fmt.Errorf("compute content-digest: %w", err)
}
```

**Option 2: Validate Existing**
```go
existingDigest := strings.TrimSpace(req.Header.Get("Content-Digest"))
if existingDigest != "" {
    // Verify existing digest matches body
    if err := validateContentDigest(req, existingDigest); err != nil {
        return fmt.Errorf("invalid existing content-digest: %w", err)
    }
} else {
    if err := ensureContentDigestHeader(req); err != nil {
        return fmt.Errorf("compute content-digest: %w", err)
    }
}
```

---

### 6. Unsupported Key Type Silent Failure

**Location:** `default_a2a_signer.go:165-174`

**Code:**
```go
func (s *DefaultA2ASigner) getAlgorithm(k sagecrypto.KeyType) string {
    switch k {
    case sagecrypto.KeyTypeSecp256k1:
        return "es256k"
    case sagecrypto.KeyTypeEd25519:
        return "ed25519"
    default:
        return ""  // ⚠️ Silent failure
    }
}
```

**Description:**
- Returns empty string for unknown key types
- No error propagation
- Signing continues with invalid algorithm

**Attack/Bug Scenario:**
```go
// Future key type added to crypto package
keyPair := generateKey(crypto.KeyTypeRSA4096)

// getAlgorithm returns ""
// Signature created with alg="" - invalid but not caught
signer.SignRequest(ctx, req, did, keyPair)
```

**Impact:**
- **Invalid Signatures:** Created with empty algorithm field
- **Verification Failures:** Recipient cannot verify signature
- **Silent Errors:** No indication of problem until verification

**Recommended Fix:**
```go
func (s *DefaultA2ASigner) getAlgorithm(k sagecrypto.KeyType) (string, error) {
    switch k {
    case sagecrypto.KeyTypeSecp256k1:
        return "es256k", nil
    case sagecrypto.KeyTypeEd25519:
        return "ed25519", nil
    default:
        return "", fmt.Errorf("unsupported key type: %v", k)
    }
}

// Update callers to handle error:
alg, err := s.getAlgorithm(keyPair.Type())
if err != nil {
    return err
}
if opts.Algorithm != "" {
    if err := s.validateAlgorithm(opts.Algorithm, keyPair.Type()); err != nil {
        return err
    }
    alg = opts.Algorithm
}
```

---

### 7. HTTP Body Transformation Side Effects

**Location:** `default_a2a_signer.go:155-157`

**Code:**
```go
req.Body = io.NopCloser(bytes.NewReader(body))
req.ContentLength = int64(len(body))
req.GetBody = func() (io.ReadCloser, error) { ... }
```

**Description:**
- Transforms request from chunked encoding to content-length
- Changes HTTP transport characteristics
- May break HTTP/2 flow control

**Issues:**

1. **Transfer-Encoding Lost:**
```go
// Original request
req.Header.Set("Transfer-Encoding", "chunked")
req.Body = streamingBody  // Streaming, not buffered

// After ensureContentDigestHeader
req.Header.Get("Transfer-Encoding")  // Still says "chunked"
req.Body  // Now buffered with ContentLength set - mismatch!
```

2. **Large Payload Behavior Change:**
```go
// Original: 500MB streamed in chunks
// After: 500MB loaded into memory (DoS risk)
```

**Impact:**
- **Protocol Violation:** Conflicting content-length and transfer-encoding
- **Memory Issues:** Streaming requests become buffered
- **Performance:** Large uploads consume memory instead of streaming

**Recommended Fix:**
```go
func ensureContentDigestHeader(req *http.Request) error {
    // Remove conflicting Transfer-Encoding header
    req.Header.Del("Transfer-Encoding")

    var body []byte
    if req.Body != nil {
        var err error
        limitedReader := io.LimitReader(req.Body, maxBodySize+1)
        body, err = io.ReadAll(limitedReader)
        if err != nil {
            return err
        }
        if len(body) > maxBodySize {
            return fmt.Errorf("request body exceeds maximum size")
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

**Documentation Needed:**
```go
// ensureContentDigestHeader computes and sets the Content-Digest header.
//
// IMPORTANT: This function buffers the entire request body in memory.
// It will:
//   - Read the complete body (up to maxBodySize)
//   - Replace req.Body with a new io.Reader
//   - Set req.ContentLength explicitly
//   - Remove Transfer-Encoding: chunked if present
//   - Set req.GetBody for HTTP client retries
//
// This transformation is necessary for RFC 9421 compliance but has
// performance implications for large payloads.
func ensureContentDigestHeader(req *http.Request) error {
```

---

## 🟢 Low Severity Issues

### 8. Insufficient Input Validation in quoteComponents

**Location:** `default_a2a_signer.go:132-143`

**Code:**
```go
func quoteComponents(components []string) []string {
    out := make([]string, 0, len(components))
    for _, c := range components {
        c = strings.ToLower(strings.TrimSpace(c))
        if len(c) > 0 && c[0] == '"' && c[len(c)-1] == '"' {
            out = append(out, c)  // ⚠️ Accepts malformed quoted strings
            continue
        }
        out = append(out, fmt.Sprintf(`"%s"`, c))
    }
    return out
}
```

**Description:**
- Only checks first and last character for quotes
- Doesn't validate quote escaping or internal quotes
- May produce invalid component identifiers

**Edge Cases:**
```go
// Malformed inputs that pass validation
inputs := []string{
    `"@met"hod"`,      // Internal unescaped quote
    `"`,               // Single quote
    `""`,              // Empty quoted string
    `"@method""`,      // Double trailing quote
}

// All produce unexpected output
```

**Impact:**
- **Signature Failures:** Malformed components may not verify
- **Injection Potential:** Special characters in component names
- **RFC Violation:** Non-compliant Signature-Input format

**Recommended Fix:**
```go
func quoteComponents(components []string) []string {
    out := make([]string, 0, len(components))
    for _, c := range components {
        c = strings.ToLower(strings.TrimSpace(c))
        if c == "" {
            continue  // Skip empty components
        }

        // Strip any existing quotes and re-quote
        c = strings.Trim(c, `"`)

        // Validate component identifier (RFC 9421 section 2.1)
        if !isValidComponentIdentifier(c) {
            // Log warning or return error
            continue
        }

        out = append(out, fmt.Sprintf(`"%s"`, c))
    }
    return out
}

func isValidComponentIdentifier(s string) bool {
    // RFC 9421: sf-string = DQUOTE *chr DQUOTE
    // chr = unescaped / escaped
    // unescaped = %x20-21 / %x23-5B / %x5D-7E
    if len(s) == 0 {
        return false
    }

    for _, r := range s {
        // Allow printable ASCII except quotes and backslash
        if r < 0x20 || r > 0x7E || r == '"' || r == '\\' {
            return false
        }
    }
    return true
}
```

---

### 9. Missing Error Context

**Multiple Locations**

**Code:**
```go
// Location 1: default_a2a_signer.go:110
return fmt.Errorf("private key does not implement crypto.Signer: %T", priv)

// Location 2: default_a2a_signer.go:116
return fmt.Errorf("rfc9421 signing failed: %w", err)
```

**Description:**
- Errors lack contextual information about the request
- Difficult to debug which request failed
- No DID or request URL in error messages

**Impact:**
- **Debugging Difficulty:** Cannot identify failing requests in logs
- **Audit Trail Gaps:** Missing context for security events
- **Operations:** Hard to correlate errors with user actions

**Recommended Fix:**
```go
// Add request context to errors
return fmt.Errorf("private key does not implement crypto.Signer: %T (did: %s, url: %s)",
    priv, agentDID, req.URL.String())

return fmt.Errorf("rfc9421 signing failed for %s (did: %s): %w",
    req.URL.String(), agentDID, err)
```

---

## Summary Table

| # | Issue | Severity | CVSS | Exploitability | Fix Complexity |
|---|-------|----------|------|----------------|----------------|
| 1 | Memory DoS | 🔴 High | 7.5 | Easy | Low |
| 2 | Algorithm Injection | 🔴 High | 8.1 | Medium | Medium |
| 3 | Timestamp Manipulation | 🔴 High | 7.4 | Medium | Low |
| 4 | Slice Mutation | 🟡 Medium | N/A | Easy | Low |
| 5 | Digest Bypass | 🟡 Medium | 6.5 | Easy | Low |
| 6 | Silent Key Type Failure | 🟡 Medium | N/A | N/A | Low |
| 7 | Body Transform Side Effects | 🟡 Medium | N/A | N/A | Low |
| 8 | Input Validation | 🟢 Low | N/A | Low | Medium |
| 9 | Missing Error Context | 🟢 Low | N/A | N/A | Low |

---

## Remediation Priority

### Immediate (Before Production)
1. **Issue #1:** Add body size limit (`maxBodySize = 10MB`)
2. **Issue #2:** Implement algorithm whitelist validation
3. **Issue #3:** Add timestamp range validation (±5 minutes)

### Short Term (Next Sprint)
4. **Issue #5:** Always recompute Content-Digest
5. **Issue #4:** Fix slice mutation with defensive copy
6. **Issue #6:** Return error for unsupported key types

### Medium Term (Future Enhancement)
7. **Issue #7:** Document body transformation behavior
8. **Issue #8:** Improve component validation
9. **Issue #9:** Add request context to errors

---

## Testing Recommendations

### Security Tests to Add

```go
// Test 1: Large body DoS protection
func TestSignRequest_BodySizeLimit(t *testing.T) {
    largeBody := make([]byte, 11*1024*1024) // 11MB
    req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeBody))

    err := signer.SignRequest(ctx, req, did, kp)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "exceeds maximum size")
}

// Test 2: Algorithm validation
func TestSignRequest_InvalidAlgorithm(t *testing.T) {
    opts := &SigningOptions{
        Algorithm: "none",
    }

    err := signer.SignRequestWithOptions(ctx, req, did, kp, opts)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "invalid algorithm")
}

// Test 3: Timestamp validation
func TestSignRequest_OldTimestamp(t *testing.T) {
    opts := &SigningOptions{
        Created: time.Now().Add(-1 * time.Hour).Unix(),
    }

    err := signer.SignRequestWithOptions(ctx, req, did, kp, opts)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "timestamp too old")
}

// Test 4: Content-Digest verification
func TestSignRequest_InvalidContentDigest(t *testing.T) {
    req := httptest.NewRequest("POST", "/test",
        strings.NewReader(`{"data":"test"}`))
    req.Header.Set("Content-Digest", "sha-256=:INVALID:")

    err := signer.SignRequest(ctx, req, did, kp)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "invalid content-digest")
}
```

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE-400: Uncontrolled Resource Consumption](https://cwe.mitre.org/data/definitions/400.html)
- [CWE-327: Use of a Broken or Risky Cryptographic Algorithm](https://cwe.mitre.org/data/definitions/327.html)
- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [CVSS v3.1 Calculator](https://www.first.org/cvss/calculator/3.1)

---

*Security Review Date: 2025-11-01*
*Reviewed By: Claude Code Assistant*
*Code Version: feature/multi-agent-demo branch*

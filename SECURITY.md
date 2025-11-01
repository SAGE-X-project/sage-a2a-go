# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Currently supported versions:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

**Note**: This project is currently in active development. The first stable release will be v1.0.0.

## Reporting a Vulnerability

We take the security of sage-a2a-go seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### Please Do NOT

- Open a public GitHub issue for security vulnerabilities
- Discuss the vulnerability in public forums, social media, or mailing lists

### Please Do

**Report security vulnerabilities by creating a private security advisory**:

👉 **https://github.com/sage-x-project/sage-a2a-go/security/advisories/new**

This will create a private, confidential issue that only you and the maintainers can see.

Please include the following information in your report:

1. **Description**: A clear description of the vulnerability
2. **Impact**: What an attacker could achieve by exploiting the vulnerability
3. **Reproduction Steps**: Detailed steps to reproduce the issue
4. **Proof of Concept**: If possible, include a PoC or code sample
5. **Suggested Fix**: If you have ideas on how to fix it
6. **Environment**:
   - sage-a2a-go version
   - Go version
   - Operating system

### What to Expect

1. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
2. **Initial Assessment**: We will provide an initial assessment within 7 days
3. **Updates**: We will keep you informed of our progress
4. **Resolution**: We aim to resolve critical vulnerabilities within 30 days
5. **Disclosure**: We will coordinate with you on the disclosure timeline

### Disclosure Policy

- We follow coordinated vulnerability disclosure
- We will not publicly disclose vulnerabilities until:
  - A fix has been released
  - 90 days have passed since the initial report (whichever comes first)
- We will credit you in the security advisory (unless you prefer to remain anonymous)

## Security Best Practices

When using sage-a2a-go, please follow these security best practices:

### 1. Key Management

```go
//  Good: Load keys from secure storage
keyPair, err := loadKeyFromVault()

// L Bad: Hard-coded keys
keyPair := &keys.Ed25519KeyPair{
    Private: []byte("hardcoded-private-key"), // Never do this!
}
```

### 2. DID Validation

```go
//  Good: Always verify DID signatures
verifier := verifier.NewDIDVerifier()
if err := verifier.VerifySignature(message, signature); err != nil {
    return fmt.Errorf("signature verification failed: %w", err)
}

// L Bad: Skipping verification
// Never skip signature verification in production!
```

### 3. Context Timeouts

```go
//  Good: Use reasonable timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// L Bad: No timeout
ctx := context.Background() // Could hang indefinitely
```

### 4. TLS Configuration

```go
//  Good: Use TLS for production
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        MinVersion: tls.VersionTLS12,
    },
}

// L Bad: Skip TLS verification
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true, // Never do this in production!
    },
}
```

### 5. Input Validation

```go
//  Good: Validate all inputs
if len(message) > maxMessageSize {
    return errors.New("message too large")
}

// L Bad: No input validation
// Always validate user inputs!
```

## Known Security Considerations

### Cryptographic Dependencies

This library depends on cryptographic implementations from:
- `github.com/sage-x-project/sage` - For DID and key management
- `crypto/ecdsa`, `crypto/ed25519` - For signature generation/verification

We regularly monitor these dependencies for security updates.

### Rate Limiting

The library does not implement rate limiting by default. In production:

```go
// Implement rate limiting at the application level
limiter := rate.NewLimiter(rate.Every(time.Second), 10)
if !limiter.Allow() {
    return errors.New("rate limit exceeded")
}
```

### Audit Log

For security-critical applications, implement audit logging:

```go
// Log all security-relevant events
logger.Info("signature verification successful",
    "did", msg.From,
    "timestamp", time.Now(),
)
```

## Security Audits

| Date | Version | Auditor | Report |
|------|---------|---------|--------|
| TBD  | TBD     | TBD     | TBD    |

*No formal security audits have been conducted yet. This section will be updated when audits are completed.*

## Security Updates

Security updates will be:
- Released as patch versions (e.g., v1.0.1)
- Announced in GitHub Security Advisories
- Published in the CHANGELOG with a `[SECURITY]` tag
- Communicated to users via GitHub Discussions

## Vulnerability Response Team

The security response team can be reached via:
- **Private Security Advisory**: https://github.com/sage-x-project/sage-a2a-go/security/advisories/new
- **GitHub Issues** (for non-sensitive security discussions): https://github.com/sage-x-project/sage-a2a-go/issues

## Bug Bounty Program

We do not currently have a bug bounty program. However, we greatly appreciate responsible disclosure and will:
- Publicly acknowledge your contribution (with your permission)
- List you in our security hall of fame
- Provide early access to fixes

## Additional Resources

- [OWASP Go Security Best Practices](https://owasp.org/www-project-go-secure-coding-practices-guide/)
- [Go Security Policy](https://golang.org/security)
- [A2A Protocol Security Considerations](https://github.com/a2aproject/a2a)

## Questions?

If you have questions about security but don't have a vulnerability to report:
- Check existing [GitHub Discussions](https://github.com/sage-x-project/sage-a2a-go/discussions)
- Create a new discussion in the Security category

Thank you for helping keep sage-a2a-go and our users safe!

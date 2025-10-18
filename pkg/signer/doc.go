// Package signer provides HTTP message signing for A2A (Agent-to-Agent) communication.
//
// This package implements RFC9421 HTTP Message Signatures with SAGE DIDs
// (Decentralized Identifiers) for secure agent-to-agent authentication.
//
// # Signing HTTP Requests
//
// Use A2ASigner to sign outgoing HTTP requests with your agent's DID:
//
//	signer := signer.NewDefaultA2ASigner()
//	req, _ := http.NewRequest("POST", "https://agent.example.com/task", body)
//
//	err := signer.SignRequest(ctx, req, agentDID, keyPair)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// This adds Signature and Signature-Input headers to the request.
//
// # Custom Signing Options
//
// Customize which components to sign and include additional parameters:
//
//	opts := &signer.SigningOptions{
//	    Components: []string{"@method", "@target-uri", "@authority", "content-type"},
//	    Created:    time.Now().Unix(),
//	    Expires:    time.Now().Add(5 * time.Minute).Unix(),
//	    Nonce:      "random-nonce-value",
//	    Algorithm:  "ES256K",
//	}
//
//	err := signer.SignRequestWithOptions(ctx, req, agentDID, keyPair, opts)
//
// # Signature Components
//
// Common HTTP components to include in signatures:
//
//   - @method - HTTP method (GET, POST, etc.)
//   - @target-uri - Request URL
//   - @authority - Host header
//   - content-type - Content-Type header
//   - content-digest - Hash of request body
//   - authorization - Authorization header
//
// # DID Integration
//
// The signer includes the agent's DID as the keyid parameter:
//
//	Signature-Input: sig1=("@method" "@target-uri");created=1234567890;keyid="did:sage:ethereum:0x..."
//
// This allows the recipient to:
//
//  1. Extract the sender's DID from the signature
//  2. Resolve the public key from blockchain
//  3. Verify the signature cryptographically
//  4. Authenticate the sender's identity
//
// # Supported Algorithms
//
// The package supports both ECDSA and Ed25519 signatures:
//
//   - ES256K (ECDSA with secp256k1) for Ethereum/EVM chains
//   - EdDSA (Ed25519) for Solana and other Ed25519-based chains
//
// The algorithm is automatically determined from the key pair type.
//
// # RFC9421 Compliance
//
// This implementation follows RFC9421 HTTP Message Signatures specification:
//
//   - JWS-style base64url encoding
//   - Signature base string construction
//   - Structured field serialization
//   - Component identifiers (@method, @target-uri, etc.)
//   - Signature parameters (created, expires, keyid, nonce)
//
// # Example: Complete Request Flow
//
//	// Create request
//	taskJSON, _ := json.Marshal(task)
//	req, _ := http.NewRequest("POST", targetURL, bytes.NewReader(taskJSON))
//	req.Header.Set("Content-Type", "application/json")
//
//	// Sign request
//	signer := signer.NewDefaultA2ASigner()
//	err := signer.SignRequest(ctx, req, myDID, myKeyPair)
//
//	// Send request
//	client := &http.Client{}
//	resp, err := client.Do(req)
//
// The recipient can then verify the signature using the verifier package.
//
// # Security Considerations
//
//   - Always use HTTPS for transport security
//   - Include timestamp (created) to prevent replay attacks
//   - Consider adding expiration (expires) for time-limited signatures
//   - Include content-digest when signing request bodies
//   - Use nonces for additional replay protection
//   - Keep private keys secure and never transmit them
//
// # Error Handling
//
// Common signing errors:
//
//   - Nil request: request parameter is nil
//   - Nil key pair: cryptographic key pair is nil
//   - Empty DID: agent DID is empty string
//   - Signing failed: cryptographic operation failed
//   - Context canceled: operation interrupted
package signer

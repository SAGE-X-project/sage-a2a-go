// Package client provides an HTTP client with automatic DID-based request signing.
//
// The client package implements an HTTP client wrapper that automatically signs
// all outgoing requests using DID (Decentralized Identifier) authentication.
// This enables secure Agent-to-Agent (A2A) communication with cryptographic
// authentication.
//
// # Features
//
//   - Automatic DID signature generation for all HTTP requests
//   - Support for POST, GET, and custom HTTP methods
//   - RFC9421 compliant HTTP message signatures
//   - Context-aware request execution
//   - Custom HTTP client injection
//   - ECDSA (secp256k1) and Ed25519 key support
//
// # Basic Usage
//
//	// Create A2A client with DID and key pair
//	agentDID := did.AgentDID("did:sage:ethereum:0x...")
//	keyPair, _ := crypto.GenerateSecp256k1KeyPair()
//	client := client.NewA2AClient(agentDID, keyPair, nil)
//
//	// Send POST request (automatically signed)
//	ctx := context.Background()
//	body := []byte(`{"task": "process"}`)
//	resp, err := client.Post(ctx, "https://agent.example.com/api/task", body)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer resp.Body.Close()
//
// # Custom HTTP Client
//
//	// Use custom HTTP client with timeout
//	httpClient := &http.Client{
//	    Timeout: 30 * time.Second,
//	}
//	client := client.NewA2AClient(agentDID, keyPair, httpClient)
//
// # GET Requests
//
//	resp, err := client.Get(ctx, "https://agent.example.com/api/status")
//
// # Custom Requests
//
//	req, _ := http.NewRequest("PUT", "https://agent.example.com/api/data", body)
//	req.Header.Set("Content-Type", "application/json")
//	resp, err := client.Do(ctx, req)
//
// # How It Works
//
// The A2AClient wraps a standard http.Client and intercepts all requests
// to add RFC9421 HTTP message signatures. The signature includes:
//
//   - Request method, URI, and headers
//   - DID as the keyid parameter
//   - Cryptographic signature using the agent's private key
//   - Timestamp for replay attack prevention
//
// The receiving server can verify the signature using the DIDAuthMiddleware
// from the server package, which resolves the public key from the DID and
// validates the signature.
//
// # Error Handling
//
//	resp, err := client.Post(ctx, url, body)
//	if err != nil {
//	    // Handle network errors, signing errors, or context cancellation
//	    log.Printf("Request failed: %v", err)
//	    return
//	}
//
//	if resp.StatusCode != http.StatusOK {
//	    // Handle HTTP errors
//	    body, _ := io.ReadAll(resp.Body)
//	    log.Printf("HTTP error %d: %s", resp.StatusCode, body)
//	}
//
// # Thread Safety
//
// A2AClient is safe for concurrent use by multiple goroutines. The underlying
// http.Client is designed for this purpose.
//
// # Context Cancellation
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	resp, err := client.Get(ctx, url)
//	if err == context.DeadlineExceeded {
//	    log.Println("Request timed out")
//	}
//
// # Integration with A2A Protocol
//
// This client is designed for use with the Agent-to-Agent (A2A) Protocol,
// which uses JSON-RPC 2.0 over HTTP(S) for agent communication. The DID
// signatures provide cryptographic authentication for all A2A messages.
//
// See the server package for the corresponding middleware that verifies
// these signatures on the receiving end.
package client

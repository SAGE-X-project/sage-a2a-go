// Package server provides HTTP middleware for DID-based request verification.
//
// The server package implements HTTP middleware that verifies DID (Decentralized
// Identifier) signatures on incoming requests. This enables secure Agent-to-Agent
// (A2A) communication with cryptographic authentication.
//
// # Features
//
//   - Automatic DID signature verification for HTTP requests
//   - RFC9421 compliant HTTP message signature validation
//   - DID extraction and context propagation
//   - Optional verification mode (allow unsigned requests)
//   - CORS preflight support (OPTIONS requests)
//   - Custom error handler support
//   - Request body preservation
//   - ECDSA (secp256k1) and Ed25519 key support
//
// # Basic Usage
//
//	// Create DID authentication middleware
//	ethereumClient, _ := ethereum.NewEthereumClientV4(config)
//	middleware := server.NewDIDAuthMiddleware(ethereumClient)
//
//	// Wrap HTTP handler
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    // Extract verified DID from context
//	    agentDID, ok := server.GetAgentDIDFromContext(r.Context())
//	    if !ok {
//	        http.Error(w, "Unauthorized", http.StatusUnauthorized)
//	        return
//	    }
//
//	    // Process authenticated request
//	    fmt.Fprintf(w, "Authenticated as: %s", agentDID)
//	})
//
//	// Apply middleware
//	http.Handle("/api/", middleware.Wrap(handler))
//
// # Optional Verification
//
//	// Allow unsigned requests to pass through
//	middleware.SetOptional(true)
//
// # Custom Error Handler
//
//	middleware.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
//	    log.Printf("Authentication failed: %v", err)
//	    http.Error(w, "Custom error message", http.StatusForbidden)
//	})
//
// # How It Works
//
// The DIDAuthMiddleware performs the following steps for each request:
//
//  1. Checks for Signature and Signature-Input headers
//  2. Skips verification for OPTIONS requests (CORS preflight)
//  3. Extracts the DID from the signature's keyid parameter
//  4. Resolves the public key from the blockchain using the DID
//  5. Verifies the RFC9421 HTTP message signature
//  6. Adds the verified DID to the request context
//  7. Calls the next handler in the chain
//
// If verification fails at any step, the middleware returns 401 Unauthorized
// and does not call the next handler.
//
// # Context Propagation
//
//	func myHandler(w http.ResponseWriter, r *http.Request) {
//	    // Extract DID from context
//	    agentDID, ok := server.GetAgentDIDFromContext(r.Context())
//	    if !ok {
//	        // This shouldn't happen if middleware is working
//	        http.Error(w, "No DID in context", http.StatusInternalServerError)
//	        return
//	    }
//
//	    // Use DID for authorization, logging, etc.
//	    log.Printf("Request from agent: %s", agentDID)
//
//	    // Check if agent has permission
//	    if !hasPermission(agentDID, "task.execute") {
//	        http.Error(w, "Forbidden", http.StatusForbidden)
//	        return
//	    }
//
//	    // Process request...
//	}
//
// # CORS Support
//
// The middleware automatically allows OPTIONS requests to pass through without
// signature verification. This is essential for browser-based clients that
// send CORS preflight requests.
//
//	// OPTIONS requests are not verified
//	// Other methods (GET, POST, etc.) require signatures
//
// # Body Preservation
//
// The middleware reads and preserves the request body so it can be used by
// downstream handlers. The body is buffered in memory during verification
// and restored before calling the next handler.
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    // Body is available even after middleware verification
//	    body, err := io.ReadAll(r.Body)
//	    if err != nil {
//	        http.Error(w, "Error reading body", http.StatusBadRequest)
//	        return
//	    }
//
//	    // Process body...
//	}
//
// # Error Handling
//
// By default, verification errors return 401 Unauthorized with an error message.
// You can customize this behavior:
//
//	middleware.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
//	    // Log error
//	    log.Printf("Auth error from %s: %v", r.RemoteAddr, err)
//
//	    // Return custom response
//	    w.Header().Set("Content-Type", "application/json")
//	    w.WriteHeader(http.StatusUnauthorized)
//	    json.NewEncoder(w).Encode(map[string]string{
//	        "error": "Authentication failed",
//	        "message": "Invalid or missing DID signature",
//	    })
//	})
//
// # Integration with A2A Protocol
//
// This middleware is designed for use with the Agent-to-Agent (A2A) Protocol,
// which uses JSON-RPC 2.0 over HTTP(S) for agent communication. The DID
// signatures provide cryptographic authentication for all A2A messages.
//
// See the client package for the corresponding client that generates these
// signatures automatically.
//
// # Thread Safety
//
// The middleware is safe for concurrent use by multiple goroutines and can
// be shared across multiple HTTP servers.
//
// # Performance Considerations
//
//   - Signature verification requires public key resolution from blockchain
//   - Consider caching public keys to reduce latency
//   - Body buffering requires memory proportional to body size
//   - For large bodies, consider streaming verification if supported
//
// # Security Best Practices
//
//   - Always use HTTPS/TLS 1.3+ in production
//   - Implement rate limiting to prevent DoS attacks
//   - Log all authentication failures for security monitoring
//   - Regularly rotate agent keys
//   - Validate DID format before blockchain lookup
//   - Consider implementing replay attack protection
//
// See the examples directory for complete usage examples.
package server

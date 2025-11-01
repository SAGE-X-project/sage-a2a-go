// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
//
// sage-a2a-go is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sage-a2a-go is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with sage-a2a-go.  If not, see <https://www.gnu.org/licenses/>.

// Package verifier provides DID-based HTTP signature verification for A2A communication.
//
// This package implements RFC9421 HTTP Message Signatures verification using
// blockchain-anchored DIDs (Decentralized Identifiers) for agent authentication.
//
// # DID Verification
//
// The DIDVerifier resolves public keys from DIDs and verifies HTTP signatures:
//
//	client := ethereum.NewEthereumClientV4(config)
//	selector := verifier.NewDefaultKeySelector(client)
//	sigVerifier := verifier.NewRFC9421Verifier()
//	didVerifier := verifier.NewDefaultDIDVerifier(client, selector, sigVerifier)
//
//	// Verify signature with known DID
//	err := didVerifier.VerifyHTTPSignature(ctx, req, agentDID)
//	if err != nil {
//	    log.Fatal("verification failed:", err)
//	}
//
//	// Verify and extract DID from request
//	agentDID, err := didVerifier.VerifyHTTPSignatureWithKeyID(ctx, req)
//
// # Key Selection
//
// The KeySelector automatically chooses the appropriate cryptographic key
// based on the blockchain protocol:
//
//	selector := verifier.NewDefaultKeySelector(client)
//
//	// Select ECDSA key for Ethereum
//	pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "ethereum")
//
//	// Select Ed25519 key for Solana
//	pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "solana")
//
//	// Automatic selection (uses first available key)
//	pubKey, keyType, err := selector.SelectKey(ctx, agentDID, "")
//
// # Protocol-Based Selection
//
// Different blockchain protocols require different cryptographic algorithms:
//
//   - ethereum → ECDSA (secp256k1)
//   - solana → Ed25519
//   - unknown/empty → First available verified key
//
// # RFC9421 HTTP Signature Verification
//
// The RFC9421Verifier implements HTTP Message Signatures verification:
//
//	sigVerifier := verifier.NewRFC9421Verifier()
//	err := sigVerifier.VerifyHTTPRequest(req, publicKey)
//
// This wraps SAGE's RFC9421 implementation with the SignatureVerifier interface.
//
// # Multi-Key Support
//
// Agents can register multiple cryptographic keys for different purposes:
//
//   - ECDSA key for Ethereum transactions
//   - Ed25519 key for Solana transactions
//   - Different keys for different protocols
//
// The KeySelector automatically selects the appropriate key based on context.
//
// # Error Handling
//
// Common verification errors:
//
//   - Invalid signature: cryptographic verification failed
//   - DID not found: agent not registered on blockchain
//   - Key not found: requested key type not available
//   - Missing headers: required signature headers not present
//   - Invalid DID format: malformed DID string
//
// # Security Considerations
//
//   - Always verify signatures before processing requests
//   - Check DID format before resolution
//   - Use HTTPS for all agent-to-agent communication
//   - Validate timestamp to prevent replay attacks
//   - Verify agent capabilities before executing tasks
package verifier

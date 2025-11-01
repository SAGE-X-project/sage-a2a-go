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

// Package protocol provides Agent Card functionality for the A2A (Agent-to-Agent) protocol.
//
// This package implements creation, signing, and verification of Agent Cards using
// JSON Web Signatures (JWS) with blockchain-anchored DIDs (Decentralized Identifiers).
//
// # Agent Cards
//
// An Agent Card is a metadata document that describes an AI agent's identity,
// capabilities, and service endpoints. It includes:
//
//   - DID (Decentralized Identifier) - blockchain-anchored identity
//   - Name and description - human-readable information
//   - Endpoint - service URL where the agent is accessible
//   - Capabilities - operations the agent can perform
//   - Public keys - cryptographic keys for verification
//   - Metadata - custom key-value pairs
//   - Timestamps - creation and expiration times
//
// # Creating Agent Cards
//
// Use the AgentCardBuilder for a fluent API to create cards:
//
//	card := protocol.NewAgentCardBuilder(
//	    did.AgentDID("did:sage:ethereum:0x..."),
//	    "MyAgent",
//	    "https://agent.example.com",
//	).
//	    WithDescription("AI agent with DID authentication").
//	    WithCapabilities("task.execute", "messaging.send").
//	    WithMetadata("region", "us-west-2").
//	    WithExpiresAt(time.Now().Add(365 * 24 * time.Hour)).
//	    Build()
//
// # Signing Agent Cards
//
// Sign cards with JWS compact serialization:
//
//	signer := protocol.NewDefaultAgentCardSigner(client)
//	signedCard, err := signer.SignAgentCard(ctx, card, keyPair)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Verifying Signatures
//
// Verify signatures using DID resolution from blockchain:
//
//	err = signer.VerifyAgentCard(ctx, signedCard)
//	if err != nil {
//	    log.Fatal("verification failed:", err)
//	}
//
// Or verify with a pre-fetched public key:
//
//	err = signer.VerifyAgentCardWithKey(ctx, signedCard, publicKey)
//
// # Validation
//
// Agent Cards provide built-in validation methods:
//
//	// Validate required fields
//	if err := card.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if expired
//	if card.IsExpired() {
//	    log.Println("Card has expired")
//	}
//
//	// Check for specific capability
//	if card.HasCapability("task.execute") {
//	    // Agent can execute tasks
//	}
//
// # Supported Algorithms
//
// The package supports both ECDSA and Ed25519 signing algorithms:
//
//   - ES256K (ECDSA with secp256k1) for Ethereum/EVM chains
//   - EdDSA (Ed25519) for Solana and other Ed25519-based chains
//
// The algorithm is automatically selected based on the key type.
package protocol

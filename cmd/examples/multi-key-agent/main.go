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

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/protocol"
	"github.com/sage-x-project/sage-a2a-go/pkg/verifier"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// mockEthereumClient simulates blockchain with multi-key support
type mockEthereumClient struct {
    publicKeys map[did.AgentDID]map[did.KeyType]interface{}
}

func (m *mockEthereumClient) ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error) {
	keys := []did.AgentKey{}
	if keyMap, found := m.publicKeys[agentDID]; found {
		for keyType, pubKey := range keyMap {
			keyData, _ := did.MarshalPublicKey(pubKey)
			keys = append(keys, did.AgentKey{
				Type:      keyType,
				KeyData:   keyData,
				Verified:  true,
				CreatedAt: time.Now(),
			})
		}
	}
	return keys, nil
}

func (m *mockEthereumClient) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
    if keyMap, found := m.publicKeys[agentDID]; found {
        if pubKey, found := keyMap[keyType]; found {
            return pubKey, nil
        }
    }
    return nil, fmt.Errorf("key type %s not found for DID %s", keyType, agentDID)
}

// Satisfy DIDResolver for key selection
func (m *mockEthereumClient) GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error) {
    d := did.AgentDID(didStr)
    meta := &did.AgentMetadataV4{DID: d, IsActive: true}
    if keyMap, ok := m.publicKeys[d]; ok {
        for kt, pk := range keyMap {
            if keyData, err := did.MarshalPublicKey(pk); err == nil {
                meta.Keys = append(meta.Keys, did.AgentKey{
                    Type:      kt,
                    KeyData:   keyData,
                    Verified:  true,
                    CreatedAt: time.Now(),
                })
            }
        }
    }
    return meta, nil
}

// This example demonstrates managing an agent with multiple cryptographic keys
func main() {
	fmt.Println("=== Multi-Key Agent Example ===")

	ctx := context.Background()

	// Step 1: Create an agent DID
	fmt.Println("Step 1: Creating multi-key agent...")
	agentDID := did.AgentDID("did:sage:ethereum:0xMULTIKEYAGENT1234567890ABCDEF1234567890")
	fmt.Printf("  Agent DID: %s\n\n", agentDID)

	// Step 2: Generate multiple keys for different protocols
	fmt.Println("Step 2: Generating cryptographic keys...")

	// Generate ECDSA key (for Ethereum/EVM chains)
	fmt.Println("  Generating ECDSA key (secp256k1) for Ethereum...")
	ecdsaPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	ecdsaPubKey := &ecdsaPrivKey.PublicKey
	fmt.Printf("  ✓ ECDSA key generated\n")

	// Generate Ed25519 key (for Solana)
	fmt.Println("  Generating Ed25519 key for Solana...")
	ed25519PubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  ✓ Ed25519 key generated\n\n")

	// Step 3: Create Agent Card with multi-key information
	fmt.Println("Step 3: Creating Agent Card with multiple public keys...")

	// Encode public keys for Agent Card
	ecdsaKeyData, _ := did.MarshalPublicKey(ecdsaPubKey)
	ed25519KeyData, _ := did.MarshalPublicKey(ed25519PubKey)

	card := protocol.NewAgentCardBuilder(
		agentDID,
		"MultiKeyAgent",
		"https://multi-key-agent.example.com",
	).
		WithDescription("AI agent supporting multiple blockchain protocols").
		WithCapabilities(
			"ethereum.sign",
			"ethereum.verify",
			"solana.sign",
			"solana.verify",
			"cross-chain.communication",
		).
		WithPublicKey(protocol.PublicKeyInfo{
			ID:      "ethereum-key-1",
			Type:    "EcdsaSecp256k1VerificationKey2019",
			KeyData: string(ecdsaKeyData),
			Purpose: []string{"authentication", "signing"},
		}).
		WithPublicKey(protocol.PublicKeyInfo{
			ID:      "solana-key-1",
			Type:    "Ed25519VerificationKey2020",
			KeyData: string(ed25519KeyData),
			Purpose: []string{"authentication", "signing"},
		}).
		WithMetadata("supported_chains", []string{"ethereum", "solana"}).
		WithMetadata("version", "2.0.0").
		Build()

	fmt.Printf("  ✓ Agent Card created with %d public keys\n\n", len(card.PublicKeys))

	// Step 4: Display public keys
	fmt.Println("Step 4: Agent's Public Keys")
	fmt.Println("  ----------------------------------------")
	for i, keyInfo := range card.PublicKeys {
		fmt.Printf("  Key %d:\n", i+1)
		fmt.Printf("    ID:      %s\n", keyInfo.ID)
		fmt.Printf("    Type:    %s\n", keyInfo.Type)
		fmt.Printf("    Purpose: %v\n", keyInfo.Purpose)
		fmt.Printf("    KeyData: %s...\n", keyInfo.KeyData[:min(40, len(keyInfo.KeyData))])
		fmt.Println()
	}
	fmt.Println("  ----------------------------------------")

	// Step 5: Set up mock blockchain with both keys
	fmt.Println("Step 5: Registering keys on blockchain (mock)...")

	mockClient := &mockEthereumClient{
		publicKeys: map[did.AgentDID]map[did.KeyType]interface{}{
			agentDID: {
				did.KeyTypeECDSA:   ecdsaPubKey,
				did.KeyTypeEd25519: ed25519PubKey,
			},
		},
	}

	fmt.Println("  ✓ ECDSA key registered for Ethereum protocol")
	fmt.Println("  ✓ Ed25519 key registered for Solana protocol")

	// Step 6: Demonstrate protocol-based key selection
	fmt.Println("Step 6: Protocol-based key selection...")

	selector := verifier.NewDefaultKeySelector(mockClient)

	// Select key for Ethereum
	fmt.Println("\n  Scenario 1: Communicating with Ethereum network")
	ethPubKey, ethKeyType, err := selector.SelectKey(ctx, agentDID, "ethereum")
	if err != nil {
		log.Fatalf("Failed to select Ethereum key: %v", err)
	}
	fmt.Printf("    Protocol: ethereum\n")
	fmt.Printf("    Selected: %s\n", ethKeyType)
	fmt.Printf("    Key Type: %T\n", ethPubKey)
	fmt.Printf("    ✓ Using ECDSA key for Ethereum transactions\n")

	// Select key for Solana
	fmt.Println("\n  Scenario 2: Communicating with Solana network")
	solPubKey, solKeyType, err := selector.SelectKey(ctx, agentDID, "solana")
	if err != nil {
		log.Fatalf("Failed to select Solana key: %v", err)
	}
	fmt.Printf("    Protocol: solana\n")
	fmt.Printf("    Selected: %s\n", solKeyType)
	fmt.Printf("    Key Type: %T\n", solPubKey)
	fmt.Printf("    ✓ Using Ed25519 key for Solana transactions\n")

	// Select key for unknown protocol (fallback)
	fmt.Println("\n  Scenario 3: Communicating with unknown protocol")
	unknownPubKey, unknownKeyType, err := selector.SelectKey(ctx, agentDID, "unknown-chain")
	if err != nil {
		log.Fatalf("Failed to select key: %v", err)
	}
	fmt.Printf("    Protocol: unknown-chain\n")
	fmt.Printf("    Selected: %s (fallback to first available)\n", unknownKeyType)
	fmt.Printf("    Key Type: %T\n", unknownPubKey)
	fmt.Printf("    ✓ Using fallback key selection\n\n")

	// Step 7: Display Agent Card JSON with all keys
	fmt.Println("Step 7: Complete Agent Card JSON representation...")
	fmt.Println("  ----------------------------------------")

	cardJSON, err := json.MarshalIndent(card, "  ", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  %s\n", string(cardJSON))
	fmt.Println("  ----------------------------------------")

	// Step 8: Demonstrate cross-chain capabilities
	fmt.Println("Step 8: Cross-chain operation simulation...")

	operations := []struct {
		chain   string
		action  string
		keyType string
	}{
		{"ethereum", "Sign smart contract transaction", "ECDSA"},
		{"solana", "Sign token transfer", "Ed25519"},
		{"ethereum", "Verify message signature", "ECDSA"},
		{"solana", "Verify program instruction", "Ed25519"},
	}

	for i, op := range operations {
		fmt.Printf("\n  Operation %d:\n", i+1)
		fmt.Printf("    Chain:   %s\n", op.chain)
		fmt.Printf("    Action:  %s\n", op.action)

		pubKey, keyType, err := selector.SelectKey(ctx, agentDID, op.chain)
		if err != nil {
			fmt.Printf("    ✗ Failed: %v\n", err)
			continue
		}

		fmt.Printf("    Key:     %s\n", keyType)
		fmt.Printf("    Status:  ✓ Ready\n")

		// Simulate operation
		time.Sleep(50 * time.Millisecond)
		_ = pubKey // Use the selected key

		fmt.Printf("    Result:  ✓ %s completed successfully\n", op.action)
	}

	fmt.Println()

	// Summary
	fmt.Println("=== Multi-Key Agent Summary ===")
	fmt.Println("  Agent Configuration:")
	fmt.Printf("    DID:              %s\n", card.DID)
	fmt.Printf("    Name:             %s\n", card.Name)
	fmt.Printf("    Total Keys:       %d\n", len(card.PublicKeys))
	fmt.Printf("    Capabilities:     %d\n", len(card.Capabilities))
	fmt.Printf("    Supported Chains: %v\n", card.Metadata["supported_chains"])
	fmt.Println()

	fmt.Println("  Key Management Benefits:")
	fmt.Println("    ✓ Protocol-specific key selection")
	fmt.Println("    ✓ Automatic fallback for unknown protocols")
	fmt.Println("    ✓ Single agent identity across multiple chains")
	fmt.Println("    ✓ Blockchain-agnostic authentication")
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey takeaways:")
	fmt.Println("  ✓ Agents can maintain multiple cryptographic keys")
	fmt.Println("  ✓ KeySelector automatically chooses the right key per protocol")
	fmt.Println("  ✓ ECDSA for Ethereum/EVM chains, Ed25519 for Solana")
	fmt.Println("  ✓ Agent Card stores all public keys with metadata")
	fmt.Println("  ✓ Enables true cross-chain agent identity and operations")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

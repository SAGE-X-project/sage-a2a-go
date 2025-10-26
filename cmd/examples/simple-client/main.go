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
	"fmt"
	"log"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/sage-x-project/sage-a2a-go/pkg/transport"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func init() {
	// Initialize crypto package
	crypto.SetKeyGenerators(
		func() (crypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
		func() (crypto.KeyPair, error) { return keys.GenerateP256KeyPair() },
	)
	crypto.SetStorageConstructors(
		func() crypto.KeyStorage { return storage.NewMemoryKeyStorage() },
	)
	crypto.SetFormatConstructors(
		func() crypto.KeyExporter { return formats.NewJWKExporter() },
		func() crypto.KeyExporter { return formats.NewPEMExporter() },
		func() crypto.KeyImporter { return formats.NewJWKImporter() },
		func() crypto.KeyImporter { return formats.NewPEMImporter() },
	)
}

func main() {
	fmt.Println("SAGE A2A Go - Simple Client Example")
	fmt.Println("=====================================")

	// Create context
	ctx := context.Background()

	// Generate client DID and key pair
	fmt.Println("\n1. Generating client DID and key pair...")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}
	clientDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678")
	fmt.Printf("   Client DID: %s\n", clientDID)

	// Create agent card for target agent
	fmt.Println("\n2. Creating agent card for target agent...")
	targetCard := &a2a.AgentCard{
		Name:               "Example Agent",
		Description:        "An example A2A agent",
		URL:                "https://agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		AdditionalInterfaces: []a2a.AgentInterface{
			{
				Transport: a2a.TransportProtocolJSONRPC,
				URL:       "https://agent.example.com/rpc",
			},
		},
	}
	fmt.Printf("   Target Agent: %s\n", targetCard.Name)
	fmt.Printf("   Target URL: %s\n", targetCard.URL)

	// Create DID-authenticated client
	fmt.Println("\n3. Creating DID-authenticated A2A client...")
	client, err := transport.NewDIDAuthenticatedClient(
		ctx,
		clientDID,
		keyPair,
		targetCard,
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Destroy()
	fmt.Println("   Client created successfully!")

	// Create a message
	fmt.Println("\n4. Creating a message...")
	message := a2a.NewMessage(
		a2a.MessageRoleUser,
		&a2a.TextPart{Text: "Hello from SAGE A2A Go!"},
	)
	fmt.Printf("   Message: %s\n", message.Parts[0].(*a2a.TextPart).Text)

	// Send message (this will fail without a real server, but demonstrates the API)
	fmt.Println("\n5. Sending message to agent...")
	fmt.Println("   (Note: This will fail without a real A2A server running)")

	params := &a2a.MessageSendParams{
		Message: message,
	}

	result, err := client.SendMessage(ctx, params)
	if err != nil {
		fmt.Printf("   ⚠️  Expected error (no server running): %v\n", err)
		fmt.Println("\n✅ Example completed successfully!")
		fmt.Println("\nTo test with a real server:")
		fmt.Println("  1. Start an A2A-compliant agent server")
		fmt.Println("  2. Update targetCard.URL to point to your server")
		fmt.Println("  3. Run this example again")
		return
	}

	// If we got a response (shouldn't happen without a server)
	fmt.Printf("   ✅ Received response: %+v\n", result)

	fmt.Println("\n✅ Example completed!")
}

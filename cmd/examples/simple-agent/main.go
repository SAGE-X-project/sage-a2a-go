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
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/protocol"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// This example demonstrates how to create a simple agent with DID and Agent Card
func main() {
	fmt.Println("=== Simple Agent with DID Example ===\n")

	// Step 1: Create an agent DID
	// In a real scenario, this would come from blockchain registration
	agentDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678")
	fmt.Printf("Step 1: Agent DID created\n")
	fmt.Printf("  DID: %s\n\n", agentDID)

	// Step 2: Create an Agent Card using the builder pattern
	fmt.Println("Step 2: Creating Agent Card...")

	card := protocol.NewAgentCardBuilder(
		agentDID,
		"SimpleAgent",
		"https://simple-agent.example.com",
	).
		WithDescription("A simple AI agent demonstrating DID-based authentication").
		WithCapabilities(
			"task.create",
			"task.execute",
			"messaging.send",
			"messaging.receive",
		).
		WithMetadata("version", "1.0.0").
		WithMetadata("region", "us-west-2").
		WithMetadata("type", "assistant").
		WithExpiresAt(time.Now().Add(365 * 24 * time.Hour)). // Valid for 1 year
		Build()

	fmt.Printf("  Agent Card created successfully!\n\n")

	// Step 3: Validate the Agent Card
	fmt.Println("Step 3: Validating Agent Card...")
	if err := card.Validate(); err != nil {
		log.Fatalf("Agent Card validation failed: %v", err)
	}
	fmt.Printf("  ✓ Validation passed\n\n")

	// Step 4: Display Agent Card details
	fmt.Println("Step 4: Agent Card Details")
	fmt.Println("  ----------------------------------------")
	fmt.Printf("  DID:          %s\n", card.DID)
	fmt.Printf("  Name:         %s\n", card.Name)
	fmt.Printf("  Description:  %s\n", card.Description)
	fmt.Printf("  Endpoint:     %s\n", card.Endpoint)
	fmt.Printf("  Capabilities: %v\n", card.Capabilities)
	fmt.Printf("  Created At:   %s\n", time.Unix(card.CreatedAt, 0).Format(time.RFC3339))
	fmt.Printf("  Expires At:   %s\n", time.Unix(card.ExpiresAt, 0).Format(time.RFC3339))
	fmt.Printf("  Metadata:     %v\n", card.Metadata)
	fmt.Println("  ----------------------------------------\n")

	// Step 5: Check capabilities
	fmt.Println("Step 5: Checking capabilities...")
	capabilities := []string{"task.execute", "messaging.send", "data.process"}
	for _, cap := range capabilities {
		hasCapability := card.HasCapability(cap)
		status := "✗"
		if hasCapability {
			status = "✓"
		}
		fmt.Printf("  %s Capability '%s': %v\n", status, cap, hasCapability)
	}
	fmt.Println()

	// Step 6: Check expiration
	fmt.Println("Step 6: Checking expiration status...")
	if card.IsExpired() {
		fmt.Println("  ✗ Agent Card has EXPIRED")
	} else {
		daysUntilExpiry := time.Unix(card.ExpiresAt, 0).Sub(time.Now()).Hours() / 24
		fmt.Printf("  ✓ Agent Card is VALID (expires in %.0f days)\n", daysUntilExpiry)
	}
	fmt.Println()

	// Step 7: Serialize Agent Card to JSON
	fmt.Println("Step 7: Serializing Agent Card to JSON...")
	cardJSON, err := json.MarshalIndent(card, "  ", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize Agent Card: %v", err)
	}
	fmt.Println("  JSON representation:")
	fmt.Printf("  %s\n\n", string(cardJSON))

	// Step 8: Demonstrate usage in context
	fmt.Println("Step 8: Using Agent Card in application context...")
	ctx := context.Background()

	// Simulate agent performing a task
	if err := performTask(ctx, card); err != nil {
		log.Fatalf("Task execution failed: %v", err)
	}

	fmt.Println("\n=== Example completed successfully! ===")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. See agent-communication example for DID-based message signing")
	fmt.Println("  2. See multi-key-agent example for managing multiple cryptographic keys")
	fmt.Println("  3. Integrate with SAGE blockchain for DID registration")
}

// performTask simulates an agent performing a task
func performTask(ctx context.Context, card *protocol.AgentCard) error {
	fmt.Printf("  Agent '%s' starting task execution...\n", card.Name)

	// Check if agent has required capability
	if !card.HasCapability("task.execute") {
		return fmt.Errorf("agent does not have 'task.execute' capability")
	}

	// Simulate task processing
	time.Sleep(100 * time.Millisecond)

	fmt.Println("  ✓ Task executed successfully")
	return nil
}

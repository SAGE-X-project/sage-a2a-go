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

// Package main demonstrates SSE (Server-Sent Events) streaming with sage-a2a-go.
//
// This example shows:
//   - Real-time message streaming
//   - Handling all 4 A2A event types
//   - Progress tracking
//   - Graceful cancellation
//   - Error recovery
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/sage-x-project/sage-a2a-go/pkg/transport"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func init() {
	// Initialize crypto subsystem with all key generators
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

// ProgressTracker tracks streaming progress
type ProgressTracker struct {
	startTime time.Time
	messages  int
	artifacts int
	status    a2a.TaskState
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		startTime: time.Now(),
	}
}

// HandleEvent processes an event and updates progress
func (p *ProgressTracker) HandleEvent(event a2a.Event) {
	switch e := event.(type) {
	case *a2a.Message:
		p.messages++
		p.printMessage(e)

	case *a2a.Task:
		p.status = e.Status.State
		p.printTask(e)

	case *a2a.TaskStatusUpdateEvent:
		p.status = e.Status.State
		p.printStatusUpdate(e)

	case *a2a.TaskArtifactUpdateEvent:
		p.artifacts++
		p.printArtifact(e)
	}
}

func (p *ProgressTracker) printMessage(msg *a2a.Message) {
	fmt.Printf("\n[MESSAGE] %s\n", msg.ID)

	for _, part := range msg.Parts {
		if textPart, ok := part.(*a2a.TextPart); ok {
			fmt.Printf("  %s\n", textPart.Text)
		}
	}
}

func (p *ProgressTracker) printTask(task *a2a.Task) {
	fmt.Printf("\n[TASK] %s\n", task.ID)
	fmt.Printf("  Status: %s\n", task.Status.State)
	fmt.Printf("  Context: %s\n", task.ContextID)
}

func (p *ProgressTracker) printStatusUpdate(update *a2a.TaskStatusUpdateEvent) {
	fmt.Printf("\n[STATUS] %s\n", update.Status.State)

	if update.Status.State.Terminal() {
		duration := time.Since(p.startTime)
		fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("Task completed in %s\n", duration.Round(time.Second))
		fmt.Printf("Messages received: %d\n", p.messages)
		fmt.Printf("Artifacts created: %d\n", p.artifacts)
		fmt.Print(strings.Repeat("=", 60) + "\n")
	}
}

func (p *ProgressTracker) printArtifact(artifact *a2a.TaskArtifactUpdateEvent) {
	fmt.Printf("\n[ARTIFACT] %s\n", artifact.Artifact.ID)
	if artifact.Artifact.Description != "" {
		fmt.Printf("  Description: %s\n", artifact.Artifact.Description)
	}
}

// streamConversation demonstrates streaming a conversation
func streamConversation(client *transport.DIDHTTPTransport, prompt string) error {
	fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Streaming conversation...\n")
	fmt.Print(strings.Repeat("=", 60) + "\n\n")

	// Setup context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\n\nCancelling stream...")
		cancel()
	}()

	// Create message params
	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: prompt}},
		},
	}

	// Track progress
	tracker := NewProgressTracker()

	// Stream events
	for event, err := range client.SendStreamingMessage(ctx, params) {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("\nStream cancelled by user")
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("stream timeout: %w", err)
			}
			return fmt.Errorf("stream error: %w", err)
		}

		tracker.HandleEvent(event)

		// Exit if task completed
		if tracker.status.Terminal() {
			break
		}
	}

	return nil
}

// demonstrateReconnection shows reconnecting to an existing task
func demonstrateReconnection(client *transport.DIDHTTPTransport, taskID a2a.TaskID) error {
	fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Reconnecting to task: %s\n", taskID)
	fmt.Print(strings.Repeat("=", 60) + "\n\n")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	params := &a2a.TaskIDParams{ID: taskID}
	tracker := NewProgressTracker()

	fmt.Println("Receiving backfill events...")

	for event, err := range client.ResubscribeToTask(ctx, params) {
		if err != nil {
			return fmt.Errorf("resubscribe error: %w", err)
		}

		tracker.HandleEvent(event)

		// Exit if task completed
		if tracker.status.Terminal() {
			break
		}
	}

	return nil
}

// demonstrateErrorHandling shows error handling patterns
func demonstrateErrorHandling(client *transport.DIDHTTPTransport) {
	fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Error Handling Demo\n")
	fmt.Print(strings.Repeat("=", 60) + "\n\n")

	// Short timeout to trigger error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{&a2a.TextPart{Text: "Tell me a long story"}},
		},
	}

	for event, err := range client.SendStreamingMessage(ctx, params) {
		if err != nil {
			// Demonstrate error type checking
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Println("✓ Timeout detected (expected)")
				return
			}
			fmt.Printf("Unexpected error: %v\n", err)
			return
		}

		fmt.Printf("Event: %T\n", event)
	}
}

func main() {
	fmt.Println("SSE Streaming Example - sage-a2a-go")
	fmt.Println()

	// NOTE: This is a demonstration example
	// In production, you would:
	// 1. Load your DID and key from secure storage
	// 2. Fetch the target agent card from a registry
	// 3. Validate the agent card signature

	// Setup demo credentials
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	myDID := did.AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Demo target agent card
	targetCard := &a2a.AgentCard{
		Name: "Demo Assistant",
		URL:  "https://demo.a2a-agent.com",
	}

	// Create DID HTTP transport directly for streaming
	httpTransport := transport.NewDIDHTTPTransport(
		targetCard.URL,
		myDID,
		keyPair,
		nil, // use default HTTP client
	).(*transport.DIDHTTPTransport)
	defer httpTransport.Destroy()

	// Example 1: Stream a conversation
	fmt.Println("Example 1: Streaming Conversation")
	prompt := "Tell me a short story about a robot learning to code"

	if err := streamConversation(httpTransport, prompt); err != nil {
		log.Printf("Stream error: %v", err)
	}

	// Example 2: Error handling
	fmt.Println("\n\nExample 2: Error Handling")
	demonstrateErrorHandling(httpTransport)

	// Example 3: Reconnection (requires a real task ID)
	// Uncomment to test reconnection
	// fmt.Println("\n\nExample 3: Task Reconnection")
	// taskID := a2a.TaskID("task-123-from-previous-run")
	// if err := demonstrateReconnection(httpTransport, taskID); err != nil {
	//     log.Printf("Reconnection error: %v", err)
	// }

	fmt.Println("\n✅ Examples completed!")
}

// Helper for strings.Repeat since it's not imported
var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: func(s string, count int) string {
		result := ""
		for i := 0; i < count; i++ {
			result += s
		}
		return result
	},
}

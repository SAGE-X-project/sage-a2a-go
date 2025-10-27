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

// Package main demonstrates a simple chat application using SSE streaming.
//
// This example shows a basic chat interface that:
//   - Sends messages to an A2A agent
//   - Streams responses in real-time
//   - Maintains conversation context
//   - Provides a simple CLI interface
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// ChatSession manages a chat conversation
type ChatSession struct {
	transport *transport.DIDHTTPTransport
	contextID string
	taskID    a2a.TaskID
}

// NewChatSession creates a new chat session
func NewChatSession(httpTransport *transport.DIDHTTPTransport) *ChatSession {
	return &ChatSession{
		transport: httpTransport,
		contextID: a2a.NewContextID(),
	}
}

// SendMessage sends a message and streams the response
func (s *ChatSession) SendMessage(userInput string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	params := &a2a.MessageSendParams{
		Message: &a2a.Message{
			Role:      a2a.MessageRoleUser,
			Parts:     []a2a.Part{&a2a.TextPart{Text: userInput}},
			ContextID: s.contextID,
		},
	}

	if s.taskID != "" {
		params.Message.TaskID = s.taskID
	}

	fmt.Printf("\n🤖 Assistant: ")

	responseText := ""

	for event, err := range s.transport.SendStreamingMessage(ctx, params) {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("stream error: %w", err)
		}

		switch e := event.(type) {
		case *a2a.Message:
			// Stream message text as it arrives
			for _, part := range e.Parts {
				if textPart, ok := part.(*a2a.TextPart); ok {
					// Print new content (simulating streaming)
					newText := textPart.Text[len(responseText):]
					fmt.Print(newText)
					responseText = textPart.Text
				}
			}

		case *a2a.Task:
			// Save task ID for context
			s.taskID = e.ID

		case *a2a.TaskStatusUpdateEvent:
			// Check if task is done
			if e.Status.State.Terminal() {
				fmt.Println() // New line after response
				return nil
			}
		}
	}

	fmt.Println() // New line after response
	return nil
}

func printWelcome() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║         A2A Chat Demo - sage-a2a-go                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Real-time streaming chat powered by SSE")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  /quit     - Exit the chat")
	fmt.Println("  /help     - Show this help")
	fmt.Println("  <message> - Chat with the assistant")
	fmt.Println()
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println()
}

func main() {
	printWelcome()

	// Setup demo credentials
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	myDID := did.AgentDID("did:sage:ethereum:0xdemo")

	targetCard := &a2a.AgentCard{
		Name: "Chat Assistant",
		URL:  "https://demo.a2a-agent.com",
	}

	// Create transport
	httpTransport := transport.NewDIDHTTPTransport(
		targetCard.URL,
		myDID,
		keyPair,
		nil,
	).(*transport.DIDHTTPTransport)
	defer httpTransport.Destroy()

	// Create chat session
	session := NewChatSession(httpTransport)

	// Demo mode: Send preset messages
	demoMessages := []string{
		"Hello! Can you help me?",
		"What can you do?",
		"Tell me a joke",
	}

	fmt.Println("🎬 Demo Mode: Sending preset messages...")

	for _, msg := range demoMessages {
		fmt.Printf("👤 You: %s\n", msg)

		if err := session.SendMessage(msg); err != nil {
			log.Printf("Error: %v", err)
			break
		}

		time.Sleep(1 * time.Second) // Pause between messages
		fmt.Println()
	}

	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println("\n💡 Demo completed!")
	fmt.Println()
	fmt.Println("To enable interactive mode:")
	fmt.Println("1. Uncomment the interactive loop below")
	fmt.Println("2. Provide real agent credentials")
	fmt.Println("3. Run the program")
	fmt.Println()

	// Interactive mode (disabled by default)
	// Uncomment to enable:
	/*
		for {
			fmt.Print("👤 You: ")

			if !scanner.Scan() {
				break
			}

			input := strings.TrimSpace(scanner.Text())

			if input == "" {
				continue
			}

			// Handle commands
			if strings.HasPrefix(input, "/") {
				switch input {
				case "/quit", "/exit":
					fmt.Println("\n👋 Goodbye!")
					return
				case "/help":
					printWelcome()
					continue
				default:
					fmt.Println("❌ Unknown command. Type /help for help.")
					continue
				}
			}

			// Send message
			if err := session.SendMessage(input); err != nil {
				log.Printf("Error: %v", err)
				break
			}

			fmt.Println()
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Scanner error: %v", err)
		}
	*/
}

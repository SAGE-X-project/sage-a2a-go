package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	stdcrypto "crypto"

	"github.com/sage-x-project/sage-a2a-go/pkg/protocol"
	"github.com/sage-x-project/sage-a2a-go/pkg/signer"
	"github.com/sage-x-project/sage-a2a-go/pkg/verifier"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// TaskRequest represents a task sent between agents
type TaskRequest struct {
	TaskID      string `json:"task_id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// TaskResponse represents a task response
type TaskResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// mockKeyPair implements crypto.KeyPair for demonstration
type mockKeyPair struct {
	pubKey  *ecdsa.PublicKey
	privKey *ecdsa.PrivateKey
}

func (m *mockKeyPair) ID() string {
	return "mock-key-id"
}

func (m *mockKeyPair) PublicKey() stdcrypto.PublicKey {
	return m.pubKey
}

func (m *mockKeyPair) PrivateKey() stdcrypto.PrivateKey {
	return m.privKey
}

func (m *mockKeyPair) Type() crypto.KeyType {
	return crypto.KeyTypeSecp256k1
}

func (m *mockKeyPair) Sign(data []byte) ([]byte, error) {
	// Simple signature for demonstration
	return []byte("mock-signature-" + string(data[:min(10, len(data))])), nil
}

func (m *mockKeyPair) Verify(data, signature []byte) error {
	// Simple verification for demonstration
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// mockEthereumClient simulates blockchain DID resolution
type mockEthereumClient struct {
	publicKeys map[did.AgentDID]*ecdsa.PublicKey
}

func (m *mockEthereumClient) ResolveAllPublicKeys(ctx context.Context, agentDID did.AgentDID) ([]did.AgentKey, error) {
	// Return empty list for simplicity in this example
	return []did.AgentKey{}, nil
}

func (m *mockEthereumClient) ResolvePublicKeyByType(ctx context.Context, agentDID did.AgentDID, keyType did.KeyType) (interface{}, error) {
	if pubKey, found := m.publicKeys[agentDID]; found {
		return pubKey, nil
	}
	return nil, fmt.Errorf("DID not found: %s", agentDID)
}

// This example demonstrates agent-to-agent communication with DID-based authentication
func main() {
	fmt.Println("=== Agent-to-Agent Communication Example ===\n")

	ctx := context.Background()

	// Step 1: Create two agents with DIDs
	fmt.Println("Step 1: Creating two agents...")

	agentADID := did.AgentDID("did:sage:ethereum:0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	agentBDID := did.AgentDID("did:sage:ethereum:0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")

	// Generate keys for both agents
	privKeyA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	keyPairA := &mockKeyPair{pubKey: &privKeyA.PublicKey, privKey: privKeyA}

	privKeyB, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	_ = &mockKeyPair{pubKey: &privKeyB.PublicKey, privKey: privKeyB} // keyPairB for Agent B (not used in this simplified example)

	fmt.Printf("  Agent A DID: %s\n", agentADID)
	fmt.Printf("  Agent B DID: %s\n\n", agentBDID)

	// Step 2: Create Agent Cards for both agents
	fmt.Println("Step 2: Creating Agent Cards...")

	cardA := protocol.NewAgentCardBuilder(agentADID, "AgentA", "https://agent-a.example.com").
		WithDescription("Sender agent").
		WithCapabilities("task.create", "messaging.send").
		Build()

	cardB := protocol.NewAgentCardBuilder(agentBDID, "AgentB", "https://agent-b.example.com").
		WithDescription("Receiver agent").
		WithCapabilities("task.execute", "messaging.receive").
		Build()

	fmt.Printf("  ✓ Agent A card created: %s\n", cardA.Name)
	fmt.Printf("  ✓ Agent B card created: %s\n\n", cardB.Name)

	// Step 3: Set up mock blockchain client for DID resolution
	fmt.Println("Step 3: Setting up DID resolution (mock blockchain)...")

	mockClient := &mockEthereumClient{
		publicKeys: map[did.AgentDID]*ecdsa.PublicKey{
			agentADID: &privKeyA.PublicKey,
			agentBDID: &privKeyB.PublicKey,
		},
	}

	fmt.Println("  ✓ Mock blockchain configured with agent public keys\n")

	// Step 4: Agent A creates and signs a task request
	fmt.Println("Step 4: Agent A creating a task request...")

	taskReq := TaskRequest{
		TaskID:      "task-12345",
		Type:        "data-processing",
		Description: "Process customer data for analytics",
		Priority:    "high",
	}

	taskJSON, _ := json.Marshal(taskReq)
	fmt.Printf("  Task Request: %s\n", string(taskJSON))

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://agent-b.example.com/tasks", bytes.NewReader(taskJSON))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request with Agent A's DID
	fmt.Println("\n  Signing request with Agent A's DID...")
	a2aSigner := signer.NewDefaultA2ASigner()
	if err := a2aSigner.SignRequest(ctx, req, agentADID, keyPairA); err != nil {
		log.Fatalf("Failed to sign request: %v", err)
	}

	fmt.Println("  ✓ Request signed successfully")
	fmt.Printf("  Signature-Input: %s\n", req.Header.Get("Signature-Input"))
	fmt.Printf("  Signature: %s\n\n", req.Header.Get("Signature"))

	// Step 5: Agent B receives and verifies the request
	fmt.Println("Step 5: Agent B receiving and verifying the request...")

	// Create a test server for Agent B
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the signature
		selector := verifier.NewDefaultKeySelector(mockClient)
		sigVerifier := verifier.NewRFC9421Verifier()
		didVerifier := verifier.NewDefaultDIDVerifier(mockClient, selector, sigVerifier)

		fmt.Println("  Verifying HTTP signature...")
		if err := didVerifier.VerifyHTTPSignature(ctx, r, agentADID); err != nil {
			fmt.Printf("  ✗ Signature verification failed: %v\n", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Println("  ✓ Signature verified successfully")
		fmt.Printf("  ✓ Authenticated as: %s\n", agentADID)

		// Read task request
		body, _ := io.ReadAll(r.Body)
		var task TaskRequest
		json.Unmarshal(body, &task)

		fmt.Printf("  ✓ Received task: %s (Priority: %s)\n\n", task.TaskID, task.Priority)

		// Process task and send response
		response := TaskResponse{
			TaskID:  task.TaskID,
			Status:  "accepted",
			Message: "Task queued for processing",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer serverB.Close()

	// Update request URL to test server
	req.URL.Host = serverB.URL[7:] // Remove "http://"
	req.URL.Scheme = "http"
	req.RequestURI = ""

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Step 6: Agent A receives the response
	fmt.Println("Step 6: Agent A receiving response from Agent B...")

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed with status: %d", resp.StatusCode)
	}

	var taskResp TaskResponse
	json.NewDecoder(resp.Body).Decode(&taskResp)

	fmt.Printf("  Response Status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	fmt.Printf("  Task Status: %s\n", taskResp.Status)
	fmt.Printf("  Message: %s\n\n", taskResp.Message)

	// Summary
	fmt.Println("=== Communication Flow Summary ===")
	fmt.Println("  1. Agent A created a task request")
	fmt.Println("  2. Agent A signed the request with its DID")
	fmt.Println("  3. Agent B received and verified the signature")
	fmt.Println("  4. Agent B authenticated Agent A via DID resolution")
	fmt.Println("  5. Agent B processed the task and responded")
	fmt.Println("  6. Secure, DID-authenticated communication completed!")
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey takeaways:")
	fmt.Println("  ✓ HTTP requests are signed with agent's DID")
	fmt.Println("  ✓ Signatures are verified using blockchain-anchored public keys")
	fmt.Println("  ✓ RFC9421 HTTP Message Signatures ensure integrity")
	fmt.Println("  ✓ Agents can verify each other's identity without centralized auth")
}

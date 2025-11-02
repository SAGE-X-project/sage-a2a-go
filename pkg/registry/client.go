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

package registry

import (
	"context"
	"fmt"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	dideth "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// RegistrationClient provides a simplified wrapper for agent registration
// using the three-phase commit-reveal-activate flow.
//
// THREE-PHASE REGISTRATION FLOW:
//
// Phase 1: COMMIT (anti-front-running)
//   - Generate salt and compute commitment hash
//   - Send commitment hash + stake to contract
//   - Wait 1-60 minutes before reveal
//
// Phase 2: REGISTER (reveal commitment)
//   - Send full registration params with salt
//   - Contract verifies commitment hash matches
//   - Agent registered but not active yet
//
// Phase 3: ACTIVATE (time-locked)
//   - Wait minimum 1 hour after registration
//   - Call activateAgent() to enable agent
//   - Stake is refunded after activation
//
// Usage:
//
//	client := registry.NewRegistrationClient(config)
//
//	// Phase 1: Commit
//	status, err := client.CommitRegistration(ctx, params)
//	// Wait 1-60 minutes
//
//	// Phase 2: Register
//	status, err = client.RegisterAgent(ctx, status)
//	// Wait 1 hour
//
//	// Phase 3: Activate
//	status, err = client.ActivateAgent(ctx, status)
type RegistrationClient struct {
	agentCardClient *dideth.AgentCardClient
	config          *ClientConfig
}

// ClientConfig configures the registration client
type ClientConfig struct {
	// RPCURL is the Ethereum RPC endpoint
	RPCURL string

	// RegistryAddress is the AgentCardRegistry contract address
	RegistryAddress string

	// PrivateKey is the Ethereum private key (hex string without 0x prefix)
	// Required for registration transactions
	PrivateKey string

	// ChainID is the Ethereum chain ID (optional)
	ChainID *int64
}

// RegistrationParams contains parameters for agent registration
type RegistrationParams = did.RegistrationParams

// CommitmentStatus tracks the current phase of registration
type CommitmentStatus = did.CommitmentStatus

// RegistrationPhase indicates the current phase
type RegistrationPhase = did.RegistrationPhase

// Registration phases
const (
	PhaseCommitted  = did.PhaseCommitted
	PhaseRegistered = did.PhaseRegistered
	PhaseActivated  = did.PhaseActivated
)

// NewRegistrationClient creates a new registration client for Ethereum-based agents
func NewRegistrationClient(config *ClientConfig) (*RegistrationClient, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL is required")
	}
	if config.RegistryAddress == "" {
		return nil, fmt.Errorf("registry address is required")
	}
	if config.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required for registration")
	}

	// Create SAGE RegistryConfig
	sageConfig := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		RPCEndpoint:     config.RPCURL,
		ContractAddress: config.RegistryAddress,
		PrivateKey:      config.PrivateKey,
	}

	// Create AgentCard v4 client
	agentCardClient, err := dideth.NewAgentCardClient(sageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AgentCard client: %w", err)
	}

	return &RegistrationClient{
		agentCardClient: agentCardClient,
		config:          config,
	}, nil
}

// CommitRegistration performs Phase 1: commit registration with hash.
//
// This generates a random salt, computes a commitment hash, and submits
// it to the blockchain with a stake (typically 0.01 ETH).
//
// The commitment prevents front-running attacks by hiding registration
// details until reveal phase.
//
// You MUST wait 1-60 minutes before calling RegisterAgent.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - params: Registration parameters (name, URL, keys, etc.)
//
// Returns:
//   - status: Commitment status with salt and timestamp
//   - error: Any errors during commitment
func (c *RegistrationClient) CommitRegistration(ctx context.Context, params *RegistrationParams) (*CommitmentStatus, error) {
	status, err := c.agentCardClient.CommitRegistration(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("commit registration failed: %w", err)
	}
	return status, nil
}

// RegisterAgent performs Phase 2: reveal commitment and register.
//
// This reveals the full registration parameters along with the salt
// from Phase 1. The contract verifies that the commitment hash matches.
//
// You MUST wait at least 1 minute and at most 60 minutes after
// CommitRegistration before calling this.
//
// After successful registration, you MUST wait at least 1 hour
// before calling ActivateAgent.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - status: Commitment status from CommitRegistration
//
// Returns:
//   - status: Updated status with registration timestamp
//   - error: Any errors during registration
func (c *RegistrationClient) RegisterAgent(ctx context.Context, status *CommitmentStatus) (*CommitmentStatus, error) {
	updatedStatus, err := c.agentCardClient.RegisterAgent(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("register agent failed: %w", err)
	}
	return updatedStatus, nil
}

// ActivateAgent performs Phase 3: activate the agent.
//
// This makes the agent active and visible in the registry.
// The stake from Phase 1 is refunded after activation.
//
// You MUST wait at least 1 hour after RegisterAgent before calling this.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - status: Status from RegisterAgent
//
// Returns error if activation fails
func (c *RegistrationClient) ActivateAgent(ctx context.Context, status *CommitmentStatus) error {
	err := c.agentCardClient.ActivateAgent(ctx, status)
	if err != nil {
		return fmt.Errorf("activate agent failed: %w", err)
	}
	return nil
}

// Helper functions for building registration parameters

// NewRegistrationParams creates a new RegistrationParams struct.
//
// This is a convenience function for building registration parameters
// with all required fields.
//
// Parameters:
//   - name: Human-readable agent name
//   - serviceURL: Agent service endpoint URL
//   - signingKey: ECDSA signing key pair (secp256k1)
//   - kemKey: X25519 KEM key pair for HPKE
//   - preferredTransport: Preferred transport protocol
//   - capabilities: List of agent capabilities
//
// Returns initialized RegistrationParams
func NewRegistrationParams(
	name string,
	serviceURL string,
	signingKey crypto.KeyPair,
	kemKey crypto.KeyPair,
	preferredTransport string,
	capabilities []string,
) *RegistrationParams {
	// Note: RegistrationParams is a SAGE type with fields:
	// DID, Name, Description, Endpoint, Capabilities (string), Keys, KeyTypes, Signatures, Salt
	// This function signature is kept for compatibility, but users should work with SAGE types directly
	params := &RegistrationParams{
		Name:         name,
		Endpoint:     serviceURL,
		Capabilities: "", // Will be filled by AgentCardClient
	}

	// Store keys for later use (these will be marshaled by AgentCardClient)
	// Note: These fields don't exist in SAGE's RegistrationParams
	// Users should use AgentCardClient methods directly for registration

	return params
}

// GetSAGEClient returns the underlying SAGE AgentCard client for advanced usage
func (c *RegistrationClient) GetSAGEClient() *dideth.AgentCardClient {
	return c.agentCardClient
}

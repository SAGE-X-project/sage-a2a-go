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

	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/sage-x-project/sage/pkg/agent/did"
	dideth "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
)

// Resolver wraps SAGE's DID resolver for agent metadata resolution
type Resolver struct {
	ethClient     *dideth.EthereumClient
	agentCardV4   *dideth.AgentCardClient
	rpcURL        string
	registryAddr  string
}

// ResolverConfig configures the DID resolver
type ResolverConfig struct {
	// RPCURL is the Ethereum RPC endpoint
	RPCURL string

	// RegistryAddress is the AgentRegistry contract address
	RegistryAddress string

	// ChainID is the Ethereum chain ID (optional)
	ChainID *int64
}

// NewResolver creates a new DID resolver for Ethereum-based agents
func NewResolver(config *ResolverConfig) (*Resolver, error) {
	if config.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL is required")
	}
	if config.RegistryAddress == "" {
		return nil, fmt.Errorf("registry address is required")
	}

	// Create SAGE RegistryConfig
	sageConfig := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		RPCEndpoint:     config.RPCURL,
		ContractAddress: config.RegistryAddress,
	}
	if config.ChainID != nil {
		// Set network based on chain ID if provided
		// This is optional for basic resolution
	}

	// Create Ethereum client for public key resolution
	ethClient, err := dideth.NewEthereumClient(sageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ethereum client: %w", err)
	}

	// Create AgentCard v4 client for metadata resolution
	agentCardV4, err := dideth.NewAgentCardClient(sageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AgentCard client: %w", err)
	}

	return &Resolver{
		ethClient:    ethClient,
		agentCardV4:  agentCardV4,
		rpcURL:       config.RPCURL,
		registryAddr: config.RegistryAddress,
	}, nil
}

// GetAgentMetadata retrieves agent metadata by DID
func (r *Resolver) GetAgentMetadata(ctx context.Context, agentDID identity.AgentDID) (*identity.AgentMetadata, error) {
	// Convert to SAGE DID type
	sageDID := did.AgentDID(agentDID)

	// Resolve using Ethereum client (v1.5.2 unified Resolve method)
	metadata, err := r.ethClient.Resolve(ctx, sageDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent metadata: %w", err)
	}

	// Convert back to sage-a2a-go type
	return (*identity.AgentMetadata)(metadata), nil
}

// ResolvePublicKey resolves the signing public key for a DID
func (r *Resolver) ResolvePublicKey(ctx context.Context, agentDID identity.AgentDID) (interface{}, error) {
	// Get full metadata
	metadata, err := r.GetAgentMetadata(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	if metadata.PublicKey == nil {
		return nil, fmt.Errorf("no public key found for DID: %s", agentDID)
	}

	return metadata.PublicKey, nil
}

// ResolveKEMKey resolves the KEM (X25519) public key for a DID
func (r *Resolver) ResolveKEMKey(ctx context.Context, agentDID identity.AgentDID) (interface{}, error) {
	// Get full metadata
	metadata, err := r.GetAgentMetadata(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	if metadata.PublicKEMKey == nil {
		return nil, fmt.Errorf("no KEM key found for DID: %s", agentDID)
	}

	return metadata.PublicKEMKey, nil
}

// IsActive checks if an agent is active in the registry
func (r *Resolver) IsActive(ctx context.Context, agentDID identity.AgentDID) (bool, error) {
	// Get metadata
	metadata, err := r.GetAgentMetadata(ctx, agentDID)
	if err != nil {
		return false, err
	}

	return metadata.IsActive, nil
}

// GetSAGEResolver returns the underlying SAGE resolver for advanced usage
// This allows interoperability with existing SAGE code during migration
// Note: Returns the AgentCardClient which implements the Resolver interface
func (r *Resolver) GetSAGEResolver() *dideth.AgentCardClient {
	return r.agentCardV4
}

// GetEthereumClient returns the underlying Ethereum client for advanced usage
func (r *Resolver) GetEthereumClient() *dideth.EthereumClient {
	return r.ethClient
}

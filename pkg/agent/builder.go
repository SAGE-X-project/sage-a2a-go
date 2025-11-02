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

package agent

import (
	"context"
	"fmt"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/sage-x-project/sage-a2a-go/pkg/transport"
)

// Agent represents a complete A2A agent with DID authentication
type Agent struct {
	DID       identity.AgentDID
	KeyPair   crypto.KeyPair
	A2AClient *a2aclient.Client
	Card      *a2a.AgentCard
}

// Builder helps construct an A2A agent with DID authentication
type Builder struct {
	did     identity.AgentDID
	keyPair crypto.KeyPair
	card    *a2a.AgentCard
}

// NewBuilder creates a new agent builder
func NewBuilder() *Builder {
	return &Builder{}
}

// WithDID sets the agent's DID
func (b *Builder) WithDID(did identity.AgentDID) *Builder {
	b.did = did
	return b
}

// WithKeyPair sets the agent's key pair
func (b *Builder) WithKeyPair(keyPair crypto.KeyPair) *Builder {
	b.keyPair = keyPair
	return b
}

// WithAgentCard sets the target agent card to connect to
func (b *Builder) WithAgentCard(card *a2a.AgentCard) *Builder {
	b.card = card
	return b
}

// Build creates the agent with all configured components
func (b *Builder) Build(ctx context.Context) (*Agent, error) {
	// Validate required fields
	if b.did == "" {
		return nil, fmt.Errorf("DID is required")
	}
	if b.keyPair == nil {
		return nil, fmt.Errorf("key pair is required")
	}
	if b.card == nil {
		return nil, fmt.Errorf("agent card is required")
	}

	// Convert identity types to SAGE types for transport layer
	sageDID := identity.AgentDID(b.did)
	sageKeyPair := crypto.KeyPair(b.keyPair)

	// Create A2A client with DID-authenticated transport
	a2aClient, err := transport.NewDIDAuthenticatedClient(
		ctx,
		sageDID,
		sageKeyPair,
		b.card,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A client: %w", err)
	}

	return &Agent{
		DID:       b.did,
		KeyPair:   b.keyPair,
		A2AClient: a2aClient,
		Card:      b.card,
	}, nil
}

// NewAgent is a convenience function to create an agent
func NewAgent(ctx context.Context, did identity.AgentDID, keyPair crypto.KeyPair, card *a2a.AgentCard) (*Agent, error) {
	return NewBuilder().
		WithDID(did).
		WithKeyPair(keyPair).
		WithAgentCard(card).
		Build(ctx)
}

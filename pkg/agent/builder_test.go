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
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	assert.NotNil(t, builder)
}

func TestBuilder_WithDID(t *testing.T) {
	builder := NewBuilder()
	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	result := builder.WithDID(testDID)

	// Verify fluent API
	assert.Equal(t, builder, result)
	assert.Equal(t, testDID, builder.did)
}

func TestBuilder_WithKeyPair(t *testing.T) {
	builder := NewBuilder()
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	result := builder.WithKeyPair(keyPair)

	// Verify fluent API
	assert.Equal(t, builder, result)
	assert.Equal(t, keyPair, builder.keyPair)
}

func TestBuilder_WithAgentCard(t *testing.T) {
	builder := NewBuilder()
	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	result := builder.WithAgentCard(card)

	// Verify fluent API
	assert.Equal(t, builder, result)
	assert.Equal(t, card, builder.card)
}

func TestBuilder_FluentAPI(t *testing.T) {
	// Test that builder methods can be chained
	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	builder := NewBuilder().
		WithDID(testDID).
		WithKeyPair(keyPair).
		WithAgentCard(card)

	assert.NotNil(t, builder)
	assert.Equal(t, testDID, builder.did)
	assert.Equal(t, keyPair, builder.keyPair)
	assert.Equal(t, card, builder.card)
}

func TestBuilder_Build_Success(t *testing.T) {
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		Description:        "A test agent",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewBuilder().
		WithDID(testDID).
		WithKeyPair(keyPair).
		WithAgentCard(card).
		Build(ctx)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, testDID, agent.DID)
	assert.Equal(t, keyPair, agent.KeyPair)
	assert.Equal(t, card, agent.Card)
	assert.NotNil(t, agent.A2AClient)
}

func TestBuilder_Build_MissingDID(t *testing.T) {
	ctx := context.Background()

	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewBuilder().
		WithKeyPair(keyPair).
		WithAgentCard(card).
		Build(ctx)

	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "DID is required")
}

func TestBuilder_Build_MissingKeyPair(t *testing.T) {
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewBuilder().
		WithDID(testDID).
		WithAgentCard(card).
		Build(ctx)

	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "key pair is required")
}

func TestBuilder_Build_MissingAgentCard(t *testing.T) {
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agent, err := NewBuilder().
		WithDID(testDID).
		WithKeyPair(keyPair).
		Build(ctx)

	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "agent card is required")
}

func TestBuilder_Build_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	_, err = NewBuilder().
		WithDID(testDID).
		WithKeyPair(keyPair).
		WithAgentCard(card).
		Build(ctx)

	// May succeed or fail depending on timing, but should not panic
	// Just verify no panic occurred
	if err != nil {
		// Error is expected with cancelled context
		assert.NotNil(t, err)
	}
	// Agent may or may not be created depending on timing
}

func TestNewAgent_Success(t *testing.T) {
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		Description:        "A test agent",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewAgent(ctx, testDID, keyPair, card)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, testDID, agent.DID)
	assert.Equal(t, keyPair, agent.KeyPair)
	assert.Equal(t, card, agent.Card)
	assert.NotNil(t, agent.A2AClient)
}

func TestNewAgent_Ed25519KeyPair(t *testing.T) {
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:solana:11111111111111111111111111111111")
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Solana Agent",
		URL:                "https://solana-agent.example.com",
		Description:        "A Solana test agent",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewAgent(ctx, testDID, keyPair, card)

	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, testDID, agent.DID)
	assert.Equal(t, keyPair, agent.KeyPair)
	assert.Equal(t, card, agent.Card)
	assert.NotNil(t, agent.A2AClient)
}

func TestNewAgent_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name    string
		did     identity.AgentDID
		keyPair crypto.KeyPair
		card    *a2a.AgentCard
		errMsg  string
	}{
		{
			name:    "Missing DID",
			did:     "",
			keyPair: mustGenerateSecp256k1(),
			card:    &a2a.AgentCard{Name: "Test", URL: "https://test.com", PreferredTransport: a2a.TransportProtocolJSONRPC},
			errMsg:  "DID is required",
		},
		{
			name:    "Missing KeyPair",
			did:     identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890"),
			keyPair: nil,
			card:    &a2a.AgentCard{Name: "Test", URL: "https://test.com", PreferredTransport: a2a.TransportProtocolJSONRPC},
			errMsg:  "key pair is required",
		},
		{
			name:    "Missing AgentCard",
			did:     identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890"),
			keyPair: mustGenerateSecp256k1(),
			card:    nil,
			errMsg:  "agent card is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent, err := NewAgent(ctx, tc.did, tc.keyPair, tc.card)

			assert.Error(t, err)
			assert.Nil(t, agent)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestAgent_Structure(t *testing.T) {
	// Test Agent struct has expected fields
	ctx := context.Background()

	testDID := identity.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	card := &a2a.AgentCard{
		Name:               "Test Agent",
		URL:                "https://test-agent.example.com",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
	}

	agent, err := NewAgent(ctx, testDID, keyPair, card)
	require.NoError(t, err)

	// Verify all fields are accessible
	_ = agent.DID
	_ = agent.KeyPair
	_ = agent.A2AClient
	_ = agent.Card

	// Verify types
	assert.IsType(t, identity.AgentDID(""), agent.DID)
	assert.NotNil(t, agent.KeyPair)
	assert.NotNil(t, agent.A2AClient)
	assert.IsType(t, &a2a.AgentCard{}, agent.Card)
}

// Helper function for test cases
func mustGenerateSecp256k1() crypto.KeyPair {
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		panic(err)
	}
	return keyPair
}

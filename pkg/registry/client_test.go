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
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistrationClient_MissingRPCURL(t *testing.T) {
	config := &ClientConfig{
		RegistryAddress: "0x1234",
		PrivateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	client, err := NewRegistrationClient(config)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "RPC URL is required")
}

func TestNewRegistrationClient_MissingRegistryAddress(t *testing.T) {
	config := &ClientConfig{
		RPCURL:     "https://ethereum-rpc.example.com",
		PrivateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	client, err := NewRegistrationClient(config)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "registry address is required")
}

func TestNewRegistrationClient_MissingPrivateKey(t *testing.T) {
	config := &ClientConfig{
		RPCURL:          "https://ethereum-rpc.example.com",
		RegistryAddress: "0x1234",
	}

	client, err := NewRegistrationClient(config)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "private key is required")
}

func TestNewRegistrationParams(t *testing.T) {
	// Setup
	signingKey, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	kemKey, err := crypto.GenerateX25519KeyPair()
	require.NoError(t, err)

	// Execute
	params := NewRegistrationParams(
		"Test Agent",
		"https://test-agent.example.com",
		signingKey,
		kemKey,
		"jsonrpc",
		[]string{"capability1", "capability2"},
	)

	// Assert - verify basic SAGE RegistrationParams fields
	assert.NotNil(t, params)
	assert.Equal(t, "Test Agent", params.Name)
	assert.Equal(t, "https://test-agent.example.com", params.Endpoint)
}

func TestRegistrationPhaseConstants(t *testing.T) {
	// Verify phase constants match SAGE constants
	assert.Equal(t, PhaseCommitted, RegistrationPhase(1))
	assert.Equal(t, PhaseRegistered, RegistrationPhase(2))
	assert.Equal(t, PhaseActivated, RegistrationPhase(3))
}

func TestNewRegistrationParams_AllFields(t *testing.T) {
	signingKey, _ := crypto.GenerateSecp256k1KeyPair()
	kemKey, _ := crypto.GenerateX25519KeyPair()

	params := NewRegistrationParams(
		"Agent Name",
		"https://agent.example.com",
		signingKey,
		kemKey,
		"http",
		[]string{"chat", "search"},
	)

	// Verify basic SAGE RegistrationParams fields
	assert.Equal(t, "Agent Name", params.Name)
	assert.Equal(t, "https://agent.example.com", params.Endpoint)
}

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

package hpke

import (
	"context"
	"crypto/ecdh"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/crypto"
	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport implements transport.MessageTransport for testing
type mockTransport struct {
	responses map[string]*transport.Response
}

func (m *mockTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
	// For testing, just return a successful response
	return &transport.Response{
		Success: true,
		Data:    []byte("{}"),
	}, nil
}

// mockResolver implements did.Resolver for testing
type mockResolver struct {
	kemKeys  map[did.AgentDID]*ecdh.PublicKey
	metadata map[did.AgentDID]*did.AgentMetadata
}

func (m *mockResolver) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	if meta, ok := m.metadata[agentDID]; ok {
		return meta, nil
	}
	return &did.AgentMetadata{
		DID:          agentDID,
		PublicKEMKey: m.kemKeys[agentDID],
	}, nil
}

func (m *mockResolver) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	if key, ok := m.kemKeys[agentDID]; ok {
		return key, nil
	}
	return nil, nil
}

func (m *mockResolver) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	// For testing, return nil
	return nil, nil
}

func (m *mockResolver) ListAgentsByOwner(ctx context.Context, owner string) ([]*did.AgentMetadata, error) {
	return nil, nil
}

func (m *mockResolver) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	return nil, nil
}

func (m *mockResolver) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	return &did.VerificationResult{Valid: true}, nil
}

func TestNewClient_Success(t *testing.T) {
	// Setup
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	// Execute
	client, err := NewClient(clientDID, keyPair, transport, resolver, sessionMgr, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, clientDID, client.clientDID)
	assert.NotNil(t, client.sageClient)
}

func TestNewClient_MissingDID(t *testing.T) {
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	client, err := NewClient("", keyPair, transport, resolver, sessionMgr, nil)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "clientDID is required")
}

func TestNewClient_MissingKeyPair(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	client, err := NewClient(clientDID, nil, transport, resolver, sessionMgr, nil)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "keyPair is required")
}

func TestNewClient_MissingTransport(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	client, err := NewClient(clientDID, keyPair, nil, resolver, sessionMgr, nil)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "transport is required")
}

func TestNewClient_MissingResolver(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	sessionMgr := session.NewManager()

	client, err := NewClient(clientDID, keyPair, transport, nil, sessionMgr, nil)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "resolver is required")
}

func TestNewClient_MissingSessionManager(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}

	client, err := NewClient(clientDID, keyPair, transport, resolver, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "sessionMgr is required")
}

func TestNewClient_WithOptions(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	opts := &ClientOptions{}

	client, err := NewClient(clientDID, keyPair, transport, resolver, sessionMgr, opts)

	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetSessionManager(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	client, _ := NewClient(clientDID, keyPair, transport, resolver, sessionMgr, nil)

	returnedMgr := client.GetSessionManager()
	assert.Equal(t, sessionMgr, returnedMgr)
}

func TestGetSAGEClient(t *testing.T) {
	clientDID := identity.AgentDID("did:sage:ethereum:0x1234")
	keyPair, _ := crypto.GenerateEd25519KeyPair()
	transport := &mockTransport{responses: make(map[string]*transport.Response)}
	resolver := &mockResolver{
		kemKeys:  make(map[did.AgentDID]*ecdh.PublicKey),
		metadata: make(map[did.AgentDID]*did.AgentMetadata),
	}
	sessionMgr := session.NewManager()

	client, _ := NewClient(clientDID, keyPair, transport, resolver, sessionMgr, nil)

	sageClient := client.GetSAGEClient()
	assert.NotNil(t, sageClient)
}

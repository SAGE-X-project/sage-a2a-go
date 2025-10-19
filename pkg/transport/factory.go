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

package transport

import (
	"context"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// WithDIDHTTPTransport returns a FactoryOption that enables DID-authenticated
// HTTP/JSON-RPC 2.0 transport for a2a-go clients.
//
// This transport automatically signs all HTTP requests with the agent's DID
// using RFC 9421 HTTP Message Signatures.
//
// Parameters:
//   - agentDID: Your agent's DID for signing requests
//   - keyPair: Your agent's private key for signing
//   - httpClient: Optional HTTP client (nil to use http.DefaultClient)
//
// Example:
//
//	client, err := a2aclient.NewFromCard(
//	    ctx,
//	    agentCard,
//	    sagea2a.WithDIDHTTPTransport(myDID, myKeyPair, nil),
//	)
//	if err != nil {
//	    return err
//	}
//	defer client.Destroy()
//
//	// All requests are automatically signed with DID
//	task, err := client.SendMessage(ctx, message)
func WithDIDHTTPTransport(
	agentDID did.AgentDID,
	keyPair crypto.KeyPair,
	httpClient *http.Client,
) a2aclient.FactoryOption {
	return a2aclient.WithTransport(
		a2a.TransportProtocolJSONRPC,
		a2aclient.TransportFactoryFn(func(ctx context.Context, url string, card *a2a.AgentCard) (a2aclient.Transport, error) {
			return NewDIDHTTPTransport(url, agentDID, keyPair, httpClient), nil
		}),
	)
}

// NewDIDAuthenticatedClientWithInterceptors creates a client with DID HTTP transport
// and custom interceptors.
//
// Example:
//
//	client, err := sagea2a.NewDIDAuthenticatedClientWithInterceptors(
//	    ctx,
//	    myDID,
//	    myKeyPair,
//	    agentCard,
//	    loggingInterceptor,
//	    metricsInterceptor,
//	)
func NewDIDAuthenticatedClientWithInterceptors(
	ctx context.Context,
	agentDID did.AgentDID,
	keyPair crypto.KeyPair,
	card *a2a.AgentCard,
	interceptors ...a2aclient.CallInterceptor,
) (*a2aclient.Client, error) {
	opts := []a2aclient.FactoryOption{
		WithDIDHTTPTransport(agentDID, keyPair, nil),
	}
	if len(interceptors) > 0 {
		opts = append(opts, a2aclient.WithInterceptors(interceptors...))
	}
	return a2aclient.NewFromCard(ctx, card, opts...)
}

// NewDIDAuthenticatedClient is a convenience function that creates an a2a-go client
// with DID-authenticated HTTP transport from an agent card URL.
//
// This is equivalent to:
//
//	a2aclient.NewFromCard(ctx, card, WithDIDHTTPTransport(did, key, nil))
//
// Example:
//
//	client, err := sagea2a.NewDIDAuthenticatedClient(
//	    ctx,
//	    myDID,
//	    myKeyPair,
//	    agentCard,
//	)
//	if err != nil {
//	    return err
//	}
//	defer client.Destroy()
//
//	task, err := client.SendMessage(ctx, message)
func NewDIDAuthenticatedClient(
	ctx context.Context,
	agentDID did.AgentDID,
	keyPair crypto.KeyPair,
	card *a2a.AgentCard,
) (*a2aclient.Client, error) {
	return a2aclient.NewFromCard(
		ctx,
		card,
		WithDIDHTTPTransport(agentDID, keyPair, nil),
	)
}

// NewDIDAuthenticatedClientWithConfig is like NewDIDAuthenticatedClient but
// allows specifying a custom Config.
func NewDIDAuthenticatedClientWithConfig(
	ctx context.Context,
	agentDID did.AgentDID,
	keyPair crypto.KeyPair,
	card *a2a.AgentCard,
	config a2aclient.Config,
) (*a2aclient.Client, error) {
	return a2aclient.NewFromCard(
		ctx,
		card,
		a2aclient.WithConfig(config),
		WithDIDHTTPTransport(agentDID, keyPair, nil),
	)
}

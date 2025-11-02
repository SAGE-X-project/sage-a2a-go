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

// Package agent provides high-level utilities for building A2A agents with DID authentication.
//
// This package provides a unified API that combines SAGE's identity management with
// A2A's agent-to-agent communication. Users can build complete agents using only
// sage-a2a-go packages without directly importing SAGE or A2A.
//
// Example:
//
//	import (
//	    "github.com/sage-x-project/sage-a2a-go/pkg/agent"
//	    "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
//	    "github.com/sage-x-project/sage-a2a-go/pkg/identity"
//	)
//
//	// Generate key pair
//	keyPair, _ := crypto.GenerateSecp256k1KeyPair()
//
//	// Create agent DID
//	did := identity.AgentDID("did:sage:ethereum:0x...")
//
//	// Build agent with DID authentication
//	agent, err := agent.NewAgent(did, keyPair, "https://a2a-server.example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use agent's A2A client
//	task, err := agent.A2AClient.GetTask(ctx, taskID)
package agent

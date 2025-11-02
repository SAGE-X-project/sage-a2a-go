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

// Package identity provides unified DID (Decentralized Identifier) management for A2A agents.
//
// This package wraps SAGE's DID functionality to provide a simple, unified API
// for sage-a2a-go users. Users should import only sage-a2a-go/pkg/identity instead
// of importing SAGE's DID packages directly.
//
// Example:
//
//	import "github.com/sage-x-project/sage-a2a-go/pkg/identity"
//
//	// Create an agent DID
//	agentDID := identity.AgentDID("did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678")
//
//	// Validate the DID
//	if err := identity.ValidateDID(string(agentDID)); err != nil {
//	    log.Fatal(err)
//	}
package identity

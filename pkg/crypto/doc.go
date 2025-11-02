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

// Package crypto provides unified cryptographic key management for A2A agents.
//
// This package wraps SAGE's crypto functionality to provide a simple, unified API
// for sage-a2a-go users. Users should import only sage-a2a-go/pkg/crypto instead
// of importing SAGE's crypto packages directly.
//
// Example:
//
//	import "github.com/sage-x-project/sage-a2a-go/pkg/crypto"
//
//	// Generate a key pair for Ethereum-based agent
//	keyPair, err := crypto.GenerateSecp256k1KeyPair()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use the key pair for signing, verification, etc.
//	signature, err := keyPair.Sign(message)
package crypto

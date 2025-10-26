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

// Package sagea2a provides version information for sage-a2a-go and its dependencies.
package sagea2a

const (
	// Version is the current version of sage-a2a-go
	Version = "2.0.0-alpha"

	// A2AProtocolVersion is the A2A Protocol specification version this library supports
	// See: https://github.com/a2aproject/A2A
	A2AProtocolVersion = "0.4.0"

	// MinA2AProtocolVersion is the minimum A2A Protocol version compatible with this library
	MinA2AProtocolVersion = "0.2.6"

	// SAGEVersion is the SAGE core version required
	SAGEVersion = "1.3.1"
)

// VersionInfo contains detailed version information
type VersionInfo struct {
	SageA2AVersion        string
	A2AProtocolVersion    string
	MinA2AProtocolVersion string
	SAGEVersion           string
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		SageA2AVersion:        Version,
		A2AProtocolVersion:    A2AProtocolVersion,
		MinA2AProtocolVersion: MinA2AProtocolVersion,
		SAGEVersion:           SAGEVersion,
	}
}

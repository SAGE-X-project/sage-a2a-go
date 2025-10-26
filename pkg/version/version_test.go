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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionConstants(t *testing.T) {
	// Verify version constants are not empty
	assert.NotEmpty(t, Version, "Version should not be empty")
	assert.NotEmpty(t, A2AProtocolVersion, "A2AProtocolVersion should not be empty")
	assert.NotEmpty(t, MinA2AProtocolVersion, "MinA2AProtocolVersion should not be empty")
	assert.NotEmpty(t, SAGEVersion, "SAGEVersion should not be empty")
	assert.NotEmpty(t, A2AGoForkVersion, "A2AGoForkVersion should not be empty")

	// Verify expected values
	assert.Equal(t, "1.0.0-dev", Version)
	assert.Equal(t, "0.4.0", A2AProtocolVersion)
	assert.Equal(t, "0.2.6", MinA2AProtocolVersion)
	assert.Equal(t, "1.3.1", SAGEVersion)
	assert.Equal(t, "v0.0.0-20251026124015-70634d9eddae", A2AGoForkVersion)
}

func TestGet(t *testing.T) {
	info := Get()

	// Verify all fields are populated
	assert.Equal(t, Version, info.SageA2AVersion)
	assert.Equal(t, A2AProtocolVersion, info.A2AProtocolVersion)
	assert.Equal(t, MinA2AProtocolVersion, info.MinA2AProtocolVersion)
	assert.Equal(t, SAGEVersion, info.SAGEVersion)
	assert.Equal(t, A2AGoForkVersion, info.A2AGoForkVersion)

	// Verify Info struct
	assert.Equal(t, "1.0.0-dev", info.SageA2AVersion)
	assert.Equal(t, "0.4.0", info.A2AProtocolVersion)
	assert.Equal(t, "0.2.6", info.MinA2AProtocolVersion)
	assert.Equal(t, "1.3.1", info.SAGEVersion)
	assert.Equal(t, "v0.0.0-20251026124015-70634d9eddae", info.A2AGoForkVersion)
}

func TestInfoStruct(t *testing.T) {
	// Test that Info struct can be created manually
	info := Info{
		SageA2AVersion:        "test-version",
		A2AProtocolVersion:    "0.4.0",
		MinA2AProtocolVersion: "0.2.6",
		SAGEVersion:           "1.3.1",
		A2AGoForkVersion:      "test-fork",
	}

	assert.Equal(t, "test-version", info.SageA2AVersion)
	assert.Equal(t, "0.4.0", info.A2AProtocolVersion)
	assert.Equal(t, "0.2.6", info.MinA2AProtocolVersion)
	assert.Equal(t, "1.3.1", info.SAGEVersion)
	assert.Equal(t, "test-fork", info.A2AGoForkVersion)
}

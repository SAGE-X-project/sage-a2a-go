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

package verifier

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"net/http"
	"testing"

	"github.com/sage-x-project/sage-a2a-go/pkg/signer"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRFC9421Verifier(t *testing.T) {
	verifier := NewRFC9421Verifier()

	assert.NotNil(t, verifier)
	assert.NotNil(t, verifier.verifier)
	assert.NotNil(t, verifier.options)
}

func TestRFC9421Verifier_VerifyHTTPRequest_Success_ECDSA(t *testing.T) {
	ctx := context.Background()

	// Generate ECDSA key pair
	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Create and sign request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	testDID := did.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair)
	require.NoError(t, err)

	// Verify signature
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair.PublicKey())
	assert.NoError(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_Success_Ed25519(t *testing.T) {
	ctx := context.Background()

	// Generate Ed25519 key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Create and sign request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	testDID := did.AgentDID("did:sage:solana:11111111111111111111111111111111")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair)
	require.NoError(t, err)

	// Verify signature
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair.PublicKey())
	assert.NoError(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_InvalidSignature(t *testing.T) {
	ctx := context.Background()

	// Generate two different key pairs
	keyPair1, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	keyPair2, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Create and sign request with keyPair1
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	testDID := did.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair1)
	require.NoError(t, err)

	// Try to verify with different public key (keyPair2)
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair2.PublicKey())
	assert.Error(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_MissingSignature(t *testing.T) {
	ctx := context.Background()

	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Create request without signing
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Try to verify unsigned request
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair.PublicKey())
	assert.Error(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_TamperedBody(t *testing.T) {
	ctx := context.Background()

	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Create and sign request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	testDID := did.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair)
	require.NoError(t, err)

	// Tamper with body after signing
	req.Body = http.NoBody
	req.ContentLength = 0

	// Verification should fail
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair.PublicKey())
	assert.Error(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_WithCryptoPublicKey(t *testing.T) {
	ctx := context.Background()

	// Generate Ed25519 key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Create and sign request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://example.com/test", bytes.NewReader([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	testDID := did.AgentDID("did:sage:solana:11111111111111111111111111111111")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair)
	require.NoError(t, err)

	// Verify using ed25519.PublicKey directly (not crypto.PublicKey interface)
	verifier := NewRFC9421Verifier()
	ed25519Key := keyPair.PublicKey().(ed25519.PublicKey)
	err = verifier.VerifyHTTPRequest(req, ed25519Key)
	assert.NoError(t, err)
}

func TestRFC9421Verifier_VerifyHTTPRequest_EmptyBody(t *testing.T) {
	ctx := context.Background()

	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Create and sign request with empty body
	req, err := http.NewRequestWithContext(ctx, "GET", "https://example.com/test", nil)
	require.NoError(t, err)

	testDID := did.AgentDID("did:sage:ethereum:0x1234567890123456789012345678901234567890")

	signer := signer.NewDefaultA2ASigner()
	err = signer.SignRequest(ctx, req, testDID, keyPair)
	require.NoError(t, err)

	// Verify signature
	verifier := NewRFC9421Verifier()
	err = verifier.VerifyHTTPRequest(req, keyPair.PublicKey())
	assert.NoError(t, err)
}

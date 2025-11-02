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

package session

import (
	"context"
	"testing"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_WithDefaultOptions(t *testing.T) {
	mgr := NewManager(nil)

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.opts)
	assert.Equal(t, 24*time.Hour, mgr.opts.SessionTTL)
	assert.Equal(t, 1*time.Hour, mgr.opts.CleanupInterval)
	assert.Equal(t, 1000, mgr.opts.MaxSessions)
}

func TestNewManager_WithCustomOptions(t *testing.T) {
	opts := &Options{
		SessionTTL:      1 * time.Hour,
		CleanupInterval: 10 * time.Minute,
		MaxSessions:     100,
	}

	mgr := NewManager(opts)

	assert.NotNil(t, mgr)
	assert.Equal(t, opts, mgr.opts)
}

func TestCreate_Success(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, err := mgr.Create(ctx, remoteDID, sharedSecret)

	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, remoteDID, session.RemoteDID)
	assert.NotNil(t, session.SymmetricKey)
	assert.Len(t, session.SymmetricKey, 32) // AES-256 key
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.ExpiresAt.IsZero())
}

func TestCreate_MissingRemoteDID(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	sharedSecret := []byte("shared-secret")

	session, err := mgr.Create(ctx, "", sharedSecret)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "remote DID is required")
}

func TestCreate_MissingSharedSecret(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")

	session, err := mgr.Create(ctx, remoteDID, nil)

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "shared secret is required")
}

func TestGet_Success(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	created, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	retrieved, err := mgr.Get(ctx, created.ID)

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, created.ID, retrieved.ID)
}

func TestGet_NotFound(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	session, err := mgr.Get(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session not found")
}

func TestDelete_Success(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	err := mgr.Delete(ctx, session.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = mgr.Get(ctx, session.ID)
	assert.Error(t, err)
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")
	plaintext := []byte("Hello, World!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	// Encrypt
	ciphertext, err := mgr.Encrypt(ctx, session.ID, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt
	decrypted, err := mgr.Decrypt(ctx, session.ID, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestRefresh_Success(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)
	originalExpiry := session.ExpiresAt

	time.Sleep(10 * time.Millisecond)

	err := mgr.Refresh(ctx, session.ID)
	require.NoError(t, err)

	refreshed, _ := mgr.Get(ctx, session.ID)
	assert.True(t, refreshed.ExpiresAt.After(originalExpiry))
}

func TestListByRemoteDID(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID1 := identity.AgentDID("did:sage:ethereum:0x1111")
	remoteDID2 := identity.AgentDID("did:sage:ethereum:0x2222")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	// Create sessions for two different DIDs
	mgr.Create(ctx, remoteDID1, sharedSecret)
	mgr.Create(ctx, remoteDID1, sharedSecret)
	mgr.Create(ctx, remoteDID2, sharedSecret)

	// List sessions for remoteDID1
	sessions := mgr.ListByRemoteDID(ctx, remoteDID1)

	assert.Len(t, sessions, 2)
	for _, s := range sessions {
		assert.Equal(t, remoteDID1, s.RemoteDID)
	}
}

func TestDeleteByRemoteDID(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID1 := identity.AgentDID("did:sage:ethereum:0x1111")
	remoteDID2 := identity.AgentDID("did:sage:ethereum:0x2222")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	// Create sessions
	mgr.Create(ctx, remoteDID1, sharedSecret)
	mgr.Create(ctx, remoteDID1, sharedSecret)
	mgr.Create(ctx, remoteDID2, sharedSecret)

	// Delete all sessions for remoteDID1
	count := mgr.DeleteByRemoteDID(ctx, remoteDID1)

	assert.Equal(t, 2, count)

	// Verify only remoteDID2 session remains
	sessions := mgr.List(ctx)
	assert.Len(t, sessions, 1)
	assert.Equal(t, remoteDID2, sessions[0].RemoteDID)
}

func TestCount(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	// Initially zero
	assert.Equal(t, 0, mgr.Count())

	// Create sessions
	mgr.Create(ctx, remoteDID, sharedSecret)
	mgr.Create(ctx, remoteDID, sharedSecret)

	assert.Equal(t, 2, mgr.Count())
}

func TestSetGetMetadata(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	// Set metadata
	err := mgr.SetMetadata(ctx, session.ID, "context", "conversation-123")
	require.NoError(t, err)

	// Get metadata
	value, err := mgr.GetMetadata(ctx, session.ID, "context")
	require.NoError(t, err)
	assert.Equal(t, "conversation-123", value)
}

func TestGetMetadata_NotFound(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	value, err := mgr.GetMetadata(ctx, session.ID, "nonexistent")

	assert.Error(t, err)
	assert.Empty(t, value)
	assert.Contains(t, err.Error(), "metadata key not found")
}

func TestExists(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	assert.True(t, mgr.Exists(ctx, session.ID))
	assert.False(t, mgr.Exists(ctx, "nonexistent"))
}

func TestGetExpiresAt(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	expiresAt, err := mgr.GetExpiresAt(ctx, session.ID)

	require.NoError(t, err)
	assert.False(t, expiresAt.IsZero())
	assert.True(t, expiresAt.After(time.Now()))
}

func TestGetRemoteDID(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	session, _ := mgr.Create(ctx, remoteDID, sharedSecret)

	retrievedDID, err := mgr.GetRemoteDID(ctx, session.ID)

	require.NoError(t, err)
	assert.Equal(t, remoteDID, retrievedDID)
}

func TestList(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	// Create multiple sessions
	mgr.Create(ctx, remoteDID, sharedSecret)
	mgr.Create(ctx, remoteDID, sharedSecret)
	mgr.Create(ctx, remoteDID, sharedSecret)

	sessions := mgr.List(ctx)

	assert.Len(t, sessions, 3)
}

func TestMaxSessions(t *testing.T) {
	opts := &Options{
		SessionTTL:      1 * time.Hour,
		CleanupInterval: 1 * time.Hour,
		MaxSessions:     2,
	}
	mgr := NewManager(opts)
	ctx := context.Background()
	remoteDID := identity.AgentDID("did:sage:ethereum:0x1234")
	sharedSecret := []byte("shared-secret-32-bytes-long!!!!!")

	// Create max sessions
	_, err1 := mgr.Create(ctx, remoteDID, sharedSecret)
	_, err2 := mgr.Create(ctx, remoteDID, sharedSecret)

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Attempt to exceed max
	_, err3 := mgr.Create(ctx, remoteDID, sharedSecret)

	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "maximum sessions limit reached")
}

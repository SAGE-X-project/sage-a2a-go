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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sage-x-project/sage-a2a-go/pkg/identity"
)

// SessionID uniquely identifies an encryption session
type SessionID string

// Session represents an encrypted communication session
type Session struct {
	ID             SessionID          // Unique session identifier (KID)
	RemoteDID      identity.AgentDID  // Remote agent's DID
	SharedSecret   []byte             // Shared secret from HPKE or key exchange
	SymmetricKey   []byte             // Derived AES-256 key
	CreatedAt      time.Time          // Session creation timestamp
	LastAccessedAt time.Time          // Last time session was used
	ExpiresAt      time.Time          // Session expiration time
	Metadata       map[string]string  // Additional session metadata
}

// Manager manages encryption sessions
type Manager struct {
	sessions sync.Map // map[SessionID]*Session
	opts     *Options
	mu       sync.RWMutex
}

// Options configures the session manager
type Options struct {
	// SessionTTL is the time-to-live for sessions
	SessionTTL time.Duration

	// CleanupInterval is how often to clean up expired sessions
	CleanupInterval time.Duration

	// MaxSessions is the maximum number of concurrent sessions
	MaxSessions int
}

// DefaultOptions returns default session manager options
func DefaultOptions() *Options {
	return &Options{
		SessionTTL:      24 * time.Hour,  // 24 hours default TTL
		CleanupInterval: 1 * time.Hour,   // Cleanup every hour
		MaxSessions:     1000,            // Max 1000 concurrent sessions
	}
}

// NewManager creates a new session manager
func NewManager(opts *Options) *Manager {
	if opts == nil {
		opts = DefaultOptions()
	}

	m := &Manager{
		opts: opts,
	}

	// Start background cleanup goroutine
	go m.cleanupExpiredSessions()

	return m
}

// Create creates a new session with the given remote DID and shared secret
func (m *Manager) Create(ctx context.Context, remoteDID identity.AgentDID, sharedSecret []byte) (*Session, error) {
	if remoteDID == "" {
		return nil, fmt.Errorf("remote DID is required")
	}
	if len(sharedSecret) == 0 {
		return nil, fmt.Errorf("shared secret is required")
	}

	// Check if we've exceeded max sessions
	sessionCount := 0
	m.sessions.Range(func(_, _ interface{}) bool {
		sessionCount++
		return true
	})

	if sessionCount >= m.opts.MaxSessions {
		return nil, fmt.Errorf("maximum sessions limit reached (%d)", m.opts.MaxSessions)
	}

	// Generate session ID (KID - Key Identifier)
	sessionID, err := m.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Derive AES-256 symmetric key from shared secret using HKDF-SHA256
	symmetricKey := m.deriveSymmetricKey(sharedSecret)

	now := time.Now()
	session := &Session{
		ID:             sessionID,
		RemoteDID:      remoteDID,
		SharedSecret:   sharedSecret,
		SymmetricKey:   symmetricKey,
		CreatedAt:      now,
		LastAccessedAt: now,
		ExpiresAt:      now.Add(m.opts.SessionTTL),
		Metadata:       make(map[string]string),
	}

	m.sessions.Store(sessionID, session)

	return session, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(ctx context.Context, sessionID SessionID) (*Session, error) {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	session := value.(*Session)

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		m.sessions.Delete(sessionID)
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	// Update last accessed time
	session.LastAccessedAt = time.Now()

	return session, nil
}

// Delete removes a session
func (m *Manager) Delete(ctx context.Context, sessionID SessionID) error {
	m.sessions.Delete(sessionID)
	return nil
}

// Encrypt encrypts plaintext using the session's symmetric key
func (m *Manager) Encrypt(ctx context.Context, sessionID SessionID, plaintext []byte) ([]byte, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(session.SymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext using the session's symmetric key
func (m *Manager) Decrypt(ctx context.Context, sessionID SessionID, ciphertext []byte) ([]byte, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(session.SymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// List returns all active sessions (for debugging/monitoring)
func (m *Manager) List(ctx context.Context) []*Session {
	var sessions []*Session

	m.sessions.Range(func(_, value interface{}) bool {
		session := value.(*Session)
		if time.Now().Before(session.ExpiresAt) {
			sessions = append(sessions, session)
		}
		return true
	})

	return sessions
}

// generateSessionID generates a unique session ID (KID)
func (m *Manager) generateSessionID() (SessionID, error) {
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Base64 encode for URL-safe session ID
	sessionID := base64.RawURLEncoding.EncodeToString(randomBytes)

	return SessionID(sessionID), nil
}

// deriveSymmetricKey derives a 256-bit AES key from shared secret using SHA-256
func (m *Manager) deriveSymmetricKey(sharedSecret []byte) []byte {
	hash := sha256.Sum256(sharedSecret)
	return hash[:]
}

// cleanupExpiredSessions periodically removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(m.opts.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.sessions.Range(func(key, value interface{}) bool {
			session := value.(*Session)
			if now.After(session.ExpiresAt) {
				m.sessions.Delete(key)
			}
			return true
		})
	}
}

// Refresh extends the session's expiration time
func (m *Manager) Refresh(ctx context.Context, sessionID SessionID) error {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = time.Now().Add(m.opts.SessionTTL)
	session.LastAccessedAt = time.Now()

	return nil
}

// ListByRemoteDID returns all active sessions for a specific remote DID
func (m *Manager) ListByRemoteDID(ctx context.Context, remoteDID identity.AgentDID) []*Session {
	var sessions []*Session

	m.sessions.Range(func(_, value interface{}) bool {
		session := value.(*Session)
		if session.RemoteDID == remoteDID && time.Now().Before(session.ExpiresAt) {
			sessions = append(sessions, session)
		}
		return true
	})

	return sessions
}

// DeleteByRemoteDID removes all sessions for a specific remote DID
func (m *Manager) DeleteByRemoteDID(ctx context.Context, remoteDID identity.AgentDID) int {
	deletedCount := 0

	m.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		if session.RemoteDID == remoteDID {
			m.sessions.Delete(key)
			deletedCount++
		}
		return true
	})

	return deletedCount
}

// Count returns the total number of active sessions
func (m *Manager) Count() int {
	count := 0
	now := time.Now()

	m.sessions.Range(func(_, value interface{}) bool {
		session := value.(*Session)
		if now.Before(session.ExpiresAt) {
			count++
		}
		return true
	})

	return count
}

// SetMetadata sets metadata for a session
func (m *Manager) SetMetadata(ctx context.Context, sessionID SessionID, key, value string) error {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Metadata[key] = value
	return nil
}

// GetMetadata retrieves metadata from a session
func (m *Manager) GetMetadata(ctx context.Context, sessionID SessionID, key string) (string, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	value, ok := session.Metadata[key]
	if !ok {
		return "", fmt.Errorf("metadata key not found: %s", key)
	}

	return value, nil
}

// Exists checks if a session exists and is valid
func (m *Manager) Exists(ctx context.Context, sessionID SessionID) bool {
	_, err := m.Get(ctx, sessionID)
	return err == nil
}

// GetExpiresAt returns the expiration time for a session
func (m *Manager) GetExpiresAt(ctx context.Context, sessionID SessionID) (time.Time, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return time.Time{}, err
	}

	return session.ExpiresAt, nil
}

// GetRemoteDID returns the remote DID for a session
func (m *Manager) GetRemoteDID(ctx context.Context, sessionID SessionID) (identity.AgentDID, error) {
	session, err := m.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}

	return session.RemoteDID, nil
}

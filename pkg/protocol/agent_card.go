package protocol

import (
	"context"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// AgentCard represents an A2A Agent Card
// An Agent Card is a JSON metadata document that describes an agent's
// identity, capabilities, and service endpoints
type AgentCard struct {
	// DID is the agent's Decentralized Identifier
	DID string `json:"did"`

	// Name is the human-readable name of the agent
	Name string `json:"name"`

	// Description provides details about the agent's purpose and functionality
	Description string `json:"description,omitempty"`

	// Endpoint is the base URL where the agent's A2A service is accessible
	Endpoint string `json:"endpoint"`

	// Capabilities lists the operations this agent can perform
	Capabilities []string `json:"capabilities,omitempty"`

	// PublicKeys contains the agent's public keys for verification
	PublicKeys []PublicKeyInfo `json:"publicKeys,omitempty"`

	// CreatedAt is the timestamp when the card was created (Unix timestamp)
	CreatedAt int64 `json:"createdAt"`

	// ExpiresAt is the optional expiration timestamp (Unix timestamp)
	ExpiresAt int64 `json:"expiresAt,omitempty"`

	// Version of the Agent Card specification
	Version string `json:"version,omitempty"`

	// Metadata contains additional custom fields
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PublicKeyInfo represents a public key in the Agent Card
type PublicKeyInfo struct {
	// ID is a unique identifier for this key
	ID string `json:"id"`

	// Type specifies the key algorithm
	// Examples: "EcdsaSecp256k1VerificationKey2019", "Ed25519VerificationKey2020"
	Type string `json:"type"`

	// KeyData is the base64-encoded public key
	KeyData string `json:"keyData"`

	// Purpose describes what this key is used for
	// Examples: "authentication", "signing", "encryption"
	Purpose []string `json:"purpose,omitempty"`
}

// SignedAgentCard represents an Agent Card with a cryptographic signature
type SignedAgentCard struct {
	// Card is the Agent Card data
	Card *AgentCard `json:"card"`

	// Signature is the JWS compact serialization of the card signature
	// Format: base64url(header).base64url(payload).base64url(signature)
	Signature string `json:"signature"`

	// SignedAt is when the signature was created (Unix timestamp)
	SignedAt int64 `json:"signedAt"`
}

// AgentCardSigner signs and verifies Agent Cards
type AgentCardSigner interface {
	// SignAgentCard signs an Agent Card with the agent's private key
	// Returns a SignedAgentCard with JWS signature
	SignAgentCard(ctx context.Context, card *AgentCard, keyPair crypto.KeyPair) (*SignedAgentCard, error)

	// VerifyAgentCard verifies a signed Agent Card's signature
	// Resolves the public key from the DID in the card
	VerifyAgentCard(ctx context.Context, signedCard *SignedAgentCard) error

	// VerifyAgentCardWithKey verifies a signed Agent Card with a provided public key
	// This is useful when you already have the public key and don't need DID resolution
	VerifyAgentCardWithKey(ctx context.Context, signedCard *SignedAgentCard, publicKey interface{}) error
}

// AgentCardBuilder helps construct Agent Cards with a fluent API
type AgentCardBuilder struct {
	card *AgentCard
}

// NewAgentCardBuilder creates a new AgentCardBuilder
func NewAgentCardBuilder(agentDID did.AgentDID, name, endpoint string) *AgentCardBuilder {
	return &AgentCardBuilder{
		card: &AgentCard{
			DID:       string(agentDID),
			Name:      name,
			Endpoint:  endpoint,
			CreatedAt: time.Now().Unix(),
			Version:   "1.0",
		},
	}
}

// WithDescription adds a description to the Agent Card
func (b *AgentCardBuilder) WithDescription(description string) *AgentCardBuilder {
	b.card.Description = description
	return b
}

// WithCapabilities adds capabilities to the Agent Card
func (b *AgentCardBuilder) WithCapabilities(capabilities ...string) *AgentCardBuilder {
	b.card.Capabilities = append(b.card.Capabilities, capabilities...)
	return b
}

// WithPublicKey adds a public key to the Agent Card
func (b *AgentCardBuilder) WithPublicKey(keyInfo PublicKeyInfo) *AgentCardBuilder {
	b.card.PublicKeys = append(b.card.PublicKeys, keyInfo)
	return b
}

// WithExpiresAt sets the expiration time for the Agent Card
func (b *AgentCardBuilder) WithExpiresAt(expiresAt time.Time) *AgentCardBuilder {
	b.card.ExpiresAt = expiresAt.Unix()
	return b
}

// WithMetadata adds custom metadata to the Agent Card
func (b *AgentCardBuilder) WithMetadata(key string, value interface{}) *AgentCardBuilder {
	if b.card.Metadata == nil {
		b.card.Metadata = make(map[string]interface{})
	}
	b.card.Metadata[key] = value
	return b
}

// Build returns the constructed Agent Card
func (b *AgentCardBuilder) Build() *AgentCard {
	return b.card
}

// IsExpired checks if the Agent Card has expired
func (c *AgentCard) IsExpired() bool {
	if c.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > c.ExpiresAt
}

// HasCapability checks if the agent has a specific capability
func (c *AgentCard) HasCapability(capability string) bool {
	for _, cap := range c.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// Validate performs basic validation on the Agent Card
func (c *AgentCard) Validate() error {
	if c.DID == "" {
		return ErrInvalidAgentCard{"DID is required"}
	}
	if c.Name == "" {
		return ErrInvalidAgentCard{"name is required"}
	}
	if c.Endpoint == "" {
		return ErrInvalidAgentCard{"endpoint is required"}
	}
	if c.CreatedAt == 0 {
		return ErrInvalidAgentCard{"createdAt is required"}
	}
	return nil
}

// ErrInvalidAgentCard is returned when an Agent Card is invalid
type ErrInvalidAgentCard struct {
	Message string
}

func (e ErrInvalidAgentCard) Error() string {
	return "invalid agent card: " + e.Message
}

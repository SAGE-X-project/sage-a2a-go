package verifier

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

type DIDResolver interface {
	GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error)
}

type DefaultKeySelector struct {
	resolver DIDResolver
}

func NewDefaultKeySelector(resolver DIDResolver) *DefaultKeySelector {
	return &DefaultKeySelector{resolver: resolver}
}

// - "ethereum"/"eth": ECDSA(secp256k1)
// - "solana"/"sol": Ed25519
// - "hpke"/"kem"/"x25519": X25519(32바이트) — HPKE용
// - 그 외: (1) Ed25519, (2) ECDSA, (3) 첫 검증된 키 순
func (s *DefaultKeySelector) SelectKey(ctx context.Context, agentDID did.AgentDID, protocol string) (crypto.PublicKey, did.KeyType, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, fmt.Errorf("context error: %w", err)
	}

	meta, err := s.resolver.GetAgentByDID(ctx, string(agentDID))

	if err != nil {
		return nil, 0, fmt.Errorf("resolve agent: %w", err)
	}
	if meta == nil || !meta.IsActive {
		return nil, 0, fmt.Errorf("agent inactive or not found: %s", agentDID)
	}

	// 빠른 HPKE/KEM 처리: 우선 KME 전용 필드, 없으면 키 배열에서 X25519 검색
	proto := strings.ToLower(strings.TrimSpace(protocol))
	switch proto {
	case "hpke", "kem", "x25519":
		if len(meta.PublicKEMKey) == 32 {
			return crypto.PublicKey(meta.PublicKEMKey), did.KeyTypeX25519, nil
		}
		if pk, ok := firstByType(meta.Keys, did.KeyTypeX25519); ok {
			// X25519는 32바이트 로우 형태를 그대로 반환
			return crypto.PublicKey(pk.KeyData), did.KeyTypeX25519, nil
		}
		return nil, 0, errors.New("no X25519 (HPKE) key registered")

	case "ethereum", "eth", "eip155":
		if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		}
		// 폴백: Ed25519 → 기타
		if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		}
		return firstAnyVerified(meta.Keys)

	case "solana", "sol":
		if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		}
		// 폴백: ECDSA → 기타
		if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		}
		return firstAnyVerified(meta.Keys)
	}

	// 기본 정책: Ed25519 > ECDSA > 첫 검증된 키
	if k, ok := firstByType(meta.Keys, did.KeyTypeEd25519); ok {
		return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
	}
	if k, ok := firstByType(meta.Keys, did.KeyTypeECDSA); ok {
		return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
	}
	return firstAnyVerified(meta.Keys)
}

func firstByType(keys []did.AgentKey, t did.KeyType) (did.AgentKey, bool) {
	for _, k := range keys {
		if k.Verified && k.Type == t {
			return k, true
		}
	}
	return did.AgentKey{}, false
}

func firstAnyVerified(keys []did.AgentKey) (crypto.PublicKey, did.KeyType, error) {
	for _, k := range keys {
		if !k.Verified {
			continue
		}
		// X25519는 로우 바이트 반환, 서명키는 Unmarshal
		switch k.Type {
		case did.KeyTypeX25519:
			if len(k.KeyData) == 32 {
				return crypto.PublicKey(k.KeyData), did.KeyTypeX25519, nil
			}
		case did.KeyTypeECDSA:
			return unmarshalByKeyType(k.KeyData, did.KeyTypeECDSA)
		case did.KeyTypeEd25519:
			return unmarshalByKeyType(k.KeyData, did.KeyTypeEd25519)
		default:
			// 모르는 타입은 스킵
		}
	}
	return nil, 0, errors.New("no verified keys available")
}

func unmarshalByKeyType(raw []byte, kt did.KeyType) (crypto.PublicKey, did.KeyType, error) {
	switch kt {
	case did.KeyTypeECDSA:
		pk, err := did.UnmarshalPublicKey(raw, "secp256k1")
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshal secp256k1: %w", err)
		}
		return pk.(crypto.PublicKey), did.KeyTypeECDSA, nil
	case did.KeyTypeEd25519:
		pk, err := did.UnmarshalPublicKey(raw, "ed25519")
		if err != nil {
			return nil, 0, fmt.Errorf("unmarshal ed25519: %w", err)
		}
		return pk.(crypto.PublicKey), did.KeyTypeEd25519, nil
	case did.KeyTypeX25519:
		// X25519는 32바이트 raw 반환 (HPKE 용도)
		if len(raw) != 32 {
			return nil, 0, fmt.Errorf("x25519: want 32 bytes, got %d", len(raw))
		}
		return crypto.PublicKey(raw), did.KeyTypeX25519, nil
	default:
		return nil, 0, fmt.Errorf("unknown key type: %d", kt)
	}
}

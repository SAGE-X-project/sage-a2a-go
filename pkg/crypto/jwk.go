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

package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// JWK represents a JSON Web Key (RFC 7517)
type JWK struct {
	// Key Type (kty)
	Kty string `json:"kty"` // "EC", "OKP", etc.

	// Key Use (use) - Optional
	Use string `json:"use,omitempty"` // "sig", "enc"

	// Key ID (kid) - Optional
	Kid string `json:"kid,omitempty"`

	// Algorithm (alg) - Optional
	Alg string `json:"alg,omitempty"` // "ES256K", "EdDSA", etc.

	// Curve (crv) - For EC and OKP keys
	Crv string `json:"crv,omitempty"` // "secp256k1", "Ed25519", "P-256", etc.

	// EC Public Key Parameters
	X string `json:"x,omitempty"` // Base64url-encoded X coordinate
	Y string `json:"y,omitempty"` // Base64url-encoded Y coordinate

	// EC Private Key Parameter
	D string `json:"d,omitempty"` // Base64url-encoded private key

	// OKP (Octet Key Pair) Public Key Parameter - for Ed25519
	// X is used for OKP public key
}

// ExportPublicKeyToJWK exports a public key to JWK format
func ExportPublicKeyToJWK(pubKey interface{}, keyID string) (*JWK, error) {
	switch key := pubKey.(type) {
	case *ecdsa.PublicKey:
		return exportECDSAPublicKeyToJWK(key, keyID)
	case ed25519.PublicKey:
		return exportEd25519PublicKeyToJWK(key, keyID)
	default:
		return nil, fmt.Errorf("unsupported public key type: %T", pubKey)
	}
}

// ExportPrivateKeyToJWK exports a private key to JWK format
func ExportPrivateKeyToJWK(privKey interface{}, keyID string) (*JWK, error) {
	switch key := privKey.(type) {
	case *ecdsa.PrivateKey:
		return exportECDSAPrivateKeyToJWK(key, keyID)
	case ed25519.PrivateKey:
		return exportEd25519PrivateKeyToJWK(key, keyID)
	default:
		return nil, fmt.Errorf("unsupported private key type: %T", privKey)
	}
}

// ImportPublicKeyFromJWK imports a public key from JWK format
func ImportPublicKeyFromJWK(jwk *JWK) (interface{}, error) {
	switch jwk.Kty {
	case "EC":
		return importECDSAPublicKeyFromJWK(jwk)
	case "OKP":
		return importOKPPublicKeyFromJWK(jwk)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

// ImportPrivateKeyFromJWK imports a private key from JWK format
func ImportPrivateKeyFromJWK(jwk *JWK) (interface{}, error) {
	switch jwk.Kty {
	case "EC":
		return importECDSAPrivateKeyFromJWK(jwk)
	case "OKP":
		return importOKPPrivateKeyFromJWK(jwk)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

// MarshalJWK marshals a JWK to JSON
func MarshalJWK(jwk *JWK) ([]byte, error) {
	return json.Marshal(jwk)
}

// UnmarshalJWK unmarshals a JWK from JSON
func UnmarshalJWK(data []byte) (*JWK, error) {
	var jwk JWK
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWK: %w", err)
	}
	return &jwk, nil
}

// exportECDSAPublicKeyToJWK exports an ECDSA public key to JWK format
func exportECDSAPublicKeyToJWK(pubKey *ecdsa.PublicKey, keyID string) (*JWK, error) {
	curve := pubKey.Curve
	var crv, alg string

	switch curve {
	case elliptic.P256():
		crv = "P-256"
		alg = "ES256"
	default:
		// Assume secp256k1 (Ethereum)
		crv = "secp256k1"
		alg = "ES256K"
	}

	// Encode coordinates as base64url
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Pad to curve byte length
	curveByteLen := (curve.Params().BitSize + 7) / 8
	xBytes = padLeft(xBytes, curveByteLen)
	yBytes = padLeft(yBytes, curveByteLen)

	return &JWK{
		Kty: "EC",
		Use: "sig",
		Kid: keyID,
		Alg: alg,
		Crv: crv,
		X:   base64.RawURLEncoding.EncodeToString(xBytes),
		Y:   base64.RawURLEncoding.EncodeToString(yBytes),
	}, nil
}

// exportECDSAPrivateKeyToJWK exports an ECDSA private key to JWK format
func exportECDSAPrivateKeyToJWK(privKey *ecdsa.PrivateKey, keyID string) (*JWK, error) {
	// First export public key
	jwk, err := exportECDSAPublicKeyToJWK(&privKey.PublicKey, keyID)
	if err != nil {
		return nil, err
	}

	// Add private key parameter
	dBytes := privKey.D.Bytes()
	curveByteLen := (privKey.Curve.Params().BitSize + 7) / 8
	dBytes = padLeft(dBytes, curveByteLen)
	jwk.D = base64.RawURLEncoding.EncodeToString(dBytes)

	return jwk, nil
}

// exportEd25519PublicKeyToJWK exports an Ed25519 public key to JWK format
func exportEd25519PublicKeyToJWK(pubKey ed25519.PublicKey, keyID string) (*JWK, error) {
	return &JWK{
		Kty: "OKP",
		Use: "sig",
		Kid: keyID,
		Alg: "EdDSA",
		Crv: "Ed25519",
		X:   base64.RawURLEncoding.EncodeToString(pubKey),
	}, nil
}

// exportEd25519PrivateKeyToJWK exports an Ed25519 private key to JWK format
func exportEd25519PrivateKeyToJWK(privKey ed25519.PrivateKey, keyID string) (*JWK, error) {
	// Ed25519 private key is 64 bytes: 32-byte seed + 32-byte public key
	// JWK uses the 32-byte seed as the 'd' parameter
	seed := privKey.Seed()
	pubKey := privKey.Public().(ed25519.PublicKey)

	return &JWK{
		Kty: "OKP",
		Use: "sig",
		Kid: keyID,
		Alg: "EdDSA",
		Crv: "Ed25519",
		X:   base64.RawURLEncoding.EncodeToString(pubKey),
		D:   base64.RawURLEncoding.EncodeToString(seed),
	}, nil
}

// importECDSAPublicKeyFromJWK imports an ECDSA public key from JWK format
func importECDSAPublicKeyFromJWK(jwk *JWK) (*ecdsa.PublicKey, error) {
	if jwk.Kty != "EC" {
		return nil, fmt.Errorf("expected EC key type, got %s", jwk.Kty)
	}

	var curve elliptic.Curve
	switch jwk.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "secp256k1":
		// Use secp256k1 curve (Ethereum)
		// Note: Go's crypto/elliptic doesn't include secp256k1, but we can use x509 marshaling
		return importSecp256k1PublicKeyFromJWK(jwk)
	default:
		return nil, fmt.Errorf("unsupported curve: %s", jwk.Crv)
	}

	// Decode coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	// Validate the public key is on the curve
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("public key not on curve")
	}

	return pubKey, nil
}

// importSecp256k1PublicKeyFromJWK imports a secp256k1 public key from JWK
func importSecp256k1PublicKeyFromJWK(jwk *JWK) (*ecdsa.PublicKey, error) {
	// Decode coordinates
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Y coordinate: %w", err)
	}

	// secp256k1 uses compressed or uncompressed format
	// Create uncompressed format: 0x04 || X || Y
	uncompressed := make([]byte, 1+len(xBytes)+len(yBytes))
	uncompressed[0] = 0x04
	copy(uncompressed[1:], xBytes)
	copy(uncompressed[1+len(xBytes):], yBytes)

	// Parse using x509 (which supports secp256k1 through btcec)
	pubKey, err := x509.ParsePKIXPublicKey(uncompressed)
	if err != nil {
		// Fallback: manually construct secp256k1 public key
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		return &ecdsa.PublicKey{
			Curve: nil, // secp256k1 curve (not in stdlib)
			X:     x,
			Y:     y,
		}, nil
	}

	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected ECDSA public key, got %T", pubKey)
	}

	return ecdsaPubKey, nil
}

// importECDSAPrivateKeyFromJWK imports an ECDSA private key from JWK format
func importECDSAPrivateKeyFromJWK(jwk *JWK) (*ecdsa.PrivateKey, error) {
	// First import public key
	pubKey, err := importECDSAPublicKeyFromJWK(jwk)
	if err != nil {
		return nil, err
	}

	// Decode private key parameter
	dBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("failed to decode D parameter: %w", err)
	}

	d := new(big.Int).SetBytes(dBytes)

	return &ecdsa.PrivateKey{
		PublicKey: *pubKey,
		D:         d,
	}, nil
}

// importOKPPublicKeyFromJWK imports an OKP (Ed25519) public key from JWK format
func importOKPPublicKeyFromJWK(jwk *JWK) (ed25519.PublicKey, error) {
	if jwk.Kty != "OKP" {
		return nil, fmt.Errorf("expected OKP key type, got %s", jwk.Kty)
	}

	if jwk.Crv != "Ed25519" {
		return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
	}

	// Decode public key
	pubKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode X parameter: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: %d", len(pubKeyBytes))
	}

	return ed25519.PublicKey(pubKeyBytes), nil
}

// importOKPPrivateKeyFromJWK imports an OKP (Ed25519) private key from JWK format
func importOKPPrivateKeyFromJWK(jwk *JWK) (ed25519.PrivateKey, error) {
	if jwk.Kty != "OKP" {
		return nil, fmt.Errorf("expected OKP key type, got %s", jwk.Kty)
	}

	if jwk.Crv != "Ed25519" {
		return nil, fmt.Errorf("unsupported OKP curve: %s", jwk.Crv)
	}

	// Decode seed (private key)
	seedBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("failed to decode D parameter: %w", err)
	}

	if len(seedBytes) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid Ed25519 seed size: %d", len(seedBytes))
	}

	// Generate private key from seed
	privKey := ed25519.NewKeyFromSeed(seedBytes)

	return privKey, nil
}

// padLeft pads a byte slice with zeros on the left to reach the target length
func padLeft(data []byte, targetLen int) []byte {
	if len(data) >= targetLen {
		return data
	}

	padded := make([]byte, targetLen)
	copy(padded[targetLen-len(data):], data)
	return padded
}

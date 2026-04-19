package mcp

import (
	"crypto/ed25519"
	"encoding/base64"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// ServerIdentity describes an MCP server's identity and verifiable credentials.
// It enables clients to verify the authenticity of the server they are
// communicating with using Ed25519 public key signatures.
type ServerIdentity struct {
	// Name is the server's human-readable name.
	Name string `json:"name"`

	// Version is the server's version string.
	Version string `json:"version"`

	// Description is an optional human-readable description of the server.
	Description string `json:"description,omitempty"`

	// Capabilities describes the server's advertised capabilities.
	Capabilities *ServerCapabilities `json:"capabilities,omitempty"`

	// PublicKey is the server's Ed25519 public key, base64-encoded (standard
	// encoding with padding).
	PublicKey string `json:"publicKey,omitempty"`
}

// Verify checks that the provided signature is a valid Ed25519 signature of
// message using this server's public key.
//
// The public key must be a base64-encoded Ed25519 public key (32 bytes decoded).
// The signature must be base64-encoded.
func (si *ServerIdentity) Verify(message []byte, signature string) error {
	if si.PublicKey == "" {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: server has no public key")
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(si.PublicKey)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: decode public key: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: invalid public key size %d, expected %d", len(pubKeyBytes), ed25519.PublicKeySize)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: decode signature: %w", err)
	}

	if len(sigBytes) != ed25519.SignatureSize {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: invalid signature size %d, expected %d", len(sigBytes), ed25519.SignatureSize)
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	if !ed25519.Verify(pubKey, message, sigBytes) {
		return core.Errorf(core.ErrInvalidInput, "mcp/identity: signature verification failed")
	}

	return nil
}

// Sign creates an Ed25519 signature of message using the provided private key.
// This is a helper for servers to sign messages. The private key must be a
// 64-byte Ed25519 private key. The returned signature is base64-encoded.
func Sign(privateKey ed25519.PrivateKey, message []byte) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", core.Errorf(core.ErrInvalidInput, "mcp/identity: invalid private key size %d, expected %d", len(privateKey), ed25519.PrivateKeySize)
	}

	sig := ed25519.Sign(privateKey, message)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// PublicKeyFromPrivate returns the base64-encoded public key corresponding to
// the given Ed25519 private key.
func PublicKeyFromPrivate(privateKey ed25519.PrivateKey) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", core.Errorf(core.ErrInvalidInput, "mcp/identity: invalid private key size %d, expected %d", len(privateKey), ed25519.PrivateKeySize)
	}

	pubKey := privateKey.Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(pubKey), nil
}

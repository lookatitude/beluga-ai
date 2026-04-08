package mcp

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
)

func TestServerIdentity_Verify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	pubKeyB64 := base64.StdEncoding.EncodeToString(pub)

	identity := &ServerIdentity{
		Name:      "test-server",
		Version:   "1.0.0",
		PublicKey: pubKeyB64,
	}

	message := []byte("hello world")
	sig := ed25519.Sign(priv, message)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	t.Run("valid signature", func(t *testing.T) {
		err := identity.Verify(message, sigB64)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		badSig := base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		err := identity.Verify(message, badSig)
		if err == nil {
			t.Fatal("expected error for invalid signature")
		}
		if !strings.Contains(err.Error(), "verification failed") {
			t.Errorf("error %q should mention 'verification failed'", err)
		}
	})

	t.Run("wrong message", func(t *testing.T) {
		err := identity.Verify([]byte("wrong message"), sigB64)
		if err == nil {
			t.Fatal("expected error for wrong message")
		}
	})

	t.Run("no public key", func(t *testing.T) {
		noKey := &ServerIdentity{Name: "test"}
		err := noKey.Verify(message, sigB64)
		if err == nil {
			t.Fatal("expected error for missing public key")
		}
		if !strings.Contains(err.Error(), "no public key") {
			t.Errorf("error %q should mention 'no public key'", err)
		}
	})

	t.Run("invalid public key encoding", func(t *testing.T) {
		badKey := &ServerIdentity{PublicKey: "not-valid-base64!!!"}
		err := badKey.Verify(message, sigB64)
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
	})

	t.Run("wrong size public key", func(t *testing.T) {
		shortKey := &ServerIdentity{
			PublicKey: base64.StdEncoding.EncodeToString([]byte("too short")),
		}
		err := shortKey.Verify(message, sigB64)
		if err == nil {
			t.Fatal("expected error for wrong key size")
		}
		if !strings.Contains(err.Error(), "invalid public key size") {
			t.Errorf("error %q should mention 'invalid public key size'", err)
		}
	})

	t.Run("invalid signature encoding", func(t *testing.T) {
		err := identity.Verify(message, "not-valid-base64!!!")
		if err == nil {
			t.Fatal("expected error for invalid signature encoding")
		}
	})

	t.Run("wrong size signature", func(t *testing.T) {
		shortSig := base64.StdEncoding.EncodeToString([]byte("short"))
		err := identity.Verify(message, shortSig)
		if err == nil {
			t.Fatal("expected error for wrong signature size")
		}
		if !strings.Contains(err.Error(), "invalid signature size") {
			t.Errorf("error %q should mention 'invalid signature size'", err)
		}
	})
}

func TestSign(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	message := []byte("test message")
	sig, err := Sign(priv, message)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if sig == "" {
		t.Fatal("expected non-empty signature")
	}

	// Verify the signature is valid base64.
	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		t.Errorf("expected signature size %d, got %d", ed25519.SignatureSize, len(sigBytes))
	}
}

func TestSign_InvalidKey(t *testing.T) {
	_, err := Sign(ed25519.PrivateKey([]byte("too short")), []byte("msg"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid private key size") {
		t.Errorf("error %q should mention 'invalid private key size'", err)
	}
}

func TestPublicKeyFromPrivate(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pubB64, err := PublicKeyFromPrivate(priv)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivate: %v", err)
	}

	expectedB64 := base64.StdEncoding.EncodeToString(pub)
	if pubB64 != expectedB64 {
		t.Errorf("expected %q, got %q", expectedB64, pubB64)
	}
}

func TestPublicKeyFromPrivate_InvalidKey(t *testing.T) {
	_, err := PublicKeyFromPrivate(ed25519.PrivateKey([]byte("short")))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSign_VerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	pubB64, err := PublicKeyFromPrivate(priv)
	if err != nil {
		t.Fatal(err)
	}

	_ = pub // not used directly

	message := []byte("round trip test")
	sig, err := Sign(priv, message)
	if err != nil {
		t.Fatal(err)
	}

	identity := &ServerIdentity{
		Name:      "test",
		PublicKey: pubB64,
	}

	if err := identity.Verify(message, sig); err != nil {
		t.Fatalf("round-trip verification failed: %v", err)
	}
}

package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	salt := make([]byte, SaltSize)
	key := DeriveKey([]byte("testpass"), salt)
	if len(key) != KeySize {
		t.Fatalf("expected key length %d, got %d", KeySize, len(key))
	}

	key2 := DeriveKey([]byte("testpass"), salt)
	for i := range key {
		if key[i] != key2[i] {
			t.Fatal("same passphrase+salt should produce same key")
		}
	}

	key3 := DeriveKey([]byte("different"), salt)
	same := true
	for i := range key {
		if key[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different passphrases should produce different keys")
	}
}

func TestEncryptDecryptFile(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "test.txt")
	enc := filepath.Join(dir, "test.enc")
	dec := filepath.Join(dir, "test.dec")

	content := []byte("secret data for red team engagement")
	os.WriteFile(plain, content, 0600)

	pass := []byte("strongpass123")

	if err := EncryptFile(plain, enc, pass); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if !IsEncrypted(enc) {
		t.Fatal("encrypted file should be detected as encrypted")
	}
	if IsEncrypted(plain) {
		t.Fatal("plaintext file should not be detected as encrypted")
	}

	if err := DecryptFile(enc, dec, pass); err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	result, _ := os.ReadFile(dec)
	if string(result) != string(content) {
		t.Fatalf("decrypted content mismatch: got %q", result)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "test.txt")
	enc := filepath.Join(dir, "test.enc")
	dec := filepath.Join(dir, "test.dec")

	os.WriteFile(plain, []byte("secret"), 0600)
	EncryptFile(plain, enc, []byte("correct"))

	err := DecryptFile(enc, dec, []byte("wrong"))
	if err != ErrInvalidPassphrase {
		t.Fatalf("expected ErrInvalidPassphrase, got %v", err)
	}
}

func TestHashEvidence(t *testing.T) {
	h1 := HashEvidence("2026-01-01T00:00:00Z", "command", "whoami", "root", 0, "[]", "op1")
	h2 := HashEvidence("2026-01-01T00:00:00Z", "command", "whoami", "root", 0, "[]", "op1")
	if h1 != h2 {
		t.Fatal("same inputs should produce same hash")
	}

	h3 := HashEvidence("2026-01-01T00:00:00Z", "command", "whoami", "admin", 0, "[]", "op1")
	if h1 == h3 {
		t.Fatal("different outputs should produce different hashes")
	}

	if len(h1) < 70 {
		t.Fatalf("hash too short: %s", h1)
	}
	if h1[:7] != "sha256:" {
		t.Fatalf("hash should start with sha256: got %s", h1[:7])
	}
}

func TestRandomHex(t *testing.T) {
	h1, err := RandomHex(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(h1) != 32 {
		t.Fatalf("expected 32 hex chars, got %d", len(h1))
	}

	h2, _ := RandomHex(16)
	if h1 == h2 {
		t.Fatal("two random hex values should differ")
	}
}

func TestZeroBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ZeroBytes(data)
	for i, b := range data {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %d", i, b)
		}
	}
}

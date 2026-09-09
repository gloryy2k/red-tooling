package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
)

const (
	SaltSize       = 32
	NonceSize      = 12 // AES-256-GCM standard nonce
	KeySize        = 32 // AES-256
	ArgonMemoryKB  = 512 * 1024 // 512 MB
	ArgonTime      = 4
	ArgonThreads   = 4
	MagicHeader    = "RTENC\x01" // 6 bytes: file format identifier + version
)

var (
	ErrInvalidPassphrase = errors.New("invalid passphrase or corrupted file")
	ErrNotEncrypted      = errors.New("file is not encrypted")
)

// DeriveKey derives an AES-256 key from a passphrase using Argon2id.
func DeriveKey(passphrase []byte, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, ArgonTime, ArgonMemoryKB, ArgonThreads, KeySize)
}

// EncryptFile encrypts srcPath and writes the result to dstPath.
// Format: MAGIC(6) | SALT(32) | NONCE(12) | CIPHERTEXT | GCM_TAG(16)
func EncryptFile(srcPath, dstPath string, passphrase []byte) error {
	plaintext, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	key := DeriveKey(passphrase, salt)
	defer ZeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	ZeroBytes(plaintext)

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	if _, err := out.Write([]byte(MagicHeader)); err != nil {
		return err
	}
	if _, err := out.Write(salt); err != nil {
		return err
	}
	if _, err := out.Write(nonce); err != nil {
		return err
	}
	if _, err := out.Write(ciphertext); err != nil {
		return err
	}

	return nil
}

// DecryptFile decrypts srcPath and writes plaintext to dstPath.
func DecryptFile(srcPath, dstPath string, passphrase []byte) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read encrypted file: %w", err)
	}

	headerLen := len(MagicHeader)
	minSize := headerLen + SaltSize + NonceSize + 16 // at least GCM tag
	if len(data) < minSize {
		return ErrNotEncrypted
	}

	if string(data[:headerLen]) != MagicHeader {
		return ErrNotEncrypted
	}

	offset := headerLen
	salt := data[offset : offset+SaltSize]
	offset += SaltSize
	nonce := data[offset : offset+NonceSize]
	offset += NonceSize
	ciphertext := data[offset:]

	key := DeriveKey(passphrase, salt)
	defer ZeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ErrInvalidPassphrase
	}

	if err := os.WriteFile(dstPath, plaintext, 0600); err != nil {
		ZeroBytes(plaintext)
		return fmt.Errorf("write decrypted file: %w", err)
	}

	ZeroBytes(plaintext)
	return nil
}

// IsEncrypted checks if a file has the RT encryption header.
func IsEncrypted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, len(MagicHeader))
	if _, err := io.ReadFull(f, header); err != nil {
		return false
	}
	return string(header) == MagicHeader
}

// HashEvidence computes SHA-256 hash for evidence chain.
func HashEvidence(timestamp, action, input, output string, exitCode int, tags, operator string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%d|%s|%s", timestamp, action, input, output, exitCode, tags, operator)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ZeroBytes overwrites a byte slice with zeros.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// RandomHex generates a random hex string of n bytes.
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

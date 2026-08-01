package newapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// NewAPIModuleEncryptionKeyEnv is the environment variable that supplies the
// module's dedicated key for encrypting relay tokens and per-user access
// tokens at rest. It is intentionally independent from the JWT secret so that
// a compromise of one does not compromise the other.
const NewAPIModuleEncryptionKeyEnv = "NEWAPI_MODULE_ENCRYPTION_KEY"

var errInvalidCiphertext = errors.New("newapi: invalid ciphertext")

// newAPICipher encrypts/decrypts managed secrets using AES-256-GCM.
type newAPICipher struct {
	aead cipher.AEAD
}

// newCipher builds the module cipher from the provided key. The key must be at
// least 16 bytes; longer keys are hashed down to 32 bytes via SHA-256 so that
// arbitrary-length secrets are accepted while AES-256 strength is guaranteed.
func newCipher(key string) (*newAPICipher, error) {
	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &newAPICipher{aead: aead}, nil
}

// Encrypt encodes plaintext as base64(nonce + ciphertext).
func (c *newAPICipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func (c *newAPICipher) Decrypt(encoded string) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errInvalidCiphertext
	}
	ns := c.aead.NonceSize()
	if len(sealed) < ns {
		return "", errInvalidCiphertext
	}
	plaintext, err := c.aead.Open(nil, sealed[:ns], sealed[ns:], nil)
	if err != nil {
		return "", errInvalidCiphertext
	}
	return string(plaintext), nil
}

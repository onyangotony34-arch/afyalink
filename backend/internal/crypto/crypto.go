// Package crypto provides AES-256-GCM field encryption for the PHI columns
// (diagnosis, allergies, discharge summary, clinical note bodies).
//
// Ciphertext layout is nonce || ciphertext || GCM tag. The nonce is drawn from
// crypto/rand on every call, so encrypting the same plaintext twice produces
// different ciphertext — deliberate, since deterministic ciphertext would let
// anyone with database access correlate patients by shared diagnosis.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// ErrInvalidCiphertext is returned when stored bytes cannot be authenticated.
// It is deliberately vague: callers must not surface decryption detail to API
// clients.
var ErrInvalidCiphertext = errors.New("ciphertext is invalid or has been tampered with")

// Cipher encrypts and decrypts PHI field values.
type Cipher struct {
	aead cipher.AEAD
}

// New builds a Cipher from a 32-byte key. Any other key length is rejected;
// silently truncating or padding a key is how AES-256 becomes AES-128 by
// accident.
func New(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: AES-256 requires a 32-byte key, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: creating AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: creating GCM: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

// Encrypt seals plaintext and returns nonce-prefixed ciphertext ready for a
// BYTEA column.
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generating nonce: %w", err)
	}

	// Seal appends to its first argument, so passing the nonce slice makes the
	// nonce the ciphertext's prefix in a single allocation.
	return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt opens nonce-prefixed ciphertext. It returns ErrInvalidCiphertext for
// any authentication failure so callers cannot distinguish tampering from
// corruption.
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}
	return plaintext, nil
}

// EncryptString is the string convenience form of Encrypt.
func (c *Cipher) EncryptString(plaintext string) ([]byte, error) {
	return c.Encrypt([]byte(plaintext))
}

// DecryptString is the string convenience form of Decrypt.
func (c *Cipher) DecryptString(ciphertext []byte) (string, error) {
	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptNullable encrypts an optional field. A nil pointer maps to a nil
// BYTEA (SQL NULL) rather than to ciphertext of the empty string, so "not
// recorded" stays distinguishable from "recorded as empty".
func (c *Cipher) EncryptNullable(plaintext *string) ([]byte, error) {
	if plaintext == nil {
		return nil, nil
	}
	return c.EncryptString(*plaintext)
}

// DecryptNullable decrypts an optional field, mapping SQL NULL back to nil.
func (c *Cipher) DecryptNullable(ciphertext []byte) (*string, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}
	plaintext, err := c.DecryptString(ciphertext)
	if err != nil {
		return nil, err
	}
	return &plaintext, nil
}

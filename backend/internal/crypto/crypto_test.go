package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestNewRejectsWrongKeyLength(t *testing.T) {
	for _, size := range []int{0, 16, 24, 31, 33, 64} {
		if _, err := New(make([]byte, size)); err == nil {
			t.Fatalf("New accepted a %d-byte key; AES-256 requires exactly 32", size)
		}
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []string{
		"",
		"Post-operative recovery following emergency appendectomy",
		"Penicillin, sulfa drugs",
		strings.Repeat("long diagnosis text ", 500),
		"unicode: dawa ya maumivu — 痛み止め",
	}

	for _, plaintext := range cases {
		ciphertext, err := c.EncryptString(plaintext)
		if err != nil {
			t.Fatalf("EncryptString(%q): %v", truncate(plaintext), err)
		}

		got, err := c.DecryptString(ciphertext)
		if err != nil {
			t.Fatalf("DecryptString: %v", err)
		}
		if got != plaintext {
			t.Fatalf("round trip mismatch for %q", truncate(plaintext))
		}
	}
}

// Plaintext must never appear in the stored bytes — the whole point of the
// _enc columns.
func TestCiphertextDoesNotContainPlaintext(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	plaintext := "community-acquired pneumonia"
	ciphertext, err := c.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	if bytes.Contains(ciphertext, []byte(plaintext)) {
		t.Fatal("ciphertext contains the plaintext")
	}
}

// Encryption must be non-deterministic, otherwise anyone with database access
// could group patients by identical diagnosis ciphertext.
func TestEncryptionIsNonDeterministic(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	first, err := c.EncryptString("hypertension")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	second, err := c.EncryptString("hypertension")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("encrypting the same plaintext twice produced identical ciphertext")
	}
}

// GCM authentication must reject any modification.
func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ciphertext, err := c.EncryptString("fractured left tibia")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	tampered := bytes.Clone(ciphertext)
	tampered[len(tampered)-1] ^= 0xFF

	if _, err := c.Decrypt(tampered); err != ErrInvalidCiphertext {
		t.Fatalf("tampered ciphertext: got err %v, want ErrInvalidCiphertext", err)
	}
}

func TestDecryptRejectsTruncatedCiphertext(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := c.Decrypt([]byte{1, 2, 3}); err != ErrInvalidCiphertext {
		t.Fatalf("truncated ciphertext: got err %v, want ErrInvalidCiphertext", err)
	}
}

// A different key must not decrypt — confirms the key is actually in use.
func TestDecryptWithWrongKeyFails(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	otherKey := testKey(t)
	otherKey[0] ^= 0xFF
	other, err := New(otherKey)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ciphertext, err := c.EncryptString("asthma")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	if _, err := other.Decrypt(ciphertext); err != ErrInvalidCiphertext {
		t.Fatalf("wrong key: got err %v, want ErrInvalidCiphertext", err)
	}
}

// A nil optional field must stay nil rather than becoming ciphertext of "".
func TestNullableRoundTrip(t *testing.T) {
	c, err := New(testKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	encrypted, err := c.EncryptNullable(nil)
	if err != nil {
		t.Fatalf("EncryptNullable(nil): %v", err)
	}
	if encrypted != nil {
		t.Fatal("EncryptNullable(nil) produced ciphertext; expected nil so the column stays SQL NULL")
	}

	decrypted, err := c.DecryptNullable(nil)
	if err != nil {
		t.Fatalf("DecryptNullable(nil): %v", err)
	}
	if decrypted != nil {
		t.Fatal("DecryptNullable(nil) returned a value; expected nil")
	}

	value := "latex allergy"
	encrypted, err = c.EncryptNullable(&value)
	if err != nil {
		t.Fatalf("EncryptNullable: %v", err)
	}

	decrypted, err = c.DecryptNullable(encrypted)
	if err != nil {
		t.Fatalf("DecryptNullable: %v", err)
	}
	if decrypted == nil || *decrypted != value {
		t.Fatalf("nullable round trip mismatch: got %v, want %q", decrypted, value)
	}
}

func truncate(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

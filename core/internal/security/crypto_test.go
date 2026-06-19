package security

import (
	"bytes"
	"os"
	"testing"
)

// TestMain allows us to set up the environment configuration safely
// before any package tests execute.
func TestMain(m *testing.M) {
	// Set a default key for the testing suite run
	os.Setenv("APP_KEY", "test-env-key-for-security-package-12345")

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

// TestEncryptDecrypt verifies the fundamental happy path:
// Data encrypted can be successfully decrypted back to its original state.
func TestEncryptDecrypt(t *testing.T) {
	secretPayload := "The quick brown fox jumps over the lazy dog."

	// 1. Run Encryption
	ciphertext, err := Encrypt(secretPayload)
	if err != nil {
		t.Fatalf("Encryption failed unexpectedly: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("Encryption returned an empty byte slice")
	}

	// 2. Run Decryption
	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed unexpectedly: %v", err)
	}

	// 3. Validate Match
	if decrypted != secretPayload {
		t.Errorf("Decryption mismatch.\nExpected: %q\nGot:      %q", secretPayload, decrypted)
	}
}

// TestCiphertextRandomness ensures that encrypting the same string twice
// produces completely different ciphertexts due to the random nonce.
func TestCiphertextRandomness(t *testing.T) {
	input := "identical string"

	cipher1, err := Encrypt(input)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	cipher2, err := Encrypt(input)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	if bytes.Equal(cipher1, cipher2) {
		t.Error("Security flaw: Encrypting the same plaintext twice produced identical ciphertext. Nonce logic is failing.")
	}
}

// TestTamperResistance validates that AES-GCM tag verification blocks
// malicious modifications to the payload or nonce.
func TestTamperResistance(t *testing.T) {
	input := "Highly sensitive accounting data"
	ciphertext, err := Encrypt(input)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	t.Run("Flipped Bit in Ciphertext", func(t *testing.T) {
		// Clone ciphertext to avoid mutating the original across subtests
		corruptedCipher := append([]byte(nil), ciphertext...)

		// Corrupt the very last byte of the payload
		corruptedCipher[len(corruptedCipher)-1] ^= 0xFF

		_, err := Decrypt(corruptedCipher)
		if err == nil {
			t.Error("Security flaw: Decrypt succeeded on altered ciphertext payload.")
		}
	})

	t.Run("Flipped Bit in Nonce", func(t *testing.T) {
		corruptedCipher := append([]byte(nil), ciphertext...)

		// Corrupt the first byte (which lives inside the nonce header)
		corruptedCipher[0] ^= 0xFF

		_, err := Decrypt(corruptedCipher)
		if err == nil {
			t.Error("Security flaw: Decrypt succeeded on altered nonce prefix.")
		}
	})

	t.Run("Truncated Ciphertext", func(t *testing.T) {
		// Pass a payload shorter than the 12-byte nonce requirement
		shortCipher := []byte{0x01, 0x02, 0x03}

		_, err := Decrypt(shortCipher)
		if err == nil {
			t.Error("Expected error when passing truncated ciphertext, got nil.")
		}
	})
}

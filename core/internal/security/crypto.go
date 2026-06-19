package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"sync"
)

type Key struct {
	aead cipher.AEAD
}

var (
	globalKey *Key
	once      sync.Once
)

func getKey() (*Key, error) {
	var err error
	once.Do(func() {
		keyStr := os.Getenv("APP_KEY")
		if keyStr == "" {
			err = errors.New("APP_KEY environment variable is not set")
			return
		}
		hasher := sha256.New()
		hasher.Write([]byte(keyStr))
		hash := hasher.Sum(nil)

		block, err := aes.NewCipher(hash)
		if err != nil {
			return
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return
		}
		globalKey = &Key{aead: gcm}
	})
	return globalKey, err
}

func Encrypt(plaintext string) ([]byte, error) {
	key, err := getKey()
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, key.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends ciphertext to the nonce, returning [nonce][ciphertext]
	return key.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func Decrypt(ciphertext []byte) (string, error) {
	key, err := getKey()
	if err != nil {
		return "", err
	}

	nonceSize := key.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("security: ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := key.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

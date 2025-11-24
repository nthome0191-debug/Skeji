package sealer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	KEY = "lfQVRuulcL2iOhOJ2r8BYTweoSKwVAJnIF9U+AL+M60="
)

func CreateOpaqueToken(buID string, scheduleID string) (string, error) {
	plaintext := []byte(buID + ":" + scheduleID)

	key, err := base64.StdEncoding.DecodeString(KEY)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ct := aesgcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ct), nil
}

func ParseOpaqueToken(token string) (string, string, error) {
	key, err := base64.StdEncoding.DecodeString(KEY)
	if err != nil {
		return "", "", err
	}

	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonceSize := aesgcm.NonceSize()
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	pt, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(pt), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token format")
	}

	return parts[0], parts[1], nil
}

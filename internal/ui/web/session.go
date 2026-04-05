package web

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
)

// getOrCreateSessionKey loads a 32-byte key from config, or from $XDG_CONFIG_HOME/dux/session.key, or creates one.
func getOrCreateSessionKey() ([]byte, error) {
	if b64Key := viper.GetString("web.session_key"); b64Key != "" {
		key, err := base64.StdEncoding.DecodeString(b64Key)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}

	keyPath, err := xdg.ConfigFile("dux/session.key")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config dir: %w", err)
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) == 32 {
			return data, nil
		}
		// if not 32 bytes, overwrite it
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return key, nil
}

func encryptSessionID(key []byte, sessionID string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(sessionID), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func decryptSessionID(key []byte, encryptedB64 string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

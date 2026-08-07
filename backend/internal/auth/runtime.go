// Package auth loads the cryptographic runtime used by admin authentication.
package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"time"

	authdomain "github.com/ilhamnugraha8944/tauco/backend/internal/auth/domain"
)

type Runtime struct {
	Tokens         *authdomain.TokenManager
	Secrets        *authdomain.SecretBox
	RecoverySecret []byte
}

func LoadRuntime(lookup func(string) (string, bool)) (Runtime, error) {
	if lookup == nil {
		return Runtime{}, errors.New("auth environment lookup is required")
	}
	required := func(name string) (string, error) {
		value, ok := lookup(name)
		if !ok || strings.TrimSpace(value) == "" {
			return "", errors.New(name + " is required")
		}
		return strings.TrimSpace(value), nil
	}
	issuer, err := required("JWT_ISSUER")
	if err != nil {
		return Runtime{}, err
	}
	audience, err := required("JWT_AUDIENCE")
	if err != nil {
		return Runtime{}, err
	}
	keyID, err := required("JWT_KEY_ID")
	if err != nil {
		return Runtime{}, err
	}
	environment := "local"
	if value, ok := lookup("APP_ENV"); ok && strings.TrimSpace(value) != "" {
		environment = strings.ToLower(strings.TrimSpace(value))
	}
	if environment != "local" && environment != "test" && environment != "staging" && environment != "production" {
		return Runtime{}, errors.New("APP_ENV must be local, test, staging, or production")
	}
	var tokens *authdomain.TokenManager
	if environment == "staging" || environment == "production" {
		encodedPrivate, loadErr := required("JWT_ED25519_PRIVATE_KEY_BASE64")
		if loadErr != nil {
			return Runtime{}, loadErr
		}
		privateKey, decodeErr := base64.RawStdEncoding.DecodeString(encodedPrivate)
		if decodeErr != nil || len(privateKey) != ed25519.PrivateKeySize {
			return Runtime{}, errors.New("JWT_ED25519_PRIVATE_KEY_BASE64 must contain one raw-base64 Ed25519 private key")
		}
		tokens, err = authdomain.NewEd25519TokenManager(ed25519.PrivateKey(privateKey), issuer, audience, keyID, 10*time.Minute)
	} else {
		privatePath, loadErr := required("JWT_PRIVATE_KEY_FILE")
		if loadErr != nil {
			return Runtime{}, loadErr
		}
		publicPath, loadErr := required("JWT_PUBLIC_KEY_FILE")
		if loadErr != nil {
			return Runtime{}, loadErr
		}
		privatePEM, readErr := os.ReadFile(privatePath)
		if readErr != nil {
			return Runtime{}, errors.New("read JWT private key failed")
		}
		publicPEM, readErr := os.ReadFile(publicPath)
		if readErr != nil {
			return Runtime{}, errors.New("read JWT public key failed")
		}
		privateKey, publicKey, parseErr := authdomain.ParseRSAKeyPair(privatePEM, publicPEM)
		if parseErr != nil {
			return Runtime{}, parseErr
		}
		tokens, err = authdomain.NewTokenManager(privateKey, publicKey, issuer, audience, keyID, 10*time.Minute)
	}
	if err != nil {
		return Runtime{}, err
	}
	encodedKey, err := required("MFA_ENCRYPTION_KEY")
	if err != nil {
		return Runtime{}, err
	}
	key, err := base64.RawStdEncoding.DecodeString(encodedKey)
	if err != nil {
		return Runtime{}, errors.New("MFA_ENCRYPTION_KEY must be raw base64")
	}
	mfaKeyID, err := required("MFA_ENCRYPTION_KEY_ID")
	if err != nil {
		return Runtime{}, err
	}
	box, err := authdomain.NewSecretBox(key, mfaKeyID)
	if err != nil {
		return Runtime{}, err
	}
	recovery, err := required("RECOVERY_CODE_HMAC_SECRET")
	if err != nil {
		return Runtime{}, err
	}
	if len(recovery) < 32 {
		return Runtime{}, errors.New("RECOVERY_CODE_HMAC_SECRET must contain at least 32 bytes")
	}
	return Runtime{Tokens: tokens, Secrets: box, RecoverySecret: []byte(recovery)}, nil
}

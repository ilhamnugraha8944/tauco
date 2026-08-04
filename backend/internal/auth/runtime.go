// Package auth loads the cryptographic runtime used by admin authentication.
package auth

import (
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
	privatePath, err := required("JWT_PRIVATE_KEY_FILE")
	if err != nil {
		return Runtime{}, err
	}
	publicPath, err := required("JWT_PUBLIC_KEY_FILE")
	if err != nil {
		return Runtime{}, err
	}
	privatePEM, err := os.ReadFile(privatePath)
	if err != nil {
		return Runtime{}, errors.New("read JWT private key failed")
	}
	publicPEM, err := os.ReadFile(publicPath)
	if err != nil {
		return Runtime{}, errors.New("read JWT public key failed")
	}
	privateKey, publicKey, err := authdomain.ParseRSAKeyPair(privatePEM, publicPEM)
	if err != nil {
		return Runtime{}, err
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
	tokens, err := authdomain.NewTokenManager(privateKey, publicKey, issuer, audience, keyID, 10*time.Minute)
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

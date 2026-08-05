// Package domain contains authentication primitives without HTTP or database dependencies.
package domain

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const (
	argonMemoryKiB   = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltBytes   = 16
	argonOutputBytes = 32
	totpPeriod       = 30 * time.Second
)

var (
	ErrInvalidCredential = errors.New("invalid credential")
	ErrInvalidToken      = errors.New("invalid access token")
)

func HashPassword(password string) (string, error) {
	if len(password) < 12 || len(password) > 128 {
		return "", ErrInvalidCredential
	}
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemoryKiB, argonParallelism, argonOutputBytes)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemoryKiB,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return false
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil ||
		memory < 8*1024 || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 8 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 64 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

type AccessClaims struct {
	SessionID string `json:"sid"`
	MFA       bool   `json:"mfa"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	private  *rsa.PrivateKey
	public   *rsa.PublicKey
	issuer   string
	audience string
	keyID    string
	ttl      time.Duration
	now      func() time.Time
}

func NewTokenManager(
	private *rsa.PrivateKey,
	public *rsa.PublicKey,
	issuer, audience, keyID string,
	ttl time.Duration,
) (*TokenManager, error) {
	if private == nil || public == nil || strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" ||
		strings.TrimSpace(keyID) == "" || ttl <= 0 || ttl > 15*time.Minute || private.N.Cmp(public.N) != 0 || private.E != public.E {
		return nil, errors.New("invalid JWT manager configuration")
	}
	return &TokenManager{private: private, public: public, issuer: issuer, audience: audience, keyID: keyID, ttl: ttl, now: time.Now}, nil
}

func (manager *TokenManager) Sign(userID, sessionID uuid.UUID, mfa bool) (string, time.Time, error) {
	if manager == nil || userID == uuid.Nil || sessionID == uuid.Nil {
		return "", time.Time{}, ErrInvalidToken
	}
	now := manager.now().UTC()
	expires := now.Add(manager.ttl)
	jti, err := uuid.NewV7()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate JWT ID: %w", err)
	}
	claims := AccessClaims{
		SessionID: sessionID.String(),
		MFA:       mfa,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    manager.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{manager.audience},
			ExpiresAt: jwt.NewNumericDate(expires),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["typ"] = "JWT"
	token.Header["kid"] = manager.keyID
	signed, err := token.SignedString(manager.private)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expires, nil
}

func (manager *TokenManager) Verify(raw string) (AccessClaims, error) {
	if manager == nil || raw == "" {
		return AccessClaims{}, ErrInvalidToken
	}
	claims := AccessClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		&claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodRS256 || token.Header["typ"] != "JWT" || token.Header["kid"] != manager.keyID {
				return nil, ErrInvalidToken
			}
			return manager.public, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(manager.issuer),
		jwt.WithAudience(manager.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(5*time.Second),
		jwt.WithTimeFunc(manager.now),
	)
	if err != nil || token == nil || !token.Valid || claims.ID == "" || claims.IssuedAt == nil || claims.NotBefore == nil ||
		claims.ExpiresAt == nil || uuid.Validate(claims.Subject) != nil || uuid.Validate(claims.SessionID) != nil {
		return AccessClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, nil, fmt.Errorf("generate RSA key: %w", err)
	}
	return private, &private.PublicKey, nil
}

func EncodeRSAKeyPair(private *rsa.PrivateKey, public *rsa.PublicKey) ([]byte, []byte, error) {
	if private == nil || public == nil || private.N.Cmp(public.N) != 0 || private.E != public.E {
		return nil, nil, errors.New("invalid RSA key pair")
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, nil, err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}), nil
}

func ParseRSAKeyPair(privatePEM, publicPEM []byte) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateBlock, _ := pem.Decode(privatePEM)
	publicBlock, _ := pem.Decode(publicPEM)
	if privateBlock == nil || privateBlock.Type != "PRIVATE KEY" || publicBlock == nil || publicBlock.Type != "PUBLIC KEY" {
		return nil, nil, errors.New("invalid RSA PEM")
	}
	privateValue, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		return nil, nil, errors.New("invalid RSA private key")
	}
	private, ok := privateValue.(*rsa.PrivateKey)
	if !ok || private.Validate() != nil {
		return nil, nil, errors.New("invalid RSA private key")
	}
	publicValue, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		return nil, nil, errors.New("invalid RSA public key")
	}
	public, ok := publicValue.(*rsa.PublicKey)
	if !ok || private.N.Cmp(public.N) != 0 || private.E != public.E {
		return nil, nil, errors.New("RSA key pair does not match")
	}
	return private, public, nil
}

type SecretBox struct {
	aead  cipher.AEAD
	keyID string
}

func NewSecretBox(key []byte, keyID string) (*SecretBox, error) {
	if len(key) != 32 || strings.TrimSpace(keyID) == "" {
		return nil, errors.New("AES-256-GCM requires a 32-byte key and key ID")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead, keyID: keyID}, nil
}

func (box *SecretBox) Seal(plaintext []byte) (ciphertext, nonce []byte, keyID string, err error) {
	if box == nil || len(plaintext) == 0 {
		return nil, nil, "", errors.New("secret plaintext is required")
	}
	nonce = make([]byte, box.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, "", err
	}
	return box.aead.Seal(nil, nonce, plaintext, []byte(box.keyID)), nonce, box.keyID, nil
}

func (box *SecretBox) Open(ciphertext, nonce []byte, keyID string) ([]byte, error) {
	if box == nil || keyID != box.keyID || len(nonce) != box.aead.NonceSize() {
		return nil, errors.New("invalid encrypted secret")
	}
	plaintext, err := box.aead.Open(nil, nonce, ciphertext, []byte(keyID))
	if err != nil {
		return nil, errors.New("invalid encrypted secret")
	}
	return plaintext, nil
}

func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

func VerifyTOTP(secret, code string, now time.Time) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	step := now.UTC().Unix() / int64(totpPeriod/time.Second)
	for offset := int64(-1); offset <= 1; offset++ {
		candidateStep := step + offset
		candidate, err := totpAt(secret, candidateStep)
		if err == nil && subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			return candidateStep, true
		}
	}
	return 0, false
}

func CurrentTOTP(secret string, now time.Time) (string, error) {
	return totpAt(secret, now.UTC().Unix()/int64(totpPeriod/time.Second))
}

func totpAt(secret string, step int64) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(decoded) < 16 || step < 0 {
		return "", ErrInvalidCredential
	}
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, uint64(step))
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(buffer)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1_000_000), nil
}

func GenerateRecoveryCodes(count int) ([]string, error) {
	if count < 1 || count > 20 {
		return nil, errors.New("invalid recovery code count")
	}
	result := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(result) < count {
		raw := make([]byte, 8)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)[:12]
		code := encoded[:4] + "-" + encoded[4:8] + "-" + encoded[8:]
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result, nil
}

func HMACRecoveryCode(secret []byte, code string) (string, error) {
	if len(secret) < 32 || len(code) != 14 {
		return "", ErrInvalidCredential
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(strings.ToUpper(code)))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func RandomToken(bytes int) (string, string, error) {
	if bytes < 32 || bytes > 128 {
		return "", "", errors.New("invalid token entropy")
	}
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:]), nil
}

func SHA256Hex(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

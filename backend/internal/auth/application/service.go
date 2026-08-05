// Package application implements admin authentication use cases.
package application

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	authdomain "github.com/ilhamnugraha8944/tauco/backend/internal/auth/domain"
)

const (
	AccessTTL     = 10 * time.Minute
	RefreshTTL    = 30 * 24 * time.Hour
	RecoveryCount = 10
	LevelPassword = "password"
	LevelMFA      = "mfa"
)

var (
	ErrAuthentication = errors.New("authentication failed")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrConflict       = errors.New("authentication state conflict")
	ErrNotFound       = errors.New("authentication record not found")
	ErrRefreshReused  = errors.New("refresh token reuse detected")
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Status       string
	MFAEnabled   bool
	Roles        []string
	Permissions  []string
}

type MFACredential struct {
	Ciphertext []byte
	Nonce      []byte
	KeyID      string
	Enabled    bool
}

type Principal struct {
	User
	SessionID        uuid.UUID
	Level            string
	CSRFTokenHash    string
	SessionExpiresAt time.Time
}

type Store interface {
	FindUserByEmail(context.Context, string) (User, error)
	FindPrincipal(context.Context, uuid.UUID, uuid.UUID) (Principal, error)
	CreateSession(context.Context, uuid.UUID, string, string, string, string, time.Time) (uuid.UUID, error)
	RotateRefresh(context.Context, string, string, string, string) (Principal, error)
	RevokeSession(context.Context, uuid.UUID, string) error
	SaveTOTPSetup(context.Context, uuid.UUID, []byte, []byte, string) error
	FindTOTP(context.Context, uuid.UUID) (MFACredential, error)
	EnableTOTP(context.Context, uuid.UUID, uuid.UUID, int64, []string) error
	ConsumeTOTPStep(context.Context, uuid.UUID, int64) (bool, error)
	ConsumeRecoveryCode(context.Context, uuid.UUID, string) (bool, error)
	ReplaceRecoveryCodes(context.Context, uuid.UUID, []string) error
	BootstrapAdmin(context.Context, uuid.UUID, string, string) error
	ResetPassword(context.Context, string, string) error
	ResetTOTP(context.Context, string) error
	RevokeAllSessions(context.Context, string, string) error
	RecordAudit(context.Context, string, *uuid.UUID, *uuid.UUID) error
}

type Service struct {
	store          Store
	tokens         *authdomain.TokenManager
	secrets        *authdomain.SecretBox
	recoverySecret []byte
	dummyHash      string
	now            func() time.Time
}

type SessionTokens struct {
	Access         string
	Refresh        string
	CSRF           string
	AccessExpires  time.Time
	SessionExpires time.Time
}

type LoginResult struct {
	Principal Principal
	Tokens    SessionTokens
}

type TOTPSetup struct {
	Secret    string
	ExpiresAt time.Time
}

type TOTPEnableResult struct {
	Codes         []string
	Access        string
	AccessExpires time.Time
}

func NewService(store Store, tokens *authdomain.TokenManager, secrets *authdomain.SecretBox, recoverySecret []byte) (*Service, error) {
	if store == nil || tokens == nil || secrets == nil || len(recoverySecret) < 32 {
		return nil, errors.New("invalid authentication service configuration")
	}
	dummyHash, err := authdomain.HashPassword("not-a-real-admin-password")
	if err != nil {
		return nil, err
	}
	return &Service{store: store, tokens: tokens, secrets: secrets, recoverySecret: append([]byte(nil), recoverySecret...), dummyHash: dummyHash, now: time.Now}, nil
}

func (service *Service) Login(
	ctx context.Context,
	email, password string,
	totpCode, recoveryCode *string,
	userAgent string,
) (LoginResult, error) {
	normalizedEmail, valid := normalizeEmail(email)
	user, findErr := service.store.FindUserByEmail(ctx, normalizedEmail)
	encoded := service.dummyHash
	if findErr == nil {
		encoded = user.PasswordHash
	}
	passwordValid := authdomain.VerifyPassword(encoded, password)
	if !valid || findErr != nil || !passwordValid || user.Status != "active" {
		_ = service.store.RecordAudit(ctx, "auth.login_failed", nil, nil)
		return LoginResult{}, ErrAuthentication
	}

	level := LevelPassword
	if user.MFAEnabled {
		if err := service.verifySecondFactor(ctx, user.ID, totpCode, recoveryCode); err != nil {
			_ = service.store.RecordAudit(ctx, "auth.login_failed", &user.ID, nil)
			return LoginResult{}, ErrAuthentication
		}
		level = LevelMFA
	}
	result, err := service.newSession(ctx, user, level, userAgent)
	if err != nil {
		return LoginResult{}, err
	}
	_ = service.store.RecordAudit(ctx, "auth.login_succeeded", &user.ID, &result.Principal.SessionID)
	return result, nil
}

func (service *Service) ValidateAccess(ctx context.Context, raw string, requireMFA bool) (Principal, error) {
	claims, err := service.tokens.Verify(raw)
	if err != nil {
		return Principal{}, ErrUnauthorized
	}
	userID, userErr := uuid.Parse(claims.Subject)
	sessionID, sessionErr := uuid.Parse(claims.SessionID)
	if userErr != nil || sessionErr != nil {
		return Principal{}, ErrUnauthorized
	}
	principal, err := service.store.FindPrincipal(ctx, userID, sessionID)
	if err != nil || principal.Status != "active" || principal.SessionExpiresAt.Before(service.now()) ||
		claims.MFA != (principal.Level == LevelMFA) || (requireMFA && principal.Level != LevelMFA) {
		return Principal{}, ErrUnauthorized
	}
	return principal, nil
}

func (service *Service) SetupTOTP(ctx context.Context, principal Principal) (TOTPSetup, error) {
	credential, err := service.store.FindTOTP(ctx, principal.ID)
	if err == nil && credential.Enabled {
		return TOTPSetup{}, ErrConflict
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return TOTPSetup{}, err
	}
	secret, err := authdomain.GenerateTOTPSecret()
	if err != nil {
		return TOTPSetup{}, err
	}
	ciphertext, nonce, keyID, err := service.secrets.Seal([]byte(secret))
	if err != nil {
		return TOTPSetup{}, err
	}
	if err := service.store.SaveTOTPSetup(ctx, principal.ID, ciphertext, nonce, keyID); err != nil {
		return TOTPSetup{}, err
	}
	_ = service.store.RecordAudit(ctx, "auth.totp_setup_started", &principal.ID, &principal.SessionID)
	return TOTPSetup{Secret: secret, ExpiresAt: service.now().Add(10 * time.Minute)}, nil
}

func (service *Service) EnableTOTP(ctx context.Context, principal Principal, code string) (TOTPEnableResult, error) {
	credential, err := service.store.FindTOTP(ctx, principal.ID)
	if err != nil || credential.Enabled {
		return TOTPEnableResult{}, ErrConflict
	}
	secret, err := service.secrets.Open(credential.Ciphertext, credential.Nonce, credential.KeyID)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	step, valid := authdomain.VerifyTOTP(string(secret), code, service.now())
	if !valid {
		return TOTPEnableResult{}, ErrAuthentication
	}
	codes, hashes, err := service.newRecoveryCodes()
	if err != nil {
		return TOTPEnableResult{}, err
	}
	if err := service.store.EnableTOTP(ctx, principal.ID, principal.SessionID, step, hashes); err != nil {
		return TOTPEnableResult{}, err
	}
	access, expires, err := service.tokens.Sign(principal.ID, principal.SessionID, true)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	_ = service.store.RecordAudit(ctx, "auth.totp_enabled", &principal.ID, &principal.SessionID)
	return TOTPEnableResult{Codes: codes, Access: access, AccessExpires: expires}, nil
}

func (service *Service) Refresh(ctx context.Context, rawRefresh, csrf string) (LoginResult, error) {
	if rawRefresh == "" || csrf == "" {
		return LoginResult{}, ErrUnauthorized
	}
	newRefresh, newHash, err := authdomain.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	newCSRF, newCSRFHash, err := authdomain.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	principal, err := service.store.RotateRefresh(ctx, authdomain.SHA256Hex(rawRefresh), authdomain.SHA256Hex(csrf), newHash, newCSRFHash)
	if err != nil {
		if errors.Is(err, ErrRefreshReused) {
			return LoginResult{}, ErrRefreshReused
		}
		return LoginResult{}, ErrUnauthorized
	}
	access, accessExpires, err := service.tokens.Sign(principal.ID, principal.SessionID, true)
	if err != nil {
		return LoginResult{}, err
	}
	_ = service.store.RecordAudit(ctx, "auth.session_refreshed", &principal.ID, &principal.SessionID)
	return LoginResult{Principal: principal, Tokens: SessionTokens{
		Access: access, Refresh: newRefresh, CSRF: newCSRF,
		AccessExpires: accessExpires, SessionExpires: principal.SessionExpiresAt,
	}}, nil
}

func (service *Service) Logout(ctx context.Context, principal Principal) error {
	if err := service.store.RevokeSession(ctx, principal.SessionID, "LOGOUT"); err != nil {
		return err
	}
	_ = service.store.RecordAudit(ctx, "auth.logout", &principal.ID, &principal.SessionID)
	return nil
}

func (service *Service) RegenerateRecoveryCodes(ctx context.Context, principal Principal, code string) ([]string, error) {
	credential, err := service.store.FindTOTP(ctx, principal.ID)
	if err != nil || !credential.Enabled {
		return nil, ErrConflict
	}
	secret, err := service.secrets.Open(credential.Ciphertext, credential.Nonce, credential.KeyID)
	if err != nil {
		return nil, err
	}
	step, valid := authdomain.VerifyTOTP(string(secret), code, service.now())
	if !valid {
		return nil, ErrAuthentication
	}
	consumed, err := service.store.ConsumeTOTPStep(ctx, principal.ID, step)
	if err != nil || !consumed {
		return nil, ErrAuthentication
	}
	codes, hashes, err := service.newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	if err := service.store.ReplaceRecoveryCodes(ctx, principal.ID, hashes); err != nil {
		return nil, err
	}
	_ = service.store.RecordAudit(ctx, "auth.recovery_codes_regenerated", &principal.ID, &principal.SessionID)
	return codes, nil
}

func (service *Service) BootstrapAdmin(ctx context.Context, email, password string) error {
	normalized, valid := normalizeEmail(email)
	if !valid {
		return ErrAuthentication
	}
	hash, err := authdomain.HashPassword(password)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return service.store.BootstrapAdmin(ctx, id, normalized, hash)
}

func (service *Service) ResetPassword(ctx context.Context, email, password string) error {
	normalized, valid := normalizeEmail(email)
	if !valid {
		return ErrNotFound
	}
	hash, err := authdomain.HashPassword(password)
	if err != nil {
		return err
	}
	return service.store.ResetPassword(ctx, normalized, hash)
}

func (service *Service) ResetTOTP(ctx context.Context, email string) error {
	normalized, valid := normalizeEmail(email)
	if !valid {
		return ErrNotFound
	}
	return service.store.ResetTOTP(ctx, normalized)
}

func (service *Service) RevokeSessions(ctx context.Context, email, reason string) error {
	normalized, valid := normalizeEmail(email)
	if !valid || strings.TrimSpace(reason) == "" {
		return ErrNotFound
	}
	return service.store.RevokeAllSessions(ctx, normalized, strings.ToUpper(strings.TrimSpace(reason)))
}

func (service *Service) newSession(ctx context.Context, user User, level, userAgent string) (LoginResult, error) {
	refresh, refreshHash, err := authdomain.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	csrf, csrfHash, err := authdomain.RandomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	sessionExpires := service.now().Add(RefreshTTL)
	sessionID, err := service.store.CreateSession(ctx, user.ID, level, csrfHash, authdomain.SHA256Hex(userAgent), refreshHash, sessionExpires)
	if err != nil {
		return LoginResult{}, err
	}
	access, accessExpires, err := service.tokens.Sign(user.ID, sessionID, level == LevelMFA)
	if err != nil {
		_ = service.store.RevokeSession(ctx, sessionID, "TOKEN_ISSUE_FAILED")
		return LoginResult{}, err
	}
	return LoginResult{Principal: Principal{User: user, SessionID: sessionID, Level: level, CSRFTokenHash: csrfHash, SessionExpiresAt: sessionExpires}, Tokens: SessionTokens{
		Access: access, Refresh: refresh, CSRF: csrf,
		AccessExpires: accessExpires, SessionExpires: sessionExpires,
	}}, nil
}

func (service *Service) verifySecondFactor(ctx context.Context, userID uuid.UUID, totpCode, recoveryCode *string) error {
	if (totpCode == nil) == (recoveryCode == nil) {
		return ErrAuthentication
	}
	if totpCode != nil {
		credential, err := service.store.FindTOTP(ctx, userID)
		if err != nil || !credential.Enabled {
			return ErrAuthentication
		}
		secret, err := service.secrets.Open(credential.Ciphertext, credential.Nonce, credential.KeyID)
		if err != nil {
			return ErrAuthentication
		}
		step, valid := authdomain.VerifyTOTP(string(secret), *totpCode, service.now())
		if !valid {
			return ErrAuthentication
		}
		consumed, err := service.store.ConsumeTOTPStep(ctx, userID, step)
		if err != nil || !consumed {
			return ErrAuthentication
		}
		return nil
	}
	hash, err := authdomain.HMACRecoveryCode(service.recoverySecret, *recoveryCode)
	if err != nil {
		return ErrAuthentication
	}
	consumed, err := service.store.ConsumeRecoveryCode(ctx, userID, hash)
	if err != nil || !consumed {
		return ErrAuthentication
	}
	return nil
}

func (service *Service) newRecoveryCodes() ([]string, []string, error) {
	codes, err := authdomain.GenerateRecoveryCodes(RecoveryCount)
	if err != nil {
		return nil, nil, err
	}
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := authdomain.HMACRecoveryCode(service.recoverySecret, code)
		if err != nil {
			return nil, nil, err
		}
		hashes = append(hashes, hash)
	}
	return codes, hashes, nil
}

func normalizeEmail(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(normalized)
	return normalized, err == nil && address.Address == normalized && len(normalized) <= 160
}

// Package repository persists authentication state in PostgreSQL.
package repository

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authapp "github.com/ilhamnugraha8944/tauco/backend/internal/auth/application"
	"gorm.io/gorm"
)

type Postgres struct {
	db *gorm.DB
}

func NewPostgres(db *gorm.DB) (*Postgres, error) {
	if db == nil {
		return nil, errors.New("auth repository requires PostgreSQL")
	}
	return &Postgres{db: db}, nil
}

func (store *Postgres) FindUserByEmail(ctx context.Context, email string) (authapp.User, error) {
	var row struct {
		ID                          uuid.UUID
		Email, PasswordHash, Status string
		MFAEnabled                  bool
	}
	result := store.db.WithContext(ctx).Raw(`
SELECT user_account.id, user_account.email, user_account.password_hash,
       user_account.status,
       EXISTS (
           SELECT 1 FROM tauco_app.mfa_credentials AS credential
           WHERE credential.admin_user_id = user_account.id
             AND credential.enabled_at IS NOT NULL
             AND credential.revoked_at IS NULL
       ) AS mfa_enabled
FROM tauco_app.admin_users AS user_account
WHERE user_account.email = ?`, email).Scan(&row)
	if result.Error != nil {
		return authapp.User{}, result.Error
	}
	if result.RowsAffected != 1 {
		return authapp.User{}, authapp.ErrNotFound
	}
	user := authapp.User{ID: row.ID, Email: row.Email, PasswordHash: row.PasswordHash, Status: row.Status, MFAEnabled: row.MFAEnabled}
	if err := store.loadAuthorization(ctx, store.db, &user); err != nil {
		return authapp.User{}, err
	}
	return user, nil
}

func (store *Postgres) FindPrincipal(ctx context.Context, userID, sessionID uuid.UUID) (authapp.Principal, error) {
	return store.findPrincipal(ctx, store.db, userID, sessionID)
}

func (store *Postgres) findPrincipal(ctx context.Context, db *gorm.DB, userID, sessionID uuid.UUID) (authapp.Principal, error) {
	var row struct {
		ID                          uuid.UUID
		Email, PasswordHash, Status string
		MFAEnabled                  bool
		SessionID                   uuid.UUID
		Level, CSRFTokenHash        string
		SessionExpiresAt            time.Time
	}
	result := db.WithContext(ctx).Raw(`
SELECT user_account.id, user_account.email, user_account.password_hash,
       user_account.status,
       EXISTS (
           SELECT 1 FROM tauco_app.mfa_credentials AS credential
           WHERE credential.admin_user_id = user_account.id
             AND credential.enabled_at IS NOT NULL
             AND credential.revoked_at IS NULL
       ) AS mfa_enabled,
       session.id AS session_id,
       session.authentication_level AS level,
       session.csrf_token_hash,
       session.expires_at AS session_expires_at
FROM tauco_app.admin_users AS user_account
JOIN tauco_app.admin_sessions AS session
  ON session.admin_user_id = user_account.id
WHERE user_account.id = ? AND session.id = ?
  AND session.status = 'active' AND session.expires_at > statement_timestamp()`, userID, sessionID).Scan(&row)
	if result.Error != nil {
		return authapp.Principal{}, result.Error
	}
	if result.RowsAffected != 1 {
		return authapp.Principal{}, authapp.ErrNotFound
	}
	user := authapp.User{ID: row.ID, Email: row.Email, PasswordHash: row.PasswordHash, Status: row.Status, MFAEnabled: row.MFAEnabled}
	if err := store.loadAuthorization(ctx, db, &user); err != nil {
		return authapp.Principal{}, err
	}
	return authapp.Principal{User: user, SessionID: row.SessionID, Level: row.Level, CSRFTokenHash: row.CSRFTokenHash, SessionExpiresAt: row.SessionExpiresAt}, nil
}

func (store *Postgres) loadAuthorization(ctx context.Context, db *gorm.DB, user *authapp.User) error {
	if err := db.WithContext(ctx).Raw(`
SELECT role.key
FROM tauco_app.user_roles AS user_role
JOIN tauco_app.roles AS role ON role.id = user_role.role_id
WHERE user_role.admin_user_id = ?
ORDER BY role.key`, user.ID).Scan(&user.Roles).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Raw(`
SELECT DISTINCT permission.key
FROM tauco_app.user_roles AS user_role
JOIN tauco_app.role_permissions AS role_permission ON role_permission.role_id = user_role.role_id
JOIN tauco_app.permissions AS permission ON permission.id = role_permission.permission_id
WHERE user_role.admin_user_id = ?
ORDER BY permission.key`, user.ID).Scan(&user.Permissions).Error
}

func (store *Postgres) CreateSession(
	ctx context.Context,
	userID uuid.UUID,
	level, csrfHash, userAgentHash, refreshHash string,
	expires time.Time,
) (uuid.UUID, error) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	refreshID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}
	err = store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
INSERT INTO tauco_app.admin_sessions (
    id, admin_user_id, authentication_level, csrf_token_hash,
    user_agent_hash, expires_at
) VALUES (?, ?, ?, ?, ?, ?)`, sessionID, userID, level, csrfHash, nullableHash(userAgentHash), expires).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
INSERT INTO tauco_app.admin_refresh_tokens (id, session_id, token_hash, expires_at)
VALUES (?, ?, ?, ?)`, refreshID, sessionID, refreshHash, expires).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE tauco_app.admin_users SET last_login_at = statement_timestamp() WHERE id = ?`, userID).Error
	})
	return sessionID, err
}

func (store *Postgres) RotateRefresh(
	ctx context.Context,
	oldHash, csrfHash, newHash, newCSRFHash string,
) (authapp.Principal, error) {
	var principal authapp.Principal
	reused := false
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			TokenID, SessionID, UserID               uuid.UUID
			TokenExpires, SessionExpires             time.Time
			UsedAt, TokenRevokedAt, SessionRevokedAt *time.Time
			SessionStatus, Level, StoredCSRF         string
		}
		result := tx.Raw(`
SELECT token.id AS token_id, token.session_id, session.admin_user_id AS user_id,
       token.expires_at AS token_expires, session.expires_at AS session_expires,
       token.used_at, token.revoked_at AS token_revoked_at,
       session.revoked_at AS session_revoked_at, session.status AS session_status,
       session.authentication_level AS level, session.csrf_token_hash AS stored_csrf
FROM tauco_app.admin_refresh_tokens AS token
JOIN tauco_app.admin_sessions AS session ON session.id = token.session_id
WHERE token.token_hash = ?
FOR UPDATE OF token, session`, oldHash).Scan(&row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return authapp.ErrUnauthorized
		}
		if row.UsedAt != nil {
			if err := tx.Exec(`
UPDATE tauco_app.admin_sessions
SET status = 'revoked', revoked_at = statement_timestamp(), revoke_reason = 'REFRESH_REUSE'
WHERE id = ? AND status <> 'revoked'`, row.SessionID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`
UPDATE tauco_app.admin_refresh_tokens
SET revoked_at = statement_timestamp()
WHERE session_id = ? AND used_at IS NULL AND revoked_at IS NULL`, row.SessionID).Error; err != nil {
				return err
			}
			reused = true
			return nil
		}
		if row.TokenRevokedAt != nil || row.SessionRevokedAt != nil || row.SessionStatus != "active" ||
			row.Level != authapp.LevelMFA || row.TokenExpires.Before(time.Now()) || row.SessionExpires.Before(time.Now()) ||
			subtle.ConstantTimeCompare([]byte(row.StoredCSRF), []byte(csrfHash)) != 1 {
			return authapp.ErrUnauthorized
		}
		if err := tx.Exec(`UPDATE tauco_app.admin_refresh_tokens SET used_at = statement_timestamp() WHERE id = ?`, row.TokenID).Error; err != nil {
			return err
		}
		newID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		if err := tx.Exec(`
INSERT INTO tauco_app.admin_refresh_tokens (id, session_id, token_hash, expires_at)
VALUES (?, ?, ?, ?)`, newID, row.SessionID, newHash, row.SessionExpires).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
UPDATE tauco_app.admin_sessions
SET csrf_token_hash = ?, last_used_at = statement_timestamp()
WHERE id = ?`, newCSRFHash, row.SessionID).Error; err != nil {
			return err
		}
		value, err := store.findPrincipal(ctx, tx, row.UserID, row.SessionID)
		if err != nil {
			return err
		}
		principal = value
		principal.CSRFTokenHash = newCSRFHash
		return nil
	})
	if err == nil && reused {
		return authapp.Principal{}, authapp.ErrRefreshReused
	}
	return principal, err
}

func (store *Postgres) RevokeSession(ctx context.Context, sessionID uuid.UUID, reason string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
UPDATE tauco_app.admin_sessions
SET status = 'revoked', revoked_at = statement_timestamp(), revoke_reason = ?
WHERE id = ? AND status <> 'revoked'`, reason, sessionID).Error; err != nil {
			return err
		}
		return tx.Exec(`
UPDATE tauco_app.admin_refresh_tokens
SET revoked_at = statement_timestamp()
WHERE session_id = ? AND used_at IS NULL AND revoked_at IS NULL`, sessionID).Error
	})
}

func (store *Postgres) SaveTOTPSetup(ctx context.Context, userID uuid.UUID, ciphertext, nonce []byte, keyID string) error {
	return store.db.WithContext(ctx).Exec(`
INSERT INTO tauco_app.mfa_credentials (
    admin_user_id, encrypted_secret, encryption_key_id, nonce
) VALUES (?, ?, ?, ?)
ON CONFLICT (admin_user_id) DO UPDATE SET
    encrypted_secret = EXCLUDED.encrypted_secret,
    encryption_key_id = EXCLUDED.encryption_key_id,
    nonce = EXCLUDED.nonce,
    last_used_step = NULL,
    enabled_at = NULL,
    revoked_at = NULL`, userID, ciphertext, keyID, nonce).Error
}

func (store *Postgres) FindTOTP(ctx context.Context, userID uuid.UUID) (authapp.MFACredential, error) {
	var row struct {
		Ciphertext, Nonce []byte
		KeyID             string
		Enabled           bool
	}
	result := store.db.WithContext(ctx).Raw(`
SELECT encrypted_secret AS ciphertext, nonce, encryption_key_id AS key_id,
       enabled_at IS NOT NULL AS enabled
FROM tauco_app.mfa_credentials
WHERE admin_user_id = ? AND revoked_at IS NULL`, userID).Scan(&row)
	if result.Error != nil {
		return authapp.MFACredential{}, result.Error
	}
	if result.RowsAffected != 1 {
		return authapp.MFACredential{}, authapp.ErrNotFound
	}
	return authapp.MFACredential{Ciphertext: row.Ciphertext, Nonce: row.Nonce, KeyID: row.KeyID, Enabled: row.Enabled}, nil
}

func (store *Postgres) EnableTOTP(ctx context.Context, userID, sessionID uuid.UUID, step int64, hashes []string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`
UPDATE tauco_app.mfa_credentials
SET enabled_at = statement_timestamp(), last_used_step = ?
WHERE admin_user_id = ? AND enabled_at IS NULL AND revoked_at IS NULL`, step, userID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return authapp.ErrConflict
		}
		if err := tx.Exec(`
UPDATE tauco_app.admin_sessions
SET authentication_level = 'mfa', last_used_at = statement_timestamp()
WHERE id = ? AND admin_user_id = ? AND status = 'active'`, sessionID, userID).Error; err != nil {
			return err
		}
		return store.insertRecoveryCodes(tx, userID, hashes)
	})
}

func (store *Postgres) ConsumeTOTPStep(ctx context.Context, userID uuid.UUID, step int64) (bool, error) {
	result := store.db.WithContext(ctx).Exec(`
UPDATE tauco_app.mfa_credentials
SET last_used_step = ?
WHERE admin_user_id = ? AND enabled_at IS NOT NULL AND revoked_at IS NULL
  AND (last_used_step IS NULL OR last_used_step < ?)`, step, userID, step)
	return result.RowsAffected == 1, result.Error
}

func (store *Postgres) ConsumeRecoveryCode(ctx context.Context, userID uuid.UUID, hash string) (bool, error) {
	result := store.db.WithContext(ctx).Exec(`
UPDATE tauco_app.mfa_recovery_codes
SET used_at = statement_timestamp()
WHERE admin_user_id = ? AND code_hash = ? AND used_at IS NULL AND revoked_at IS NULL`, userID, hash)
	return result.RowsAffected == 1, result.Error
}

func (store *Postgres) ReplaceRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes []string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
UPDATE tauco_app.mfa_recovery_codes
SET revoked_at = statement_timestamp()
WHERE admin_user_id = ? AND used_at IS NULL AND revoked_at IS NULL`, userID).Error; err != nil {
			return err
		}
		return store.insertRecoveryCodes(tx, userID, hashes)
	})
}

func (store *Postgres) insertRecoveryCodes(tx *gorm.DB, userID uuid.UUID, hashes []string) error {
	for _, hash := range hashes {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		if err := tx.Exec(`
INSERT INTO tauco_app.mfa_recovery_codes (id, admin_user_id, code_hash)
VALUES (?, ?, ?)`, id, userID, hash).Error; err != nil {
			return err
		}
	}
	return nil
}

func (store *Postgres) BootstrapAdmin(ctx context.Context, id uuid.UUID, email, passwordHash string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
INSERT INTO tauco_app.admin_users (id, email, password_hash)
VALUES (?, ?, ?)`, id, email, passwordHash).Error; err != nil {
			return fmt.Errorf("insert admin user: %w", err)
		}
		if err := tx.Exec(`
INSERT INTO tauco_app.user_roles (admin_user_id, role_id)
SELECT ?, id FROM tauco_app.roles WHERE key = 'super_admin'`, id).Error; err != nil {
			return err
		}
		return store.insertAudit(tx, "auth.admin_bootstrapped", &id, nil)
	})
}

func (store *Postgres) ResetPassword(ctx context.Context, email, passwordHash string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userID, err := store.userIDByEmail(ctx, tx, email)
		if err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE tauco_app.admin_users SET password_hash = ? WHERE id = ?`, passwordHash, userID).Error; err != nil {
			return err
		}
		if err := store.revokeUserSessions(tx, userID, "PASSWORD_RESET"); err != nil {
			return err
		}
		return store.insertAudit(tx, "auth.password_reset", &userID, nil)
	})
}

func (store *Postgres) ResetTOTP(ctx context.Context, email string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userID, err := store.userIDByEmail(ctx, tx, email)
		if err != nil {
			return err
		}
		if err := tx.Exec(`
UPDATE tauco_app.mfa_credentials
SET enabled_at = NULL, revoked_at = statement_timestamp()
WHERE admin_user_id = ? AND revoked_at IS NULL`, userID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
UPDATE tauco_app.mfa_recovery_codes
SET revoked_at = statement_timestamp()
WHERE admin_user_id = ? AND used_at IS NULL AND revoked_at IS NULL`, userID).Error; err != nil {
			return err
		}
		if err := store.revokeUserSessions(tx, userID, "TOTP_RESET"); err != nil {
			return err
		}
		return store.insertAudit(tx, "auth.totp_reset", &userID, nil)
	})
}

func (store *Postgres) RevokeAllSessions(ctx context.Context, email, reason string) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		userID, err := store.userIDByEmail(ctx, tx, email)
		if err != nil {
			return err
		}
		if err := store.revokeUserSessions(tx, userID, reason); err != nil {
			return err
		}
		return store.insertAudit(tx, "auth.sessions_revoked", &userID, nil)
	})
}

func (store *Postgres) revokeUserSessions(tx *gorm.DB, userID uuid.UUID, reason string) error {
	if err := tx.Exec(`
UPDATE tauco_app.admin_sessions
SET status = 'revoked', revoked_at = statement_timestamp(), revoke_reason = ?
WHERE admin_user_id = ? AND status <> 'revoked'`, reason, userID).Error; err != nil {
		return err
	}
	return tx.Exec(`
UPDATE tauco_app.admin_refresh_tokens AS token
SET revoked_at = statement_timestamp()
FROM tauco_app.admin_sessions AS session
WHERE token.session_id = session.id AND session.admin_user_id = ?
  AND token.used_at IS NULL AND token.revoked_at IS NULL`, userID).Error
}

func (store *Postgres) userIDByEmail(ctx context.Context, tx *gorm.DB, email string) (uuid.UUID, error) {
	var id uuid.UUID
	result := tx.WithContext(ctx).Raw(`SELECT id FROM tauco_app.admin_users WHERE email = ?`, email).Scan(&id)
	if result.Error != nil {
		return uuid.Nil, result.Error
	}
	if result.RowsAffected != 1 {
		return uuid.Nil, authapp.ErrNotFound
	}
	return id, nil
}

func (store *Postgres) RecordAudit(ctx context.Context, event string, actorID, sessionID *uuid.UUID) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return store.insertAudit(tx, event, actorID, sessionID)
	})
}

func (store *Postgres) insertAudit(tx *gorm.DB, event string, actorID, sessionID *uuid.UUID) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return tx.Exec(`
INSERT INTO tauco_app.activity_logs (
    id, event_type, entity_type, entity_id, actor_type, actor_id, metadata_json
) VALUES (?, ?, 'admin_session', ?, 'admin', ?, '{}'::jsonb)`, id, event, sessionID, actorID).Error
}

func nullableHash(value string) any {
	if value == "" {
		return nil
	}
	return value
}

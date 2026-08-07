package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	authapp "github.com/ilhamnugraha8944/tauco/backend/internal/auth/application"
	authdomain "github.com/ilhamnugraha8944/tauco/backend/internal/auth/domain"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	accessCookie  = "tauco_admin_access"
	refreshCookie = "tauco_admin_refresh"
	csrfCookie    = "tauco_admin_csrf"
	principalKey  = "tauco.admin.principal"
	bffHeader     = "X-Tauco-Admin-BFF-Secret"
)

type AdminAuthConfig struct {
	AllowedOrigins []string
	RateSecret     []byte
	BFFSecret      []byte
	SecureCookies  bool
	RequestID      func(*gin.Context) string
}

type AdminAuthHandler struct {
	service   *authapp.Service
	limiter   rateLimiter
	origins   map[string]struct{}
	rateKey   []byte
	bffKey    []byte
	secure    bool
	requestID func(*gin.Context) string
}

type rateLimiter interface {
	Allow(context.Context, string, int, time.Duration) (bool, time.Duration)
}

func NewAdminAuthHandler(service *authapp.Service, limiter rateLimiter, config AdminAuthConfig) (*AdminAuthHandler, error) {
	if service == nil || limiter == nil || len(config.RateSecret) < 32 || len(config.BFFSecret) < 32 || len(config.AllowedOrigins) == 0 {
		return nil, errors.New("invalid admin auth handler configuration")
	}
	origins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, raw := range config.AllowedOrigins {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("admin origin must be an exact HTTP(S) origin")
		}
		origins[parsed.Scheme+"://"+parsed.Host] = struct{}{}
	}
	requestID := config.RequestID
	if requestID == nil {
		requestID = func(c *gin.Context) string { return c.GetHeader("X-Request-ID") }
	}
	return &AdminAuthHandler{
		service: service, limiter: limiter, origins: origins,
		rateKey: append([]byte(nil), config.RateSecret...), bffKey: append([]byte(nil), config.BFFSecret...),
		secure: config.SecureCookies, requestID: requestID,
	}, nil
}

func (handler *AdminAuthHandler) Register(router gin.IRouter) {
	group := router.Group("/api/v1/admin/auth")
	group.Use(handler.bff())
	group.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	group.POST("/login", handler.browserMutation(), handler.login)
	group.POST("/totp/setup", handler.browserMutation(), handler.require(false, ""), handler.csrf(), handler.setupTOTP)
	group.POST("/totp/enable", handler.browserMutation(), handler.require(false, ""), handler.csrf(), handler.enableTOTP)
	group.POST("/refresh", handler.browserMutation(), handler.refresh)
	group.POST("/logout", handler.browserMutation(), handler.require(false, ""), handler.csrf(), handler.logout)
	group.GET("/me", handler.require(false, ""), handler.me)
	group.POST("/recovery-codes/regenerate", handler.browserMutation(), handler.require(true, "account.manage"), handler.csrf(), handler.regenerate)
}

func (handler *AdminAuthHandler) bff() gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := []byte(c.GetHeader(bffHeader))
		if len(provided) != len(handler.bffKey) || subtle.ConstantTimeCompare(provided, handler.bffKey) != 1 {
			handler.problem(c, http.StatusNotFound, "ROUTE_NOT_FOUND", "Route tidak ditemukan.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (handler *AdminAuthHandler) login(c *gin.Context) {
	var request AdminLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handler.problem(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Permintaan tidak valid.")
		return
	}
	key := handler.rateLimitKey("admin-login", c.ClientIP()+"\x00"+strings.ToLower(strings.TrimSpace(string(request.Email))))
	if allowed, retry := handler.limiter.Allow(c, key, 5, 15*time.Minute); !allowed {
		c.Header("Retry-After", strconv.Itoa(max(1, int(retry.Seconds()))))
		handler.problem(c, http.StatusTooManyRequests, "RATE_LIMITED", "Coba kembali setelah waktu tunggu berakhir.")
		return
	}
	result, err := handler.service.Login(c, string(request.Email), request.Password, request.TotpCode, request.RecoveryCode, c.Request.UserAgent())
	if err != nil {
		handler.problem(c, http.StatusUnauthorized, "AUTHENTICATION_FAILED", "Email, password, atau faktor kedua tidak valid.")
		return
	}
	handler.setSessionCookies(c, result.Tokens)
	status := Authenticated
	if result.Principal.Level != authapp.LevelMFA {
		status = MfaSetupRequired
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, AdminAuthResponse{Data: AdminAuthData{ExpiresAt: result.Tokens.AccessExpires, Status: status, User: adminUser(result.Principal.User)}, Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) setupTOTP(c *gin.Context) {
	principal := currentPrincipal(c)
	setup, err := handler.service.SetupTOTP(c, principal)
	if err != nil {
		handler.authError(c, err)
		return
	}
	label := url.PathEscape("Tauco Cap Badak:" + principal.Email)
	uri := "otpauth://totp/" + label + "?secret=" + url.QueryEscape(setup.Secret) + "&issuer=" + url.QueryEscape("Tauco Cap Badak") + "&algorithm=SHA1&digits=6&period=30"
	c.JSON(http.StatusOK, AdminTotpSetupResponse{Data: AdminTotpSetupData{ExpiresAt: setup.ExpiresAt, ManualKey: setup.Secret, OtpauthUri: uri}, Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) enableTOTP(c *gin.Context) {
	var request AdminTotpCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handler.problem(c, 422, "VALIDATION_FAILED", "Kode TOTP wajib diisi.")
		return
	}
	result, err := handler.service.EnableTOTP(c, currentPrincipal(c), request.TotpCode)
	if err != nil {
		handler.authError(c, err)
		return
	}
	handler.setAccessCookie(c, result.Access, int(time.Until(result.AccessExpires).Seconds()))
	c.JSON(http.StatusOK, AdminRecoveryCodesResponse{Data: AdminRecoveryCodesData{Codes: result.Codes}, Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) refresh(c *gin.Context) {
	rawRefresh, refreshErr := c.Cookie(refreshCookie)
	csrf, csrfErr := c.Cookie(csrfCookie)
	if refreshErr != nil || csrfErr != nil || subtle.ConstantTimeCompare([]byte(csrf), []byte(c.GetHeader("X-CSRF-Token"))) != 1 {
		handler.problem(c, http.StatusUnauthorized, "UNAUTHORIZED", "Session tidak valid.")
		return
	}
	result, err := handler.service.Refresh(c, rawRefresh, csrf)
	if err != nil {
		handler.problem(c, http.StatusUnauthorized, "UNAUTHORIZED", "Session tidak valid.")
		return
	}
	handler.setSessionCookies(c, result.Tokens)
	c.JSON(http.StatusOK, AdminAuthResponse{Data: AdminAuthData{ExpiresAt: result.Tokens.AccessExpires, Status: Authenticated, User: adminUser(result.Principal.User)}, Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) logout(c *gin.Context) {
	if err := handler.service.Logout(c, currentPrincipal(c)); err != nil {
		handler.authError(c, err)
		return
	}
	handler.clearCookies(c)
	c.Status(http.StatusNoContent)
}

func (handler *AdminAuthHandler) me(c *gin.Context) {
	c.JSON(http.StatusOK, AdminUserResponse{Data: adminUser(currentPrincipal(c).User), Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) regenerate(c *gin.Context) {
	var request AdminTotpCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		handler.problem(c, 422, "VALIDATION_FAILED", "Kode TOTP wajib diisi.")
		return
	}
	codes, err := handler.service.RegenerateRecoveryCodes(c, currentPrincipal(c), request.TotpCode)
	if err != nil {
		handler.authError(c, err)
		return
	}
	c.JSON(http.StatusOK, AdminRecoveryCodesResponse{Data: AdminRecoveryCodesData{Codes: codes}, Meta: handler.responseMeta(c)})
}

func (handler *AdminAuthHandler) require(requireMFA bool, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(accessCookie)
		if err != nil {
			handler.problem(c, 401, "UNAUTHORIZED", "Authentication diperlukan.")
			c.Abort()
			return
		}
		principal, err := handler.service.ValidateAccess(c, raw, requireMFA)
		if err != nil {
			handler.problem(c, 401, "UNAUTHORIZED", "Authentication diperlukan.")
			c.Abort()
			return
		}
		if allowed, retry := handler.limiter.Allow(c, handler.rateLimitKey("admin", principal.ID.String()), 120, time.Minute); !allowed {
			c.Header("Retry-After", strconv.Itoa(max(1, int(retry.Seconds()))))
			handler.problem(c, 429, "RATE_LIMITED", "Coba kembali setelah waktu tunggu berakhir.")
			c.Abort()
			return
		}
		if permission != "" && !contains(principal.Permissions, permission) {
			handler.problem(c, 403, "FORBIDDEN", "Permission tidak mencukupi.")
			c.Abort()
			return
		}
		c.Set(principalKey, principal)
		c.Next()
	}
}

func (handler *AdminAuthHandler) csrf() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := currentPrincipal(c)
		cookie, err := c.Cookie(csrfCookie)
		header := c.GetHeader("X-CSRF-Token")
		if err != nil || cookie == "" || subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 || subtle.ConstantTimeCompare([]byte(authdomain.SHA256Hex(cookie)), []byte(principal.CSRFTokenHash)) != 1 {
			handler.problem(c, 403, "CSRF_REJECTED", "CSRF token tidak valid.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (handler *AdminAuthHandler) browserMutation() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
			handler.problem(c, 403, "CROSS_SITE_REJECTED", "Cross-site request ditolak.")
			c.Abort()
			return
		}
		origin := c.GetHeader("Origin")
		if origin == "" {
			if referer, err := url.Parse(c.GetHeader("Referer")); err == nil && referer.Scheme != "" && referer.Host != "" {
				origin = referer.Scheme + "://" + referer.Host
			}
		}
		if _, allowed := handler.origins[origin]; !allowed {
			handler.problem(c, 403, "ORIGIN_DENIED", "Origin permintaan tidak diizinkan.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (handler *AdminAuthHandler) setSessionCookies(c *gin.Context, tokens authapp.SessionTokens) {
	handler.setAccessCookie(c, tokens.Access, int(time.Until(tokens.AccessExpires).Seconds()))
	handler.setCookie(c, refreshCookie, tokens.Refresh, int(time.Until(tokens.SessionExpires).Seconds()), true)
	handler.setCookie(c, csrfCookie, tokens.CSRF, int(time.Until(tokens.SessionExpires).Seconds()), false)
}

func (handler *AdminAuthHandler) setAccessCookie(c *gin.Context, value string, maxAge int) {
	handler.setCookie(c, accessCookie, value, maxAge, true)
}
func (handler *AdminAuthHandler) setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	if maxAge >= 0 {
		maxAge = max(1, maxAge)
	}
	c.SetCookie(name, value, maxAge, "/", "", handler.secure, httpOnly)
}
func (handler *AdminAuthHandler) clearCookies(c *gin.Context) {
	for _, cookie := range []struct {
		name     string
		httpOnly bool
	}{{accessCookie, true}, {refreshCookie, true}, {csrfCookie, false}} {
		handler.setCookie(c, cookie.name, "", -1, cookie.httpOnly)
	}
}

func (handler *AdminAuthHandler) rateLimitKey(prefix, value string) string {
	mac := hmac.New(sha256.New, handler.rateKey)
	_, _ = mac.Write([]byte(value))
	return prefix + ":" + hex.EncodeToString(mac.Sum(nil))
}
func (handler *AdminAuthHandler) problem(c *gin.Context, status int, code, detail string) {
	c.Header("Content-Type", "application/problem+json")
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{
		"type":  "urn:tauco-cap-badak:problem:" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		"title": http.StatusText(status), "status": status, "detail": detail,
		"instance": c.Request.URL.EscapedPath(), "code": code,
		"requestId": handler.requestID(c),
	})
}
func (handler *AdminAuthHandler) authError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, authapp.ErrAuthentication), errors.Is(err, authapp.ErrUnauthorized):
		handler.problem(c, 401, "AUTHENTICATION_FAILED", "Authentication tidak valid.")
	case errors.Is(err, authapp.ErrConflict):
		handler.problem(c, 409, "AUTH_STATE_CONFLICT", "Status authentication telah berubah.")
	default:
		handler.problem(c, 500, "INTERNAL_ERROR", "Permintaan tidak dapat diproses.")
	}
}
func (handler *AdminAuthHandler) responseMeta(c *gin.Context) ResponseMeta {
	return ResponseMeta{ApiVersion: ResponseMetaApiVersionV1, RequestId: handler.requestID(c)}
}
func currentPrincipal(c *gin.Context) authapp.Principal {
	value, _ := c.Get(principalKey)
	principal, _ := value.(authapp.Principal)
	return principal
}
func adminUser(user authapp.User) AdminUser {
	return AdminUser{Id: user.ID, Email: openapi_types.Email(user.Email), Status: AdminUserStatus(user.Status), MfaEnabled: user.MFAEnabled, Roles: append([]string(nil), user.Roles...), Permissions: append([]string(nil), user.Permissions...)}
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

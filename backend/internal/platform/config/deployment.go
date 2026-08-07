package config

import (
	"net/url"
	"strings"
)

const MaxDeploymentEnvironmentBytes = 3072

type DeploymentConfig struct {
	AdminRemoteEnabled bool
	ContactAPIEnabled  bool
	FormSyncEnabled    bool
	AdminCookieSecure  bool
	AdminBFFSecret     []byte
	AdminOrigins       []string
	CORSOrigins        []string
	MediaStorageDriver string
	MediaS3Endpoint    string
	MediaS3Region      string
	MediaS3Bucket      string
	MediaS3Prefix      string
	MediaS3AccessKeyID string
	MediaS3SecretKey   string
}

func LoadDeployment(lookup LookupEnv, environment Environment) (DeploymentConfig, error) {
	if lookup == nil {
		return DeploymentConfig{}, &ValidationError{Problems: []Problem{{Field: "environment", Reason: "lookup function is required"}}}
	}
	reader := envReader{lookup: lookup}
	adminEnabled := reader.strictBool("ADMIN_REMOTE_ENABLED", false)
	contactEnabled := reader.strictBool("CONTACT_API_ENABLED", false)
	formSyncEnabled := reader.strictBool("FORM_SYNC_ENABLED", false)
	secureCookies := reader.strictBool("ADMIN_COOKIE_SECURE", false)
	corsOrigins := reader.origins("CORS_ALLOWED_ORIGINS", environment.IsProductionLike())
	adminOrigins := reader.origins("ADMIN_ALLOWED_ORIGINS", environment.IsProductionLike())
	mediaDriver, _ := reader.string("MEDIA_STORAGE_DRIVER", "local")
	mediaDriver = strings.ToLower(mediaDriver)
	if mediaDriver != "local" && mediaDriver != "s3" {
		reader.add("MEDIA_STORAGE_DRIVER", "must be either local or s3")
	}
	mediaS3Endpoint, _ := reader.raw("MEDIA_S3_ENDPOINT")
	mediaS3Region, _ := reader.raw("MEDIA_S3_REGION")
	mediaS3Bucket, _ := reader.raw("MEDIA_S3_BUCKET")
	mediaS3Prefix, _ := reader.raw("MEDIA_S3_PREFIX")
	mediaS3AccessKey, _ := reader.raw("MEDIA_S3_ACCESS_KEY_ID")
	mediaS3SecretKey, _ := reader.raw("MEDIA_S3_SECRET_ACCESS_KEY")
	if mediaDriver == "s3" {
		parsedEndpoint, endpointErr := url.Parse(mediaS3Endpoint)
		if endpointErr != nil || parsedEndpoint.Host == "" || parsedEndpoint.User != nil ||
			parsedEndpoint.RawQuery != "" || parsedEndpoint.Fragment != "" ||
			(parsedEndpoint.Scheme != "http" && parsedEndpoint.Scheme != "https") ||
			(environment.IsProductionLike() && parsedEndpoint.Scheme != "https") {
			reader.add("MEDIA_S3_ENDPOINT", "must be an absolute "+map[bool]string{true: "HTTPS", false: "HTTP(S)"}[environment.IsProductionLike()]+" URL without credentials, query, or fragment")
		}
		for name, value := range map[string]string{
			"MEDIA_S3_REGION":            mediaS3Region,
			"MEDIA_S3_BUCKET":            mediaS3Bucket,
			"MEDIA_S3_ACCESS_KEY_ID":     mediaS3AccessKey,
			"MEDIA_S3_SECRET_ACCESS_KEY": mediaS3SecretKey,
		} {
			if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) {
				reader.add(name, "is required without surrounding whitespace when MEDIA_STORAGE_DRIVER is s3")
			}
		}
		if mediaS3Prefix != strings.Trim(mediaS3Prefix, "/") || strings.Contains(mediaS3Prefix, "..") || strings.Contains(mediaS3Prefix, "\\") {
			reader.add("MEDIA_S3_PREFIX", "must be an optional safe object-key prefix")
		}
	}

	bffSecret, _ := reader.raw("ADMIN_BFF_SHARED_SECRET")
	if adminEnabled {
		if value, ok := reader.raw("ADMIN_DATABASE_URL"); !ok || value == "" {
			reader.add("ADMIN_DATABASE_URL", "is required when ADMIN_REMOTE_ENABLED is true")
		}
		if len([]byte(bffSecret)) < 32 {
			reader.add("ADMIN_BFF_SHARED_SECRET", "must contain at least 32 bytes when ADMIN_REMOTE_ENABLED is true")
		}
		if len(adminOrigins) == 0 {
			reader.add("ADMIN_ALLOWED_ORIGINS", "must contain at least one exact origin when ADMIN_REMOTE_ENABLED is true")
		}
		allowed := make(map[string]struct{}, len(corsOrigins))
		for _, origin := range corsOrigins {
			allowed[origin] = struct{}{}
		}
		for _, origin := range adminOrigins {
			if _, ok := allowed[origin]; !ok {
				reader.add("ADMIN_ALLOWED_ORIGINS", "must be a subset of CORS_ALLOWED_ORIGINS")
				break
			}
		}
	}

	if environment.IsProductionLike() {
		if len(corsOrigins) == 0 {
			reader.add("CORS_ALLOWED_ORIGINS", "must contain at least one exact HTTPS origin")
		}
		if adminEnabled && !secureCookies {
			reader.add("ADMIN_COOKIE_SECURE", "must be true when remote admin is enabled")
		}
		if adminEnabled && insecureExampleSecret(bffSecret) {
			reader.add("ADMIN_BFF_SHARED_SECRET", "must not use an example or local development value")
		}
		if contactEnabled {
			reader.add("CONTACT_API_ENABLED", "must remain false; production contact transport is Netlify Forms")
		}
		if mediaDriver != "s3" {
			reader.add("MEDIA_STORAGE_DRIVER", "must be s3 in staging or production")
		}
		for _, name := range []string{"CURSOR_HMAC_SECRET", "RATE_LIMIT_HMAC_SECRET", "METRICS_BEARER_TOKEN"} {
			reader.productionSecret(name)
		}
		if contactEnabled {
			reader.productionSecret("CONTACT_HMAC_SECRET")
		}
		redisURL, found := reader.raw("REDIS_URL")
		parsedRedis, redisErr := url.Parse(redisURL)
		if !found || redisErr != nil || parsedRedis.Scheme != "rediss" || parsedRedis.Host == "" || parsedRedis.Fragment != "" {
			reader.add("REDIS_URL", "must be a TLS rediss URL in staging or production")
		}
		if used := deploymentEnvironmentBytes(lookup); used > MaxDeploymentEnvironmentBytes {
			reader.add("environment", "application variables exceed the 3072-byte deployment budget")
		}
	}

	if len(reader.problems) > 0 {
		return DeploymentConfig{}, &ValidationError{Problems: reader.problems}
	}
	return DeploymentConfig{
		AdminRemoteEnabled: adminEnabled, ContactAPIEnabled: contactEnabled,
		FormSyncEnabled: formSyncEnabled, AdminCookieSecure: secureCookies,
		AdminBFFSecret: []byte(bffSecret), AdminOrigins: adminOrigins,
		CORSOrigins: corsOrigins, MediaStorageDriver: mediaDriver,
		MediaS3Endpoint: mediaS3Endpoint, MediaS3Region: mediaS3Region,
		MediaS3Bucket: mediaS3Bucket, MediaS3Prefix: mediaS3Prefix,
		MediaS3AccessKeyID: mediaS3AccessKey, MediaS3SecretKey: mediaS3SecretKey,
	}, nil
}

func (r *envReader) productionSecret(name string) {
	value, found := r.lookup(name)
	if !found || value == "" || value != strings.TrimSpace(value) || len([]byte(value)) < 32 {
		r.add(name, "must contain at least 32 bytes without surrounding whitespace")
		return
	}
	if insecureExampleSecret(value) {
		r.add(name, "must not use an example or local development value")
	}
}

func (r *envReader) strictBool(name string, fallback bool) bool {
	value, found := r.raw(name)
	if !found {
		return fallback
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	default:
		r.add(name, "must be either true or false")
		return fallback
	}
}

func (r *envReader) origins(name string, requireHTTPS bool) []string {
	value, found := r.raw(name)
	if !found || value == "" {
		return nil
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		parsed, err := url.Parse(strings.TrimSpace(item))
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") || (requireHTTPS && parsed.Scheme != "https") {
			r.add(name, "must contain comma-separated exact "+map[bool]string{true: "HTTPS", false: "HTTP(S)"}[requireHTTPS]+" origins")
			continue
		}
		origin := parsed.Scheme + "://" + parsed.Host
		if _, duplicate := seen[origin]; duplicate {
			r.add(name, "must not contain duplicate origins")
			continue
		}
		seen[origin] = struct{}{}
		result = append(result, origin)
	}
	return result
}

func insecureExampleSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "" || strings.Contains(normalized, "change-me") || strings.Contains(normalized, "example") || strings.Contains(normalized, "local-")
}

var deploymentEnvironmentKeys = []string{
	"APP_ENV", "SERVICE_NAME", "HTTP_HOST", "PORT", "LOG_LEVEL", "LOG_FORMAT",
	"HTTP_READ_HEADER_TIMEOUT", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT", "HTTP_IDLE_TIMEOUT", "SHUTDOWN_GRACE_PERIOD",
	"DATABASE_URL", "ADMIN_DATABASE_URL", "DATABASE_DEPLOYMENT_PROFILE",
	"DATABASE_MAX_OPEN_CONNS", "DATABASE_MAX_IDLE_CONNS", "DATABASE_CONN_MAX_LIFETIME", "DATABASE_CONN_MAX_IDLE_TIME", "REDIS_URL",
	"CURSOR_HMAC_SECRET", "RATE_LIMIT_HMAC_SECRET", "METRICS_BEARER_TOKEN",
	"CONTACT_HMAC_SECRET", "JWT_ED25519_PRIVATE_KEY_BASE64", "JWT_KEY_ID",
	"JWT_ISSUER", "JWT_AUDIENCE", "MFA_ENCRYPTION_KEY", "MFA_ENCRYPTION_KEY_ID",
	"RECOVERY_CODE_HMAC_SECRET", "ADMIN_ALLOWED_ORIGINS", "CORS_ALLOWED_ORIGINS",
	"TRUSTED_PROXY_CIDRS", "ADMIN_BFF_SHARED_SECRET", "ADMIN_REMOTE_ENABLED",
	"CONTACT_API_ENABLED", "FORM_SYNC_ENABLED", "MEDIA_STORAGE_DRIVER",
	"MEDIA_S3_ENDPOINT", "MEDIA_S3_REGION", "MEDIA_S3_BUCKET", "MEDIA_S3_PREFIX", "MEDIA_S3_ACCESS_KEY_ID", "MEDIA_S3_SECRET_ACCESS_KEY",
	"NEXT_PUBLIC_SITE_URL", "GOOGLE_SITE_VERIFICATION", "ADMIN_CMS_ENABLED", "ADMIN_API_ORIGIN", "CONTENT_SOURCE",
	"REVALIDATION_SECRET", "FORM_SYNC_SECRET",
}

func deploymentEnvironmentBytes(lookup LookupEnv) int {
	total := 0
	for _, key := range deploymentEnvironmentKeys {
		if value, found := lookup(key); found {
			total += len(key) + 2 + len(value)
		}
	}
	return total
}

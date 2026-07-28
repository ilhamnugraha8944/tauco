// Package config loads and validates the backend process configuration.
//
// Configuration is read exclusively from environment variables. Loading a
// .env file is deliberately left to the process supervisor or local developer
// tooling so production behavior cannot depend on files in the container.
package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPHost            = "127.0.0.1"
	defaultHTTPPort            = 8080
	defaultReadHeaderTimeout   = 5 * time.Second
	defaultReadTimeout         = 15 * time.Second
	defaultWriteTimeout        = 15 * time.Second
	defaultIdleTimeout         = 60 * time.Second
	defaultShutdownGracePeriod = 10 * time.Second
	defaultServiceName         = "tauco-api"
)

var (
	serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)
	hostLabelPattern   = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
)

// Environment identifies the runtime deployment class.
type Environment string

const (
	EnvironmentLocal      Environment = "local"
	EnvironmentTest       Environment = "test"
	EnvironmentStaging    Environment = "staging"
	EnvironmentProduction Environment = "production"
)

// IsProductionLike reports whether the process is running in a remotely
// deployed environment. These environments use stricter defaults and require
// explicit network binding configuration.
func (e Environment) IsProductionLike() bool {
	return e == EnvironmentStaging || e == EnvironmentProduction
}

// Config contains process-level settings that are safe to pass through the
// composition root. Secret-bearing dependency configuration belongs in its
// own typed section and must never be formatted as a whole value.
type Config struct {
	Environment Environment
	HTTP        HTTPConfig
	Log         LogConfig
}

// HTTPConfig controls the public HTTP server and graceful shutdown behavior.
type HTTPConfig struct {
	Host                string
	Port                int
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownGracePeriod time.Duration
}

// Address returns a valid host:port listen address, including for IPv6 hosts.
func (c HTTPConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// LogConfig controls the structured application logger.
type LogConfig struct {
	Level   string
	Format  string
	Service string
}

// LookupEnv matches os.LookupEnv and permits deterministic unit testing without
// mutating process-wide environment state.
type LookupEnv func(key string) (value string, found bool)

// Problem identifies a single invalid environment-backed configuration field.
// Values are intentionally omitted so errors are always safe to log.
type Problem struct {
	Field  string
	Reason string
}

// ValidationError contains all configuration issues found during one load.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Problems) == 0 {
		return "invalid configuration"
	}

	parts := make([]string, 0, len(e.Problems))
	for _, problem := range e.Problems {
		parts = append(parts, problem.Field+": "+problem.Reason)
	}

	return "invalid configuration: " + strings.Join(parts, "; ")
}

// Has reports whether a field is represented in the validation failure.
func (e *ValidationError) Has(field string) bool {
	if e == nil {
		return false
	}
	for _, problem := range e.Problems {
		if problem.Field == field {
			return true
		}
	}
	return false
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return LoadWithLookup(os.LookupEnv)
}

// LoadWithLookup reads, parses, and validates configuration through lookup.
// Error messages never include rejected environment values.
func LoadWithLookup(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		return Config{}, &ValidationError{
			Problems: []Problem{{Field: "environment", Reason: "lookup function is required"}},
		}
	}

	reader := envReader{lookup: lookup}
	environment := reader.environment()

	defaultLogLevel := "debug"
	defaultLogFormat := "console"
	if environment == EnvironmentTest {
		defaultLogLevel = "error"
	}
	if environment.IsProductionLike() {
		defaultLogLevel = "info"
		defaultLogFormat = "json"
	}

	host, hostSet := reader.string("HTTP_HOST", defaultHTTPHost)
	port, portSet := reader.port()
	service, serviceSet := reader.string("SERVICE_NAME", defaultServiceName)

	cfg := Config{
		Environment: environment,
		HTTP: HTTPConfig{
			Host:                host,
			Port:                port,
			ReadHeaderTimeout:   reader.duration("HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout),
			ReadTimeout:         reader.duration("HTTP_READ_TIMEOUT", defaultReadTimeout),
			WriteTimeout:        reader.duration("HTTP_WRITE_TIMEOUT", defaultWriteTimeout),
			IdleTimeout:         reader.duration("HTTP_IDLE_TIMEOUT", defaultIdleTimeout),
			ShutdownGracePeriod: reader.duration("SHUTDOWN_GRACE_PERIOD", defaultShutdownGracePeriod),
		},
		Log: LogConfig{
			Level:   reader.normalizedString("LOG_LEVEL", defaultLogLevel),
			Format:  reader.normalizedString("LOG_FORMAT", defaultLogFormat),
			Service: service,
		},
	}

	if environment.IsProductionLike() {
		if !hostSet {
			reader.add("HTTP_HOST", "must be explicitly configured for staging or production")
		}
		if !portSet {
			reader.add("PORT", "must be explicitly configured for staging or production")
		}
		if !serviceSet {
			reader.add("SERVICE_NAME", "must be explicitly configured for staging or production")
		}
	}

	reader.problems = append(reader.problems, cfg.validationProblems()...)
	if len(reader.problems) > 0 {
		return Config{}, &ValidationError{Problems: reader.problems}
	}

	return cfg, nil
}

// Validate verifies a programmatically constructed configuration.
func (c Config) Validate() error {
	problems := c.validationProblems()
	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: problems}
}

func (c Config) validationProblems() []Problem {
	var problems []Problem

	switch c.Environment {
	case EnvironmentLocal, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
	default:
		problems = append(problems, Problem{
			Field:  "APP_ENV",
			Reason: "must be one of local, test, staging, or production",
		})
	}

	if reason := invalidHostReason(c.HTTP.Host); reason != "" {
		problems = append(problems, Problem{Field: "HTTP_HOST", Reason: reason})
	}
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		problems = append(problems, Problem{Field: "PORT", Reason: "must be an integer between 1 and 65535"})
	}

	problems = append(problems,
		durationProblem("HTTP_READ_HEADER_TIMEOUT", c.HTTP.ReadHeaderTimeout, 30*time.Second),
		durationProblem("HTTP_READ_TIMEOUT", c.HTTP.ReadTimeout, 2*time.Minute),
		durationProblem("HTTP_WRITE_TIMEOUT", c.HTTP.WriteTimeout, 5*time.Minute),
		durationProblem("HTTP_IDLE_TIMEOUT", c.HTTP.IdleTimeout, 10*time.Minute),
		durationProblem("SHUTDOWN_GRACE_PERIOD", c.HTTP.ShutdownGracePeriod, time.Minute),
	)
	problems = compactProblems(problems)

	if c.HTTP.ReadTimeout > 0 &&
		c.HTTP.ReadHeaderTimeout > 0 &&
		c.HTTP.ReadTimeout < c.HTTP.ReadHeaderTimeout {
		problems = append(problems, Problem{
			Field:  "HTTP_READ_TIMEOUT",
			Reason: "must be greater than or equal to HTTP_READ_HEADER_TIMEOUT",
		})
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, Problem{
			Field:  "LOG_LEVEL",
			Reason: "must be one of debug, info, warn, or error",
		})
	}

	switch c.Log.Format {
	case "json", "console":
	default:
		problems = append(problems, Problem{
			Field:  "LOG_FORMAT",
			Reason: "must be either json or console",
		})
	}

	if !serviceNamePattern.MatchString(c.Log.Service) {
		problems = append(problems, Problem{
			Field:  "SERVICE_NAME",
			Reason: "must be 3-63 lowercase letters, digits, or hyphens and start with a letter",
		})
	}

	if c.Environment.IsProductionLike() && c.Log.Format != "json" {
		problems = append(problems, Problem{
			Field:  "LOG_FORMAT",
			Reason: "must be json in staging or production",
		})
	}
	if c.Environment == EnvironmentProduction && c.Log.Level == "debug" {
		problems = append(problems, Problem{
			Field:  "LOG_LEVEL",
			Reason: "debug logging is not allowed in production",
		})
	}

	return problems
}

type envReader struct {
	lookup   LookupEnv
	problems []Problem
}

func (r *envReader) add(field, reason string) {
	r.problems = append(r.problems, Problem{Field: field, Reason: reason})
}

func (r *envReader) raw(name string) (string, bool) {
	value, found := r.lookup(name)
	if !found {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func (r *envReader) string(name, fallback string) (string, bool) {
	value, found := r.raw(name)
	if !found {
		return fallback, false
	}
	if value == "" {
		r.add(name, "must not be empty when configured")
	}
	return value, true
}

func (r *envReader) normalizedString(name, fallback string) string {
	value, found := r.string(name, fallback)
	if !found {
		return fallback
	}
	return strings.ToLower(value)
}

func (r *envReader) environment() Environment {
	value, found := r.raw("APP_ENV")
	if !found {
		return EnvironmentLocal
	}

	switch Environment(strings.ToLower(value)) {
	case EnvironmentLocal:
		return EnvironmentLocal
	case EnvironmentTest:
		return EnvironmentTest
	case EnvironmentStaging:
		return EnvironmentStaging
	case EnvironmentProduction:
		return EnvironmentProduction
	default:
		r.add("APP_ENV", "must be one of local, test, staging, or production")
		return Environment(value)
	}
}

func (r *envReader) port() (int, bool) {
	portValue, portSet := r.raw("PORT")
	httpPortValue, httpPortSet := r.raw("HTTP_PORT")

	if portSet && httpPortSet && portValue != httpPortValue {
		r.add("PORT", "must match HTTP_PORT when both variables are configured")
	}

	value := strconv.Itoa(defaultHTTPPort)
	switch {
	case portSet:
		value = portValue
	case httpPortSet:
		value = httpPortValue
	}

	port, err := strconv.Atoi(value)
	if err != nil {
		r.add("PORT", "must be an integer between 1 and 65535")
		return 0, portSet || httpPortSet
	}
	return port, portSet || httpPortSet
}

func (r *envReader) duration(name string, fallback time.Duration) time.Duration {
	value, found := r.raw(name)
	if !found {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		r.add(name, "must be a valid Go duration such as 5s or 1m")
		return 0
	}
	return duration
}

func durationProblem(field string, value, maximum time.Duration) Problem {
	if value <= 0 || value > maximum {
		return Problem{
			Field:  field,
			Reason: fmt.Sprintf("must be greater than zero and no more than %s", maximum),
		}
	}
	return Problem{}
}

func compactProblems(problems []Problem) []Problem {
	result := problems[:0]
	for _, problem := range problems {
		if problem.Field != "" {
			result = append(result, problem)
		}
	}
	return result
}

func invalidHostReason(host string) string {
	if host == "" {
		return "must not be empty"
	}
	if len(host) > 253 {
		return "must be a valid hostname or IP address without a port"
	}
	if strings.ContainsAny(host, `/\?#@`) ||
		strings.Contains(host, "://") ||
		strings.ContainsAny(host, " \t\r\n") {
		return "must be a valid hostname or IP address without a port"
	}
	if host == "*" {
		return "wildcard host is not allowed; configure an explicit bind address"
	}
	if net.ParseIP(host) != nil || strings.EqualFold(host, "localhost") {
		return ""
	}

	trimmed := strings.TrimSuffix(host, ".")
	if trimmed == "" {
		return "must be a valid hostname or IP address without a port"
	}
	for _, label := range strings.Split(trimmed, ".") {
		if !hostLabelPattern.MatchString(label) {
			return "must be a valid hostname or IP address without a port"
		}
	}
	return ""
}

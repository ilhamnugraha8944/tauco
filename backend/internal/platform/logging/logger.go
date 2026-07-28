// Package logging provides a Zap-backed structured logger with a deliberately
// small field allowlist. Callers should log static event messages and identifiers
// only; request bodies, headers, configuration values, and contact PII are never
// valid log fields.
package logging

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps Zap so application code cannot casually attach arbitrary keyed
// strings. Use the typed field constructors below.
type Logger struct {
	core *zap.Logger
}

// Field is a logging field created by an allowlisted constructor.
type Field struct {
	value zap.Field
}

// New builds a stdout structured logger. It does not add caller paths or
// automatic stack traces because those can expose local paths or unchecked
// error text.
func New(logConfig config.LogConfig, environment config.Environment) (*Logger, error) {
	level, err := parseLevel(logConfig.Level)
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	var encoder zapcore.Encoder
	switch logConfig.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("create logger: unsupported log format")
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)
	return newLogger(core, logConfig.Service, environment), nil
}

func newLogger(core zapcore.Core, service string, environment config.Environment) *Logger {
	base := zap.New(core).With(
		zap.String("service", safeIdentifier(service, 63)),
		zap.String("environment", safeIdentifier(string(environment), 16)),
	)
	return &Logger{core: base}
}

func parseLevel(value string) (zapcore.Level, error) {
	switch value {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, errors.New("create logger: unsupported log level")
	}
}

// Named creates a child logger. The name is normalized to a bounded identifier.
func (l *Logger) Named(name string) *Logger {
	if l == nil || l.core == nil {
		return l
	}
	return &Logger{core: l.core.Named(safeIdentifier(name, 63))}
}

// With creates a child logger carrying allowlisted fields.
func (l *Logger) With(fields ...Field) *Logger {
	if l == nil || l.core == nil {
		return l
	}
	return &Logger{core: l.core.With(zapFields(fields)...)}
}

func (l *Logger) Debug(message string, fields ...Field) {
	if l != nil && l.core != nil {
		l.core.Debug(RedactString(message), zapFields(fields)...)
	}
}

func (l *Logger) Info(message string, fields ...Field) {
	if l != nil && l.core != nil {
		l.core.Info(RedactString(message), zapFields(fields)...)
	}
}

func (l *Logger) Warn(message string, fields ...Field) {
	if l != nil && l.core != nil {
		l.core.Warn(RedactString(message), zapFields(fields)...)
	}
}

func (l *Logger) Error(message string, fields ...Field) {
	if l != nil && l.core != nil {
		l.core.Error(RedactString(message), zapFields(fields)...)
	}
}

// Sync flushes buffered log entries.
func (l *Logger) Sync() error {
	if l == nil || l.core == nil {
		return nil
	}
	return l.core.Sync()
}

func zapFields(fields []Field) []zap.Field {
	result := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		if field.value.Type != zapcore.SkipType && field.value.Key != "" {
			result = append(result, field.value)
		}
	}
	return result
}

// RequestID records a generated request identifier.
func RequestID(value string) Field {
	return stringField("request_id", safeIdentifier(value, 128))
}

// TraceID records a W3C-compatible trace identifier.
func TraceID(value string) Field {
	return stringField("trace_id", safeIdentifier(value, 64))
}

// JobID records a durable background job identifier.
func JobID(value string) Field {
	return stringField("job_id", safeIdentifier(value, 128))
}

// WorkerID records a bounded worker identifier.
func WorkerID(value string) Field {
	return stringField("worker_id", safeIdentifier(value, 128))
}

// Route records a route template, never a raw URL or query string.
func Route(template string) Field {
	return stringField("route", truncateUTF8(RedactString(template), 256))
}

// Method records an HTTP method.
func Method(value string) Field {
	return stringField("method", safeIdentifier(value, 16))
}

// Status records an HTTP status code.
func Status(value int) Field {
	return Field{value: zap.Int("status", value)}
}

// Latency records request or job latency in milliseconds.
func Latency(value time.Duration) Field {
	return Field{value: zap.Int64("latency_ms", value.Milliseconds())}
}

// ErrorCode records a stable, non-sensitive application error code.
func ErrorCode(value string) Field {
	return stringField("stable_error_code", safeIdentifier(value, 128))
}

// Cause records redacted error text. Prefer ErrorCode whenever the text itself
// is unnecessary. A nil error produces no field.
func Cause(err error) Field {
	if err == nil {
		return Field{value: zap.Skip()}
	}
	return stringField("error", RedactString(err.Error()))
}

// Component records a static subsystem name.
func Component(value string) Field {
	return stringField("component", safeIdentifier(value, 63))
}

// Operation records a static operation name.
func Operation(value string) Field {
	return stringField("operation", safeIdentifier(value, 63))
}

// JobKind records a registered durable job kind.
func JobKind(value string) Field {
	return stringField("job_kind", safeIdentifier(value, 63))
}

// Attempt records a durable job attempt number.
func Attempt(value int) Field {
	return Field{value: zap.Int("attempt", value)}
}

// Result records a bounded static result such as success, retry, or dead.
func Result(value string) Field {
	return stringField("result", safeIdentifier(value, 32))
}

// CacheOutcome records a bounded static cache result such as hit, miss, or
// fallback. Cache keys and arbitrary slugs are intentionally unsupported.
func CacheOutcome(value string) Field {
	return stringField("cache_outcome", safeIdentifier(value, 32))
}

// Count records a non-negative aggregate count. It must never represent a
// customer identifier or other direct PII.
func Count(value int64) Field {
	return Field{value: zap.Int64("count", value)}
}

func stringField(key, value string) Field {
	return Field{value: zap.String(key, value)}
}

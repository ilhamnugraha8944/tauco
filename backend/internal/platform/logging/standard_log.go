package logging

import (
	"errors"
	"log"
	"strings"
)

// StandardLogger adapts the structured logger for standard-library components
// such as http.Server. Untrusted text is passed through the redaction layer.
func (l *Logger) StandardLogger(component string) *log.Logger {
	return log.New(
		standardLogWriter{logger: l.Named(component)},
		"",
		0,
	)
}

type standardLogWriter struct {
	logger *Logger
}

func (w standardLogWriter) Write(payload []byte) (int, error) {
	message := strings.TrimSpace(string(payload))
	if message != "" {
		w.logger.Error(
			"standard library error",
			Cause(errors.New(message)),
		)
	}
	return len(payload), nil
}

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ServerConfig defines defensive HTTP timeouts independently from the
// application config loader.
type ServerConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
	ErrorLog          *log.Logger
}

// DefaultServerConfig returns safe defaults for the Phase 1B public API.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Address:           ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// Server owns the standard HTTP server and its graceful-shutdown policy.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

// NewServer validates the HTTP policy and creates a server. It does not bind a
// port, which keeps construction deterministic and testable.
func NewServer(handler http.Handler, config ServerConfig) (*Server, error) {
	if handler == nil {
		return nil, errors.New("http handler is required")
	}
	if err := validateServerConfig(config); err != nil {
		return nil, err
	}

	return &Server{
		httpServer: &http.Server{
			Addr:              config.Address,
			Handler:           handler,
			ReadHeaderTimeout: config.ReadHeaderTimeout,
			ReadTimeout:       config.ReadTimeout,
			WriteTimeout:      config.WriteTimeout,
			IdleTimeout:       config.IdleTimeout,
			MaxHeaderBytes:    config.MaxHeaderBytes,
			ErrorLog:          config.ErrorLog,
		},
		shutdownTimeout: config.ShutdownTimeout,
	}, nil
}

// Run binds the configured address and serves until the context is canceled or
// the server fails.
func (s *Server) Run(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("http server is not initialized")
	}

	return s.run(ctx, s.httpServer.ListenAndServe)
}

// Serve is Run with a caller-provided listener, useful for tests and runtimes
// that create their own listener.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	if s == nil || s.httpServer == nil {
		return errors.New("http server is not initialized")
	}
	if listener == nil {
		return errors.New("http listener is required")
	}

	return s.run(ctx, func() error {
		return s.httpServer.Serve(listener)
	})
}

// Shutdown gracefully stops accepting new connections using the caller's
// deadline. Run and Serve normally call this automatically.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("http server is not initialized")
	}
	if ctx == nil {
		return errors.New("shutdown context is required")
	}

	return s.httpServer.Shutdown(ctx)
}

func (s *Server) run(ctx context.Context, serve func() error) error {
	if ctx == nil {
		return errors.New("server context is required")
	}

	serveError := make(chan error, 1)
	go func() {
		serveError <- normalizeServeError(serve())
	}()

	select {
	case err := <-serveError:
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			s.shutdownTimeout,
		)
		defer cancel()

		shutdownErr := s.httpServer.Shutdown(shutdownContext)
		if shutdownErr != nil {
			// Ensure Serve exits even when graceful shutdown reaches its
			// deadline. Preserve the graceful-shutdown error for the caller.
			_ = s.httpServer.Close()
		}

		serveErr := <-serveError
		if shutdownErr != nil {
			return fmt.Errorf("shutdown http server: %w", shutdownErr)
		}
		return serveErr
	}
}

func normalizeServeError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func validateServerConfig(config ServerConfig) error {
	switch {
	case strings.TrimSpace(config.Address) == "":
		return errors.New("http server address is required")
	case config.ReadHeaderTimeout <= 0:
		return errors.New("http read header timeout must be greater than zero")
	case config.ReadTimeout <= 0:
		return errors.New("http read timeout must be greater than zero")
	case config.ReadTimeout < config.ReadHeaderTimeout:
		return errors.New(
			"http read timeout must not be shorter than read header timeout",
		)
	case config.WriteTimeout <= 0:
		return errors.New("http write timeout must be greater than zero")
	case config.IdleTimeout <= 0:
		return errors.New("http idle timeout must be greater than zero")
	case config.ShutdownTimeout <= 0:
		return errors.New("http shutdown timeout must be greater than zero")
	case config.MaxHeaderBytes <= 0:
		return errors.New("http max header bytes must be greater than zero")
	default:
		return nil
	}
}

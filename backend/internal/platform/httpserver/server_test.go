package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewServerUsesDefensiveTimeouts(t *testing.T) {
	config := DefaultServerConfig()
	server, err := NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
	}), config)
	if err != nil {
		t.Fatalf("NewServer(): %v", err)
	}

	if server.httpServer.Addr != config.Address {
		t.Fatalf(
			"address = %q, want %q",
			server.httpServer.Addr,
			config.Address,
		)
	}
	if server.httpServer.ReadHeaderTimeout != config.ReadHeaderTimeout {
		t.Fatalf(
			"read header timeout = %v, want %v",
			server.httpServer.ReadHeaderTimeout,
			config.ReadHeaderTimeout,
		)
	}
	if server.httpServer.ReadTimeout != config.ReadTimeout {
		t.Fatalf(
			"read timeout = %v, want %v",
			server.httpServer.ReadTimeout,
			config.ReadTimeout,
		)
	}
	if server.httpServer.WriteTimeout != config.WriteTimeout {
		t.Fatalf(
			"write timeout = %v, want %v",
			server.httpServer.WriteTimeout,
			config.WriteTimeout,
		)
	}
	if server.httpServer.IdleTimeout != config.IdleTimeout {
		t.Fatalf(
			"idle timeout = %v, want %v",
			server.httpServer.IdleTimeout,
			config.IdleTimeout,
		)
	}
	if server.httpServer.MaxHeaderBytes != config.MaxHeaderBytes {
		t.Fatalf(
			"max header bytes = %d, want %d",
			server.httpServer.MaxHeaderBytes,
			config.MaxHeaderBytes,
		)
	}
}

func TestNewServerValidatesConfiguration(t *testing.T) {
	valid := DefaultServerConfig()
	tests := []struct {
		name   string
		mutate func(*ServerConfig)
	}{
		{
			name: "missing address",
			mutate: func(config *ServerConfig) {
				config.Address = " "
			},
		},
		{
			name: "missing read header timeout",
			mutate: func(config *ServerConfig) {
				config.ReadHeaderTimeout = 0
			},
		},
		{
			name: "read timeout shorter than header timeout",
			mutate: func(config *ServerConfig) {
				config.ReadTimeout = config.ReadHeaderTimeout - time.Millisecond
			},
		},
		{
			name: "missing write timeout",
			mutate: func(config *ServerConfig) {
				config.WriteTimeout = 0
			},
		},
		{
			name: "missing idle timeout",
			mutate: func(config *ServerConfig) {
				config.IdleTimeout = 0
			},
		},
		{
			name: "missing shutdown timeout",
			mutate: func(config *ServerConfig) {
				config.ShutdownTimeout = 0
			},
		},
		{
			name: "missing max header bytes",
			mutate: func(config *ServerConfig) {
				config.MaxHeaderBytes = 0
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)

			_, err := NewServer(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
				config,
			)
			if err == nil {
				t.Fatal("NewServer() error = nil, want validation error")
			}
		})
	}

	if _, err := NewServer(nil, valid); err == nil {
		t.Fatal("NewServer(nil) error = nil, want validation error")
	}
}

func TestServerStopsCleanlyWhenContextIsCanceled(t *testing.T) {
	config := DefaultServerConfig()
	config.Address = "127.0.0.1:0"
	config.ShutdownTimeout = time.Second

	server, err := NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusNoContent)
		}),
		config,
	)
	if err != nil {
		t.Fatalf("NewServer(): %v", err)
	}

	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runError := make(chan error, 1)
	go func() {
		runError <- server.Serve(ctx, listener)
	}()

	cancel()

	select {
	case err := <-runError:
		if err != nil {
			t.Fatalf("Serve() after cancellation: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop within graceful shutdown period")
	}
}

func TestServerDrainsInFlightRequestBeforeStopping(t *testing.T) {
	config := DefaultServerConfig()
	config.Address = "127.0.0.1:0"
	config.ShutdownTimeout = time.Second

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	server, err := NewServer(
		http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			close(requestStarted)
			<-releaseRequest
			response.WriteHeader(http.StatusNoContent)
		}),
		config,
	)
	if err != nil {
		t.Fatalf("NewServer(): %v", err)
	}

	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runError := make(chan error, 1)
	go func() {
		runError <- server.Serve(ctx, listener)
	}()

	clientError := make(chan error, 1)
	go func() {
		response, requestErr := (&http.Client{Timeout: 2 * time.Second}).Get(
			"http://" + listener.Addr().String(),
		)
		if requestErr == nil {
			requestErr = response.Body.Close()
		}
		clientError <- requestErr
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach handler")
	}

	cancel()
	select {
	case err := <-runError:
		t.Fatalf("Serve() returned before the request drained: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseRequest)
	select {
	case err := <-clientError:
		if err != nil {
			t.Fatalf("in-flight request failed during drain: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
	select {
	case err := <-runError:
		if err != nil {
			t.Fatalf("Serve() after drain: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after the request drained")
	}
}

func TestServerForcesCloseAfterShutdownDeadline(t *testing.T) {
	config := DefaultServerConfig()
	config.Address = "127.0.0.1:0"
	config.ShutdownTimeout = 25 * time.Millisecond

	requestStarted := make(chan struct{})
	requestCanceled := make(chan struct{})
	server, err := NewServer(
		http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(requestStarted)
			<-request.Context().Done()
			close(requestCanceled)
		}),
		config,
	)
	if err != nil {
		t.Fatalf("NewServer(): %v", err)
	}

	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runError := make(chan error, 1)
	go func() {
		runError <- server.Serve(ctx, listener)
	}()

	clientDone := make(chan struct{})
	go func() {
		response, _ := (&http.Client{Timeout: time.Second}).Get(
			"http://" + listener.Addr().String(),
		)
		if response != nil {
			_ = response.Body.Close()
		}
		close(clientDone)
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach handler")
	}

	cancel()
	select {
	case err := <-runError:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Serve() error = %v, want shutdown deadline exceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not force close after the shutdown deadline")
	}
	select {
	case <-requestCanceled:
	case <-time.After(time.Second):
		t.Fatal("forced close did not cancel the request context")
	}
	select {
	case <-clientDone:
	case <-time.After(time.Second):
		t.Fatal("client did not observe forced close")
	}
}

func TestServerReturnsListenerFailure(t *testing.T) {
	config := DefaultServerConfig()
	server, err := NewServer(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		config,
	)
	if err != nil {
		t.Fatalf("NewServer(): %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	err = server.Serve(context.Background(), listener)
	if err == nil {
		t.Fatal("Serve() error = nil, want listener failure")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want concrete listener failure", err)
	}
}

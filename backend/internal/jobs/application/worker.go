package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/jobs/domain"
)

const (
	DefaultBatchSize       = 10
	DefaultWorkerCount     = 2
	DefaultChannelCapacity = 20
	DefaultLease           = 120 * time.Second
	DefaultHeartbeat       = 30 * time.Second
	DefaultInitialBackoff  = 30 * time.Second
	DefaultMaxBackoff      = 30 * time.Minute
)

type Repository interface {
	Claim(context.Context, string, int, time.Duration) ([]domain.Job, error)
	Heartbeat(context.Context, string, string, time.Duration) error
	Succeed(context.Context, string, string) error
	Fail(context.Context, string, string, time.Time, string) error
	Release(context.Context, string, string) error
	Replay(context.Context, string, string) error
}

type Handler func(context.Context, domain.Job) error

type Config struct {
	Owner           string
	BatchSize       int
	Workers         int
	ChannelCapacity int
	Lease           time.Duration
	Heartbeat       time.Duration
	PollInterval    time.Duration
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	Jitter          func() float64
}

func DefaultConfig(owner string) Config {
	return Config{
		Owner: owner, BatchSize: DefaultBatchSize, Workers: DefaultWorkerCount,
		ChannelCapacity: DefaultChannelCapacity, Lease: DefaultLease,
		Heartbeat: DefaultHeartbeat, PollInterval: time.Second,
		InitialBackoff: DefaultInitialBackoff, MaxBackoff: DefaultMaxBackoff,
		Jitter: func() float64 { return rand.Float64() },
	}
}

type Worker struct {
	repository Repository
	handlers   map[string]Handler
	config     Config
	now        func() time.Time
}

func NewWorker(repository Repository, handlers map[string]Handler, config Config) (*Worker, error) {
	if repository == nil || len(handlers) == 0 {
		return nil, errors.New("worker requires a repository and handlers")
	}
	if config.Owner == "" || config.BatchSize < 1 || config.BatchSize > 100 ||
		config.Workers < 1 || config.Workers > 32 || config.ChannelCapacity < config.Workers ||
		config.Lease <= 0 || config.Heartbeat <= 0 || config.Heartbeat >= config.Lease ||
		config.PollInterval <= 0 || config.InitialBackoff <= 0 ||
		config.MaxBackoff < config.InitialBackoff {
		return nil, errors.New("invalid worker configuration")
	}
	if config.Jitter == nil {
		config.Jitter = func() float64 { return 0.5 }
	}
	cloned := make(map[string]Handler, len(handlers))
	for kind, handler := range handlers {
		if kind == "" || handler == nil {
			return nil, errors.New("invalid worker handler")
		}
		cloned[kind] = handler
	}
	return &Worker{repository: repository, handlers: cloned, config: config, now: time.Now}, nil
}

// Run polls until cancellation, then waits for the current bounded batch.
func (worker *Worker) Run(ctx context.Context) error {
	for {
		if err := worker.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		timer := time.NewTimer(worker.config.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

// RunOnce claims and drains one bounded batch.
func (worker *Worker) RunOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	jobs, err := worker.repository.Claim(
		ctx, worker.config.Owner, worker.config.BatchSize, worker.config.Lease,
	)
	if err != nil {
		return fmt.Errorf("claim jobs: %w", err)
	}
	queue := make(chan domain.Job, worker.config.ChannelCapacity)
	var wait sync.WaitGroup
	for range worker.config.Workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for job := range queue {
				worker.process(ctx, job)
			}
		}()
	}
	for _, job := range jobs {
		select {
		case queue <- job:
		case <-ctx.Done():
			_ = worker.repository.Release(context.Background(), job.ID, worker.config.Owner)
		}
	}
	close(queue)
	wait.Wait()
	return ctx.Err()
}

func (worker *Worker) process(ctx context.Context, job domain.Job) {
	if err := job.Validate(); err != nil {
		_ = worker.repository.Fail(ctx, job.ID, worker.config.Owner, worker.now(), "INVALID_JOB")
		return
	}
	handler, found := worker.handlers[job.Kind]
	if !found {
		_ = worker.repository.Fail(ctx, job.ID, worker.config.Owner, worker.now(), "HANDLER_NOT_FOUND")
		return
	}

	handlerContext, cancel := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(worker.config.Heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-handlerContext.Done():
				return
			case <-ticker.C:
				_ = worker.repository.Heartbeat(
					handlerContext, job.ID, worker.config.Owner, worker.config.Lease,
				)
			}
		}
	}()
	err := handler(handlerContext, job)
	cancel()
	<-heartbeatDone
	if ctx.Err() != nil {
		_ = worker.repository.Release(context.Background(), job.ID, worker.config.Owner)
		return
	}
	if err == nil {
		_ = worker.repository.Succeed(ctx, job.ID, worker.config.Owner)
		return
	}
	_ = worker.repository.Fail(
		ctx, job.ID, worker.config.Owner, worker.now().Add(worker.backoff(job.Attempts)),
		"HANDLER_FAILED",
	)
}

func (worker *Worker) backoff(attempt int) time.Duration {
	exponent := math.Max(0, float64(attempt-1))
	delay := float64(worker.config.InitialBackoff) * math.Pow(2, exponent)
	if delay > float64(worker.config.MaxBackoff) {
		delay = float64(worker.config.MaxBackoff)
	}
	// Uniform jitter in the frozen +/-20 percent range.
	factor := 0.8 + 0.4*worker.config.Jitter()
	return time.Duration(delay * factor)
}

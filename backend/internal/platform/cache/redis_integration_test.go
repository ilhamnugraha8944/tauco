package cache

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRedisAdapterIntegration(t *testing.T) {
	rawURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if rawURL == "" {
		t.Skip("set REDIS_URL to run Redis integration")
	}
	store, err := NewRedis(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Redis Ping() error = %v", err)
	}
	suffix := fmt.Sprintf("integration:%d", time.Now().UnixNano())
	if _, err := store.Get(ctx, suffix); err != ErrMiss {
		t.Fatalf("initial Get() error = %v", err)
	}
	if err := store.Set(ctx, suffix, []byte("cached"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if value, err := store.Get(ctx, suffix); err != nil || string(value) != "cached" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	if generation, err := store.Generation(ctx, suffix); err != nil || generation != 0 {
		t.Fatalf("Generation() = %d, %v", generation, err)
	}
	if err := store.Bump(ctx, suffix); err != nil {
		t.Fatal(err)
	}
	if generation, _ := store.Generation(ctx, suffix); generation != 1 {
		t.Fatalf("generation after bump = %d", generation)
	}
	for request := 1; request <= 3; request++ {
		allowed, _, err := store.Take(ctx, suffix, 2, time.Minute)
		if err != nil || allowed != (request <= 2) {
			t.Fatalf("Take(%d) allowed=%t err=%v", request, allowed, err)
		}
	}
	concurrentKey := suffix + ":concurrent"
	var allowedCount atomic.Int32
	var wait sync.WaitGroup
	for range 50 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			allowed, _, takeErr := store.Take(context.Background(), concurrentKey, 10, time.Minute)
			if takeErr != nil {
				t.Errorf("concurrent Take() error = %v", takeErr)
				return
			}
			if allowed {
				allowedCount.Add(1)
			}
		}()
	}
	wait.Wait()
	if allowedCount.Load() != 10 {
		t.Fatalf("atomic concurrent allowed count = %d, want 10", allowedCount.Load())
	}
	_ = store.Delete(ctx, suffix)
}

package app

import (
	"context"
	"testing"
	"time"
)

func TestBootstrap_fastAndValid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rt, err := Bootstrap(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	if rt.State.SessionID() == "" {
		t.Fatal("empty session id")
	}
	if rt.State.Cwd() == "" {
		t.Fatal("empty cwd")
	}
	if rt.RootCAs == nil {
		t.Fatal("nil cert pool")
	}
	if rt.GlobalConfigDir == "" {
		t.Fatal("empty global config dir")
	}
}

func TestBootstrap_underOneSecond(t *testing.T) {
	t.Parallel()
	start := time.Now()
	rt, err := Bootstrap(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rt.Close()
	if time.Since(start) > time.Second {
		t.Fatalf("bootstrap took %s", time.Since(start))
	}
}

func TestBootstrap_contextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Bootstrap(ctx)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

type slowPrefetch struct{}

func (slowPrefetch) Prefetch(ctx context.Context) error {
	select {
	case <-time.After(5 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestParallelPrefetch_respectsCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	sp := slowPrefetch{}
	err := ParallelPrefetch(ctx, sp, sp, sp)
	if err == nil {
		t.Fatal("expected cancel error")
	}
}

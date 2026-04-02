package app

import (
	"context"
	"sync"
)

// Keychain abstracts OS credential reads (macOS Keychain, etc.).
type Keychain interface {
	Prefetch(ctx context.Context) error
}

// OAuthPrefetch abstracts OAuth token warmup.
type OAuthPrefetch interface {
	Prefetch(ctx context.Context) error
}

// APIKeyPrefetch abstracts API key resolution from env/files.
type APIKeyPrefetch interface {
	Prefetch(ctx context.Context) error
}

// NoopPrefetch implements all three interfaces with no work.
// Bootstrap uses NoopPrefetch for Keychain and OAuth until optional Phase 4 ACs require real warmup.
type NoopPrefetch struct{}

func (NoopPrefetch) Prefetch(context.Context) error { return nil }

// ParallelPrefetch runs keychain, oauth, apikey hooks concurrently (main.tsx parallel intent).
// Today only APIKeyPrefetch is non-noop (APIKeyFilePrefetch). Keychain and OAuth remain Phase 4
// optional tail: implement Keychain.Prefetch / OAuthPrefetch.Prefetch and pass them from Bootstrap
// when acceptance requires parity with upstream parallel credential loading.
func ParallelPrefetch(ctx context.Context, k Keychain, o OAuthPrefetch, a APIKeyPrefetch) error {
	var wg sync.WaitGroup
	errs := make([]error, 3)
	run := func(i int, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ctx.Err() != nil {
				errs[i] = ctx.Err()
				return
			}
			errs[i] = fn()
		}()
	}
	run(0, func() error {
		if k == nil {
			return nil
		}
		return k.Prefetch(ctx)
	})
	run(1, func() error {
		if o == nil {
			return nil
		}
		return o.Prefetch(ctx)
	})
	run(2, func() error {
		if a == nil {
			return nil
		}
		return a.Prefetch(ctx)
	})
	wg.Wait()
	for _, err := range errs {
		if err != nil && err != context.Canceled {
			return err
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

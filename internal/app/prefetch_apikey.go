package app

import "context"

// APIKeyFilePrefetch warms API key resolution (env vars + global api_key file) during parallel bootstrap prefetch.
type APIKeyFilePrefetch struct {
	GlobalConfigDir string
}

// Prefetch implements APIKeyPrefetch.
func (p APIKeyFilePrefetch) Prefetch(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_ = ReadAPIKey(p.GlobalConfigDir)
	return nil
}

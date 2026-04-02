package app

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net/http"

	"github.com/2456868764/rabbit-code/internal/anthropic"
)

func runAPIPreconnect(ctx context.Context, pool *x509.CertPool, log *slog.Logger) {
	preconnectRT, outboundErr := anthropic.ResolveAPIOutboundTransport(ctx, pool)
	if outboundErr != nil && log != nil {
		log.Debug("preconnect: API outbound transport unavailable, falling back to proxy+roots only", "err", outboundErr)
	}
	_ = anthropic.PreconnectHEAD(ctx, &http.Client{Transport: preconnectRT}, anthropic.BaseURL(anthropic.DetectProvider()))
}

// RunAPIPreconnect repeats API preconnect using rt.RootCAs and current process env (e.g. after LoadAndApplyMergedConfig
// updates the trust pool or managed_env switches cloud provider).
func RunAPIPreconnect(ctx context.Context, rt *Runtime) {
	if rt == nil {
		return
	}
	runAPIPreconnect(ctx, rt.RootCAs, rt.Log)
}

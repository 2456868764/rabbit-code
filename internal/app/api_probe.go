package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/anthropic/services"
)

// ProbeServiceAPI issues a request shaped like the named services/api TS module (AC4-7 / P4.6.1) using
// the same outbound RoundTripper resolution as Bootstrap (ResolveAPIOutboundTransport). On success, the
// caller must close resp.Body.
func ProbeServiceAPI(ctx context.Context, rt *Runtime, tsFile string, pol anthropic.Policy) (*http.Response, error) {
	if rt == nil {
		return nil, fmt.Errorf("app: nil runtime")
	}
	out, _ := anthropic.ResolveAPIOutboundTransport(ctx, rt.RootCAs)
	return services.RoundTripProbe(ctx, out, tsFile, anthropic.BaseURL(anthropic.DetectProvider()), anthropic.OAuthAPIBase(), pol)
}

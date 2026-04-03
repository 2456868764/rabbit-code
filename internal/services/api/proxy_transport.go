package anthropic

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// HTTPTransportWithProxyFromEnv returns a new *http.Transport that respects standard proxy
// environment variables (HTTPS_PROXY, HTTP_PROXY, ALL_PROXY, lower-case variants, NO_PROXY),
// matching the intent of utils/proxy.ts getProxyFetchOptions / client.ts outbound fetch.
//
// Use as the base RoundTripper for AuthTransport / TLS customizations (P4.3.3).
func HTTPTransportWithProxyFromEnv() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	t := base.Clone()
	t.Proxy = http.ProxyFromEnvironment
	return t
}

// HTTPTransportWithProxyFromEnvAndRoots is HTTPTransportWithProxyFromEnv with optional TLS RootCAs (e.g. app.Bootstrap pool + RABBIT_CODE_EXTRA_CA_BUNDLE).
// It does not read client cert env vars — use HTTPTransportAPIOutbound for mTLS (RABBIT_CODE_CLIENT_CERT / KEY).
func HTTPTransportWithProxyFromEnvAndRoots(pool *x509.CertPool) *http.Transport {
	t := HTTPTransportWithProxyFromEnv()
	if pool == nil {
		return t
	}
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	t.TLSClientConfig.RootCAs = pool
	return t
}

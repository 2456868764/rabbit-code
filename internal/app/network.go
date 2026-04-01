package app

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

const (
	envHTTPProxy  = "HTTP_PROXY"
	envHTTPSProxy = "HTTPS_PROXY"
	envNoProxy    = "NO_PROXY"
	envExtraCA    = "RABBIT_CODE_EXTRA_CA_BUNDLE"
)

// ProxyConfig carries proxy strings for a future HTTP transport (Phase 4).
type ProxyConfig struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

// LoadProxyFromEnv reads standard proxy variables (upper and lower case).
func LoadProxyFromEnv() ProxyConfig {
	return ProxyConfig{
		HTTPProxy:  firstNonEmpty(os.Getenv(envHTTPProxy), os.Getenv("http_proxy")),
		HTTPSProxy: firstNonEmpty(os.Getenv(envHTTPSProxy), os.Getenv("https_proxy")),
		NoProxy:    firstNonEmpty(os.Getenv(envNoProxy), os.Getenv("no_proxy")),
	}
}

func firstNonEmpty(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}

// SystemCertPool loads the system trust store and optionally appends RABBIT_CODE_EXTRA_CA_BUNDLE PEM file.
func SystemCertPool() (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	path := strings.TrimSpace(os.Getenv(envExtraCA))
	if path == "" {
		return pool, nil
	}
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !pool.AppendCertsFromPEM(pem) {
		// still return pool; caller may log
	}
	return pool, nil
}

// AppendPEMFiles appends PEM-encoded certificates from each path into pool (union with existing roots).
// Used for merged config extra_ca_paths together with SystemCertPool (SPEC §1.1 并集).
func AppendPEMFiles(pool *x509.CertPool, paths []string) error {
	if pool == nil {
		return fmt.Errorf("cert pool is nil")
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		pem, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read extra CA %q: %w", path, err)
		}
		if len(pem) > 0 {
			pool.AppendCertsFromPEM(pem)
		}
	}
	return nil
}

// TLSClientConfig returns a minimal tls.Config for Phase 4 reuse (no client certs yet).
func TLSClientConfig(pool *x509.CertPool) *tls.Config {
	if pool == nil {
		return &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}
}

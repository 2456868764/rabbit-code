package anthropic

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// HTTPTransportAPIOutbound returns a proxy-aware *http.Transport, optionally loading a TLS client
// certificate from RABBIT_CODE_CLIENT_CERT + RABBIT_CODE_CLIENT_KEY (client.ts mTLS paths; P4.3.3).
// If only one of the two paths is set, returns an error.
func HTTPTransportAPIOutbound() (*http.Transport, error) {
	return httpTransportAPIOutbound(nil)
}

// HTTPTransportAPIOutboundWithRoots is like HTTPTransportAPIOutbound but sets TLS RootCAs when pool != nil
// (align with app.Bootstrap / SystemCertPool).
func HTTPTransportAPIOutboundWithRoots(pool *x509.CertPool) (*http.Transport, error) {
	return httpTransportAPIOutbound(pool)
}

func httpTransportAPIOutbound(rootCAs *x509.CertPool) (*http.Transport, error) {
	t := HTTPTransportWithProxyFromEnvAndRoots(rootCAs)
	certPath := strings.TrimSpace(os.Getenv("RABBIT_CODE_CLIENT_CERT"))
	keyPath := strings.TrimSpace(os.Getenv("RABBIT_CODE_CLIENT_KEY"))
	if certPath == "" && keyPath == "" {
		return t, nil
	}
	if certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("anthropic: RABBIT_CODE_CLIENT_CERT and RABBIT_CODE_CLIENT_KEY must both be set for mTLS")
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("anthropic: load client cert: %w", err)
	}
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	t.TLSClientConfig.Certificates = []tls.Certificate{cert}
	return t, nil
}

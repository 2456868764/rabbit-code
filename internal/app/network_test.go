package app

import (
	"crypto/x509"
	"testing"
)

func TestAppendPEMFiles_missing(t *testing.T) {
	t.Parallel()
	pool := x509.NewCertPool()
	err := AppendPEMFiles(pool, []string{"/nonexistent/rabbit-code-test-ca.pem"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTLSClientConfig_nonNil(t *testing.T) {
	t.Parallel()
	pool := x509.NewCertPool()
	cfg := TLSClientConfig(pool)
	if cfg == nil || cfg.MinVersion == 0 {
		t.Fatal("expected tls config")
	}
}

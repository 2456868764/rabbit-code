package anthropic

import (
	"crypto/x509"
	"testing"
)

func TestHTTPTransportAPIOutbound_NoMTLS(t *testing.T) {
	t.Setenv("CLAUDE_CODE_CLIENT_CERT", "")
	t.Setenv("CLAUDE_CODE_CLIENT_KEY", "")
	tr, err := HTTPTransportAPIOutbound()
	if err != nil || tr == nil {
		t.Fatal(err)
	}
}

func TestHTTPTransportAPIOutbound_PartialPathsError(t *testing.T) {
	t.Setenv("CLAUDE_CODE_CLIENT_CERT", "/nonexistent/cert.pem")
	t.Setenv("CLAUDE_CODE_CLIENT_KEY", "")
	_, err := HTTPTransportAPIOutbound()
	if err == nil {
		t.Fatal("expected error when only cert path set")
	}
}

func TestHTTPTransportAPIOutboundWithRoots(t *testing.T) {
	t.Setenv("CLAUDE_CODE_CLIENT_CERT", "")
	t.Setenv("CLAUDE_CODE_CLIENT_KEY", "")
	pool := x509.NewCertPool()
	tr, err := HTTPTransportAPIOutboundWithRoots(pool)
	if err != nil || tr == nil {
		t.Fatal(err)
	}
	if tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs != pool {
		t.Fatal("expected RootCAs on transport")
	}
}

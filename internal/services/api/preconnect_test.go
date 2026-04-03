package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestShouldSkipPreconnect_ProxyEnvVariants(t *testing.T) {
	for _, k := range []string{
		"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy",
		"ALL_PROXY", "all_proxy",
	} {
		t.Run(k, func(t *testing.T) {
			t.Setenv(k, "http://127.0.0.1:9")
			if !ShouldSkipPreconnect() {
				t.Fatal("expected skip with " + k)
			}
		})
	}
}

func TestShouldSkipPreconnect_Bedrock(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "1")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for bedrock")
	}
}

func TestShouldSkipPreconnect_Vertex(t *testing.T) {
	t.Setenv(features.EnvUseVertex, "1")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for vertex")
	}
}

func TestShouldSkipPreconnect_Foundry(t *testing.T) {
	t.Setenv(features.EnvUseFoundry, "1")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for foundry")
	}
}

func TestShouldSkipPreconnect_UnixSocket(t *testing.T) {
	t.Setenv("RABBIT_CODE_UNIX_SOCKET", "/tmp/x.sock")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for unix socket")
	}
}

func TestShouldSkipPreconnect_ClientMTLS(t *testing.T) {
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "/path/cert.pem")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for client cert")
	}
}

func TestShouldSkipPreconnect_ClientKey(t *testing.T) {
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "/path/key.pem")
	if !ShouldSkipPreconnect() {
		t.Fatal("expected skip for client key")
	}
}

func TestPreconnectHEAD_Happy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")
	t.Setenv("RABBIT_CODE_UNIX_SOCKET", "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatal(r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if ShouldSkipPreconnect() {
		t.Skip("environment forces skip")
	}
	err := PreconnectHEAD(context.Background(), http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
}

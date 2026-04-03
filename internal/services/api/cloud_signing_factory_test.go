package anthropic

import (
	"context"
	"net/http"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestNewSigningTransportForProvider_anthropicNoWrap(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	base := http.DefaultTransport
	rt, err := NewSigningTransportForProvider(context.Background(), base, ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if rt != base {
		t.Fatalf("expected same round tripper, got %T", rt)
	}
}

func TestNewSigningTransportForProvider_bedrockSkipWraps(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "1")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvSkipBedrockAuth, "1")
	rt, err := NewSigningTransportForProvider(context.Background(), http.DefaultTransport, DetectProvider())
	if err != nil {
		t.Fatal(err)
	}
	st, ok := rt.(*SigningTransport)
	if !ok || st.Signer == nil {
		t.Fatalf("want *SigningTransport with signer, got %T", rt)
	}
}

func TestNewAPIOutboundTransport_1PNoSigningLayer(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	t.Setenv("RABBIT_CODE_CLIENT_CERT", "")
	t.Setenv("RABBIT_CODE_CLIENT_KEY", "")
	rt, err := NewAPIOutboundTransport(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rt.(*SigningTransport); ok {
		t.Fatal("1P should not use SigningTransport wrapper")
	}
	if rt == nil {
		t.Fatal("nil transport")
	}
}

package anthropic

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestUserAgent_Default(t *testing.T) {
	t.Setenv(features.EnvHTTPUserAgent, "")
	if got := UserAgent(); got != defaultUserAgent {
		t.Fatalf("got %q", got)
	}
}

func TestUserAgent_EnvOverride(t *testing.T) {
	t.Setenv(features.EnvHTTPUserAgent, "custom-agent/9")
	if got := UserAgent(); got != "custom-agent/9" {
		t.Fatalf("got %q", got)
	}
}

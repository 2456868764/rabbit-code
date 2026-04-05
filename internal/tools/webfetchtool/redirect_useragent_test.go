package webfetchtool

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestRedirectCodeText(t *testing.T) {
	if redirectCodeText(301) != "Moved Permanently" || redirectCodeText(302) != "Found" ||
		redirectCodeText(307) != "Temporary Redirect" || redirectCodeText(308) != "Permanent Redirect" {
		t.Fatal("mismatch upstream WebFetchTool.ts redirect labels")
	}
}

func TestWebFetchUserAgent_format(t *testing.T) {
	t.Setenv(features.EnvHTTPUserAgent, "")
	ua := WebFetchUserAgent()
	if !strings.HasPrefix(ua, "Claude-User (") || !strings.Contains(ua, "rabbit-code/api") {
		t.Fatalf("%q", ua)
	}
}

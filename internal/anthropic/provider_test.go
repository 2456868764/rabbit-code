package anthropic

import (
	"os"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestBaseURL_Anthropic(t *testing.T) {
	_ = os.Unsetenv("ANTHROPIC_BASE_URL")
	u := BaseURL(ProviderAnthropic)
	if !strings.HasPrefix(u, "http") {
		t.Fatal(u)
	}
}

func TestBaseURL_CustomAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://proxy.example/")
	u := BaseURL(ProviderAnthropic)
	if u != "https://proxy.example" {
		t.Fatal(u)
	}
}

func TestBaseURL_Bedrock(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	u := BaseURL(ProviderBedrock)
	if !strings.Contains(u, "eu-west-1") {
		t.Fatal(u)
	}
}

func TestMessagesPath_AllProviders(t *testing.T) {
	for _, p := range []Provider{ProviderAnthropic, ProviderBedrock, ProviderVertex, ProviderFoundry} {
		mp := MessagesPath(p)
		if mp == "" {
			t.Fatal(p)
		}
	}
}

func TestBedrockStreamPath(t *testing.T) {
	if p := BedrockStreamPath(""); p != "/model/invoke-with-response-stream" {
		t.Fatal(p)
	}
	want := "/model/anthropic.claude-3-5-sonnet-20241022-v2:0/invoke-with-response-stream"
	if p := BedrockStreamPath("anthropic.claude-3-5-sonnet-20241022-v2:0"); p != want {
		t.Fatalf("got %q want %q", p, want)
	}
}

func TestDetectProvider_Env(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "1")
	t.Cleanup(func() { _ = os.Unsetenv(features.EnvUseBedrock) })
	if DetectProvider() != ProviderBedrock {
		t.Fatal()
	}
}

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

func TestBaseURL_Vertex_sdkShape(t *testing.T) {
	t.Setenv("ANTHROPIC_VERTEX_BASE_URL", "")
	t.Setenv("CLOUD_ML_REGION", "europe-west4")
	u := BaseURL(ProviderVertex)
	if u != "https://europe-west4-aiplatform.googleapis.com/v1" {
		t.Fatal(u)
	}
	t.Setenv("CLOUD_ML_REGION", "global")
	u = BaseURL(ProviderVertex)
	if u != "https://aiplatform.googleapis.com/v1" {
		t.Fatal(u)
	}
}

func TestBaseURL_Foundry_anthropicSuffix(t *testing.T) {
	t.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", "")
	t.Setenv("ANTHROPIC_FOUNDRY_RESOURCE", "my-resource")
	u := BaseURL(ProviderFoundry)
	want := "https://my-resource.services.ai.azure.com/anthropic"
	if u != want {
		t.Fatalf("got %q want %q", u, want)
	}
}

func TestVertexStreamPath(t *testing.T) {
	want := "/projects/p1/locations/us-central1/publishers/anthropic/models/mymodel:streamRawPredict"
	if g := VertexStreamPath("p1", "us-central1", "mymodel"); g != want {
		t.Fatalf("got %q", g)
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

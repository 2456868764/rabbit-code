package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Provider selects Anthropic API transport shape (services/api/client.ts branches).
type Provider int

const (
	ProviderAnthropic Provider = iota
	ProviderBedrock
	ProviderVertex
	ProviderFoundry
)

// DetectProvider from rabbit-code feature env (maps CLAUDE_CODE_USE_* → RABBIT_CODE_USE_*).
func DetectProvider() Provider {
	switch {
	case features.UseBedrock():
		return ProviderBedrock
	case features.UseVertex():
		return ProviderVertex
	case features.UseFoundry():
		return ProviderFoundry
	default:
		return ProviderAnthropic
	}
}

// BaseURL returns default REST base for the provider (override with ANTHROPIC_BASE_URL for 1P).
func BaseURL(p Provider) string {
	if v := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")); v != "" && p == ProviderAnthropic {
		return strings.TrimRight(v, "/")
	}
	switch p {
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	case ProviderBedrock:
		if v := os.Getenv("AWS_BEDROCK_ANTHROPIC_BASE"); v != "" {
			return strings.TrimRight(v, "/")
		}
		r := os.Getenv("AWS_REGION")
		if r == "" {
			r = os.Getenv("AWS_DEFAULT_REGION")
		}
		if r == "" {
			r = "us-east-1"
		}
		return fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", r)
	case ProviderVertex:
		// Placeholder host pattern; real URL is region/project specific.
		proj := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")
		loc := os.Getenv("CLOUD_ML_REGION")
		if loc == "" {
			loc = "us-east5"
		}
		if proj != "" {
			return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s", loc, proj, loc)
		}
		return "https://us-east5-aiplatform.googleapis.com"
	case ProviderFoundry:
		if v := os.Getenv("ANTHROPIC_FOUNDRY_BASE_URL"); v != "" {
			return strings.TrimRight(v, "/")
		}
		res := os.Getenv("ANTHROPIC_FOUNDRY_RESOURCE")
		if res != "" {
			return fmt.Sprintf("https://%s.services.ai.azure.com", res)
		}
		return "https://example.azure.com"
	default:
		return "https://api.anthropic.com"
	}
}

// BedrockStreamPath returns the Bedrock Runtime invoke-with-response-stream path for modelID.
// When modelID is empty, returns the legacy placeholder used by MessagesPath(ProviderBedrock).
func BedrockStreamPath(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "/model/invoke-with-response-stream"
	}
	return fmt.Sprintf("/model/%s/invoke-with-response-stream", url.PathEscape(modelID))
}

// MessagesPath returns HTTP path for Messages create (1P vs cloud differ; tests use 1P).
func MessagesPath(p Provider) string {
	switch p {
	case ProviderAnthropic:
		return "/v1/messages"
	case ProviderBedrock:
		return BedrockStreamPath("") // placeholder without model; use BedrockStreamPath(model) for real calls
	case ProviderVertex, ProviderFoundry:
		return "/v1/messages" // Foundry/Vertex often expose Anthropic-compatible subpaths; tests mock host
	default:
		return "/v1/messages"
	}
}

// NewTransportChain builds Base → Auth for tests (proxy/TLS via http.DefaultTransport or caller-provided).
func NewTransportChain(base http.RoundTripper, apiKey, bearer string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &AuthTransport{Base: base, APIKey: apiKey, Bearer: bearer}
}

// NewTransportChainOAuth builds Auth with dynamic Bearer and optional one-shot 401 refresh (withOAuth401Retry).
func NewTransportChainOAuth(base http.RoundTripper, getBearer func() string, refresh func(context.Context) error) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &AuthTransport{Base: base, GetBearer: getBearer, Refresh: refresh}
}

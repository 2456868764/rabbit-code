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

// VertexDefaultAnthropicVersion is the JSON body anthropic_version for Vertex streamRawPredict (@anthropic-ai/vertex-sdk client.mjs).
const VertexDefaultAnthropicVersion = "vertex-2023-10-16"

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
		if v := strings.TrimSpace(os.Getenv("ANTHROPIC_VERTEX_BASE_URL")); v != "" {
			return strings.TrimRight(v, "/")
		}
		loc := strings.TrimSpace(os.Getenv("CLOUD_ML_REGION"))
		if loc == "" {
			loc = "us-east5"
		}
		if loc == "global" {
			return "https://aiplatform.googleapis.com/v1"
		}
		return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", loc)
	case ProviderFoundry:
		if v := strings.TrimSpace(os.Getenv("ANTHROPIC_FOUNDRY_BASE_URL")); v != "" {
			return strings.TrimRight(v, "/")
		}
		res := strings.TrimSpace(os.Getenv("ANTHROPIC_FOUNDRY_RESOURCE"))
		if res != "" {
			return fmt.Sprintf("https://%s.services.ai.azure.com/anthropic", res)
		}
		return "https://example.azure.com/anthropic"
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

func envVertexProjectID() string {
	return strings.TrimSpace(os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"))
}

func vertexRegion() string {
	r := strings.TrimSpace(os.Getenv("CLOUD_ML_REGION"))
	if r == "" {
		return "us-east5"
	}
	return r
}

// VertexStreamPath returns the Vertex AI publishers path for streaming Messages (vertex-sdk :streamRawPredict).
// modelID is passed through url.PathEscape; characters like '@' in Vertex model names may remain unescaped per Go's rules (same as SDK string interpolation).
func VertexStreamPath(projectID, region, modelID string) string {
	projectID = strings.TrimSpace(projectID)
	region = strings.TrimSpace(region)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = "unknown"
	}
	return fmt.Sprintf("/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict",
		projectID, region, url.PathEscape(modelID))
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

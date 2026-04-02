package app

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Bootstrap must still complete when Bedrock is selected but auth is skipped (mock/E2E), using NewAPIOutboundTransport path.
func TestBootstrap_preconnectBedrockSkipAuth(t *testing.T) {
	t.Setenv(features.EnvUseBedrock, "1")
	t.Setenv(features.EnvSkipBedrockAuth, "1")
	t.Setenv(features.EnvUseVertex, "")
	t.Setenv(features.EnvUseFoundry, "")
	rt, err := Bootstrap(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
}

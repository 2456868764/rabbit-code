package features

import (
	"os"
	"strings"
)

// EnvDisableWebSearch when truthy unregisters WebSearch (overrides all other gates).
const EnvDisableWebSearch = "RABBIT_CODE_DISABLE_WEB_SEARCH"

// EnvForceWebSearch when truthy registers WebSearch even on providers where upstream disables it (e.g. Bedrock).
const EnvForceWebSearch = "RABBIT_CODE_FORCE_WEB_SEARCH"

// EnvWebSearchHeadless when truthy skips wiring live Messages API web search in the engine (WebSearch.Run stays stub unless ExecuteSearch injected).
const EnvWebSearchHeadless = "RABBIT_CODE_WEB_SEARCH_HEADLESS"

// EnvWebSearchPlumVx3 mirrors GrowthBook tengu_plum_vx3: use small fast model + tool_choice web_search for inner search request.
const EnvWebSearchPlumVx3 = "RABBIT_CODE_WEB_SEARCH_PLUM_VX3"

// WebSearchHeadlessOnly returns true when live API wiring should be skipped (headless stub path).
func WebSearchHeadlessOnly() bool {
	return truthy(os.Getenv(EnvWebSearchHeadless))
}

// WebSearchPlumVx3 returns true when inner web search should use the Haiku/small-fast + forced tool_choice path.
func WebSearchPlumVx3() bool {
	return truthy(os.Getenv(EnvWebSearchPlumVx3))
}

// WebSearchToolEnabled mirrors WebSearchTool.isEnabled() in WebSearchTool.ts (firstParty / Vertex 4.x / Foundry).
// mainLoopModel should be the configured main loop model id (e.g. cfg.Model); when empty, callers often pass
// ResolveMainLoopModel("") from the query package.
func WebSearchToolEnabled(mainLoopModel string) bool {
	if truthy(os.Getenv(EnvDisableWebSearch)) {
		return false
	}
	if truthy(os.Getenv(EnvForceWebSearch)) {
		return true
	}
	m := strings.TrimSpace(mainLoopModel)
	switch {
	case UseBedrock():
		return false
	case UseVertex():
		return vertexModelSupportsWebSearch(m)
	case UseFoundry():
		return true
	default:
		return true
	}
}

func vertexModelSupportsWebSearch(model string) bool {
	return strings.Contains(model, "claude-opus-4") ||
		strings.Contains(model, "claude-sonnet-4") ||
		strings.Contains(model, "claude-haiku-4")
}

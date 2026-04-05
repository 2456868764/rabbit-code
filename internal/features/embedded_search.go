package features

import (
	"os"
	"strings"
)

// EnvEmbeddedSearchTools matches restored-src embeddedTools.ts (ant-native bfs/ugrep in binary).
// When truthy and CodeEntrypoint is not an SDK/local-agent entrypoint, dedicated Glob/Grep tools are omitted (tools.ts getAllBaseTools).
const EnvEmbeddedSearchTools = "EMBEDDED_SEARCH_TOOLS"

// CodeEntrypoint returns RABBIT_CODE_ENTRYPOINT or CLAUDE_CODE_ENTRYPOINT (trimmed), for embedded-search and similar gates.
func CodeEntrypoint() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_CODE_ENTRYPOINT")); s != "" {
		return s
	}
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT"))
}

// HasEmbeddedSearchTools mirrors hasEmbeddedSearchTools() in restored-src/src/utils/embeddedTools.ts.
func HasEmbeddedSearchTools() bool {
	if !truthy(os.Getenv(EnvEmbeddedSearchTools)) {
		return false
	}
	switch CodeEntrypoint() {
	case "sdk-ts", "sdk-py", "sdk-cli", "local-agent":
		return false
	default:
		return true
	}
}

// ReplModeEnabled mirrors the env-driven parts of isReplModeEnabled() in REPLTool/constants.ts (no USER_TYPE=ant default-on).
func ReplModeEnabled() bool {
	if replEnvDefinedFalsy("CLAUDE_CODE_REPL") || replEnvDefinedFalsy("RABBIT_CODE_REPL") {
		return false
	}
	if truthy(os.Getenv("CLAUDE_REPL_MODE")) || truthy(os.Getenv("RABBIT_CODE_REPL_MODE")) {
		return true
	}
	return false
}

func replEnvDefinedFalsy(key string) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	return envDefinedFalsyString(v)
}

// UseShellGrepForMemoryPrompts mirrors memdir.ts: hasEmbeddedSearchTools() || isReplModeEnabled().
func UseShellGrepForMemoryPrompts() bool {
	return HasEmbeddedSearchTools() || ReplModeEnabled()
}

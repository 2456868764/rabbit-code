package features

// ClaudeCodeRemote is true when CLAUDE_CODE_REMOTE or RABBIT_CODE_REMOTE is truthy (claude.ts remote session branch).
func ClaudeCodeRemote() bool {
	return truthy(firstNonEmptyEnvPair("RABBIT_CODE_REMOTE", "CLAUDE_CODE_REMOTE"))
}

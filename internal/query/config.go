package query

import (
	"os"
	"strconv"
	"strings"
)

// Env keys aligned with query/config.ts buildQueryConfig (Rabbit prefixes preferred; Claude names accepted where noted).

const (
	envStreamingToolExecutionRabbit = "RABBIT_CODE_STREAMING_TOOL_EXECUTION"
	envStreamingToolExecutionClaude = "CLAUDE_CODE_STREAMING_TOOL_EXECUTION"
	envEmitToolUseSummariesRabbit   = "RABBIT_CODE_EMIT_TOOL_USE_SUMMARIES"
	envEmitToolUseSummariesClaude   = "CLAUDE_CODE_EMIT_TOOL_USE_SUMMARIES"
	envUserType                     = "USER_TYPE"
	envUserTypeRabbit               = "RABBIT_CODE_USER_TYPE"
	envDisableFastModeRabbit        = "RABBIT_CODE_DISABLE_FAST_MODE"
	envDisableFastModeClaude        = "CLAUDE_CODE_DISABLE_FAST_MODE"
)

// QueryConfig mirrors query/config.ts QueryConfig: immutable snapshot at query entry.
// Feature() gates stay at call sites (TS comment); this holds only session id and env/statsig-style gates.
type QueryConfig struct {
	SessionID string
	Gates     QueryRuntimeGates
}

// QueryRuntimeGates mirrors QueryConfig["gates"] in config.ts.
type QueryRuntimeGates struct {
	// StreamingToolExecution: TS uses Statsig tengu_streaming_tool_execution2; headless uses env when Statsig is absent (default true if unset).
	StreamingToolExecution bool
	EmitToolUseSummaries   bool
	IsAnt                  bool
	FastModeEnabled        bool
}

func envTruthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "on":
		return true
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return false
}

func firstNonEmptyEnv(keys ...string) string {
	for _, k := range keys {
		if s := strings.TrimSpace(os.Getenv(k)); s != "" {
			return s
		}
	}
	return ""
}

func streamingToolExecutionFromEnv() bool {
	v, ok := os.LookupEnv(envStreamingToolExecutionRabbit)
	if !ok || strings.TrimSpace(v) == "" {
		v, ok = os.LookupEnv(envStreamingToolExecutionClaude)
	}
	if !ok || strings.TrimSpace(v) == "" {
		return true
	}
	return envTruthy(v)
}

func emitToolUseSummariesFromEnv() bool {
	s := firstNonEmptyEnv(envEmitToolUseSummariesRabbit, envEmitToolUseSummariesClaude)
	return envTruthy(s)
}

func isAntUserFromEnv() bool {
	ut := firstNonEmptyEnv(envUserTypeRabbit, envUserType)
	return strings.EqualFold(strings.TrimSpace(ut), "ant")
}

func fastModeEnabledFromEnv() bool {
	s := firstNonEmptyEnv(envDisableFastModeRabbit, envDisableFastModeClaude)
	return s == "" || !envTruthy(s)
}

// BuildQueryConfig mirrors buildQueryConfig (src/query/config.ts).
// sessionID should match bootstrap getSessionId(); pass "" when no session id is wired yet.
func BuildQueryConfig(sessionID string) QueryConfig {
	return QueryConfig{
		SessionID: strings.TrimSpace(sessionID),
		Gates: QueryRuntimeGates{
			StreamingToolExecution: streamingToolExecutionFromEnv(),
			EmitToolUseSummaries:   emitToolUseSummariesFromEnv(),
			IsAnt:                  isAntUserFromEnv(),
			FastModeEnabled:        fastModeEnabledFromEnv(),
		},
	}
}

// StopHooksUpstreamModule names src/query/stopHooks.ts for parity notes (PHASE_ITERATION_RULES §3.1).
//
// The upstream module is one large async generator (handleStopHooks) with user/teammate hooks and extract-memory flows.
// Headless rabbit-code implements hook slots on engine.Engine (StopHookFunc, StopHooksAfterSuccessfulTurn,
// StopHookBlockingContinue) and memory extraction in engine/extract_hook.go plus memdir; extend those surfaces
// rather than splitting stop-hook behavior into many small query-root files.
const StopHooksUpstreamModule = "stopHooks.ts"

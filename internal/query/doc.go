// Package query implements query loop state and transitions (query.ts parity, Phase 5).
// Subpackage engine holds QueryEngine-style orchestration (PHASE_ITERATION_RULES §4.4: src/QueryEngine.ts).
//
// Upstream layout (src/query/*.ts) vs this package (PHASE_ITERATION_RULES §3.1 — TS module basenames still map logically;
// several Go files group related parity to avoid a large sprawl of tiny sources):
//
//   - config.ts          → config.go (+ StopHooksUpstreamModule for stopHooks.ts notes)
//   - deps.ts            → deps.go (same package; TurnAssistant / test adapters / StreamingCompactExecutor wiring live alongside loop)
//   - (TS services/api)  → internal/services/api: AnthropicAssistant, stream compact summary, prompt-cache-break context, fork wiring
//   - shared turn shapes → internal/types (ToolUseCall, TurnResult) to avoid query↔api import cycles
//   - stopHooks.ts       → engine + config constant; hook slots / extract in engine
//   - tokenBudget.ts     → token_budget.go (also utils/tokenBudget + microCompact estimate + H5 submit helpers)
//
// Consolidated Go modules (multiple former files):
//
//   - state.go           — LoopState, transitions, LoopContinue, RecoveryPhase
//   - loop.go            — LoopDriver, observers, blocking limit, prompt cache break handling (tests: loop_test.go)
//   - snip.go            — transcript snip / H7 replay / UUID sidecar
//   - analyze_context.go — reactive gates, QuerySource re-exports, BuildHeadlessContextReport (delegates to services/compact)
//   - messages.go        — append user/assistant/tool messages
//   - assistant_adapters.go — StreamAssistantFunc, Sequence* (tests; no standalone TS file)
//   - bash_tool_runner.go — BashStubToolRunner, BashExecToolRunner（H9 / Phase 6 桥；**PARITY**：docs/phases/PARITY_H9_BASH_PERMISSIONS.md）
//   - streaming_compact_executor.go — StreamingCompactExecutor* (stays in query: avoids compact↔api cycle)
//   - transcript.go      — trim prefix, strip cache_control, user hints, template appendix
//
// Most loop/orchestration logic still corresponds to src/query.ts and src/QueryEngine.ts; see engine/doc.go.
package query

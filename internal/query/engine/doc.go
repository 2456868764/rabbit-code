// Package engine implements QueryEngine-style orchestration: user submit, assistant stream events, cancel (Phase 5).
// Import path: …/internal/query/engine (mirrors src/QueryEngine.ts next to query/).
// Does not import TUI (ARCHITECTURE_BOUNDARIES).
// Production wiring: internal/app.ApplyEngineCompactIntegration + (*Engine).InstallAnthropicStreamingCompact (see compact_install.go).
// Command lifecycle: Config.CommandLifecycleNotify + SubmitWithOptions.ConsumedCommandUUIDs mirror query.ts notifyCommandLifecycle after successful query().
// Host hooks: ProcessUserInputHook (processUserInput), ExtraTemplateNames (template appendix / classifier), AfterToolResultsHook (post–tool_result collect timing).
package engine

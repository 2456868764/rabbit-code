// Package engine implements QueryEngine-style orchestration: user submit, assistant stream events, cancel (Phase 5).
// Import path: …/internal/query/engine (mirrors src/QueryEngine.ts next to query/).
// Does not import TUI (ARCHITECTURE_BOUNDARIES).
// Production wiring: internal/app.ApplyEngineCompactIntegration + (*Engine).InstallAnthropicStreamingCompact (see compact_install.go).
// Command lifecycle: Config.CommandLifecycleNotify + SubmitWithOptions.ConsumedCommandUUIDs mirror query.ts notifyCommandLifecycle after successful query().
// Host hooks: ProcessUserInputHook (processUserInput), ExtraTemplateNames (template appendix / classifier), AfterToolResultsHook (post–tool_result collect timing).
// Shutdown: call internal/app.(*Runtime).RegisterEngineShutdown(engine) after New so forked extract drains before process exit (print.ts drainPendingExtraction).
// rabbit-code main uses app.QuitRuntime(rt, code) so Runtime.Close runs before os.Exit when a host registers that cleanup.
package engine

// Package compact implements headless subsets of restored-src/src/services/compact (Phase 4–5).
//
// Upstream directory: src/services/compact/*.ts. Target PARITY layout is one Go basename per TS module
// (snake_case.go); see table below.
//
// TS module (restored-src/src/services/compact/)     Go file (this package)
// -------------------------------------------------------------------------------------------------
// apiMicrocompact.ts                                 api_microcompact.go, api_context_management.go — MicroCompactRequested; getAPIContextManagement JSON
// autoCompact.ts                                     auto_compact.go — thresholds, TokenWarningState, proactive gates, AutoCompactTracking JSON, RecompactionMeta
// compact.ts                                         compact.go — RunPhase, ExecuteStub, FormatStubCompactSummary, ExecuteStubWithMeta
// microCompact.ts                                    micro_compact.go, time_based_trigger.go, time_based_microcompact.go — COMPACTABLE_TOOLS, EvaluateTimeBasedTrigger, maybeTimeBasedMicrocompact (CC JSON + Messages API JSON via wall-clock trigger)
// postCompactCleanup.ts                              post_compact_cleanup.go — main-thread post-compact source, reset microcompact state
// prompt.ts                                          prompt_compact.go — FormatCompactSummary (subset; full prompts TBD)
// grouping.ts                                        grouping.go — GroupMessagesByApiRound, ApiRoundMessage
// compactWarningState.ts                             auto_compact.go — CalculateTokenWarningState (numeric); compact_warning_state.go — suppress/clear store
// compactWarningHook.ts                              (not ported in this package)
// sessionMemoryCompact.ts                            internal/query/engine — Engine.SessionMemoryCompact callback wiring
// timeBasedMCConfig.ts                               time_based_mc_config.go — GetTimeBasedMCConfig (via features env)
//
// Related TS outside services/compact/:
//   - services/api/promptCacheBreakDetection.ts      api_microcompact.go — PromptCacheBreakActive (via features)
//   - internal/query — blocking ladder / strip cache_control — query/loop.go, query/transcript.go (prompt cache break path)
//
// Package query imports compact for LoopState.AutoCompactTracking and BuildHeadlessContextReport; compact must not import query.
//
// Tool names for COMPACTABLE_TOOLS: internal/tools/* (mirrors src/tools/*/prompt.ts|constants.ts) and
// internal/utils/shell.ShellToolNames (mirrors src/utils/shell/shellToolUtils.ts).
package compact

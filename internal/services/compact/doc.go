// Package compact implements headless subsets of restored-src/src/services/compact (Phase 4–5).
//
// Upstream directory: src/services/compact/*.ts. Target PARITY layout is one Go basename per TS module
// (snake_case.go); see table below.
//
// TS module (restored-src/src/services/compact/)     Go file (this package)
// -------------------------------------------------------------------------------------------------
// apiMicrocompact.ts                                 api_microcompact.go — MicroCompactRequested; buffer notes on micro_compact.go
// autoCompact.ts                                     auto_compact.go — thresholds, TokenWarningState, proactive gates, AutoCompactTracking JSON, RecompactionMeta
// compact.ts                                         compact.go — RunPhase, ExecuteStub, FormatStubCompactSummary, ExecuteStubWithMeta
// microCompact.ts                                    micro_compact.go — COMPACTABLE_TOOLS, collect IDs, main-thread source, microcompact buffer
// postCompactCleanup.ts                              post_compact_cleanup.go — main-thread post-compact source, reset microcompact state
// prompt.ts                                          (not ported in this package)
// grouping.ts                                        (not ported in this package)
// compactWarningState.ts                             auto_compact.go — CalculateTokenWarningState (numeric core)
// compactWarningHook.ts                              (not ported in this package)
// sessionMemoryCompact.ts                            internal/query/engine — Engine.SessionMemoryCompact callback wiring
// timeBasedMCConfig.ts                               (not ported in this package)
//
// Related TS outside services/compact/:
//   - services/api/promptCacheBreakDetection.ts      api_microcompact.go — PromptCacheBreakActive (via features)
//   - internal/query — blocking ladder / strip cache_control — query/loop.go, query/transcript.go (prompt cache break path)
//
// Package query imports compact for LoopState.AutoCompactTracking and BuildHeadlessContextReport; compact must not import query.
package compact

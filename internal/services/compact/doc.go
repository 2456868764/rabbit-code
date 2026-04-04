// Package compact implements headless subsets of restored-src/src/services/compact (Phase 4–5).
//
// Upstream directory: src/services/compact/*.ts. Target PARITY layout is one Go basename per TS module
// (snake_case.go); see table below.
//
// TS module (restored-src/src/services/compact/)     Go file (this package)
// -------------------------------------------------------------------------------------------------
// apiMicrocompact.ts                                 api_microcompact.go — entire module (getAPIContextManagement, defaults, tool lists); MicroCompactRequested / PromptCacheBreakActive colocated; HTTP merge in internal/services/api (client.go, betas_merge.go)
// autoCompact.ts                                     auto_compact.go — getAutoCompactThreshold/calculateTokenWarningState 数值核心, ProactiveAutocompactFromUsage + ProactiveAutoCompactPreflight（shouldAutoCompact）, AfterTurnProactiveAutocompactFromUsage（熔断）, AfterTurnReactiveCompactSuggested（DISABLE_COMPACT + 字节/启发 token + hasAttemptedReactive）；internal/query/engine/auto_compact_chain.go — runCompactSuggestAfterSuccessfulTurn（成功回合尾：DISABLE_COMPACT 早退 / advisor / 上门控 / session-memory compact / suggest）；CONTEXT_COLLAPSE 运行时门控见 features.ContextCollapseSuppressesProactiveAutocompact；engine 用 query 侧 Messages JSON token 估计
// compact.ts                                         compact*.go + compact_context.go (ExecutorSuggestMeta + DefaultCompactStreamingToolsJSON) + post_compact_attachments.go — static JSON, post-compact attachment payloads; querydeps: StreamCompactSummary(Detailed), SessionActivityPing + RemoteSendKeepalivesEnabled, ForkCompactSummary, StreamingCompactExecutorWithConfig (PostCompactAttachmentsJSON), engine suggest meta. Statsig/tengu_* + full normalizeMessagesForAPI: not in package.
// microCompact.ts                                    micro_compact.go, cached_microcompact.go, time_based_trigger.go (含 timeBasedMCConfig / GetTimeBasedMCConfig), time_based_microcompact.go — COMPACTABLE_TOOLS; MicrocompactMessagesAPIJSON; EstimateMessageTokensFromAPIMessagesJSON; EvaluateTimeBasedTrigger; time-based MC JSON mutators
// postCompactCleanup.ts                              post_compact_cleanup.go — RunPostCompactCleanup (+ PostCompactCleanupHooks), IsMainThreadPostCompactSource; engine wires optional hooks + legacy PostCompactCleanup callback
// prompt.ts                                          prompt_compact.go — FormatCompactSummary, GetCompactPrompt, GetPartialCompactPrompt, GetCompactUserSummaryMessage (proactive extra gated by features.KairosDailyLogMemoryEnabled)
// grouping.ts                                        grouping.go — GroupMessagesByApiRound, ApiRoundMessage
// compactWarningState.ts + compactWarningHook.ts     compact_warning.go — suppress/clear store + SubscribeCompactWarningSuppression（useSyncExternalStore analogue）；CalculateTokenWarningState（数值）仍在 auto_compact.go
// sessionMemoryCompact.ts                            session_memory_compact.go — config, truncate, keep-index, TrySessionMemoryCompactionTranscriptJSON, NewSessionMemoryCompactExecutor; engine wires SessionMemoryCompact to hooks
// timeBasedMCConfig.ts                               time_based_trigger.go — GetTimeBasedMCConfig（features env，与 evaluate 同文件）
//
// Related TS outside services/compact/:
//   - services/api/promptCacheBreakDetection.ts      api_microcompact.go — PromptCacheBreakActive (via features); notifyCacheDeletion (cacheDeletionsPending) not ported — no Go phase-2 cache-read comparison yet
//   - internal/query — blocking ladder / strip cache_control — query/loop.go, query/transcript.go (prompt cache break path)
//   - internal/query/engine — time-based MC on API transcript; LastAssistantAtForPersistence / RestoredSessionLastAssistantAt for session sidecars
//
// Package query imports compact for LoopState.AutoCompactTracking and BuildHeadlessContextReport; compact must not import query.
//
// Tool names for COMPACTABLE_TOOLS: internal/tools/* (mirrors src/tools/*/prompt.ts|constants.ts) and
// internal/utils/shell.ShellToolNames (mirrors src/utils/shell/shellToolUtils.ts).
package compact

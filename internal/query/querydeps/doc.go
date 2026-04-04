// Package querydeps holds injectable dependencies for the query loop (query.ts / QueryEngine.ts parity, Phase 5).
// deps.go is the Go counterpart of src/query/deps.ts (QueryDeps + productionDeps): callModel/autocompact/microcompact/uuid
// in TS map to StreamAssistant, TurnAssistant, and ToolRunner wiring in headless builds.
// anthropic_compact.go + compact_executor.go: StreamCompactSummary(Detailed), optional ForkCompactSummary, tools on MessagesStreamBody, StreamingCompactExecutorWithConfig (hooks + next transcript); engine sets compact.ExecutorSuggestMeta on ctx before CompactExecutor.
// Import path: github.com/2456868764/rabbit-code/internal/query/querydeps (subdirectory of internal/query).
package querydeps

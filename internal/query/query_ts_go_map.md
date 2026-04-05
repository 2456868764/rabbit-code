# query / QueryEngine ↔ restored-src mapping (executable index)

Authority tree: `claude-code-sourcemap/restored-src/src/`.

**全量功能对照与 PARITY 状态**：`docs/phases/PARITY_QUERY_QUERYENGINE.md`（遵守 **PARITY_PHASE5_DEFERRED.md**）。

## `src/query/*.ts`

| TS | Go |
|----|-----|
| `deps.ts` | `internal/query/deps.go` |
| `config.ts` | `internal/query/config.go` |
| `stopHooks.ts` | `internal/query/engine` + `config.go` (constants / hooks) |
| `tokenBudget.ts` | `internal/query/token_budget.go` |

## `src/query.ts` (monolith)

| Area | Go (primary) |
|------|----------------|
| Loop / tools / cache break | `internal/query/loop.go`, `state.go`, `messages.go`, `transcript.go`, `snip.go` |
| `skipCacheWrite`（断点下标） | `transcript.go` **`RemapPromptCacheBreakpointsForSkipCacheWrite`**；`engine.Config.SkipCacheWrite` |
| `taskBudget` → Messages API | `engine.Config.TaskBudgetTotal` → `LoopDriver.TaskBudgetTotal` → `internal/services/api/task_budget_context.go` + `anthropic_assistant.go` |
| Streaming compact summary | `internal/services/api/anthropic_stream_compact.go` |
| Compact executor + hooks wiring | `internal/query/streaming_compact_executor.go` |
| Deps / turn types | `internal/query/deps.go`, `internal/types/assistant_turn.go` |

## `src/QueryEngine.ts`

| TS | Go |
|----|-----|
| 包整体 | `internal/query/engine/*.go` |
| `taskBudget.total` | `engine.Config.TaskBudgetTotal` → `query.LoopDriver` → `anthropic.WithPerTurnTaskBudget`（`internal/services/api/task_budget_context.go`） |

## Import rules (enforced)

- `internal/services/compact` **must not** import `internal/services/api` (use `prompt_too_long_parse.go` for PTL token parse; prefix `PromptTooLongErrorPrefix`).
- `internal/services/api` may import `internal/services/compact` (assistant + stream compact).
- `internal/query` may import both `api` and `compact`; **`StreamingCompactExecutor`** stays in `query` to avoid `compact`↔`api` cycles.

## Verification

```bash
go test ./internal/query/... ./internal/services/api/... ./internal/services/compact/... ./internal/types/... -count=1 -short
go test ./... -count=1 -short
```

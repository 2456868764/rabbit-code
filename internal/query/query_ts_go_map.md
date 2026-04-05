# query / QueryEngine ↔ restored-src mapping (executable index)

Authority tree: `claude-code-sourcemap/restored-src/src/`.

**全量功能对照与 PARITY 状态**：`docs/phases/PARITY_QUERY_QUERYENGINE.md`（**`QueryEngineConfig` 字段映射 §4**、**cost §5**、**JSONL §6**；遵守 **PARITY_PHASE5_DEFERRED.md**）。

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
| `notifyCommandLifecycle`（成功返回后） | `internal/query/engine` **`Config.CommandLifecycleNotify`**、**`SubmitWithOptions.ConsumedCommandUUIDs`** |
| `skillPrefetch` / `taskSummary`（tool 结果后） | `internal/query/loop.go` **`LoopObservers.OnAfterToolResults`**；**`engine.Config.AfterToolResultsHook`** |
| `jobClassifier` → 模板名 | **`engine.Config.ExtraTemplateNames`** + **`mergedTemplateNames`**（附录与 **`EventKindTemplatesActive`**） |
| `processUserInput` | **`engine.Config.ProcessUserInputHook`**（`runTurnLoop` 内 memdir 之前） |

## `src/QueryEngine.ts`

| TS | Go |
|----|-----|
| 包整体 | `internal/query/engine/*.go` |
| 退出 drain（extract fork） | `Engine.DrainExtractMemories`；**`app.WireHeadlessEngineForShutdown`**（**`Bootstrap` 后**）或 **`RegisterEngineShutdown`**；**`cmd/rabbit-code`** 经 **`QuitRuntime`/`FailBootstrap`** 保证 **`Runtime.Close`** 先于 **`os.Exit`** |
| `taskBudget.total` | `engine.Config.TaskBudgetTotal` → `query.LoopDriver` → `anthropic.WithPerTurnTaskBudget`（`internal/services/api/task_budget_context.go`） |
| `processUserInput` / 模板附录扩展 | `engine.Config.ProcessUserInputHook`、`ExtraTemplateNames`（`engine.go`） |

## Import rules (enforced)

- `internal/services/compact` **must not** import `internal/services/api` (use `prompt_too_long_parse.go` for PTL token parse; prefix `PromptTooLongErrorPrefix`).
- `internal/services/api` may import `internal/services/compact` (assistant + stream compact).
- `internal/query` may import both `api` and `compact`; **`StreamingCompactExecutor`** stays in `query` to avoid `compact`↔`api` cycles.

## Verification

```bash
go test ./internal/query/... ./internal/services/api/... ./internal/services/compact/... ./internal/types/... -count=1 -short
go test ./... -count=1 -short
```

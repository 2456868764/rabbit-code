# `query.ts` / `QueryEngine.ts` ↔ Go 全量对照

**TS 源树**：`claude-code-sourcemap/restored-src/src/query.ts`、`QueryEngine.ts`  
**Go**：`internal/query`、`internal/query/engine`（索引：`internal/query/query_ts_go_map.md`）

## 与 PARITY 规则的关系

- **PARITY_PHASE5_DEFERRED.md** 定义：哪些 P5.F.* 已在 **headless** 路径落地，哪些在 **Follow-on**（非 Phase 5 阻断项）。
- **PHASE05_CONTINUATION.md** 定义：相对 `src/` **全量**的推荐顺序（H1–H9、T*）。
- 本文件做 **符号与职责对齐表**：**[x]** headless 已对齐、**[~]** 子集/表面不同但语义部分覆盖、**[ ]** 未做或明确 defer（见上两份 PARITY 文档）。

「全量对齐 TS」**不等于**本仓库当前承诺：Follow-on 行在落地前应更新 **PARITY_PHASE5_DEFERRED.md** 与对应 Phase **SPEC §6**。

---

## 1. `query.ts`

| 区域 | TS | Go | 状态 |
|------|-----|-----|------|
| 入口 | `query(params)`、`queryLoop` | `query.LoopDriver.RunTurnLoop`；由 `engine.Engine` 在 `Submit` 中驱动 | **[x]** |
| 参数 | `QueryParams.messages` | `LoopState.MessagesJSON`、`RunTurnLoopFromMessages` | **[x]** |
| 参数 | `QueryParams.systemPrompt`、`userContext`、`systemContext` | `AnthropicAssistant` / memdir 系统块、`prependUserContext` 类逻辑在 Go 侧分散（`messages`、`memdir`、`engine`） | **[~]** |
| 参数 | `QueryParams.canUseTool`、`toolUseContext` | `ToolUseContextMirror` + `engine.Config` 部分字段；**全量** `ToolUseContext` 见 PARITY Follow-on | **[~]** |
| 参数 | `QueryParams.querySource` | `engine.Config.QuerySource` | **[x]** |
| 参数 | `QueryParams.maxOutputTokensOverride`、`maxTurns` | `LoopState` / `Config.MaxAssistantTurns` | **[x]** |
| 参数 | `QueryParams.taskBudget` | **`engine.Config.TaskBudgetTotal`** → **`query.LoopDriver.TaskBudgetTotal`** → **`anthropic.WithPerTurnTaskBudget`** → 主轮 **`AssistantTurn` / `StreamAssistant`** 写入 **`output_config.task_budget`**（`internal/services/api/task_budget_context.go`） | **[x]** headless |
| 参数 | `QueryParams.skipCacheWrite` | 无对应开关 | **[ ]** |
| 参数 | `QueryParams.deps` | `query.Deps`（`Tools` / `Assistant` / `Turn`） | **[x]** headless 注入模型 |
| 状态 | `State.messages`、`transition`、auto-compact / max-output / pending summary / stopHook | `LoopState` 各字段（见 `state.go` 注释对 `query.ts`） | **[x]** H6 文档口径 |
| 状态 | `consumedCommandUuids`、`notifyCommandLifecycle` | **`Config.CommandLifecycleNotify`** + **`SubmitWithOptions{ ConsumedCommandUUIDs }`**：成功 **`EventKindDone`** 后对每个 UUID 调 **`notify(uuid,"completed")`**（与 **`query()`** 正常返回后尾部循环一致）；**队列消费填充 UUID** 仍属 REPL/TUI | **[~]** headless |
| 流式产物 | `StreamEvent`、`Message`、… | `engine.EngineEvent`（`EventKind*`） | **[x]** headless 子集 |
| 特性门 | `feature('REACTIVE_COMPACT')` 等 | `internal/features/rabbit_env.go` + `PARITY_PHASE5_DEFERRED` 表 | **[x]** / **[~]** 见 DEFERRED |
| 侧路 | `jobClassifier`、`skillPrefetch`、`taskSummaryModule` 等 | **`Config.ExtraTemplateNames`** 合并进模板附录 / **`EventKindTemplatesActive`**；**`Config.AfterToolResultsHook`**（经 **`LoopObservers.OnAfterToolResults`**）在每轮 tool 结果写入 transcript 之后；**分类器 / 预取 / 摘要** 的具体逻辑仍由宿主在 hook 内实现 | **[~]** headless 挂点 |

---

## 2. `QueryEngine.ts`

| 区域 | TS | Go | 状态 |
|------|-----|-----|------|
| 类型 | `QueryEngineConfig`（cwd、tools、MCP、AppState、…） | `engine.Config` + `query.Deps` + 宿主注入（无单一 1:1 结构体）；**`taskBudget.total`** 见 **`Config.TaskBudgetTotal`**；**字段级映射见下 §4** | **[~]**（**§4** 文档化；全量单 struct 仍 defer） |
| 生命周期 | `new QueryEngine(config)` | `engine.New` / `NewWithConfig` | **[x]** |
| 回合 | `submitMessage(prompt, options)` → `AsyncGenerator<SDKMessage>` | `Engine.Submit(string)` + `Events() <-chan EngineEvent` | **[~]** 表面不同；headless 事件模型 |
| 入口 | `ask({...})`（SDK 聚合） | `internal/app` 等宿主拼装 Config + Deps（非 `engine` 单函数） | **[~]** |
| 系统提示 | `fetchSystemPromptParts`、`loadMemoryPrompt`、coordinator | `memdir` / `memory_system_engine.go` / settings；**全量** `queryContext` 仍 PARITY | **[~]** |
| 用户输入 | `processUserInput`、slash commands | **`Config.ProcessUserInputHook`**：在 memdir / 模板 / 主轮之前可改写 submit 正文；**slash / 完整 REPL** 仍 TUI / Phase 9 | **[~]** headless |
| Snip | `snipReplay`、`snipProjection` | `messages.StripHistorySnipPieces`、H7 snip 日志；**SDK snip 重放** 仍部分 defer | **[~]** |
| 用量 / 费用 | `accumulateUsage`、`cost-tracker` | **`internal/cost`**（**`Usage`**、**`MergeUsage`**、**`ApplyUsageToBootstrap`**、流式 **`FromStreamUsage`**）对齐 **`claude.ts` / `logging.ts`** 子集；**`getTotalCost` / `getModelUsage`**（USD 与按模型聚合）仍宿主或 TUI | **[~]**（**§5**） |
| 会话 | `recordTranscript`、`flushSessionStorage` | H7 侧车 JSON + **`Config.RestoredSnipRemovalLog`**；**JSONL `Map<UUID, Message>`、`parentUuid` 重链** 仍 **Phase 8**（**PARITY_PHASE5_DEFERRED**） | **[~]**（**§6**） |

---

## 4. `QueryEngineConfig`（TS）↔ Go 头部映射（headless）

TS 定义见 **`restored-src/src/QueryEngine.ts`** **`QueryEngineConfig`**。Go 刻意拆在 **`engine.Config`**、**`query.Deps`**、**`bootstrap.State`** 与宿主，避免 import 环；下表为 **逐项去向**（**[x]** 本仓库 headless 已接、**[ ]** 仍 REPL/TUI/Phase 6+）。

| TS 字段 | Go / 说明 | 状态 |
|---------|-----------|------|
| `cwd` | **`engine.Config.MemdirProjectRoot`**、进程 **cwd**、**`bootstrap.State`**（**`SetCwd`**）；无单一 `Config.Cwd` | **[~]** |
| `tools` | **`query.Deps.Tools`** | **[x]** |
| `commands` | REPL **`commands.ts`**；headless 用 **`Config.ProcessUserInputHook`** / 子命令 CLI | **[ ]** |
| `mcpClients` | Phase 6+ MCP；**`ToolUseContextMirror`** defer | **[ ]** |
| `agents` | Agent 工具目录；未 mirror | **[ ]** |
| `canUseTool` | **`ToolUseContextMirror`** + 权限顾问钩子；全量见 DEFERRED | **[~]** |
| `getAppState` / `setAppState` | **`engine.Config`** 显式字段（**`InitialSettings`**、memdir 等）+ 宿主 | **[~]** |
| `initialMessages` | **`RunTurnLoopFromMessages`** / **`LoopState.MessagesJSON`** | **[x]** |
| `readFileCache` | 文件工具层 / **`filereadtool`**；非 Engine | **[ ]** |
| `customSystemPrompt` / `appendSystemPrompt` | **`AnthropicAssistant.SystemPrompt`**、**`memdir.LoadMemorySystemPrompt`**、模板附录 | **[~]** |
| `userSpecifiedModel` / `fallbackModel` | **`Config.Model`**、**`query` model 解析**；fallback 策略部分在 API 客户端 | **[~]** |
| `thinkingConfig` | **`RABBIT_CODE_*` / features** 与 **`FormatHeadlessModeTags`** 等 | **[~]** |
| `maxTurns` | **`Config.MaxAssistantTurns`** | **[x]** |
| `maxBudgetUsd` | 未在 Engine 强制；可宿主 | **[ ]** |
| `taskBudget` | **`Config.TaskBudgetTotal`** → **`output_config.task_budget`** | **[x]** |
| `jsonSchema` | structured output / 工具 schema；defer | **[ ]** |
| `verbose` | 日志 / **`engine.Config.Debug`** | **[~]** |
| `replayUserMessages` | H7 / 消息管线；部分 | **[~]** |
| `handleElicitation` | MCP **-32042**；defer | **[ ]** |
| `includePartialMessages` | 流式事件；**`EngineEvent`** 子集 | **[~]** |
| `setSDKStatus` | **`EngineEvent`** 总线替代 SDK 状态回调 | **[~]** |
| `abortController` | **`context.Context`** + **`ToolUseContextMirror.AbortSignalAborted`** | **[~]** |
| `orphanedPermission` | **`OrphanPermissionAdvisor`**、**`EventKindOrphanPermission`** | **[x]** headless 子集 |
| `snipReplay` | **`messages.StripHistorySnipPieces`**、H7 **snip 日志**；SDK 边界重放仍 **[~]** |

## 5. 用量 / 费用（`accumulateUsage`、`cost-tracker.ts`）

| TS | Go | 状态 |
|----|-----|------|
| **`accumulateUsage` / `updateUsage`**（**`claude.js`**） | **`internal/cost`**：**`Usage`**、**`Merge`**、**`ApplyUsageToBootstrap`**；流式增量 **`FromUsageDelta`**（**`from_stream.go`**） | **[x]** headless 子集 |
| **`getTotalCost` / `getModelUsage` / `getTotalAPIDuration`**（**`cost-tracker.ts`**） | 无 Engine 内建聚合；宿主消费 **`EngineEvent`** + **`bootstrap.State`** 令牌字段或日后 T3 meter | **[ ]** / **[~]** |

## 6. 会话与 JSONL Map（`recordTranscript`、`flushSessionStorage`）

| TS | Go | 状态 |
|----|-----|------|
| **侧车 / snip 元数据** | **`LoopState.SnipRemovalLog`**、**`MarshalSnipRemovalLogJSON`**、**`Config.RestoredSnipRemovalLog`**（H7） | **[x]** headless |
| **JSONL 文件 + `Map<UUID, TranscriptMessage>` + `parentUuid`** | **未实现**；目标 **Phase 8**（**PARITY_PHASE5_DEFERRED** Follow-on） | **[ ]** |

---

## 3. 维护

- 落地 Follow-on 某行后：**改 PARITY_PHASE5_DEFERRED.md**、**PHASE05_SPEC_AND_ACCEPTANCE.md §6**，并更新上表对应格为 **[x]** 或 **[~]**。
- **Import 环与分层**仍以 `query_ts_go_map.md` **Import rules** 为准。

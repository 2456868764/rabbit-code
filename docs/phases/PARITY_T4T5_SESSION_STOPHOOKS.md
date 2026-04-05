# PARITY T4 / T5（穿插）：session restore todos + stop-hooks 目录清单

对照 **`claude-code-sourcemap/restored-src/src/utils/sessionRestore.ts`** **`extractTodosFromTranscript`** 与 **`query/stopHooks.ts`** 磁盘钩子目录概念；**全量 coordinator / `executeStopHooks` 工具执行** 仍见 **`PARITY_PHASE5_DEFERRED.md`**。

## T4（`sessionRestore` 子集）

- **Go**：**`query.ExtractTodosFromTranscriptJSON`**、**`query.TodoResumeItem`**（**`content` / `status` / `activeForm`**），自 Messages API JSON 自新向旧扫描 **assistant** 消息，在 **content** 块中取最后一个 **`type":"tool_use"`** 且 **`name":"TodoWrite"`** 的 **`input.todos`**；跳过 **`content` 为字符串** 的 assistant；校验 **`status`** ∈ **`pending` / `in_progress` / `completed`** 且 **`content`/`activeForm` 非空**（对齐 **`TodoListSchema`** 思路）。
- **仍 defer**：会话恢复协调器、TUI 恢复流、其它 **`sessionRestore`** 工具链。

## T5（`stopHooks` 子集）

- **Env**：**`RABBIT_CODE_STOP_HOOKS_DIR`** → **`features.StopHooksDir()`**。
- **CLI**：**`rabbit-code stop-hooks list`**（**`-dir`** 或上述 env）→ stdout 一行 JSON **`{"markdown":["…"]}`**（目录内 **`.md`** 基名，排序）。
- **仍 defer**：**`executeStopHooks`**、job 分类器与模板任务全链。

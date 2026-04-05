# H9：Bash / 权限栈 ↔ Go（Phase 6 工具层）

**规则**：`PHASE_ITERATION_RULES.md` **§三**（清单牵引、测绿、一提交、文档同步）。**§3.1** 要求上游模块整包对照时，本文件列 **`src/tools/BashTool/`** 全量 **`.ts`**；Go 侧当前为 **headless 桥接**（**`internal/query/bash_tool_runner.go`** 等），非整目录 1:1 迁移完成声明。

**主进度表**：`PHASE05_CONTINUATION.md` **Headless 行 9（H9）**、**§3.0 H9 子计划**。

---

## §3.0 H9 有序迭代计划（执行顺序）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`BashExecToolRunner`**：拒绝命令串中的 **null 字节**（与 TS 路径卫生同类）；**`RABBIT_CODE_BASH_EXEC`** 开启时生效 | **`go test ./internal/query/... -short`** |
| 2 | ☑ | **`readOnlyValidation` / `readOnlyCommandValidation`** ↔ **`memdir.IsExtractReadOnlyBash`**（扩展只读 **git** 子命令、**`stash list` / `remote` / `config --get`**、**NUL 拒绝**）；**`pathValidation` / `bashPermissions`** 仍 **Phase 6**（见 **§4**） | **`go test ./internal/memdir/... -short`** |
| 3 | ☑ | **`canUseTool` / 孤儿 tool_use** ↔ **`query.OrphanPermissionError`**、**`engine.Config.OrphanPermissionAdvisor`**、**`EventKindOrphanPermission`**（**§5**）；全量 **`canUseTool`** 仍 **PARITY_QUERY / DEFERRED** | 文档 + 现有 **`engine_test`** |

---

## §3.1-1 上游 TS 清单（`src/tools/BashTool/`，平铺）

| # | `restored-src/src/tools/BashTool/*.ts` |
|---|----------------------------------------|
| 1 | `bashCommandHelpers.ts` |
| 2 | `bashPermissions.ts` |
| 3 | `bashSecurity.ts` |
| 4 | `commandSemantics.ts` |
| 5 | `commentLabel.ts` |
| 6 | `destructiveCommandWarning.ts` |
| 7 | `modeValidation.ts` |
| 8 | `pathValidation.ts` |
| 9 | `prompt.ts` |
| 10 | `readOnlyValidation.ts` |
| 11 | `sedEditParser.ts` |
| 12 | `sedValidation.ts` |
| 13 | `shouldUseSandbox.ts` |
| 14 | `toolName.ts` |
| 15 | `utils.ts` |

---

## Go 对照（当前 headless）

| 职责 | TS 参考 | Go | 状态 |
|------|---------|-----|------|
| Bash 工具执行（env 门控） | **`BashTool`** 管线 | **`query.BashExecToolRunner`** / **`BashStubToolRunner`**；**`RABBIT_CODE_BASH_EXEC`**（**`features.BashExecEnabled`**） | **[~]** |
| Extract 子代理只读 bash | **`readOnlyValidation.ts`** + **`readOnlyCommandValidation.ts`** | **`memdir.IsExtractReadOnlyBash`**（**§4**） | **[~]**（extract 子集；非完整 BashTool） |
| 孤儿权限 | hooks / **`useCanUseTool`** | **`query.OrphanPermissionError`**、**`engine.Config.OrphanPermissionAdvisor`**、**`EventKindOrphanPermission`**（**§5**） | **[~]** headless |

---

## §4 `readOnlyValidation` / `pathValidation` ↔ Go（headless）

| TS 区域 | 行为摘要 | Go | 状态 |
|---------|----------|-----|------|
| **`readOnlyCommandValidation.ts`** | 多词 **git** 命令、flag 白名单、危险 flag 拦截 | **`memdir.isExtractReadOnlyShellCommand`**：无管道/重定向；**`&&`/`||`/`;** 分段；**`git`** 子命令白名单 + **`stash list`**、**`remote`/`remote -v`/`remote show`**、**`config --get`**；**`blame`/`merge-base`/`shortlog`/`reflog`/`rev-list`/`cat-file`/`for-each-ref`/`whatchanged`/`name-rev`** 等 | **[~]** |
| **`readOnlyValidation.ts`** | 复合命令、**cd**+**git**、bare repo 探测等 | **未镜像**（extract 仅单管道拒绝 + 分段） | **[ ]** Phase 6+ |
| **`pathValidation.ts`** | 工作目录、删除路径、**cd** 写组合 | **`BashExecToolRunner`** 无 cwd/allowlist；**`AutoMemToolRunner`** 仅记忆目录写 | **[ ]** Phase 6 |
| **`bashPermissions.ts`** / **`bashSecurity.ts`** | 权限模式、沙箱 | **`RABBIT_CODE_BASH_EXEC`** 门控 + **NUL** 拒绝；无沙箱 | **[ ]** Phase 6 |

**`query.BashExecToolRunner`**：不设只读校验（与 TS「全量 Bash 工具」分离）；extract 路径用 **`IsExtractReadOnlyBash`** 闸门。

## §5 `canUseTool` / 孤儿权限 ↔ Go（headless）

| TS | Go | 状态 |
|----|-----|------|
| **`canUseTool`** 拒绝、孤儿 **tool_use** | **`RunTool`** 返回 **`query.OrphanPermissionError{ToolUseID}`**；**`query.OrphanToolUseID(err)`** | **[x]** 子集 |
| 成功后顾问扫描 | **`engine.Config.OrphanPermissionAdvisor`** → **`EventKindOrphanPermission`**（**`engine_test` `TestEngine_OrphanPermission_advisor`**） | **[x]** headless |
| 全量 **`ToolUseContext` / MCP / 规则引擎** | **`ToolUseContextMirror`**、DEFERRED | **[ ]** / **[~]** 见 **PARITY_QUERY_QUERYENGINE.md** |

---

## 维护

- 完成 §3.0 一项：更新上表 **状态**、**`PHASE05_CONTINUATION.md`** H9 段、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
- 整包迁移 **`BashTool` → `internal/tools/bashtool`**（或等价包）时遵守 **§3.1** 文件名 **`snake_case.go` ↔ camelCase.ts**。

# H9：Bash / 权限栈 ↔ Go（Phase 6 工具层）

**规则**：`PHASE_ITERATION_RULES.md` **§三**（清单牵引、测绿、一提交、文档同步）。**§3.1** 要求上游模块整包对照时，本文件列 **`src/tools/BashTool/`** 全量 **`.ts`**；Go 侧当前为 **headless 桥接**（**`internal/query/bash_tool_runner.go`** 等），非整目录 1:1 迁移完成声明。

**主进度表**：`PHASE05_CONTINUATION.md` **Headless 行 9（H9）**、**§3.0 H9 子计划**。

---

## §3.0 H9 有序迭代计划（执行顺序）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`BashExecToolRunner`**：拒绝命令串中的 **null 字节**（与 TS 路径卫生同类）；**`RABBIT_CODE_BASH_EXEC`** 开启时生效 | **`go test ./internal/query/... -short`** |
| 2 | ☐ | **`readOnlyValidation` / `pathValidation` / `bashPermissions`**：只读与路径规则与 **`BashExecToolRunner` / `memdir.IsExtractReadOnlyBash`** 对照表 | 表 + 测 |
| 3 | ☐ | **`OrphanPermission` / `canUseTool`**：与 **`query.OrphanPermissionAdvisor`**、Phase 6 真实 **`ToolRunner`** 接线 | 见 **PARITY_QUERY_QUERYENGINE.md**、**PARITY_PHASE5_DEFERRED** |

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
| Extract 子代理只读 bash | **`readOnlyValidation.ts`** 子集 | **`memdir.IsExtractReadOnlyBash`** 等 | **[~]** |
| 孤儿权限 | hooks / **`useCanUseTool`** | **`query.ErrOrphanPermission`**、**`OrphanPermissionAdvisor`** | **[~]** |

---

## 维护

- 完成 §3.0 一项：更新上表 **状态**、**`PHASE05_CONTINUATION.md`** H9 段、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
- 整包迁移 **`BashTool` → `internal/tools/bashtool`**（或等价包）时遵守 **§3.1** 文件名 **`snake_case.go` ↔ camelCase.ts**。

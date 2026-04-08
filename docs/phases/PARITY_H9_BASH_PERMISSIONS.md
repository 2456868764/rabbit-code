# H9：Bash / 权限栈 ↔ Go（Phase 6 工具层）

**规则**：`PHASE_ITERATION_RULES.md` **§三**（清单牵引、测绿、一提交、文档同步）。**§3.1** 要求上游模块整包对照时 **主本映射以本节为准**；Go 侧为 **headless 桥接**（**`internal/query/bash_tool_runner.go`** + **`internal/tools/bashtool`**），与 Ink/React **不等价**处已标 **defer**。

**主进度表**：`PHASE05_CONTINUATION.md` **Headless 行 9（H9）**、**§3.0 H9 子计划**。

---

## §3.0 H9 有序迭代计划（执行顺序）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`BashExecToolRunner`**：命令串 **NUL** 拒绝；**`RABBIT_CODE_BASH_EXEC`**（**`features.BashExecEnabled`**） | **`go test ./internal/query/... -short`** |
| 2 | ☑ | **Extract 只读**：**`memdir.IsExtractReadOnlyBash`** / **`extractbash`**（保守子集 + NUL）；与 **bashtool 只读模式**解耦（bashtool 见序 **4**） | **`go test ./internal/memdir/... ./internal/extractbash/... -short`** |
| 3 | ☑ | **孤儿 tool_use**：**`OrphanPermissionError`**、**`OrphanPermissionAdvisor`**、**`EventKindOrphanPermission`** | **`go test ./internal/engine/... -short`** |
| 4 | ☑ | **`readOnlyCommandValidation.ts`**（**`src/utils/shell/`**）↔ **`internal/readonlycmd`**（**`go:embed` JSON** + **`validateFlags`**）+ **`bashtool/read_only_gate.go`**（pipe / **`&&`/`||`/`;`** + **`IsCommandSafeViaFlagParsing`** 或 **`extractbash`**）；**`sed`** 钩子 → **`SedCommandAllowedByAllowlist`** | **`go test ./internal/readonlycmd/... ./internal/tools/bashtool/... -short`** |
| 5 | ☑ | **`bashSecurity.ts`** 补强：**`bash_security.go`**（regex + **`HEREDOC_IN_SUBSTITUTION`** + **`ZSH_DANGEROUS_COMMANDS`**）+ **`bash_security_sh_parse.go`**（**`mvdan.cc/sh/v3`** **`CmdSubst`/`ProcSubst`**）；**`kernel_sandbox.go`**（**firejail**/**bwrap**）；**`sedEditParser.ts`** **`applySedSubstitution`** → **`ApplySedSubstitution`** | 同上 + **`bash_security_more_test`** |
| 6 | **[~]** | **`bashCommandIsSafe_DEPRECATED`**：已接 **incomplete、`isSafeHeredoc`/`validateSafeCommandSubstitution` 早放行**（**`bash_security_heredoc.go`**）、comment-quote-desync、quoted-newline、CR、newlines、redirections、backslash-ws/operators、mid-word #、shell metacharacters、obfuscated quotes 扩展、zsh precmd 修饰符 + 前述 jq/brace/IFS/proc/Unicode**（**`bash_security_remaining_validators.go`** 等）；仍 **defer**：**`validateGitCommit` 早放行**（headless 只读门有意不接：会放行 `.git` 写）、**`validateObfuscatedFlags` 引号内 flag 扫描全校验**、**`validateMalformedTokenInjection`**（**`tryParseShellCommand`**）；**`BashTool.tsx`/`UI.tsx`**、**`LocalShellTask`**、**SandboxManager** | headless 测绿；TUI Phase 9+ |

---

## §3.1-1 上游清单（`src/tools/BashTool/`，平铺 + TSX）

| # | `restored-src/src/tools/BashTool/*` |
|---|-------------------------------------|
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
| 16 | `BashTool.tsx` |
| 17 | `BashToolResultMessage.tsx` |
| 18 | `UI.tsx` |

**邻包（Bash 只读白名单源）**

| # | `restored-src/src/utils/shell/readOnlyCommandValidation.ts` |
|---|-------------------------------------------------------------|
| A | 导出 **`GIT_READ_ONLY_COMMANDS`** 等 + **`validateFlags`** 数据；Go：**`internal/readonlycmd`** + 再生脚本 **`tools/dump_allowlist_from_bundle.mjs`**（输出 **`internal/readonlycmd/allowlist_shared.json`**） |

**邻包（拆词 / 路径）**

| # | `restored-src/src/utils/bash/commands.ts` |
|---|-------------------------------------------|
| B | **`SplitCommandDeprecated`** 等；Go：**`internal/tools/bashtool/commands.go`** |

---

## §3.1-2 Go 文件名 ↔ TS 主映射（`internal/tools/bashtool` + 卫星包）

**说明**：与 **§3.1**「单 TS ↔ 单 `snake_case.go`」理想形态相比，历史实现将 **`bashPermissions.ts`** 等拆为多个 **`*.go`**；下表为**权威主本**，后续整包迁移时优先**合并回单文件**而非再增按功能命名文件。

| TS / TSX | Go 交付物 |
|----------|-----------|
| `BashTool.tsx` | `bash_tool.go`；后台：`background.go`；超时：`timeouts.go` |
| `BashToolResultMessage.tsx` | **defer**（消息渲染） |
| `UI.tsx` | `ui.go` |
| `bashCommandHelpers.ts` | `bash_command_helpers.go` |
| `bashPermissions.ts` | `bash_permissions_strip.go`、`bash_permissions_identify.go`、`bash_pipe_preflight.go` |
| `bashSecurity.ts` | `bash_security.go`、`bash_security_legacy_validators.go`、`bash_security_remaining_validators.go`、`bash_security_heredoc.go`、`bash_security_sh_parse.go` |
| `commandSemantics.ts` | `command_semantics.go` |
| `commentLabel.ts` | `comment_label.go` |
| `destructiveCommandWarning.ts` | `destructive_command_warning.go` |
| `modeValidation.ts` | `mode_validation.go` |
| `pathValidation.ts` | `path_validation.go` |
| `prompt.ts` | `prompt.go` |
| `readOnlyValidation.ts` | `read_only_structural.go`、`read_only_gate.go`、`read_only_validation.go`；**`extractbash`**（共享子集）；**`internal/readonlycmd`**（白名单 + flags） |
| `readOnlyCommandValidation.ts` | **`internal/readonlycmd/*.go`** + **`allowlist_*.json`** |
| `sedEditParser.ts` | `sed_edit_parser.go`（**`ParseSedEditCommand`**、**`ApplySedSubstitution`**） |
| `sedValidation.ts` | `sed_validation.go` |
| `shouldUseSandbox.ts` | `should_use_sandbox.go`、`kernel_sandbox.go` |
| `toolName.ts` | `toolname.go` |
| `utils.ts` | `utils_bash_tool.go` |
| `commands.ts`（utils/bash） | `commands.go`、`commands_test.go` |
| （TS 静默命令辅助） | `silent_bash.go`、`search_read.go` |
| （bare/UNC/git-internal） | `git_bare_detect.go`、`git_internal_writes.go`、`unc_path.go` |

**`src/tasks/LocalShellTask/*`（查询层引用）**

| TS | Go |
|----|-----|
| 任务模型占位 | `internal/tui/local_shell_task.go` |

---

## Go 对照（headless）

| 职责 | TS 参考 | Go | 状态 |
|------|---------|-----|------|
| Bash 执行（env 门控） | **`BashTool.tsx`** | **`query.BashExecToolRunner`** / **`BashStubToolRunner`**；**`bashtool.Bash.Run`**（**`sh -c`**）；**`RABBIT_MONITOR_TOOL`** → sleep 拦截 | **[~]** |
| 只读模式（全量白名单 + flags） | **`readOnlyCommandValidation.ts`** + **`readOnlyValidation.ts`** | **`readonlycmd`** + **`ReadOnlyCommandLineAllowed`** + **`CheckReadOnlyStructuralConstraints`**；回落 **`extractbash`** | **[~]** |
| Extract 子代理只读 | 同上（保守子集） | **`memdir.IsExtractReadOnlyBash`** → **`extractbash`** | **[x]** 子集 |
| 孤儿权限 | hooks | **`OrphanPermissionError`**、**`OrphanPermissionAdvisor`** | **[~]** |

---

## §4 `readOnlyValidation` / `pathValidation` / `bashPermissions` ↔ Go（headless）

| TS 区域 | 行为摘要 | Go | 状态 |
|---------|----------|-----|------|
| **`readOnlyCommandValidation.ts`** | 多词 **git**、**COMMAND_ALLOWLIST**、**validateFlags**、hooks（**sed**/**ps**/…） | **`internal/readonlycmd`**（embed **`allowlist_shared.json`** + **`allowlist_extras.json`**）+ **`bashtool/read_only_gate.go`** | **[~]**（数据由上游 bundle 再生；与 TS 仍可能有 ANT_ONLY / 平台差） |
| **`readOnlyValidation.ts`** | 复合、**cd**+**git**、bare、UNC、sandbox cwd | **`read_only_gate`** + **`read_only_structural.go`** + **`extractbash`** 回落 | **[~]** |
| **`pathValidation.ts`** | 进程替换、危险 **rm**、**cd**+写、工作区根 | **`path_validation.go`** + **`features.BashWorkdirRoot`** | **[~]** |
| **`bashPermissions.ts`** / **`bashSecurity.ts`** | 权限条带、危险模式 | **`bash_permissions_*`**、**`bash_security*.go`**、**`mvdan.cc/sh`**  walk | **[~]** |
| **内核沙箱** | Claude **sandbox** 运行时 | **`kernel_sandbox.go`**（**firejail**/**bwrap**，可选） | **[~]** |

**`bashtool` 只读门**：**`RABBIT_CODE_BASH_READ_ONLY`** 时 **`IsExtractReadOnlyBashInputJSON`** → **`ReadOnlyCommandLineAllowed`**（见 **`read_only_validation.go`**），**不是**仅 **`memdir.IsExtractReadOnlyBash`**。

---

## §5 `canUseTool` / 孤儿权限 ↔ Go（headless）

| TS | Go | 状态 |
|----|-----|------|
| **`canUseTool`** 拒绝、孤儿 **tool_use** | **`OrphanPermissionError{ToolUseID}`**；**`OrphanToolUseID(err)`** | **[x]** 子集 |
| 成功后顾问扫描 | **`OrphanPermissionAdvisor`** → **`EventKindOrphanPermission`** | **[x]** |
| 全量 **`ToolUseContext` / MCP** | **`ToolUseContextMirror`**，DEFERRED | **[ ]** 见 **PARITY_QUERY_QUERYENGINE.md** |

---

## 维护

- 完成 §3.0 一项：更新本节 **状态**、**`PHASE05_CONTINUATION.md`** H9 段、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
- 再生 allowlist：**`tools/README_READONLY_ALLOWLIST.md`**（输出 **`internal/readonlycmd/allowlist_shared.json`**）。

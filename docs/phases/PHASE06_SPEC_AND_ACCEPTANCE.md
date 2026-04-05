# Phase 6 功能规格与验收标准

**Phase 6：工具层（全量）**，主计划 [GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md) **§6.6** 与 [CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md](../CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md) **§4**。E2E：`PHASE06_E2E_ACCEPTANCE.md`。

**强制**：与 **`claude-code-sourcemap/restored-src/src/`**（简称 **`src/`**）语义对齐；平台豁免依 **[PARITY_CHECKLIST.md](../PARITY_CHECKLIST.md)**、**[ARCHITECTURE_BOUNDARIES.md](../ARCHITECTURE_BOUNDARIES.md)** 及本 SPEC 显式条款。

---

## 0. 迭代前核对声明（[PHASE_ITERATION_RULES.md](./PHASE_ITERATION_RULES.md)）

| 门槛 | 状态 | 说明 |
|------|------|------|
| **§1** 全量核对功能清单 + 验收 + E2E | **已执行（2026-04-01）** | 本文 **§2 / §3** 与 **`PHASE06_E2E_ACCEPTANCE.md`** 已对照主计划 **§6.6** 与 **`SOURCE_FEATURE_FLAGS.md` §3 Phase 6**、**§2** 工具相关行通读；缺口保留在 **§2** **`[ ]`** 与 **§6** 基线行，后续迭代逐条收口。 |
| **§2** 与还原树 **路径对照** | **已执行** | 见 **§4** 表（**`src/Tool.ts` / `tools.ts` / `tools/*`** ↔ Go 交付物 ↔ 状态）；细粒度工具目录与 **`PARITY_CHECKLIST.md`**「Phase 6 推进」说明一致。 |
| **§3** **实现进度 / 迭代记录** | **已建立** | 见 **§6**；后续每次可合并增量 **追加行**，并同步 **§2 / §3** 勾选或状态列。 |

**主计划对齐**：目标与设计要点以 **§6.6.1–6.6.6** 为准；工具枚举与架构文档 **§4** 一致。

**迭代中**：执行计划列表 → 逐项实现 → **`make test-phase6`**（或等价 **`go test ./internal/tools/...`**）→ **`git commit`**，直至 **§2 / §3** 与 E2E 收口（见 **`PHASE_ITERATION_RULES.md` §二、§三**）。

---

## 1. 总览与交付物

| 步骤 | 模块 | 还原参考（`src/`） | 交付物（Go） |
|------|------|-------------------|--------------|
| 6.1 | Registry | **`Tool.ts`**（契约类型）、**`tools.ts`**（**`getTools()`** 聚合、MCP 动态增删） | **`internal/tools/registry`**（规划；与 Phase 7 MCP 衔接） |
| 6.2 | 内置工具全集 | **`tools/*`**（各 `*Tool/` 目录，见架构 **§4**） | **`internal/tools/<name>/`**，**`*.go` 基名 ↔ TS 文件** 遵守 **`PHASE_ITERATION_RULES.md` §4.3、§三-3.1** |
| 6.3 | Tool 契约 | 同上 + **`ToolUseContext`**、JSON Schema、Progress | 统一 **`Tool` 接口**（**`Name` / `Schema` / `Run(ctx, input, ToolCtx)`** 等与主计划 **§6.6.2** 一致） |

---

## 2. 功能清单（按域罗列，逐项打勾）

**图例**：**`[x]`** 已达本行可验收程度 · **`[~]`** 部分完成 · **`[ ]`** 未实现。

| 状态 | 编号 | 功能项 | 说明 |
|------|------|--------|------|
| [~] | P6.0.1 | Registry List/ByName/动态 MCP 注册 | **`internal/tools.Tool`** + **`registry.Registry`**（**`ListNames` / `ByName` / `RegisterMCP` / `UnregisterMCP` / `RunTool`→**`query.ToolRunner`**）；**`getTools()`** 全量 feature 排序、**`refreshTools`**、API schema 聚合仍 defer |
| [ ] | P6.1 | 文件：read/write/edit/glob/grep/notebook | 各工具独立单测；Go 侧已有子包占位 / 少量 **prompt/constants**，缺统一 **`Run`** 与 **AC6-1** 矩阵 |
| [ ] | P6.2 | 执行：bash、powershell(平台)、lsp | 沙箱策略可配置；**`querydeps.Bash*`** 为 Phase 5 桥接，本 Phase 对齐 **`BashTool`/`PowerShellTool`/`LSPTool`** 全栈 |
| [ ] | P6.3 | 网络：web_fetch、web_search | 代理与 allowlist |
| [ ] | P6.4 | 任务：task_*、todo、brief、plan、worktree | 与 **`Task.ts` / `tasks/`** 包一致 |
| [ ] | P6.5 | Agent/团队：agent、team、send_message |  |
| [ ] | P6.6 | MCP 封装工具 | list/read resource、调用 MCP |
| [ ] | P6.7 | 其它：ask_user、config、tool_search、tungsten、synthetic、cron 系列等 | 以 **架构 §4** 与 PARITY 为准 |
| [ ] | P6.F.1 | `WEB_BROWSER_TOOL` | 浏览器工具注册与策略（`tools.ts`、`screens/REPL`） |
| [ ] | P6.F.2 | `MONITOR_TOOL` | 监控类工具（Bash/PowerShell/Agent/tasks） |
| [ ] | P6.F.3 | `WORKFLOW_SCRIPTS` | 工作流脚本工具暴露 |
| [ ] | P6.F.4 | `OVERFLOW_TEST_TOOL` | 溢出测试工具（权限/注册） |
| [ ] | P6.F.5 | `FORK_SUBAGENT` | 子代理 fork（`tools/AgentTool`） |
| [ ] | P6.F.6 | `VERIFICATION_AGENT` | 校验代理（Todo/Task/prompts） |
| [ ] | P6.F.7 | `AGENT_MEMORY_SNAPSHOT` | 代理目录快照 |
| [ ] | P6.F.8 | `TERMINAL_PANEL` | 终端面板相关工具/权限 |
| [ ] | P6.F.9 | `BUILTIN_EXPLORE_PLAN_AGENTS` | 内置 explore/plan agents |
| [ ] | P6.F.10 | `EXPERIMENTAL_SKILL_SEARCH` | 技能搜索实验路径 |
| [ ] | P6.F.11 | `MCP_RICH_OUTPUT` | MCP 工具富输出形态 |
| [ ] | P6.F.12 | `COORDINATOR_MODE` | 协调模式允许的工具集 |
| [ ] | P6.F.13 | `KAIROS` | 工具侧 KAIROS 行为 |
| [ ] | P6.F.14 | `KAIROS_BRIEF` | Brief 工具链 |
| [ ] | P6.F.15 | `KAIROS_CHANNELS` | 通道/plan 工具 |
| [ ] | P6.F.16 | `PROACTIVE` | 主动工具侧行为 |
| [ ] | P6.F.17 | `COMMIT_ATTRIBUTION` | git/worktree 归因工具 |
| [ ] | P6.F.18 | `BASH_CLASSIFIER` | Bash 权限分类执行路径 |
| [ ] | P6.F.19 | `TREE_SITTER_BASH` | Bash 解析主路径 |
| [ ] | P6.F.20 | `TREE_SITTER_BASH_SHADOW` | Bash shadow 解析 |
| [ ] | P6.F.21 | `AGENT_TRIGGERS` | cron/触发器工具注册 |
| [ ] | P6.F.22 | `AGENT_TRIGGERS_REMOTE` | 远程触发器 |
| [ ] | P6.F.23 | `UDS_INBOX` | 对等/收件工具（`ListPeersTool` 等；与 Phase 12 协同） |

### 2.1 与 Phase 5 的边界

- **query 循环**已在 Phase 5 通过 **`querydeps` 桩 / 部分真实 Runner** 调用工具名；Phase 6 负责 **工具实现、Schema、权限语义** 与 **registry** 对齐 **`tools.ts` / `Tool.ts`**。
- **全量 Bash 权限 / sandbox** 与 **`PARITY_H9_BASH_PERMISSIONS.md`** 交叉；本 Phase **§2** 勾选须与 H9 文档一致或显式 defer。

---

## 3. 验收标准

| 状态 | 编号 | 要求 |
|------|------|------|
| [ ] | **AC6-1** | **每个工具** ≥3 单测（成功/拒绝/坏输入）。 |
| [~] | **AC6-2** | registry 动态增删 MCP 工具后 query 可见（**`engine.Config.Deps.Tools`** 设为 **`registry.Registry`** 即 **`RunTool`** 可路由到新工具；默认宿主仍 **`BashStubToolRunner`** / **`BashExecToolRunner`**）。 |
| [ ] | **AC6-3** | E2E：**`PHASE06_E2E_ACCEPTANCE.md` §2** fixture 矩阵与 SPEC 附录同步勾选。 |
| [ ] | **AC6-F1**–**AC6-F23** | **§2 `P6.F.*`** 各标志在工具注册/执行路径上行为与 [SOURCE_FEATURE_FLAGS.md](../SOURCE_FEATURE_FLAGS.md) §2 对应行一致，或 PARITY **豁免**说明。 |

**单测入口**：**`make test-phase6`**（与 **`PHASE06_E2E_ACCEPTANCE.md` §1** 一致）。

---

## 4. 与 claude-code-sourcemap 路径对照（`src/`）

**状态说明**：**未创建** = 尚无该 Go 包或仅有目录占位；**部分** = 有源码但未达 **§2** 行描述；**完成** = 本 Phase 范围内可验收。

### 4.1 工具契约与注册

| 还原路径（`src/`） | Go 交付物 | 状态 |
|-------------------|-----------|------|
| **`Tool.ts`** | **`internal/tools`** 包内（或子包）统一 **`Tool` 接口**与类型；与 **`querydeps`** 消费侧衔接 | **未创建**（尚无共享接口包） |
| **`tools.ts`**（**`getTools()`**、feature 门控、MCP 去重） | **`internal/tools/registry`** + **`internal/features`** env | **未创建** |
| **`tools/utils.ts`**、**`tools/shared/*`** | **`internal/tools/shared`**（规划，与 **§三-3.1** 整目录迁移一致） | **未创建** |

### 4.2 按域：架构 §4 工具族 ↔ Go 子包

命名：**`src/tools/GlobTool/`** → **`internal/tools/globtool/`**（**`GlobTool.ts` → `glob_tool.go`** 等，见 **`PHASE_ITERATION_RULES.md` §4.3**）。

| 域（架构 **§4**） | 还原路径（`src/tools/` 下目录） | Go 路径（`internal/tools/`） | 状态 |
|------------------|--------------------------------|-----------------------------|------|
| 文件与代码 | `FileReadTool`、`FileWriteTool`、`FileEditTool`、`GlobTool`、`GrepTool`、`NotebookEditTool` | `filereadtool`、`filewritetool`、`fileedittool`、`globtool`、`greptool`、`notebookedittool` | **部分**（多数为 prompt/constants 或空包，缺 **AC6-1**） |
| 执行环境 | `BashTool`、`PowerShellTool`、`LSPTool` | `bashtool`、`powershelltool`、`lsptool` | **部分**（**`bashtool`/`powershelltool`** 仅 **toolname** 等；**`lsptool`** 占位） |
| 网络 | `WebFetchTool`、`WebSearchTool` | `webfetchtool`、`websearchtool` | **部分**（prompt 级） |
| 任务与规划 | `Task*`、`TodoWriteTool`、`EnterPlanModeTool`、`ExitPlanModeV2Tool`、`BriefTool` 等 | `taskcreatetool`、`taskgettool`、…、`todowritetool`、`enterplanmodetool`、`exitplanmodetool`、`brieftool` 等 | **未创建**（多为 **`.gitkeep` 占位**） |
| Agent / 团队 | `AgentTool`、`TeamCreateTool`、`TeamDeleteTool`、`SendMessageTool` | `agenttool`、`teamcreatetool`、`teamdeletetool`、`sendmessagetool` | **未创建**（占位） |
| MCP | `MCPTool`、`McpAuthTool`、`ListMcpResourcesTool`、`ReadMcpResourceTool` | `mcptool`、`mcpauthtool`、`listmcpresourcestool`、`readmcpresourcetool` | **未创建**（占位） |
| 配置与搜索 | `ConfigTool`、`ToolSearchTool` | `configtool`、`toolsearchtool` | **未创建**（占位） |
| 工作区 | `EnterWorktreeTool`、`ExitWorktreeTool` | `enterworktreetool`、`exitworktreetool` | **未创建**（占位） |
| 技能 / 交互 / 其它 | `SkillTool`、`AskUserQuestionTool`、`SyntheticOutputTool`、`ScheduleCronTool`、`RemoteTriggerTool`、`SleepTool`、`REPLTool` 等 | 同名子包（小写、去连字符） | **未创建**或 **占位**（随 **P6.7** 与 feature 行迭代） |

**TS 文件计数**：**`restored-src/src/tools/`** 下 **约 149** 个 **`.ts`** 文件（含各工具 **prompt/constants**）；迭代某模块时须按 **`PHASE_ITERATION_RULES.md` §三-3.1** 列全量 **`.ts`** 清单后再动 Go 文件。

### 4.3 横切依赖（非本 Phase 包内实现）

| 还原路径 | 说明 |
|----------|------|
| **`hooks/useCanUseTool`**（类型源头）、**`types/permissions.ts`** | Phase 7 权限与 **TUI** 审批；Phase 6 工具 **Run** 须接受可注入 **`CanUseTool`** 或等价物 |
| **`services/mcp/*`** | MCP 客户端；**P6.0.1** / **P6.6** 与之对接 |

---

## 5. 引用

- **迭代规则**：[PHASE_ITERATION_RULES.md](./PHASE_ITERATION_RULES.md)
- **E2E**：`PHASE06_E2E_ACCEPTANCE.md`
- **Feature 全量表**：[SOURCE_FEATURE_FLAGS.md](../SOURCE_FEATURE_FLAGS.md) §2、§3 **Phase 6**
- **架构工具清单**：[CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md](../CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md) §4
- **PARITY 总表**：[PARITY_CHECKLIST.md](../PARITY_CHECKLIST.md)（Phase 6 行随本 Phase 扩展）
- **模块边界**：[ARCHITECTURE_BOUNDARIES.md](../ARCHITECTURE_BOUNDARIES.md)

---

## 6. 实现进度（迭代记录）

**与 §2、§3 状态列同步修订。** 本 Phase 的交付与代码迭代 **统一记在本表**（勿写入 **`PHASE_ITERATION_RULES.md`** 文末修订记录）。

| 日期 | 提交 / 标签 | 摘要 | 后续 |
|------|---------------|------|------|
| **2026-04-01** | —（迭代前准备） | 补 **§0**、扩展 **§1 / §4**（`Tool.ts`·`tools.ts`·`tools/*` ↔ Go）、**§6** 基线；**`PHASE06_E2E_ACCEPTANCE.md` §0**；**`Makefile`** **`test-phase6`**。 | 自 **§三-3.0** 生成有序计划后从 **P6.0.1** 或架构选定的首模块开工 |
| **2026-04-01** | commit | **Phase 6 迭代 1（P6.0.1 子集）**：**`internal/tools`** **`Tool`**；**`internal/tools/registry`**（**`ListNames` / `ByName` / `RegisterMCP` / `UnregisterMCP` / `RunTool`**）；单测 **`registry_test`** 断言 **`query.ToolRunner`**；**`query.Deps`** 注释接线。 | **P6.0.1** 收口 **getTools** 门控 / **P6.1** 首工具 **`Run`** + **AC6-1** |

（后续行：每完成可合并条目追加一行。）

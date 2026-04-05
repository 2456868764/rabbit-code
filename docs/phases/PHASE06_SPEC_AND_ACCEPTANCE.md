# Phase 6 功能规格与验收标准

**Phase 6：工具层（全量）**，主计划 [GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md) **§6.6** 与 [CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md](../CLAUDE_CODE_SOURCEMAP_ARCHITECTURE.md) **§4**。E2E：`PHASE06_E2E_ACCEPTANCE.md`。

**强制**：与 **`claude-code-sourcemap/restored-src/src/`**（简称 **`src/`**）语义对齐；平台豁免依 **[PARITY_CHECKLIST.md](../PARITY_CHECKLIST.md)**、**[ARCHITECTURE_BOUNDARIES.md](../ARCHITECTURE_BOUNDARIES.md)** 及本 SPEC 显式条款。

### 工具实现原则：全量上游对齐（禁止 headless 子集）

Phase 6 中**每一个**内置工具的 Go 实现，都必须以 **`src/tools/<ToolName>/`** 下对应 TypeScript 的**完整行为**为验收基准：与上游 **`call` / `validateInput` / 输入输出 Schema 及分支**对齐（错误类型与面向模型的提示在可比对范围内与 TS 一致）。**不得**把工具实现成或文档化成仅面向无 TUI 的 **headless 子集**（例如「先只做文本路径、其余 defer / stub」）。**Headless** 在本 Phase 只约束**呈现层**（如 **`UI.tsx` → `ui.go`** 可不绑 Ink、不强制复刻终端 UI），**不缩小**工具语义、JSON 结果形态或可执行分支。

**`Read` + query 主循环**：**`internal/query/loop.go`** 对 **`Read`** 调用 **`filereadtool.MapReadResultForMessagesAPI(out, MapReadResultOptions{MainLoopModel: LoopDriver.Model})`**，对齐 **`mapToolResultToToolResultBlockParam`** + **`newMessages`**：**`pdf`** → 摘要 + 跟进 **`document`**；**`parts`** → 摘要 + 跟进多 **`image`**；**`image`** → **`tool_result.content`** 为 **`image`** 块数组；**`text`** → **`AddLineNumbers`** + 可选 **`CyberRiskMitigationReminder`**（**`ShouldIncludeCyberMitigation(MainLoopModel)`**），空内容/offset 越界为 TS 同款 **`system-reminder`** 文案；**`notebook`** → **`mapNotebookCellsToToolResult`** 等价（cell XML + outputs 的 text/image 块、相邻 **`text`** 合并）；**`file_unchanged`** → **`FileUnchangedStub`**。**`Write`**：**`filewritetool.MapWriteToolResultForMessagesAPI`** 对齐 **`mapToolResultToToolResultBlockParam`**（create/update 短文案）；**`query/loop.go`** 已接线。**`Edit`**：**`fileedittool.MapEditToolResultForMessagesAPI`** 对齐 **`mapToolResultToToolResultBlockParam`**（replaceAll / userModified 文案）；**`query/loop.go`** 已接线。**`Glob`**：**`globtool.MapGlobToolResultForMessagesAPI`** + **loop** 接线。**`Grep`**：**`greptool.MapGrepToolResultForMessagesAPI`** + **loop** 接线。**`NotebookEdit`**：**`notebookedittool.MapNotebookEditToolResultForMessagesAPI`** + **loop** 接线。与 TS 的细微差异记在 **§6 / PARITY**。

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
| [~] | P6.1 | 文件：read/write/edit/glob/grep/notebook | **Read**/**Write**：**`FileReadTool.ts`/`limits.ts`/`imageProcessor.ts`/`prompt.ts`/`UI.tsx`** → **`file_read_tool.go`/`limits.go`/`image_processor.go`/`prompt.go`/`ui.go`**（**`doc.go`** 列映射）；**`prompt.ts` `renderPromptTemplate`/`DESCRIPTION`/`MAX_LINES_TO_READ`** → **`RenderReadToolPrompt`**/**常量**；**`Run`** 输入 **JSON `DisallowUnknownFields`**（**`z.strictObject`**）。**`FileWriteTool`**：**`prompt.ts` `getWriteToolDescription`/`DESCRIPTION`** → **`GetWriteToolDescription`/`Description`**；**`UI.tsx`→`ui.go`**；**`Run`** 严格 JSON。**`Edit**：**`fileedittool.FileEdit`** 对齐 **`FileEditTool.ts`** **`validateInput`/`call`**（**`findActualString`**/**`preserveQuoteStyle`**/**`applyEditToFile`**/**`getPatchFromContents`** 等价 **`GetPatchForDisplay`** 于前后全文、**`readFileSyncWithMetadata`** 经 **`filewritetool.ReadNormalizedFileWithContext`**、**`writeTextContent`**、**`ReadFileState`** staleness（**`existsNow && !isUncPath(abs)`** 时 **`criticalEditStaleness`**，与 **`FileWrite`** 对称）、**`.ipynb`→NotebookEdit**、**1 GiB** 上限、**`FindSimilarFile`**、**`validateSettingsFileEdit`**：嵌入 **SchemaStore** **`settings_schema.json`**（与 **`types.ts` `CLAUDE_CODE_SETTINGS_SCHEMA_URL`** 同源；**`make gen-settings-schema`** 刷新）、**`jsonschema/v6`** 校验 + **`regexp2` ECMAScript** 编译 **`pattern`**（**`permissionRule`** 等 lookahead）；根级键白名单对齐 **Zod `.strict()`**；**`mapToolResult`** 已接 **loop**；服务钩复用 **`filewritetool.WriteContext`**。**`normalize_input.go`** 对齐 **`api.ts` `normalizeFileEditInput`**（**`StripTrailingWhitespace`**、**.md/.mdx** 不 strip、**`DesanitizeMatchString`**），在 **`Run`** 内 **`ExpandPath`** 之后执行；**`suggest_cwd.go`** 对齐 **`utils/file.ts` `suggestPathUnderCwd`**，文件不存在时先于 **`FindSimilarFile`** 追加 **Did you mean**；**UNC** 路径跳过本地 **validate**（体积/stat/读盘预检等），**call** 仍读写。**`globtool`**：**`GlobTool.ts`/`UI.tsx`**/**`prompt.ts`** → **`glob_tool.go`/`ui.go`/`prompt.go`**；**`utils/glob.ts`** 行为（**`rg --files`**、**`extractGlobBaseDirectory`**、**`CLAUDE_CODE_GLOB_*`**、**`GlobContext`** 上限/忽略/可选 **`DenyRead`**）合入 **`glob_tool.go`**；**`getGlobExclusionsForPluginCache`** 未接。**`greptool`**：**`GrepTool.ts`/`UI.tsx`/`prompt.ts`** → **`grep_tool.go`/`ui.go`/`prompt.go`**（**`rg`** 参数与 **`mapToolResult`** 对齐；无 **`rg`** 时单测 **Skip**）。**`notebookedittool`**：**`NotebookEditTool.ts`/`UI.tsx`/`prompt.ts`/`constants.ts`** → **`notebook_edit_tool.go`/`ui.go`/`prompt.go`/`constants.go`**（**`src/utils/notebook.ts` `parseCellId`** → **`parse_cell_id.go`**）；**`validateInput`/`call`** 对齐（**Read** 前置、**mtime** staleness、**UNC** 跳过本地校验、**replace→insert** 越界、**nbformat≥4.5** 的 **`cell_id`** 生成/回传；成功响应在 **nbformat 4 且 minor 小于 5** 时省略 **`cell_id`** 与 TS 一致）；**Zod `.strict()`** → 输入 **JSON `DisallowUnknownFields`**；**`MapNotebookEditToolResultForMessagesAPI`** + **loop**；**`toolsearchtool`** 描述用 **`ToolDescription`**。TUI-only **`UI.tsx`** 渲染 headless defer。Glob/Grep 的完整 **`preparePermissionMatcher`** 与宿主 **Phase 7** defer |
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
| [~] | **AC6-1** | **每个工具** ≥3 单测：**Read**（**`file_read_tool_test`** + **`prompt_test`**；严格 JSON 未知字段）、**Write**（**`prompt_test`**；严格 JSON）、**Edit**（**`prompt_test`**；严格 JSON）、**Glob**（**`glob_tool_test`**；无 **`rg`** 时部分用例 **Skip**）、**Grep**（**`grep_tool_test`**；无 **`rg`** 时 **Skip**）、**NotebookEdit**（**`notebook_edit_tool_test`** / **`parse_cell_id_test`**；replace/insert/map/严格 JSON/**nbformat** 分支）已覆盖；**Edit** 另含 **`normalize_input_test`** / **`suggest_cwd_test`** / **`settings_validate_test`** / **`prompt_test`**；其余内置工具按 **§2** 逐包收口。 |
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
| 文件与代码 | `FileReadTool`、`FileWriteTool`、`FileEditTool`、`GlobTool`、`GrepTool`、`NotebookEditTool` | `filereadtool`、`filewritetool`、`fileedittool`、`globtool`、`greptool`、`notebookedittool` | **部分**（**`filereadtool`** / **`filewritetool`** / **`fileedittool`** / **`globtool`** / **`greptool`** / **`notebookedittool`**：**`call`+`mapToolResult`** + **loop**（**Glob**/**Grep** 依赖 **`rg`**）；Glob **plugin-cache exclusions**、权限 matcher 全量等与 **§2 P6.1** 一致 defer） |
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
| **2026-04-01** | commit | **Phase 6 迭代 2（P6.1 / FileReadTool）**：**`filereadtool`** **`limits` / `image_processor` / `ui` / `file_read_tool`** + **`FileRead.Run`**（文本、**`offset`/`limit`**、设备路径拒绝、扩展名二进制拒绝）；**`registry_test`** **`Read`** 路由；**AC6-1**/**P6.1** 标 **`[~]`**。 | **P6.1** 其余工具；Read 的 PDF/图/权限/dedup/API tokenizer |
| **2026-04-01** | commit | **Phase 6 迭代 3（P6.1 / Read 全量）**：**`filereadtool`** 对齐 **`FileReadTool.ts`**：**`notebook.go`**、**`image_read.go`**（**`golang.org/x/image`**）、**`pdf.go`**（**`pdfcpu`** 页数 + **`pdftoppm`** **`parts`**）、**`validate`/`token`/`path`**；**`WithRunContext`** 限额与 dedup；删除 **`image_processor` defer**；单测含 **`.ipynb`/`.png`**。 | **P6.1** 其余文件工具 |
| **2026-04-01** | commit | **Phase 6 迭代 4（Read → transcript / Messages API）**：**`filereadtool/map_api_message.go`**（**`MapReadResultForMessagesAPI`**、**`PartsOutputDirToImageContentBlocks`**）；**`query.AppendUserMessageContentBlocks`**；**`LoopDriver`** 每轮工具结果对 **`Read`** 附加 **`document`/`image`** 跟进 **`user`** 消息。 | **`Read` 文本/notebook** 与 TS **`mapToolResult`** 字符串形态对齐；其它工具的 **`mapToolResult`** 层 |
| **2026-04-01** | commit | **Phase 6 迭代 5（Read `mapToolResult` 文本/notebook）**：**`MapReadResultOptions{MainLoopModel}`**；**`text`**/**`notebook`**/**`file_unchanged`** 分支；**`query/loop.go`** 传入 **`d.Model`**；单测覆盖行号/cyber/空文件/offset、notebook 合并、**`FileUnchangedStub`**。 | **P6.1** **`Write`/`Edit`/glob/grep** 等文件工具；其它内置工具的 **`mapToolResult`** |
| **2026-04-01** | commit | **Phase 6 迭代 6（P6.1 / Write 核心）**：**`internal/tools/filewritetool`**（**`FileWrite.Run`**、**`WriteContext`**、**`StructuredPatchFullReplace`**、staleness）；**`file_write_tool_test`**（新建/无 state 失败/有 state 更新/deny/坏 JSON）；**`registry_test`** **Read+Write** 集成。 | **Write** 的 **`mapToolResult`** 短文案（**`query/loop.go`**）；**Edit**/glob/grep/notebook |
| **2026-04-01** | commit | **Phase 6 迭代 7（Write 与上游全量对齐）**：**`validateInput`/`call`** staleness 分两段；**`GetPatchForDisplay`**（**`diff`/`convertLeadingTabsToSpaces`/escape）；**`WriteContext`** 扩展（**`CheckTeamMemSecrets`**、**`BeforeFileEdited`**、**`AfterWrite`**、**`FileHistoryTrack`**、**`FetchGitDiff`**）；**`MapWriteToolResultForMessagesAPI`** + **loop** 接线；单测含 **mtime 严格校验**、**gitDiff**、**map**、patch。 | **Edit**/glob/grep/notebook；其它内置工具 **`mapToolResult`** |
| **2026-04-01** | commit | **Phase 6 迭代 8（P6.1 / Edit）**：**`internal/tools/fileedittool`**（**`FileEdit.Run`**、**`utils.go`** 对齐 **`FileEditTool/utils.ts`**、**`settings_validate`**、**`MapEditToolResultForMessagesAPI`**）；**`filewritetool.ReadNormalizedFileWithContext`**、**`EncodeTextToFileBytes`**、**`EncUTF8`/`EncUTF16LE`**、**`UserModified`**；**`loop.go`** **Edit** **`mapToolResult`**；**`registry_test`** **Edit**；**AC6-1** / **§2** 更新。 | **Glob**/**Grep**/**NotebookEdit**；**Edit** 与 TS **AJV settings** 差异常驻 **PARITY** |
| **2026-04-01** | commit | **Phase 6 迭代 9（P6.1 / Edit 补齐）**：**`normalize_input.go`** / **`suggest_cwd.go`** 接线 **`Run`**；**UNC** 与 **`FileWrite`** 一致的本地跳过 + **staleness** 门控；**`normalize_input_test`**、**`suggest_cwd_test`**、**`TestFileEdit_notFoundIncludesSuggestPathUnderCwd`**；**§2 P6.1** / **AC6-1** / **§4.2** 文档同步。 | **Glob**/**Grep**/**NotebookEdit**；**Edit** 与 TS **Zod settings** 细微差异（SchemaStore 版本漂移等） |
| **2026-04-01** | commit | **Phase 6 迭代 10（P6.1 / settings.json 校验）**：**`settings_schema.json`**（**`gen-settings-schema`**）、**`settings_validate.go`**（**`jsonschema/v6`** + **`regexp2`**）、**`settings_validate_test`**；**§2 / §4.2 / §6** 文档。 | **Glob**/**Grep**/**NotebookEdit** |
| **2026-04-05** | commit | **Phase 6 迭代 11（P6.1 / GlobTool）**：**`globtool`** **`glob_tool.go`**（**`validateInput`/`call`** 对齐 **`GlobTool.ts`**；**`utils/glob.ts`** 合入）、**`ui.go`**、**`prompt.go`**、**`doc.go`**；**`query/loop.go`** **`MapGlobToolResultForMessagesAPI`**；**`glob_tool_test`**；删占位 **`run_context.go`**/**`.gitkeep`**；**§2/§3/§4/§6** 同步。 | **Grep**/**NotebookEdit**；Glob **plugin-cache exclusions**、完整 **`checkReadPermissionForTool`** 与 TS **`preparePermissionMatcher`** 仍宿主/Phase 7 |
| **2026-04-05** | commit | **Phase 6 迭代 12（P6.1 / GrepTool）**：**`greptool`** **`GrepTool.ts`→`grep_tool.go`**、**`UI.tsx`→`ui.go`**、**`prompt.ts`→`prompt.go`** 与 **`doc.go`** 映射说明；**`MapGrepToolResultForMessagesAPI`**（content/count/files、分页文案）；**`grep_tool_test`** 扩充 **map** / **validate**；**§2/§3/§4/§6** 与 **E2E §1** 同步。 | **NotebookEdit** 单测矩阵；Glob/Grep **preparePermissionMatcher** / plugin-cache |
| **2026-04-06** | commit | **Phase 6 迭代 13（P6.1 / NotebookEditTool）**：**`map_notebook_edit_message.go`→`ui.go`**；**`prompt.go`**（**`ToolDescription`/`ToolPrompt`**）；**`doc.go`**；输入 **严格 JSON**；成功输出 **`cell_id`** 与 **nbformat 4.5+** 规则对齐 TS；**`notebook_edit_tool_test`** 扩充；**`toolsearchtool`** 引用 **`ToolDescription`**；**§2/§3/§6** 与 **E2E §1** 同步。 | **P6.2** 起执行类工具；Glob/Grep **preparePermissionMatcher** / plugin-cache |
| **2026-04-06** | commit | **Phase 6 迭代 14（P6.1 / Read·Write·Edit 再对齐）**：**`image_read.go`→`image_processor.go`**；**`map_write_message.go`/`map_edit_message.go`→`ui.go`**；**`filereadtool` `prompt.go`**（**`RenderReadToolPrompt`**/**`Description`** 等）；**`CompactLinePrefixEnabled`**；**`filewritetool`/`fileedittool` `prompt.go`**（**`GetWriteToolDescription`/`GetEditToolPrompt`/`ShortDescription`**）；**Read**/**Write**/**Edit** **`Run`** **`DisallowUnknownFields`**；**`doc.go`** 三包；**`toolsearchtool`** 目录引用上游文案；单测 **strict JSON** + **prompt_test**。 | GrowthBook **`tengu_compact_line_prefix_killswitch`** / **`getDefaultFileReadingLimits`** 动态段落仍简化；**P6.2** |

（后续行：每完成可合并条目追加一行。）

# 还原树 `feature('…')` 标志 ↔ Go 迁移映射

本文档将 **`claude-code-sourcemap/restored-src/src`** 中通过 **`feature('FLAG')`** 门控的能力（构建期 DCE，由 `bun:bundle` 等注入）映射到 **rabbit-code** 的 **Go 包**与 **Phase**。与第三方盘点 [Claude Code: Hidden & Unreleased Features Report](https://aired.sh/p/Zlm4dmW4ED) 对照使用；**以还原源码与下表为准**，报告仅作索引参考。

---

## 1. Go 侧对等策略（强制规划）

| 策略 | 说明 |
|------|------|
| **显式注册表** | 使用 `internal/features`（或等价包）存放 **标志名常量** + **默认值**；禁止散落魔法字符串。 |
| **编译期裁剪** | 对标 TS 的 DCE：可选用 **`//go:build`** 标签 **或** 发布流水线用 **`-ldflags -X`** 注入 `features.Enabled(FLAG)=true/false`；文档须说明 **发行版 / 企业版 / 开源版** 各组合。 |
| **运行时覆盖** | 与 TS 中 `GrowthBook`、环境变量并列的行为，在 Go 中进入 **`internal/config`** + **`internal/telemetry`**，**不得**与核心 `query` 逻辑隐式耦合。 |
| **Phase 1 已实现 env** | `RABBIT_CODE_HARD_FAIL`、`RABBIT_CODE_SLOW_OPERATION_LOGGING`（可选 `RABBIT_CODE_SLOW_LOG_FILE`）、`RABBIT_CODE_FILE_PERSISTENCE`、`RABBIT_CODE_LODESTONE`（实现见 **`internal/features`**）；**Undercover**：`CLAUDE_CODE_UNDERCOVER`（`internal/app`，非下表 `feature()` 行）。 |
| **验收** | 每个非纯内部调试标志：在对应 Phase 的 SPEC **§2 / §3**（`P#.F.*`、`AC#-F*`）中有条目；在 **PARITY** 或本表维护 **`单测 ID`/`E2E ID`**（可后填）。 |

**非 `feature()` 机制**（须单独对标，不占用下表行）：`utils/undercover.ts`（`CLAUDE_CODE_UNDERCOVER`）、`USER_TYPE === 'ant'` 门控、HTTP **beta 头**（`constants/betas.ts` / `services/api`）。

---

## 2. 全量标志表（主责 Phase + 还原路径示例）

「主责 Phase」表示 **该标志核心业务** 应在该 Phase **完成对标或文档豁免**；跨 Phase 引用是正常的。

| 标志 | 还原路径（示例） | 规划 Go 包 | 主责 Phase |
|------|------------------|------------|------------|
| `KAIROS` | `main.tsx`, `assistant/*`, `query.ts`, `tools/AgentTool/*`, `screens/REPL.tsx`, `memdir/*`, `services/compact/*` | `internal/app`, `internal/engine`, `internal/tools/agent`, `internal/tui`, `internal/memdir` | 5 / 6 / 9 / 11 |
| `PROACTIVE` | `systemPrompt.ts`, `sessionStorage.ts`, `tools/AgentTool`, `screens/REPL.tsx`, `services/compact/prompt.ts` | `internal/engine`, `internal/session`, `internal/tools` | 5 / 8 / 9 / 11 |
| `KAIROS_BRIEF` | `tools/BriefTool/*`, `utils/permissions/permissionRuleParser.ts`, `utils/conversationRecovery.ts` | `internal/tools/brief`, `internal/permissions`, `internal/session` | 6 / 7 / 8 |
| `KAIROS_CHANNELS` | `utils/messages.ts`, `utils/messageQueueManager.ts`, `services/mcp/*`, plan 工具 | `internal/messages`, `internal/mcp`, `internal/tools` | 3 / 6 / 7 |
| `KAIROS_DREAM` | `skills/bundled/index.ts` | `internal/skills` | 11 |
| `KAIROS_GITHUB_WEBHOOKS` | `tools.ts`, `commands.ts`, `hooks/useReplBridge.tsx` | `internal/tools`, `internal/cli`, `internal/bridge` | 10 / 11 / 12 |
| `KAIROS_PUSH_NOTIFICATION` | `tools/ConfigTool/supportedSettings.ts`, `components/Settings/Config.tsx` | `internal/config`, `internal/tui` | 2 / 9 |
| `COORDINATOR_MODE` | `main.tsx`, `coordinator/*`, `utils/toolPool.ts`, `utils/sessionRestore.ts` | `internal/engine`, `internal/tools`, `internal/session` | 5 / 6 / 8 / 11 |
| `FORK_SUBAGENT` | `tools/AgentTool/forkSubagent.ts`, `tools/ToolSearchTool/prompt.ts` | `internal/tools/agent` | 6 / 11 |
| `VERIFICATION_AGENT` | `tools/TodoWriteTool`, `tools/TaskUpdateTool`, `constants/prompts.ts` | `internal/tools`, `internal/engine` | 6 / 11 |
| `AGENT_MEMORY_SNAPSHOT` | `tools/AgentTool/loadAgentsDir.ts`, `main.tsx` | `internal/tools/agent` | 6 / 11 |
| `AGENT_TRIGGERS` | `tools/ScheduleCronTool`, `tools.ts`, `screens/REPL.tsx` | `internal/tools`, `internal/tasks`, `internal/tui` | 6 / 11 |
| `AGENT_TRIGGERS_REMOTE` | `tools.ts`, `skills/bundled/index.ts` | `internal/tools`, `internal/tasks` | 6 / 11 |
| `ULTRATHINK` | `utils/thinking.ts` | `internal/query` 或 `internal/engine` | 5 |
| `ULTRAPLAN` | `utils/processUserInput`, `screens/REPL.tsx` | `internal/query`, `internal/tui` | 5 / 9 |
| `TOKEN_BUDGET` | `query.ts`, `utils/attachments.ts`, `screens/REPL.tsx` | `internal/query`, `internal/tui` | 5 / 9 |
| `EXTRACT_MEMORIES` | `utils/backgroundHousekeeping.ts`, `services/extractMemories/*` | `internal/session` 或 `internal/app` | 8 / 12 |
| `TEAMMEM` | `utils/sessionFileAccessHooks.ts`, `utils/collapseReadSearch.ts`, `utils/claudemd.ts`, `utils/config.ts` | `internal/session`, `internal/memdir`, `internal/config` | 2 / 8 |
| `MEMORY_SHAPE_TELEMETRY` | `utils/sessionFileAccessHooks.ts` | `internal/telemetry` | 12 |
| `CACHED_MICROCOMPACT` | `services/compact/microCompact.ts`, `services/api/claude.ts`, `query.ts` | `internal/compact`, `internal/anthropic`, `internal/query` | 4 / 5 |
| `REACTIVE_COMPACT` | `utils/analyzeContext.ts`, `services/compact/autoCompact.ts`, `query.ts` | `internal/compact`, `internal/query` | 5 |
| `CONTEXT_COLLAPSE` | `query.ts`, `utils/sessionRestore.ts`, `screens/REPL.tsx`, `tools.ts` | `internal/query`, `internal/session`, `internal/tools` | 5 / 6 / 8 / 9 |
| `COMPACTION_REMINDERS` | `utils/attachments.ts` | `internal/messages` | 3 |
| `CONNECTOR_TEXT` | `utils/messages.ts`, `services/api/claude.ts`, `constants/betas.ts` | `internal/messages`, `internal/anthropic` | 3 / 4 |
| `ANTI_DISTILLATION_CC` | `services/api/claude.ts` | `internal/anthropic` | 4 |
| `NATIVE_CLIENT_ATTESTATION` | `constants/system.ts` | `internal/anthropic` | 4 / 12 |
| `BASH_CLASSIFIER` | `utils/permissions/*`, `utils/messages.ts`, `tools/BashTool/*` | `internal/permissions`, `internal/tools/bash` | 6 / 7 |
| `TRANSCRIPT_CLASSIFIER` | `utils/permissions/*`, `utils/settings/*`, `utils/attachments.ts` | `internal/permissions`, `internal/config`, `internal/messages` | 2 / 3 / 7 |
| `TREE_SITTER_BASH` | `utils/bash/parser.ts` | `internal/tools/bash` | 6 |
| `TREE_SITTER_BASH_SHADOW` | `utils/bash/parser.ts` | `internal/tools/bash` | 6 |
| `BUDDY` | `buddy/*`, `utils/attachments.ts` | `internal/tui` | 9 |
| `VOICE_MODE` | `voice/*`, `hooks/useVoiceIntegration.tsx`, `services/voiceStreamSTT.ts` | `internal/voice`, `internal/tui` | 9 / 12 |
| `AUTO_THEME` | `tools/ConfigTool/supportedSettings.ts` | `internal/config`, `internal/tui/theme` | 2 / 9 |
| `HISTORY_PICKER` | `hooks/useHistorySearch.ts` | `internal/history`, `internal/tui` | 8 / 9 |
| `HISTORY_SNIP` | `utils/messages.ts`, `utils/collapseReadSearch.ts` | `internal/messages`, `internal/query` | 3 / 5 |
| `STREAMLINED_OUTPUT` | `cli/print.ts` | `internal/cli` | 10 |
| `MESSAGE_ACTIONS` | `screens/REPL.tsx`, `keybindings/defaultBindings.ts` | `internal/tui` | 9 |
| `QUICK_SEARCH` | `keybindings/defaultBindings.ts` | `internal/tui`, `internal/history` | 8 / 9 |
| `TERMINAL_PANEL` | `utils/permissions/classifierDecision.ts`, `tools.ts`, `keybindings/*` | `internal/tools`, `internal/tui` | 6 / 9 |
| `EXPERIMENTAL_SKILL_SEARCH` | `utils/attachments.ts`, `tools/SkillTool/*` | `internal/skills`, `internal/tools` | 6 / 11 |
| `MCP_SKILLS` | `services/mcp/client.ts` | `internal/mcp` | 7 |
| `MCP_RICH_OUTPUT` | `tools/MCPTool/UI.tsx` | `internal/tools/mcp`, `internal/tui` | 6 / 9 |
| `SKILL_IMPROVEMENT` | `utils/hooks/skillImprovement.ts` | `internal/skills` | 11 |
| `RUN_SKILL_GENERATOR` | `skills/bundled/index.ts` | `internal/skills` | 11 |
| `MONITOR_TOOL` | `tools/BashTool`, `tools/PowerShellTool`, `tools/AgentTool`, `tasks/*` | `internal/tools`, `internal/tasks`；**`bashtool`**：**`RABBIT_MONITOR_TOOL`** → **`features.MonitorToolEnabled`**（**`validateInput` sleep**） | 6 / 11 |
| `WEB_BROWSER_TOOL` | `tools.ts`, `screens/REPL.tsx`, `main.tsx` | `internal/tools`, `internal/tui` | 6 / 9 |
| `BUILTIN_EXPLORE_PLAN_AGENTS` | `tools/AgentTool/builtInAgents.ts` | `internal/tools/agent` | 6 / 11 |
| `BRIDGE_MODE` | `main.tsx`, `hooks/useReplBridge.tsx`, `entrypoints/cli.tsx` | `internal/bridge`, `internal/cli`, `internal/tui` | 9 / 10 / 12 |
| `BG_SESSIONS` | `utils/conversationRecovery.ts`, `utils/concurrentSessions.ts` | `internal/session` | 8 |
| `CCR_AUTO_CONNECT` | `utils/config.ts` | `internal/config`, `internal/bridge` | 2 / 12 |
| `CCR_MIRROR` | `main.tsx`, `bridge/*` | `internal/bridge` | 12 |
| `CCR_REMOTE_SETUP` | `commands.ts` | `internal/cli` | 10 |
| `DIRECT_CONNECT` | `main.tsx` | `internal/cli`, `internal/server` | 10 |
| `SSH_REMOTE` | `main.tsx` | `internal/cli`, `internal/remote` | 10 |
| `DAEMON` | `entrypoints/cli.tsx`, `commands.ts` | `internal/cli` | 10 |
| `SELF_HOSTED_RUNNER` | `entrypoints/cli.tsx` | `internal/cli` | 10 |
| `UDS_INBOX` | `utils/messages/systemInit.ts`, `tools/SendMessageTool/*`, `main.tsx` | `internal/messages`, `internal/tools`, `internal/bridge` | 3 / 6 / 12 |
| `ABLATION_BASELINE` | `entrypoints/cli.tsx` | `internal/cli` | 10 |
| `DUMP_SYSTEM_PROMPT` | `entrypoints/cli.tsx` | `internal/cli` | 10 |
| `ENHANCED_TELEMETRY_BETA` | `utils/telemetry/sessionTracing.ts` | `internal/telemetry` | 12 |
| `HARD_FAIL` | `utils/log.ts`, `main.tsx` | `internal/app` | 1 |
| `OVERFLOW_TEST_TOOL` | `utils/permissions/classifierDecision.ts`, `tools.ts` | `internal/permissions`, `internal/tools` | 6 / 7 |
| `PERFETTO_TRACING` | `utils/telemetry/perfettoTracing.ts` | `internal/telemetry` | 12 |
| `SHOT_STATS` | `utils/stats.ts`, `components/Stats.tsx` | `internal/telemetry`, `internal/tui` | 9 / 12 |
| `SLOW_OPERATION_LOGGING` | `utils/slowOperations.ts` | `internal/app`, `internal/telemetry` | 1 / 12 |
| `TORCH` | `commands.ts` | `internal/cli` | 10 |
| `DOWNLOAD_USER_SETTINGS` | `services/settingsSync/index.ts` | `internal/config` | 2 |
| `UPLOAD_USER_SETTINGS` | `services/settingsSync/index.ts`, `main.tsx` | `internal/config`, `internal/app` | 2 |
| `NEW_INIT` | `commands/init.ts` | `internal/cli` | 10 |
| `TEMPLATES` | `utils/markdownConfigLoader.ts`, `query/stopHooks.ts`, `entrypoints/cli.tsx` | `internal/config`, `internal/query`, `internal/cli` | 2 / 5 / 10 |
| `REVIEW_ARTIFACT` | `skills/bundled`, `components/permissions/*` | `internal/skills`, `internal/permissions` | 7 / 11 |
| `WORKFLOW_SCRIPTS` | `tools.ts`, `tasks/*`, `components/tasks/*`, `commands.ts` | `internal/tools`, `internal/tasks`, `internal/cli` | 6 / 10 / 11 |
| `PROMPT_CACHE_BREAK_DETECTION` | `services/compact/*`, `services/api/claude.ts`, `tools/AgentTool` | `internal/compact`, `internal/anthropic`, `internal/query`（**H1** SSE trim+resend / optional auto-compact）, `internal/tools` | 4 / 5 / 6 |
| `BREAK_CACHE_COMMAND` | `context.ts` | `internal/engine` 或 `internal/query` | 5 |
| `CHICAGO_MCP` | `utils/computerUse/*`（注释门控） | `internal/mcp` | 7 |
| `LODESTONE` | `utils/backgroundHousekeeping.ts`, `main.tsx` | `internal/app`, `internal/bridge` | 1 / 12 |
| `UNATTENDED_RETRY` | `services/api/withRetry.ts` | `internal/anthropic` | 4 |
| `FAST_RETRY`（运行时 env：`CLAUDE_CODE_FAST_RETRY` / **`RABBIT_CODE_FAST_RETRY`**） | `services/api/withRetry.ts`（快路径退避） | `internal/features`、`internal/anthropic` | 4 |
| **529 前景白名单**（运行时 env：**`RABBIT_CODE_STRICT_FOREGROUND_529`** → **`DefaultPolicy().StrictForeground529`**） | `services/api/withRetry.ts` **`FOREGROUND_529_RETRY_SOURCES`** | `internal/features`、`internal/anthropic` | 4 |
| **OAuth beta 字符串**（**`BetaOAuth`** / `constants/oauth.ts` **`OAUTH_BETA_HEADER`**，供 HTTP **`anthropic-beta`** 拼装） | `constants/oauth.ts`、`services/api/client.ts` | `internal/anthropic`（`betas.go`） | 4 |
| `POWERSHELL_AUTO_MODE` | `utils/permissions/*` | `internal/permissions` | 7 |
| `IS_LIBC_GLIBC` | `utils/envDynamic.ts` | `internal/app` | 1 |
| `IS_LIBC_MUSL` | `utils/envDynamic.ts` | `internal/app` | 1 |
| `NATIVE_CLIPBOARD_IMAGE` | `utils/imagePaste.ts` | `internal/tui` 或 `internal/app` | 9 / 12 |
| `BYOC_ENVIRONMENT_RUNNER` | `entrypoints/cli.tsx` | `internal/cli` | 10 |
| `BUILDING_CLAUDE_APPS` | `skills/bundled/index.ts` | `internal/skills` | 11 |
| `FILE_PERSISTENCE` | `utils/filePersistence/*` | `internal/session` 或 `internal/app` | 1 / 8 |
| `HOOK_PROMPTS` | `screens/REPL.tsx` | `internal/tui`, `internal/engine` | 5 / 9 |
| `COWORKER_TYPE_TELEMETRY` | `services/analytics/metadata.ts` | `internal/telemetry` | 12 |
| `COMMIT_ATTRIBUTION` | `utils/worktree.ts`, `utils/attribution.ts`, `utils/sessionRestore.ts`, `screens/REPL.tsx` | `internal/session`, `internal/tools`, `internal/tui` | 6 / 8 / 9 |
| `AWAY_SUMMARY` | `hooks/useAwaySummary.ts`, `screens/REPL.tsx` | `internal/tui` | 9 |
| `ALLOW_TEST_VERSIONS` | `utils/nativeInstaller/download.ts` | `internal/cli` 或 `internal/app` | 10 |

---

## 3. 按 Phase 速查（勾选规划用）

| Phase | 标志（节选） |
|-------|----------------|
| **0** | 维护本文档 + 各 Phase SPEC 中「feature」节；`internal/features` 骨架可选 |
| **1** | `HARD_FAIL`, `SLOW_OPERATION_LOGGING`, `IS_LIBC_*`, `FILE_PERSISTENCE`（早期）, `LODESTONE`（交互注册前置）, 对标 `undercover` 行为文档 |
| **2** | `AUTO_THEME`, `DOWNLOAD_USER_SETTINGS`, `UPLOAD_USER_SETTINGS`, `TEAMMEM`（类型/路径）, `TRANSCRIPT_CLASSIFIER`（设置 schema）, `TEMPLATES`（配置键）, `KAIROS_PUSH_NOTIFICATION`（设置项） |
| **3** | `HISTORY_SNIP`, `CONNECTOR_TEXT`, `COMPACTION_REMINDERS`, `KAIROS`/`KAIROS_CHANNELS`/`KAIROS_BRIEF`（消息形态）, `UDS_INBOX`（消息路由说明） |
| **4** | `ANTI_DISTILLATION_CC`, `CACHED_MICROCOMPACT`, `PROMPT_CACHE_BREAK_DETECTION`, `UNATTENDED_RETRY`, `NATIVE_CLIENT_ATTESTATION`, beta 头 |
| **5** | `TOKEN_BUDGET`, `REACTIVE_COMPACT`, `CONTEXT_COLLAPSE`, `ULTRATHINK`, `ULTRAPLAN`, `BREAK_CACHE_COMMAND`, `TEMPLATES`（job classifier） |
| **6** | `WEB_BROWSER_TOOL`, `MONITOR_TOOL`, `WORKFLOW_SCRIPTS`, `OVERFLOW_TEST_TOOL`, `FORK_SUBAGENT`, `VERIFICATION_AGENT`, `AGENT_MEMORY_SNAPSHOT`, `TERMINAL_PANEL`, `BUILTIN_EXPLORE_PLAN_AGENTS`, `EXPERIMENTAL_SKILL_SEARCH`, `MCP_RICH_OUTPUT`（工具输出）, `KAIROS*` 工具侧, `COMMIT_ATTRIBUTION`（git）, `BASH_CLASSIFIER`/`TREE_SITTER_*` 执行路径 |
| **7** | `TRANSCRIPT_CLASSIFIER`, `BASH_CLASSIFIER`, `POWERSHELL_AUTO_MODE`, `MCP_SKILLS`, `CHICAGO_MCP`, `REVIEW_ARTIFACT`（审批） |
| **8** | `TEAMMEM`, `MEMORY_SHAPE_TELEMETRY`, `BG_SESSIONS`, `EXTRACT_MEMORIES`, `COMMIT_ATTRIBUTION`, `CONTEXT_COLLAPSE`（恢复）, `HISTORY_PICKER`/`QUICK_SEARCH`（数据层） |
| **9** | `BUDDY`, `VOICE_MODE`, `MESSAGE_ACTIONS`, `QUICK_SEARCH`, `HISTORY_PICKER`, `TERMINAL_PANEL`, `ULTRAPLAN`, `WEB_BROWSER_TOOL`（面板）, `PROACTIVE`/`KAIROS` REPL, `AWAY_SUMMARY`, `HOOK_PROMPTS`, `BRIDGE_MODE`（UI）, `SHOT_STATS`（展示）, `AUTO_THEME` |
| **10** | `DAEMON`, `DIRECT_CONNECT`, `SSH_REMOTE`, `TORCH`, `DUMP_SYSTEM_PROMPT`, `BYOC_ENVIRONMENT_RUNNER`, `SELF_HOSTED_RUNNER`, `NEW_INIT`, `CCR_REMOTE_SETUP`, `STREAMLINED_OUTPUT`, `ALLOW_TEST_VERSIONS`, `TEMPLATES` 子命令, `WORKFLOW_SCRIPTS` 子命令, `ABLATION_BASELINE` |
| **11** | `KAIROS*`, `COORDINATOR_MODE`, `PROACTIVE`, `AGENT_TRIGGERS*`, `KAIROS_DREAM`, `SKILL_IMPROVEMENT`, `RUN_SKILL_GENERATOR`, `BUILDING_CLAUDE_APPS`, `KAIROS_GITHUB_WEBHOOKS`, `WORKFLOW_SCRIPTS`, `MONITOR_TOOL`, `VERIFICATION_AGENT` |
| **12** | `BRIDGE_MODE`, `CCR_AUTO_CONNECT`, `CCR_MIRROR`, `UDS_INBOX`, `VOICE_MODE`（服务）, `ENHANCED_TELEMETRY_BETA`, `PERFETTO_TRACING`, `COWORKER_TYPE_TELEMETRY`, `MEMORY_SHAPE_TELEMETRY`, `NATIVE_CLIPBOARD_IMAGE`, `FILE_PERSISTENCE`, `LODESTONE`（协议） |

---

## 4. 维护

- **主计划**：[GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](./GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md) §2.6。  
- **阶段 SPEC**：各 `docs/phases/PHASEXX_SPEC_AND_ACCEPTANCE.md` **§2 功能清单**（`P#.F.*` 等）与 **§3 验收标准**（`AC#-F*` 等）列出本 Phase 标志子集。  
- 还原树新增 `feature('X')` 时：**先增本表一行**，再更新对应 Phase SPEC 的 **§2 / §3**。

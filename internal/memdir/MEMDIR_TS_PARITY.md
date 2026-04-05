# `src/memdir` 模块 ↔ `internal/memdir`（§3.1 模块级对齐）

**规则**：`docs/phases/PHASE_ITERATION_RULES.md` **§3.1** — 以 **`restored-src/src/<area>/`** 整目录为交付边界；Go **`*.go`** 基名与 TS **`*.ts`** 一一 **`camelCase` → `snake_case`**；禁止把**同一 TS 文件**拆成多个按小功能命名的 `.go`；**多 TS 合并为一个 `.go`** 时须在本表写明且文件名取主模块。

---

## §3.0 有序迭代计划（`PHASE_ITERATION_RULES.md` §三-3.0）

由 **本 PARITY 表 `[~]` / `[ ]`**、**`PHASE05_CONTINUATION.md` H8**、**`restored-src/src/memdir/*.ts`** 导出；**严格按序**执行；完成一项即更新下表 **状态** 与上表符号列。排序与 **`PARITY` + §3.2-3** 冲突时以强制规则为准并回写说明。

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`findRelevantMemories`**：TS 形参 ↔ **`FindRelevantMemoriesClassic`**（`ctx`↔`AbortSignal`）+ 通用 **`FindRelevantMemoriesOpts`** | 下表 **`findRelevantMemories`** 为 **[x]**；**`go test ./internal/memdir/... -short`** |
| 2 | ☑ | **`memoryScan`**：**`MemoryHeader` / `scanMemoryFiles`** — `description: null` ↔ **`Description == ""`**；**`AbortSignal` ↔ `context`**；**`FormatMemoryManifest`** 与 TS 一致（单测 **`TestFormatMemoryManifest_noDescriptionLikeTSNull`**） | 下表 **`MemoryHeader` / `scanMemoryFiles`** 为 **[x]** |
| 3 | ☑ | **`memdir.ts`**：**`loadMemoryPrompt` / `buildMemoryPrompt` / `buildSearchingPastContextSection`** — 见下 **「§3.0 序 3–4：`memdir.ts`」** | 下表 **`memdir.ts`** 行为 **[x]** |
| 4 | ☑ | **`paths.ts`**：**门控与解析** — 见下 **「§3.0 序 3–4：`paths.ts`」** | 下表 **`paths.ts`** 余项 **[x]** |

## §3.2 单次迭代核对（`PHASE_ITERATION_RULES.md` §三-3.2）

每轮迭代按仓库强制规则执行，不另向发起人征集排序。

| 阶段 | 内容 |
|------|------|
| **3.2-1 P1** | 还原树 **`claude-code-sourcemap/restored-src/src/memdir/*.ts`** 可打开；范围在本文件 / Phase SPEC **§4** / **§6** 有记录。 |
| **3.2-1 P2** | 改前 **`go test ./... -short`** 绿；本 PARITY 主表与映射行完整。 |
| **3.2-2 #1** | 对照 **§3.0 当前首项**：改 Go / 接线；**`go build ./...`**；改动包 **`go test -short`** 绿。 |
| **3.2-2 #2** | 无 import 环；**README / doc.go / PARITY** 与路径一致（**§3.2-3**）。 |
| **3.2-2 #3** | 全量 **`go test ./... -short`**；**一增量一提交**（**§三-3**）。 |

---

## §3.1-1 上游 TS 清单（`src/memdir/`，8 个，无子目录）

| # | `restored-src/src/memdir/*.ts` |
|---|--------------------------------|
| 1 | `findRelevantMemories.ts` |
| 2 | `memdir.ts` |
| 3 | `memoryAge.ts` |
| 4 | `memoryScan.ts` |
| 5 | `memoryTypes.ts` |
| 6 | `paths.ts` |
| 7 | `teamMemPaths.ts` |
| 8 | `teamMemPrompts.ts` |

---

## §3.1-3 Go ↔ TS 文件名映射（1:1）

| TS 基名 | Go 文件 |
|---------|---------|
| `findRelevantMemories.ts` | `find_relevant_memories.go` |
| `memdir.ts` | `memdir.go` |
| `memoryAge.ts` | `memory_age.go` |
| `memoryScan.ts` | `memory_scan.go` |
| `memoryTypes.ts` | `memory_types.go` |
| `paths.ts` | `paths.go` |
| `teamMemPaths.ts` | `team_mem_paths.go` |
| `teamMemPrompts.ts` | `team_mem_prompts.go` |

---

## §3.1-3 多 TS → 单 Go（显式声明）

| Go 文件 | 合并的上游 TS（主名取第一列） |
|---------|-------------------------------|
| **`extract_memories.go`** | **`services/extractMemories/extractMemories.ts`** + **`services/extractMemories/prompts.ts`** |

（Go 放在 `internal/memdir` 以避免 query/compact/api import 环；行为见 `extract_memories.go` 注释。）

---

## 跨模块接线（文件名仍对齐主 TS）

| Go 文件 | 主 TS 参考 | 说明 |
|---------|------------|------|
| **`session_memory_compact.go`** | **`services/compact/sessionMemoryCompact.ts`** | 仅实现读 auto-mem 下 **`MEMORY.md`** 的 **`compact.SessionMemoryCompactHooks`**；完整 session-memory compaction 在 **`internal/services/compact`**。 |

---

## §3.0 序 3–4：`memdir.ts` / `paths.ts` 行为对照（文档收口）

### `memdir.ts`

| 主题 | TS | Go | 说明 |
|------|----|----|------|
| **Searching past context 门控** | `getFeatureValue_CACHED_MAY_BE_STALE('tengu_coral_fern')` | **`features.MemorySearchPastContextEnabled`**（**`RABBIT_CODE_MEMORY_SEARCH_PAST_CONTEXT`**） | headless 用显式 env，非 GrowthBook |
| **工程根目录** | `getProjectDir(getOriginalCwd())` | **`BuildMemoryPromptInput.ProjectRoot` / `MemorySystemPromptInput.ProjectRoot`**；空则 **`os.Getwd`** | 宿主传入或 cwd |
| **grep 文案分支** | `hasEmbeddedSearchTools() \|\| isReplModeEnabled()` | **`useShellGrep`**（**`BuildMemoryPromptInput` / combined prompt opts**） | 由 engine 等按环境设置 |
| **buildMemoryPrompt** | **`buildMemoryLines`**（默认 **`skipIndex=false`**）+ 读 **`MEMORY.md`**；无 mkdir | **`BuildMemoryPrompt`**：**`BuildMemoryLinesAutoOnly`** + **`filepath.Join`** 读入口；**`SkipIndex`** 可选 | Go 用 **`filepath.Join`** 拼入口路径 |
| **loadMemoryPrompt** | async；**auto-only** 返回 **`buildMemoryLines`** 字符串（**`MEMORY.md` 正文由 claudemd 另载**） | **`LoadMemorySystemPrompt`**：KAIROS / TEAM / auto-only 分支 + **`AppendClaudeMdStyleMemoryEntrypoints`**（Messages **system** 单字段合一） | Go 把私享/Team **`MEMORY.md`** 截断块塞进同一 system 串，对齐 **claudemd 式**装载 |
| **遥测** | **`logMemoryDirCounts` / `logEvent`** | 无对等 | headless 不强制 |
| **skipIndex** | GrowthBook **`tengu_moth_copse`** | **`features.MemoryPromptSkipIndex`**（同 **`RABBIT_CODE_EXTRACT_MEMORIES_SKIP_INDEX`**，与 extract 共用；见 **`features/env.go`**） | headless env |
| **KAIROS 日志模式** | **`feature('KAIROS') && getKairosActive()`** | **`features.KairosDailyLogMemoryEnabled`** | env 门控 |
| **daily-log 内 search 段** | **`buildSearchingPastContextSection`** 用 embedded/REPL 启发式 | **`BuildAssistantDailyLogMemoryPrompt`** 固定 **`useShellGrep=false`** | 与 TS 在 Ant/REPL 下可有差异；可后续由宿主传参收紧 |

### `paths.ts`

| TS | Go | 说明 |
|----|-----|------|
| **`isAutoMemoryEnabled`** | **`features.AutoMemoryEnabled` / `AutoMemoryEnabledFromMerged`**（合并 settings 用 **`config.LoadMerged`**） | 顺序对齐：**`DISABLE_AUTO_MEMORY`** → **`SIMPLE`** → **`REMOTE` 且无 `REMOTE_MEMORY_DIR`** → **`autoMemoryEnabled`** → 默认 true；变量名见 **`internal/features/env.go`**（**`RABBIT_*` / `CLAUDE_*`**） |
| **`isExtractModeActive`** | **`features.ExtractMemoriesAllowed(nonInteractive)`**；宿主传 **nonInteractive** | TS：**`tengu_passport_quail`** → Go：**`RABBIT_CODE_EXTRACT_MEMORIES`**；**`tengu_slate_thimble`** → **`RABBIT_CODE_EXTRACT_MEMORIES_NON_INTERACTIVE`** |
| **`getAutoMemPath()`** | **`ResolveAutoMemDir` / `ResolveAutoMemDirWithOptions`** + **`AutoMemResolveOptions.TrustedAutoMemoryDirectory`**（**`config.LoadTrustedAutoMemoryDirectory`**） | Go 多一层 **可信 settings 路径**；其余为 override env、**`MemoryBaseDir`/`projects/<sanitized>/memory/`**（见 **`paths.go`**） |
| **`getMemoryBaseDir`** 等 | **`MemoryBaseDir`**、**`AutoMemDailyLogPath*`**、**`IsAutoMemPath`** … | 已在主表 **[x]** |

---

## 主要 export / 符号对齐（`src/memdir` 内）

Legend：**[x]** 已对齐或 env 等价，**[~]** 子集/签名差异，**[ ]** 非本包范围。

### `paths.ts`

| TS | Go | 状态 |
|----|-----|------|
| `isAutoMemoryEnabled` | **`features.AutoMemoryEnabled` / `AutoMemoryEnabledFromMerged`** + merged **`autoMemoryEnabled`** | **[x]**（见上 **§3.0 序 3–4 `paths.ts`**） |
| `isExtractModeActive` | **`IsExtractModeActive` → `features.ExtractMemoriesAllowed`** | **[x]**（GrowthBook → **`RABBIT_CODE_EXTRACT_MEMORIES*`**） |
| `getMemoryBaseDir` | `MemoryBaseDir` | **[x]** |
| `getAutoMemPath` | **`ResolveAutoMemDir` / `ResolveAutoMemDirWithOptions`** + trusted dir | **[x]**（见上 **§3.0 序 3–4**） |
| `getAutoMemDailyLogPath` | `AutoMemDailyLogPath*` | **[x]** |
| `getAutoMemEntrypoint` | `AutoMemEntrypointPath*` | **[x]** |
| `isAutoMemPath` | `IsAutoMemPath` | **[x]** |
| `hasAutoMemPathOverride` | `HasAutoMemPathOverride` | **[x]** |

### `memoryTypes.ts`

| TS | Go | 状态 |
|----|-----|------|
| `MEMORY_TYPES` / `parseMemoryType` | `MemoryTypes` / `ParseMemoryType` / `ParseMemoryTypeFromAny` | **[x]**（`unknown` → `ParseMemoryTypeFromAny`） |
| 各 `*_SECTION` | `TypesSection*`、`WhatNotToSaveSection` 等 | **[x]** embed |
| `MEMORY_DRIFT_CAVEAT` | `MemoryDriftCaveat` | **[x]** |

### `memoryScan.ts` / `memoryAge.ts`

| TS | Go | 状态 |
|----|-----|------|
| `MemoryHeader` / `scanMemoryFiles` | `MemoryHeader` / `ScanMemoryFiles` | **[x]**（**`context.Context`** ↔ **`AbortSignal`**；**`description: null`** ↔ **`Description == ""`**；**`type` undefined** ↔ **`Type == ""`**；见 **`memory_scan.go`** 注释） |
| `formatMemoryManifest` | `FormatMemoryManifest` | **[x]** |
| `memoryAge*` | `MemoryAge*` 等 | **[x]** |

### `memdir.ts`

| TS | Go | 状态 |
|----|-----|------|
| entrypoint / `truncateEntrypointContent` | `EntrypointName` / `TruncateEntrypointContent` | **[x]** |
| `loadMemoryPrompt` / `buildMemoryPrompt` | `LoadMemorySystemPrompt` / `BuildMemoryPrompt` | **[x]**（见 **§3.0 序 3–4 `memdir.ts`**） |
| `buildSearchingPastContextSection` | `BuildSearchingPastContextSection` | **[x]**（门控：**`MEMORY_SEARCH_PAST_CONTEXT`** ↔ TS **`tengu_coral_fern`**） |

### `teamMemPaths.ts` / `teamMemPrompts.ts`

| TS | Go | 状态 |
|----|-----|------|
| `getTeamMemPath` / `isTeamMemPath` / `isTeamMemFile` | `GetTeamMemPath` / `IsTeamMemPath` / `IsTeamMemFile` | **[x]** |
| `buildCombinedMemoryPrompt` | `BuildCombinedMemoryPrompt` | **[x]** |

### `findRelevantMemories.ts`

| TS | Go | 状态 |
|----|-----|------|
| `findRelevantMemories` | `FindRelevantMemoriesDetailed` / `FindRelevantMemories` / `FindRelevantMemoriesClassic` | **[x]**（`ctx`↔`AbortSignal`；**`FindRelevantMemoriesClassic`** ≈ TS 形参顺序；通用逻辑用 **`FindRelevantMemoriesOpts`**） |
| `RelevantMemory` | `RelevantMemory` | **[x]** |

---

## 验收（§3.1-4）

```bash
go test ./internal/memdir/... -count=1 -short
```

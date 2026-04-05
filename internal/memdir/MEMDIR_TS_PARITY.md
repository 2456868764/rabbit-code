# `src/memdir` 模块 ↔ `internal/memdir`（§3.1 模块级对齐）

**规则**：`docs/phases/PHASE_ITERATION_RULES.md` **§3.1** — 以 **`restored-src/src/<area>/`** 整目录为交付边界；Go **`*.go`** 基名与 TS **`*.ts`** 一一 **`camelCase` → `snake_case`**；禁止把**同一 TS 文件**拆成多个按小功能命名的 `.go`；**多 TS 合并为一个 `.go`** 时须在本表写明且文件名取主模块。

---

## §3.0 有序迭代计划（`PHASE_ITERATION_RULES.md` §三-3.0）

由 **本 PARITY 表 `[~]` / `[ ]`**、**`PHASE05_CONTINUATION.md` H8**、**`restored-src/src/memdir/*.ts`** 导出；**严格按序**执行；完成一项即更新下表 **状态** 与上表符号列。排序与 **`PARITY` + §3.2-3** 冲突时以强制规则为准并回写说明。

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`findRelevantMemories`**：TS 形参 ↔ **`FindRelevantMemoriesClassic`**（`ctx`↔`AbortSignal`）+ 通用 **`FindRelevantMemoriesOpts`** | 下表 **`findRelevantMemories`** 为 **[x]**；**`go test ./internal/memdir/... -short`** |
| 2 | ☑ | **`memoryScan`**：**`MemoryHeader` / `scanMemoryFiles`** — `description: null` ↔ **`Description == ""`**；**`AbortSignal` ↔ `context`**；**`FormatMemoryManifest`** 与 TS 一致（单测 **`TestFormatMemoryManifest_noDescriptionLikeTSNull`**） | 下表 **`MemoryHeader` / `scanMemoryFiles`** 为 **[x]** |
| 3 | ☐ | **`memdir.ts`**：**`buildMemoryPrompt` / `buildSearchingPastContextSection`** 与 TS 差异表 | 表行或 defer |
| 4 | ☐ | **`paths.ts`**：**`isAutoMemoryEnabled` / `isExtractModeActive` / `getAutoMemPath`** 与 **`internal/features`**、env 对照 | 表行或 defer |

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

## 主要 export / 符号对齐（`src/memdir` 内）

Legend：**[x]** 已对齐或 env 等价，**[~]** 子集/签名差异，**[ ]** 非本包范围。

### `paths.ts`

| TS | Go | 状态 |
|----|-----|------|
| `isAutoMemoryEnabled` | `features` + `config` | **[~]** |
| `isExtractModeActive` | `IsExtractModeActive` | **[~]** GrowthBook → env |
| `getMemoryBaseDir` | `MemoryBaseDir` | **[x]** |
| `getAutoMemPath` | `ResolveAutoMemDir*` | **[~]** |
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
| `loadMemoryPrompt` / `buildMemoryPrompt` | `LoadMemorySystemPrompt` / `BuildMemoryPrompt` | **[~]** |
| `buildSearchingPastContextSection` | `BuildSearchingPastContextSection` | **[~]** |

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

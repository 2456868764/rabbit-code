# `src/memdir` 模块 ↔ `internal/memdir`（§3.1 模块级对齐）

**规则**：`docs/phases/PHASE_ITERATION_RULES.md` **§3.1** — 以 **`restored-src/src/<area>/`** 整目录为交付边界；Go **`*.go`** 基名与 TS **`*.ts`** 一一 **`camelCase` → `snake_case`**；禁止把**同一 TS 文件**拆成多个按小功能命名的 `.go`；**多 TS 合并为一个 `.go`** 时须在本表写明且文件名取主模块。

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
| `MEMORY_TYPES` / `parseMemoryType` | `MemoryTypes` / `ParseMemoryType` | **[x]** / **[~]** |
| 各 `*_SECTION` | `TypesSection*`、`WhatNotToSaveSection` 等 | **[x]** embed |
| `MEMORY_DRIFT_CAVEAT` | `MemoryDriftCaveat` | **[x]** |

### `memoryScan.ts` / `memoryAge.ts`

| TS | Go | 状态 |
|----|-----|------|
| `MemoryHeader` / `scanMemoryFiles` | `MemoryHeader` / `ScanMemoryFiles` | **[~]** ctx vs AbortSignal |
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
| `getTeamMemPath` / `isTeamMemFile` | `GetTeamMemPath` / `IsTeamMemFile` | **[x]** |
| `buildCombinedMemoryPrompt` | `BuildCombinedMemoryPrompt` | **[x]** |

### `findRelevantMemories.ts`

| TS | Go | 状态 |
|----|-----|------|
| `findRelevantMemories` | `FindRelevantMemories*` | **[~]** opts 结构体 |
| `RelevantMemory` | `RelevantMemory` | **[x]** |

---

## 验收（§3.1-4）

```bash
go test ./internal/memdir/... -count=1 -short
```

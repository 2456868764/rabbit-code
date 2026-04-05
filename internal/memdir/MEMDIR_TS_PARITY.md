# `src/memdir/*.ts` ↔ `internal/memdir` parity

**Authority:** `claude-code-sourcemap/restored-src/src/memdir/`. **Rules:** `docs/phases/PHASE_ITERATION_RULES.md` §3.1 (one Go file per TS basename, snake_case).

## File map

| TS module | Go file | Notes |
|-----------|---------|--------|
| `paths.ts` | `paths.go` | + `FindGitRoot`, `SanitizePath` (helpers for resolution) |
| `memoryTypes.ts` | `memory_types.go` | `//go:embed promptdata/*.txt` for section bodies |
| `memoryScan.ts` | `memory_scan.go` | |
| `memoryAge.ts` | `memory_age.go` | + attachment header / session fragment helpers used by engine |
| `memdir.ts` | `memdir.go` | `LoadMemorySystemPrompt`, `BuildMemoryPrompt` = TS `loadMemoryPrompt` / `buildMemoryPrompt` analogue |
| `teamMemPaths.ts` | `team_mem_paths.go` | + secret scan / `TeamMemSecretGuardRunner` |
| `teamMemPrompts.ts` | `team_mem_prompts.go` | |
| `findRelevantMemories.ts` | `find_relevant_memories.go` | + `FindRelevantMemoryPaths`, LLM path, `RecallShapeHook` |
| `extractMemories.ts` (+ prompts) | `extract_memories.go` | lives under `services/extractMemories` in TS; Go colocates in `memdir` to avoid cycles |
| _(compact hooks)_ | `session_memory_compact_hooks.go` | file-backed `compact.SessionMemoryCompactHooks` |

## Export / symbol alignment

Legend: **[x]** behaviour matched or documented env analogue, **[~]** subset / different signature, **[ ]** not in `src/memdir` scope.

### `paths.ts`

| TS | Go | Status |
|----|-----|--------|
| `isAutoMemoryEnabled` | `features.AutoMemoryEnabled` / `AutoMemoryEnabledFromMerged` | **[~]** settings in `internal/features` + `config` |
| `isExtractModeActive` | `IsExtractModeActive(nonInteractive bool)` | **[~]** GrowthBook → `RABBIT_CODE_EXTRACT_MEMORIES*` (`features.ExtractMemoriesAllowed`) |
| `getMemoryBaseDir` | `MemoryBaseDir` | **[x]** |
| `getAutoMemPath` | `ResolveAutoMemDir` / `ResolveAutoMemDirWithOptions` | **[~]** no lodash memoize; explicit options |
| `getAutoMemDailyLogPath` | `AutoMemDailyLogPath` | **[x]** |
| `getAutoMemEntrypoint` | `AutoMemEntrypointPath` | **[x]** |
| `isAutoMemPath` | `IsAutoMemPath` | **[x]** |
| `hasAutoMemPathOverride` | `HasAutoMemPathOverride` | **[x]** |

### `memoryTypes.ts`

| TS | Go | Status |
|----|-----|--------|
| `MEMORY_TYPES` | `MemoryTypes` | **[x]** |
| `parseMemoryType` | `ParseMemoryType` (returns `""` if unknown) | **[~]** TS returns `undefined` |
| `TYPES_SECTION_*` | `TypesSectionCombined`, `TypesSectionIndividual` | **[x]** embed |
| `WHAT_NOT_TO_SAVE_SECTION` | `WhatNotToSaveSection` | **[x]** |
| `MEMORY_DRIFT_CAVEAT` | `MemoryDriftCaveat` + embedded in `WhenToAccessSection` | **[x]** |
| `WHEN_TO_ACCESS_SECTION` | `WhenToAccessSection`, `WhenToAccessCombinedSection` | **[x]** |
| `TRUSTING_RECALL_SECTION` | `TrustingRecallSection` | **[x]** |
| `MEMORY_FRONTMATTER_EXAMPLE` | `MemoryFrontmatterExample`, `MemoryFrontmatterExampleBlock` | **[x]** |

### `memoryScan.ts`

| TS | Go | Status |
|----|-----|--------|
| `MemoryHeader` | `MemoryHeader` (`Description` empty string vs TS `null`) | **[~]** |
| `scanMemoryFiles(dir, signal)` | `ScanMemoryFiles(ctx, dir)` | **[~]** context vs AbortSignal |
| `formatMemoryManifest` | `FormatMemoryManifest` | **[x]** |

### `memoryAge.ts`

| TS | Go | Status |
|----|-----|--------|
| `memoryAgeDays` | `MemoryAgeDays`, `MemoryAgeDaysAt` | **[x]** wall-clock injectable |
| `memoryAge` / freshness | `MemoryAge`, `MemoryFreshnessText`, `MemoryFreshnessNote` (+ `*At`) | **[x]** |
| _(attachments)_ | `MemoryAttachmentHeader`, `SessionFragmentsFromPaths*` | **[~]** TS in other modules |

### `memdir.ts`

| TS | Go | Status |
|----|-----|--------|
| `ENTRYPOINT_NAME` … `truncateEntrypointContent` | `EntrypointName`, `TruncateEntrypointContent` | **[x]** warning sizes use `formatFileSizeBytes` ≡ `formatFileSize` |
| `DIR_EXISTS_GUIDANCE` | `DirExistsGuidance` | **[x]** |
| `ensureMemoryDirExists` | `EnsureMemoryDirExists` | **[x]** |
| `buildMemoryLines` | `BuildMemoryLinesAutoOnly` | **[x]** |
| `buildMemoryPrompt` / `loadMemoryPrompt` | `BuildMemoryPrompt` / `LoadMemorySystemPrompt` | **[~]** async settings → struct input |
| `buildSearchingPastContextSection` | `BuildSearchingPastContextSection(auto, project, useShellGrep)` | **[~]** TS uses feature + embedded grep detection |
| KAIROS daily log | `BuildAssistantDailyLogMemoryPrompt` | **[~]** feature gate in `features` |

### `teamMemPaths.ts`

| TS | Go | Status |
|----|-----|--------|
| `getTeamMemPath` | `GetTeamMemPath` / `TeamMemDirFromAutoMemDir` | **[x]** |
| `isTeamMemPath` | `IsTeamMemPathUnderAutoMem` / `TeamMemPathResolved` | **[~]** naming |
| `validateTeamMemWritePath` | `ValidateTeamMemWritePath`, `ValidateTeamMemWritePathFull` | **[~]** sync vs async |
| `validateTeamMemKey` | `ValidateTeamMemKey` | **[~]** |
| `isTeamMemFile` | `IsTeamMemFile`, `IsTeamMemFileActive` | **[x]** |
| `isTeamMemoryEnabled` | `features.TeamMemoryEnabledFromMerged` | **[~]** |

### `teamMemPrompts.ts`

| TS | Go | Status |
|----|-----|--------|
| `buildCombinedMemoryPrompt` | `BuildCombinedMemoryPrompt` | **[x]** |

### `findRelevantMemories.ts`

| TS | Go | Status |
|----|-----|--------|
| `findRelevantMemories` | `FindRelevantMemoriesDetailed` / `FindRelevantMemories` | **[~]** opts struct vs positional args |
| `RelevantMemory` | `RelevantMemory` | **[x]** |
| `MEMORY_SHAPE_TELEMETRY` | `RecallShapeHook` | **[~]** callback, no analytics module |

## Related TS outside `src/memdir/`

`claudemd.ts`, `utils/path.ts`, `services/teamMemorySync/*`, `services/extractMemories/*` — partial analogues in Go `memdir`, `engine`, `config`; see `docs/phases/PHASE05_CONTINUATION.md` §H8 and `PARITY_PHASE5_DEFERRED.md`.

## Verification

```bash
go test ./internal/memdir/... -count=1 -short
```

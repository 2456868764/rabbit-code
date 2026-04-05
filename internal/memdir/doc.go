// Package memdir implements the restored-src **src/memdir** module in Go (headless parity), plus documented
// **§3.1 多 TS → 单 Go** extensions that live in this package for import-layering reasons.
//
// # PHASE_ITERATION_RULES.md §3.1（模块级强制）
//
// **上游目录**：`claude-code-sourcemap/restored-src/src/memdir/`（平铺，无子目录）。
//
// **§3.1-1 TS 上游文件清单（逐文件，共 8）**
//
//	findRelevantMemories.ts
//	memdir.ts
//	memoryAge.ts
//	memoryScan.ts
//	memoryTypes.ts
//	paths.ts
//	teamMemPaths.ts
//	teamMemPrompts.ts
//
// **§3.1-3 Go 源文件 ↔ TS 基名（camelCase → snake_case，一一对应）**
//
//	find_relevant_memories.go  — findRelevantMemories.ts
//	memdir.go                  — memdir.ts
//	memory_age.go              — memoryAge.ts
//	memory_scan.go             — memoryScan.ts（ScanMemoryFiles：ctx ↔ AbortSignal；Description 空 ↔ TS description null）
//	memory_types.go            — memoryTypes.ts
//	paths.go                   — paths.ts
//	team_mem_paths.go          — teamMemPaths.ts
//	team_mem_prompts.go        — teamMemPrompts.ts
//
// **§3.1-3 多 TS → 单 Go（须在 PARITY 写明；文件名取主模块）**
//
//	extract_memories.go — services/extractMemories/extractMemories.ts + services/extractMemories/prompts.ts
//
// **跨目录接线（非 src/memdir；文件名仍与主 TS 基名一致）**
//
//	session_memory_compact.go — services/compact/sessionMemoryCompact.ts（Go 侧仅为 SessionMemory 读 MEMORY.md 的 hooks 子集；完整 compact 在 internal/services/compact）
//
// **符号级对照表**：MEMDIR_TS_PARITY.md（export ↔ Go、[x]/[~]；**§3.0 / §3.2**；**§3.0 序 3–4** memdir.ts/paths.ts 行为对照表）。
//
// Related: internal/features（env 门控）；engine 接线 FindRelevantMemories（Opts）；FindRelevantMemoriesClassic 为 findRelevantMemories.ts 形参顺序的薄委托；extract stop hook、SessionMemoryCompactHooks。
// Trusted autoMemoryDirectory：config.LoadTrustedAutoMemoryDirectory ↔ paths.ts getAutoMemPathSetting。
//
// Tests：与源文件同基名 `*_test.go`（§3.1 验收：`go test ./internal/memdir/... -short`）。
package memdir

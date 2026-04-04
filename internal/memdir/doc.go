// Package memdir mirrors claude-code-sourcemap/restored-src/src/memdir for headless parity.
//
// Go file layout (aligned with TS modules):
//
//	paths.go                 — paths.ts (+ FindGitRoot, SanitizePath)
//	memory_types.go          — memoryTypes.ts (taxonomy + //go:embed promptdata)
//	memory_scan.go           — memoryScan.ts (scan + FormatMemoryManifest)
//	memory_age.go            — memoryAge.ts (+ attachment headers, session file fragments)
//	memdir.go                — memdir.ts (guidance, ensure dir, entrypoint, searching past context, LoadMemorySystemPrompt ≈ loadMemoryPrompt, BuildMemoryPrompt ≈ buildMemoryPrompt, SessionFragments stub)
//	team_mem_paths.go        — teamMemPaths.ts (+ secret scan, Write/Edit guard runner)
//	team_mem_prompts.go      — teamMemPrompts.ts (combined private+team prompt)
//	find_relevant_memories.go — findRelevantMemories.ts (+ heuristic scoring, LLM JSON selection)
//	extract_memories.go      — extractMemories.ts / prompts (fork, controller, transcript helpers, gated tools)
//
// Related: internal/features and rabbit env gates for auto-memory; engine wires FindRelevantMemories and extract stop hook.
// Trusted autoMemoryDirectory: config.LoadTrustedAutoMemoryDirectory ↔ paths.ts getAutoMemPathSetting (policy → flag → local → user; no project).
//
// Tests use the same basename as sources: paths_test.go, memory_types_test.go, memory_scan_test.go, memory_age_test.go,
// memdir_test.go, team_mem_paths_test.go, team_mem_prompts_test.go, find_relevant_memories_test.go, extract_memories_test.go.
package memdir

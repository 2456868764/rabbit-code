package memdir

import (
	"context"
	"path/filepath"
	"strings"
)

// RelevanceMode selects heuristic vs LLM path (H8.3).
type RelevanceMode string

const (
	RelevanceModeHeuristic RelevanceMode = "heuristic"
	RelevanceModeLLM       RelevanceMode = "llm"
)

// RelevantMemory mirrors findRelevantMemories.ts RelevantMemory (path + mtime for freshness).
type RelevantMemory struct {
	Path    string
	MtimeMs int64
}

// RecallShapeHook is invoked after selection with (candidates, selected) counts — TS MEMORY_SHAPE_TELEMETRY analogue.
type RecallShapeHook func(candidates, selected int)

// FindRelevantMemoriesOpts configures H8 scan + selection.
type FindRelevantMemoriesOpts struct {
	Mode            RelevanceMode
	Limit           int
	RecentTools     []string
	AlreadySurfaced map[string]struct{}
	TextComplete    TextCompleteFunc
	// StrictLLM when true skips heuristic fallback after LLM error or bad JSON (aligns with TS returning [] on failure).
	StrictLLM bool
	// OnRecallShape optional; fires once per call with candidate count and final selected count (including heuristic path).
	OnRecallShape RecallShapeHook
}

// FindRelevantMemories scans memory dir, then selects paths by heuristic or LLM (H8.1–H8.4).
func FindRelevantMemories(ctx context.Context, query, memoryDir string, opts FindRelevantMemoriesOpts) ([]string, error) {
	rel, err := FindRelevantMemoriesDetailed(ctx, query, memoryDir, opts)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rel))
	for i := range rel {
		out[i] = rel[i].Path
	}
	return out, nil
}

// FindRelevantMemoriesDetailed returns paths with mtimes (findRelevantMemories.ts).
func FindRelevantMemoriesDetailed(ctx context.Context, query, memoryDir string, opts FindRelevantMemoriesOpts) ([]RelevantMemory, error) {
	if memoryDir == "" {
		return nil, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 5
	}
	memories, err := ScanMemoryFiles(ctx, memoryDir)
	if err != nil {
		return nil, err
	}
	memories = filterAlreadySurfaced(memories, opts.AlreadySurfaced)
	candidates := len(memories)
	fire := func(selected int) {
		if opts.OnRecallShape != nil {
			opts.OnRecallShape(candidates, selected)
		}
	}
	if len(memories) == 0 {
		fire(0)
		return nil, nil
	}

	mode := opts.Mode
	if mode == "" {
		mode = RelevanceModeHeuristic
	}

	if mode == RelevanceModeLLM && opts.TextComplete != nil {
		manifest := FormatMemoryManifest(memories)
		user := buildMemdirUserPayload(query, manifest, opts.RecentTools)
		text, err := opts.TextComplete(ctx, SelectMemoriesSystemPrompt, user)
		if err == nil {
			names, perr := ParseSelectedMemoriesJSON(text)
			if perr == nil {
				res := resolveSelectedMemories(memories, names, limit)
				fire(len(res))
				return res, nil
			}
		}
		if opts.StrictLLM {
			fire(0)
			return nil, nil
		}
	}

	paths := HeuristicMemoryPathsFromHeaders(ctx, memories, query, limit)
	res := relevantFromPaths(memories, paths)
	fire(len(res))
	return res, nil
}

func relevantFromPaths(memories []MemoryHeader, paths []string) []RelevantMemory {
	by := make(map[string]MemoryHeader, len(memories))
	for _, m := range memories {
		by[m.FilePath] = m
	}
	var out []RelevantMemory
	for _, p := range paths {
		if m, ok := by[p]; ok {
			out = append(out, RelevantMemory{Path: m.FilePath, MtimeMs: m.MtimeMs})
		}
	}
	return out
}

func filterAlreadySurfaced(memories []MemoryHeader, already map[string]struct{}) []MemoryHeader {
	if len(already) == 0 {
		return memories
	}
	out := memories[:0]
	for _, m := range memories {
		if surfacedContains(already, m.FilePath) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func surfacedContains(already map[string]struct{}, filePath string) bool {
	if _, ok := already[filePath]; ok {
		return true
	}
	ap, err := filepath.Abs(filePath)
	if err == nil {
		if _, ok := already[ap]; ok {
			return true
		}
	}
	return false
}

func buildMemoryLookup(memories []MemoryHeader) (byRel map[string]MemoryHeader, baseUnique map[string]MemoryHeader) {
	byRel = make(map[string]MemoryHeader, len(memories))
	baseCount := make(map[string]int)
	for _, m := range memories {
		rel := filepath.ToSlash(m.Filename)
		byRel[strings.ToLower(rel)] = m
		b := strings.ToLower(filepath.Base(rel))
		baseCount[b]++
	}
	baseUnique = make(map[string]MemoryHeader)
	for _, m := range memories {
		rel := filepath.ToSlash(m.Filename)
		b := strings.ToLower(filepath.Base(rel))
		if baseCount[b] == 1 {
			baseUnique[b] = m
		}
	}
	return byRel, baseUnique
}

func resolveSelectedMemories(memories []MemoryHeader, selected []string, limit int) []RelevantMemory {
	byRel, baseUnique := buildMemoryLookup(memories)
	var out []RelevantMemory
	seen := make(map[string]struct{})
	for _, name := range selected {
		if len(out) >= limit {
			break
		}
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		relKey := strings.ToLower(filepath.ToSlash(n))
		var m MemoryHeader
		var ok bool
		if mm, hit := byRel[relKey]; hit {
			m, ok = mm, true
		} else if mm, hit := baseUnique[strings.ToLower(filepath.Base(n))]; hit {
			m, ok = mm, true
		}
		if !ok {
			continue
		}
		if _, dup := seen[m.FilePath]; dup {
			continue
		}
		seen[m.FilePath] = struct{}{}
		out = append(out, RelevantMemory{Path: m.FilePath, MtimeMs: m.MtimeMs})
	}
	return out
}

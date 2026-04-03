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

// FindRelevantMemoriesOpts configures H8 scan + selection.
type FindRelevantMemoriesOpts struct {
	Mode            RelevanceMode
	Limit           int
	RecentTools     []string
	AlreadySurfaced map[string]struct{}
	TextComplete    TextCompleteFunc
}

// FindRelevantMemories scans memory dir, then selects paths by heuristic or LLM (H8.1–H8.4).
func FindRelevantMemories(ctx context.Context, query, memoryDir string, opts FindRelevantMemoriesOpts) ([]string, error) {
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
	if len(memories) == 0 {
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
				return resolveSelectedFilenames(memories, names, limit), nil
			}
		}
		// fall through to heuristic on LLM failure (H8.6)
	}

	return HeuristicMemoryPathsFromHeaders(ctx, memories, query, limit), nil
}

func filterAlreadySurfaced(memories []MemoryHeader, already map[string]struct{}) []MemoryHeader {
	if len(already) == 0 {
		return memories
	}
	out := memories[:0]
	for _, m := range memories {
		if _, ok := already[m.FilePath]; ok {
			continue
		}
		out = append(out, m)
	}
	return out
}

func resolveSelectedFilenames(memories []MemoryHeader, selected []string, limit int) []string {
	byName := make(map[string]string, len(memories))
	for _, m := range memories {
		byName[strings.ToLower(m.Filename)] = m.FilePath
	}
	var paths []string
	for _, name := range selected {
		if len(paths) >= limit {
			break
		}
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		base := filepath.Base(n)
		if p, ok := byName[strings.ToLower(base)]; ok {
			paths = append(paths, p)
		}
	}
	return paths
}

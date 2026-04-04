package memdir

// Corresponds to restored-src/src/memdir/findRelevantMemories.ts (LLM selection + heuristic scoring).

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// SelectMemoriesSystemPrompt mirrors findRelevantMemories.ts SELECT_MEMORIES_SYSTEM_PROMPT (H8.2).
const SelectMemoriesSystemPrompt = `You are selecting memories that will be useful to Claude Code as it processes a user's query. You will be given the user's query and a list of available memory files with their filenames and descriptions.

Return a JSON object with key "selected_memories" whose value is an array of filenames (strings) for the memories that will clearly be useful (up to 5). Only include memories that you are certain will be helpful based on their name and description.
- If you are unsure if a memory will be useful in processing the user's query, then do not include it in your list. Be selective and discerning.
- If there are no memories in the list that would clearly be useful, return {"selected_memories":[]}.
- If a list of recently-used tools is provided, do not select memories that are usage reference or API documentation for those tools (Claude Code is already exercising them). DO still select memories containing warnings, gotchas, or known issues about those tools — active use is exactly when those matter.`

// TextCompleteFunc performs one assistant-style completion (H8.2 side-query); engine wires Anthropic streaming read.
type TextCompleteFunc func(ctx context.Context, systemPrompt, userMessage string) (assistantText string, err error)

// ParseSelectedMemoriesJSON extracts selected_memories from model output (tolerates markdown fences).
func ParseSelectedMemoriesJSON(assistantText string) ([]string, error) {
	s := strings.TrimSpace(assistantText)
	if s == "" {
		return nil, errors.New("memdir: empty model output")
	}
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		if j := strings.Index(rest, "```"); j >= 0 {
			inner := strings.TrimSpace(rest[:j])
			if strings.HasPrefix(inner, "json") {
				inner = strings.TrimSpace(strings.TrimPrefix(inner, "json"))
			}
			s = inner
		}
	}
	var obj struct {
		SelectedMemories []string `json:"selected_memories"`
	}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		re := regexp.MustCompile(`\{[\s\S]*"selected_memories"[\s\S]*\}`)
		if m := re.FindString(s); m != "" {
			if err2 := json.Unmarshal([]byte(m), &obj); err2 == nil {
				return obj.SelectedMemories, nil
			}
		}
		return nil, err
	}
	return obj.SelectedMemories, nil
}

func buildMemdirUserPayload(query, manifest string, recentTools []string) string {
	var b strings.Builder
	b.WriteString("Query: ")
	b.WriteString(query)
	b.WriteString("\n\nAvailable memories:\n")
	b.WriteString(manifest)
	if len(recentTools) > 0 {
		b.WriteString("\n\nRecently used tools: ")
		b.WriteString(strings.Join(recentTools, ", "))
	}
	b.WriteString("\n\nRespond with ONLY valid JSON: {\"selected_memories\":[\"filename.md\",...]}")
	return b.String()
}

// FindRelevantMemoryPaths returns up to limit memory file paths under memoryDir scored by simple token overlap with queryText (deterministic, no LLM — item 13).
// Uses recursive ScanMemoryFiles (skips MEMORY.md) and scores filename + frontmatter meta + file head.
func FindRelevantMemoryPaths(queryText, memoryDir string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	memories, _ := ScanMemoryFiles(context.Background(), memoryDir)
	return HeuristicMemoryPathsFromHeaders(context.Background(), memories, queryText, limit), nil
}

// HeuristicMemoryPathsFromHeaders scores memories by alphanumeric token overlap with query (newest-first tie-break).
func HeuristicMemoryPathsFromHeaders(ctx context.Context, memories []MemoryHeader, queryText string, limit int) []string {
	if limit <= 0 {
		limit = 5
	}
	queryText = strings.ToLower(strings.TrimSpace(queryText))
	tokens := tokenizeAlnum(queryText)
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		path  string
		score int
		mtime int64
	}
	var out []scored
	for _, m := range memories {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		head := readFileHead(m.FilePath, 4096)
		low := strings.ToLower(m.Filename + " " + m.Description + " " + head)
		sc := 0
		for _, tok := range tokens {
			if len(tok) < 2 {
				continue
			}
			if strings.Contains(low, tok) {
				sc++
			}
		}
		if sc > 0 {
			out = append(out, scored{m.FilePath, sc, m.MtimeMs})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].mtime > out[j].mtime
	})
	var paths []string
	for i := range out {
		if i >= limit {
			break
		}
		paths = append(paths, out[i].path)
	}
	return paths
}

func tokenizeAlnum(s string) []string {
	var cur strings.Builder
	var toks []string
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return toks
}

func readFileHead(path string, max int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, max)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}
	if n <= 0 {
		return ""
	}
	return string(buf[:n])
}

// DedupePathsStable returns paths in first-seen order.
func DedupePathsStable(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var out []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		ap, err := filepath.Abs(p)
		if err == nil {
			p = ap
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

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
	StrictLLM       bool
	OnRecallShape   RecallShapeHook
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
	memories, _ := ScanMemoryFiles(ctx, memoryDir)
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

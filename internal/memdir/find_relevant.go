package memdir

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// FindRelevantMemoryPaths returns up to limit memory file paths under memoryDir scored by simple token overlap with queryText (deterministic, no LLM — item 13).
// Uses recursive ScanMemoryFiles (skips MEMORY.md) and scores filename + frontmatter meta + file head.
func FindRelevantMemoryPaths(queryText, memoryDir string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	memories, err := ScanMemoryFiles(context.Background(), memoryDir)
	if err != nil {
		return nil, err
	}
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

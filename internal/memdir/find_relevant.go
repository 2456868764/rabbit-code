package memdir

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// FindRelevantMemoryPaths returns up to limit memory file paths under memoryDir scored by simple token overlap with queryText (deterministic, no LLM — item 13).
func FindRelevantMemoryPaths(queryText, memoryDir string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	queryText = strings.ToLower(strings.TrimSpace(queryText))
	tokens := tokenizeAlnum(queryText)
	if len(tokens) == 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return nil, err
	}
	type scored struct {
		path  string
		score int
	}
	var out []scored
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		p := filepath.Join(memoryDir, name)
		header := readFileHead(p, 4096)
		low := strings.ToLower(name + " " + header)
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
			out = append(out, scored{path: p, score: sc})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].path < out[j].path
	})
	var paths []string
	for i := range out {
		if i >= limit {
			break
		}
		paths = append(paths, out[i].path)
	}
	return paths, nil
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

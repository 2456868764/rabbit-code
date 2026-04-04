package memdir

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxMemoryFiles caps scan results (findRelevantMemories.ts / memoryScan.ts).
const MaxMemoryFiles = 200

// FrontmatterMaxLines limits how many lines we read for YAML frontmatter parsing.
const FrontmatterMaxLines = 30

// MemoryHeader mirrors memdir/memoryScan.ts MemoryHeader (headless subset).
type MemoryHeader struct {
	Filename    string // path relative to memoryDir
	FilePath    string // absolute path
	MtimeMs     int64
	Description string
	Type        string // optional frontmatter type string
}

// ScanMemoryFiles walks memoryDir recursively for .md files, skips basename MEMORY.md,
// reads frontmatter headers, returns up to MaxMemoryFiles sorted newest-first.
//
// It always returns a nil error. On failure (missing directory, walk errors, context
// cancellation), it returns (nil, nil), matching memoryScan.ts scanMemoryFiles outer
// try/catch that resolves to an empty array.
func ScanMemoryFiles(ctx context.Context, memoryDir string) ([]MemoryHeader, error) {
	out := scanMemoryFilesCollect(ctx, memoryDir)
	return out, nil
}

func scanMemoryFilesCollect(ctx context.Context, memoryDir string) []MemoryHeader {
	memoryDir = filepath.Clean(strings.TrimSpace(memoryDir))
	if memoryDir == "" {
		return nil
	}
	fi, err := os.Stat(memoryDir)
	if err != nil || !fi.IsDir() {
		return nil
	}
	var out []MemoryHeader
	_ = filepath.WalkDir(memoryDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			out = nil
			return fs.SkipAll
		default:
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "MEMORY.md") {
			return nil
		}
		rel, err := filepath.Rel(memoryDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		st, err := os.Stat(path)
		if err != nil {
			return nil
		}
		desc, typRaw := readMemoryFrontmatterMeta(path)
		out = append(out, MemoryHeader{
			Filename:    rel,
			FilePath:    path,
			MtimeMs:     st.ModTime().UnixMilli(),
			Description: desc,
			Type:        ParseMemoryType(typRaw),
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].MtimeMs != out[j].MtimeMs {
			return out[i].MtimeMs > out[j].MtimeMs
		}
		return out[i].Filename < out[j].Filename
	})
	if len(out) > MaxMemoryFiles {
		out = out[:MaxMemoryFiles]
	}
	return out
}

func readMemoryFrontmatterMeta(path string) (description, memType string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var lines []string
	for sc.Scan() && len(lines) < FrontmatterMaxLines {
		lines = append(lines, sc.Text())
	}
	content := strings.Join(lines, "\n")
	return parseFrontmatterDescriptionAndType(content)
}

// FormatMemoryManifest mirrors memoryScan.ts formatMemoryManifest (one line per memory).
func FormatMemoryManifest(memories []MemoryHeader) string {
	var b strings.Builder
	for i, m := range memories {
		if i > 0 {
			b.WriteByte('\n')
		}
		tag := ""
		if m.Type != "" {
			tag = fmt.Sprintf("[%s] ", m.Type)
		}
		ts := time.UnixMilli(m.MtimeMs).UTC().Format(time.RFC3339)
		line := fmt.Sprintf("- %s%s (%s)", tag, m.Filename, ts)
		if m.Description != "" {
			line += ": " + m.Description
		}
		b.WriteString(line)
	}
	return b.String()
}

// parseFrontmatterDescriptionAndType extracts description and type from leading YAML frontmatter or loose keys.
func parseFrontmatterDescriptionAndType(content string) (description, memType string) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return looseYAMLKeys(content)
	}
	rest := strings.TrimPrefix(content, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return looseYAMLKeys(content)
	}
	block := rest[:end]
	return looseYAMLKeys(block)
}

func looseYAMLKeys(block string) (description, memType string) {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "description:"); ok {
			description = strings.TrimSpace(unquoteYAMLString(after))
			continue
		}
		if after, ok := strings.CutPrefix(line, "type:"); ok {
			memType = strings.TrimSpace(unquoteYAMLString(after))
		}
	}
	return description, memType
}

func unquoteYAMLString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.Trim(s, `"`)
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.Trim(s, "'")
	}
	return s
}

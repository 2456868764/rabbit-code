package globtool

import (
	"path/filepath"
	"strings"
)

// NormalizeFileReadDenyPatternsToSearchDir mirrors normalizePatternsToPath(getFileReadIgnorePatterns output, searchDir)
// from filesystem.ts. patternsByRoot: key "" = global patterns (TS null root); other keys = absolute pattern roots.
func NormalizeFileReadDenyPatternsToSearchDir(patternsByRoot map[string][]string, searchDir string) []string {
	if len(patternsByRoot) == 0 {
		return nil
	}
	searchDir = filepath.Clean(searchDir)
	seen := make(map[string]struct{})
	for root, pats := range patternsByRoot {
		if root == "" {
			// TS: null-root patterns are added verbatim (normalizePatternsToPath Set init).
			for _, p := range pats {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if !strings.HasPrefix(p, "!") {
					p = "!" + p
				}
				seen[p] = struct{}{}
			}
			continue
		}
		pr := filepath.Clean(root)
		for _, pattern := range pats {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if g := normalizeOneDenyPatternToSearchDir(pr, pattern, searchDir); g != "" {
				seen[g] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for g := range seen {
		out = append(out, g)
	}
	return out
}

func normalizeOneDenyPatternToSearchDir(patternRoot, pattern, rootPath string) string {
	pr := filepath.Clean(patternRoot)
	rp := filepath.Clean(rootPath)
	fullPattern := filepath.Clean(filepath.Join(pr, filepath.FromSlash(pattern)))
	patSlash := filepath.ToSlash(pattern)

	var inner string
	if pr == rp {
		inner = ensureLeadingSlashGlob(patSlash)
	} else {
		prefix := rp + string(filepath.Separator)
		if strings.HasPrefix(fullPattern, prefix) {
			rel := fullPattern[len(rp):]
			rel = strings.TrimPrefix(rel, string(filepath.Separator))
			inner = ensureLeadingSlashGlob(filepath.ToSlash(rel))
		} else {
			relPr, err := filepath.Rel(rp, pr)
			if err != nil {
				return ""
			}
			rs := filepath.ToSlash(relPr)
			if rs == ".." || strings.HasPrefix(rs, "../") {
				return ""
			}
			inner = ensureLeadingSlashGlob(filepath.ToSlash(filepath.Join(relPr, filepath.FromSlash(pattern))))
		}
	}
	return "!" + inner
}

func ensureLeadingSlashGlob(s string) string {
	s = filepath.ToSlash(strings.TrimSpace(s))
	if s == "" {
		return "/"
	}
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

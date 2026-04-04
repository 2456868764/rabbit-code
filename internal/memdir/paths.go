package memdir

// This file corresponds to restored-src/src/memdir/paths.ts (plus git root + sanitizePath helpers used by getAutoMemPath).

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FindGitRoot returns the absolute directory containing .git (paths.ts getAutoMemBase analogue).
// If start is a file path, search begins at its parent. Returns ("", false) if no ancestor has .git.
func FindGitRoot(start string) (abs string, ok bool) {
	start, err := filepath.Abs(filepath.Clean(start))
	if err != nil {
		return "", false
	}
	fi, err := os.Stat(start)
	if err == nil && !fi.IsDir() {
		start = filepath.Dir(start)
	}
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// MaxSanitizedLength matches sessionStoragePortable.ts MAX_SANITIZED_LENGTH (room for hash suffix).
const MaxSanitizedLength = 200

// djb2Hash matches utils/hash.ts djb2Hash for ASCII; non-ASCII uses Unicode code points (TS uses UTF-16 code units).
func djb2Hash(s string) int32 {
	var hash int32
	for _, r := range s {
		hash = ((hash << 5) - hash + int32(r)) | 0
	}
	return hash
}

// SanitizePath makes a single filesystem path component safe (sessionStoragePortable.ts sanitizePath).
func SanitizePath(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	sanitized := b.String()
	if len(sanitized) <= MaxSanitizedLength {
		return sanitized
	}
	h := djb2Hash(name)
	if h < 0 {
		h = -h
	}
	suffix := strconv.FormatInt(int64(h), 36)
	return sanitized[:MaxSanitizedLength] + "-" + suffix
}

// ConfigHomeDir resolves Claude-style config home (envUtils.getClaudeConfigHomeDir): RABBIT_CODE_CONFIG_DIR, CLAUDE_CONFIG_DIR, else ~/.claude.
func ConfigHomeDir() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_CODE_CONFIG_DIR")); s != "" {
		return filepath.Clean(s)
	}
	if s := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); s != "" {
		return filepath.Clean(s)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// MemoryBaseDir returns the root for projects/* layout (paths.ts getMemoryBaseDir).
func MemoryBaseDir() string {
	if s := strings.TrimSpace(os.Getenv("RABBIT_CODE_REMOTE_MEMORY_DIR")); s != "" {
		return filepath.Clean(s)
	}
	if s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); s != "" {
		return filepath.Clean(s)
	}
	return ConfigHomeDir()
}

// validateMemoryPath returns an absolute directory with trailing separator, or ("", false) if rejected (paths.ts validateMemoryPath).
func validateMemoryPath(raw string, expandTilde bool, homeDir string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	candidate := raw
	if expandTilde && (strings.HasPrefix(candidate, "~/") || strings.HasPrefix(candidate, `~\`)) {
		rest := candidate[2:]
		restNorm := filepath.Clean(rest)
		if restNorm == "." || restNorm == ".." {
			return "", false
		}
		if homeDir == "" {
			var err error
			homeDir, err = os.UserHomeDir()
			if err != nil || homeDir == "" {
				return "", false
			}
		}
		candidate = filepath.Join(homeDir, rest)
	}
	normalized := filepath.Clean(candidate)
	for len(normalized) > 1 {
		last := normalized[len(normalized)-1]
		if last == '/' || last == '\\' {
			normalized = normalized[:len(normalized)-1]
			continue
		}
		break
	}
	if !filepath.IsAbs(normalized) {
		return "", false
	}
	if len(normalized) < 3 {
		return "", false
	}
	if len(normalized) == 2 && normalized[1] == ':' {
		return "", false
	}
	if strings.HasPrefix(normalized, `\\`) || strings.HasPrefix(normalized, "//") {
		return "", false
	}
	if strings.ContainsRune(normalized, 0) {
		return "", false
	}
	out := normalized + string(filepath.Separator)
	return out, true
}

func autoMemPathOverride() (string, bool) {
	for _, k := range []string{"RABBIT_CODE_MEMORY_PATH_OVERRIDE", "CLAUDE_COWORK_MEMORY_PATH_OVERRIDE"} {
		if s := strings.TrimSpace(os.Getenv(k)); s != "" {
			if p, ok := validateMemoryPath(s, false, ""); ok {
				return p, true
			}
		}
	}
	return "", false
}

// HasAutoMemPathOverride is true when a validated full-path override env is set (paths.ts hasAutoMemPathOverride).
func HasAutoMemPathOverride() bool {
	_, ok := autoMemPathOverride()
	return ok
}

// AutoMemResolveOptions mirrors optional inputs for paths.ts getAutoMemPath (layer 2: trusted autoMemoryDirectory).
type AutoMemResolveOptions struct {
	TrustedAutoMemoryDirectory string
}

// ResolveAutoMemDir returns the auto-memory directory for a project (paths.ts getAutoMemPath).
func ResolveAutoMemDir(projectRoot string) (string, error) {
	return ResolveAutoMemDirWithOptions(projectRoot, AutoMemResolveOptions{})
}

// ResolveAutoMemDirWithOptions applies full resolution order: env full-path override, trusted
// autoMemoryDirectory (expandTilde), then <memoryBase>/projects/<sanitized-root>/memory/.
func ResolveAutoMemDirWithOptions(projectRoot string, opt AutoMemResolveOptions) (string, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", os.ErrInvalid
	}
	if p, ok := autoMemPathOverride(); ok {
		return p, nil
	}
	if s := strings.TrimSpace(opt.TrustedAutoMemoryDirectory); s != "" {
		if p, ok := validateMemoryPath(s, true, ""); ok {
			return p, nil
		}
	}
	base := MemoryBaseDir()
	keyRoot := projectRoot
	if g, ok := FindGitRoot(projectRoot); ok {
		keyRoot = g
	}
	absKey, err := filepath.Abs(filepath.Clean(keyRoot))
	if err != nil {
		return "", err
	}
	seg := SanitizePath(absKey)
	out := filepath.Join(base, "projects", seg, "memory")
	out = filepath.Clean(out) + string(filepath.Separator)
	return out, nil
}

// AutoMemEntrypointPath returns MEMORY.md inside the auto-memory dir.
func AutoMemEntrypointPath(projectRoot string) (string, error) {
	return AutoMemEntrypointPathWithOptions(projectRoot, AutoMemResolveOptions{})
}

// AutoMemEntrypointPathWithOptions is like AutoMemEntrypointPath but uses the same resolution options as ResolveAutoMemDirWithOptions.
func AutoMemEntrypointPathWithOptions(projectRoot string, opt AutoMemResolveOptions) (string, error) {
	dir, err := ResolveAutoMemDirWithOptions(projectRoot, opt)
	if err != nil {
		return "", err
	}
	return filepath.Join(strings.TrimSuffix(dir, string(filepath.Separator)), EntrypointName), nil
}

// AutoMemDailyLogPath returns <autoMem>/logs/YYYY/MM/YYYY-MM-DD.md (paths.ts getAutoMemDailyLogPath).
func AutoMemDailyLogPath(projectRoot string, t time.Time) (string, error) {
	return AutoMemDailyLogPathWithOptions(projectRoot, t, AutoMemResolveOptions{})
}

// AutoMemDailyLogPathWithOptions is like AutoMemDailyLogPath with explicit resolve options.
func AutoMemDailyLogPathWithOptions(projectRoot string, t time.Time, opt AutoMemResolveOptions) (string, error) {
	dir, err := ResolveAutoMemDirWithOptions(projectRoot, opt)
	if err != nil {
		return "", err
	}
	root := strings.TrimSuffix(dir, string(filepath.Separator))
	y, mo, d := t.Date()
	yyyy := fmt.Sprintf("%04d", y)
	mm := fmt.Sprintf("%02d", int(mo))
	dd := fmt.Sprintf("%02d", d)
	return filepath.Join(root, "logs", yyyy, mm, fmt.Sprintf("%s-%s-%s.md", yyyy, mm, dd)), nil
}

// IsAutoMemPath reports whether absolutePath is under autoMemDir (paths.ts isAutoMemPath).
func IsAutoMemPath(absolutePath, autoMemDir string) bool {
	p, err := filepath.Abs(absolutePath)
	if err != nil {
		return false
	}
	root := strings.TrimSuffix(autoMemDir, string(filepath.Separator))
	root, err = filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

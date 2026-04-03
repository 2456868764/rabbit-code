package memdir

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// TeamMemSubdir is the subdirectory under auto-memory for shared team files (teamMemPaths.ts getTeamMemPath).
const TeamMemSubdir = "team"

// ErrTeamMemPathKey indicates a rejected relative key for team memory (subset of teamMemPaths.sanitizePathKey).
var ErrTeamMemPathKey = errors.New("memdir: invalid team memory path key")

// ErrTeamMemPathEscape means resolved path left the team memory root (symlink / traversal guard).
var ErrTeamMemPathEscape = errors.New("memdir: team memory path escapes team root")

// TeamMemDirFromAutoMemDir returns <autoMem>/team/ with trailing separator. autoMemDir should be ResolveAutoMemDir output.
func TeamMemDirFromAutoMemDir(autoMemDir string) string {
	root := strings.TrimSuffix(strings.TrimSpace(autoMemDir), string(filepath.Separator))
	if root == "" {
		return ""
	}
	return filepath.Join(root, TeamMemSubdir) + string(filepath.Separator)
}

// TeamMemEntrypointFromAutoMemDir returns <autoMem>/team/MEMORY.md.
func TeamMemEntrypointFromAutoMemDir(autoMemDir string) string {
	root := strings.TrimSuffix(strings.TrimSpace(autoMemDir), string(filepath.Separator))
	if root == "" {
		return ""
	}
	return filepath.Join(root, TeamMemSubdir, EntrypointName)
}

// IsTeamMemPathUnderAutoMem reports whether absolutePath resolves under the team subdirectory of autoMemDir (string containment; no symlink resolution).
func IsTeamMemPathUnderAutoMem(absolutePath, autoMemDir string) bool {
	teamRoot := strings.TrimSuffix(TeamMemDirFromAutoMemDir(autoMemDir), string(filepath.Separator))
	if teamRoot == "" {
		return false
	}
	p, err := filepath.Abs(absolutePath)
	if err != nil {
		return false
	}
	teamAbs, err := filepath.Abs(teamRoot)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(teamAbs, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// SanitizeTeamMemPathKey rejects null bytes, backslashes, absolute keys, and obvious URL-encoded traversals (teamMemPaths.sanitizePathKey subset).
// Keys are normalized with Unicode NFKC before checks (homograph / compatibility forms).
func SanitizeTeamMemPathKey(key string) error {
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("%w: null byte", ErrTeamMemPathKey)
	}
	key = norm.NFKC.String(key)
	decoded := key
	if d, err := url.PathUnescape(key); err == nil {
		decoded = d
	}
	if decoded != key && (strings.Contains(decoded, "..") || strings.Contains(decoded, "/") || strings.Contains(decoded, "\\")) {
		return fmt.Errorf("%w: url-encoded traversal", ErrTeamMemPathKey)
	}
	if strings.Contains(key, "\\") {
		return fmt.Errorf("%w: backslash", ErrTeamMemPathKey)
	}
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("%w: absolute path", ErrTeamMemPathKey)
	}
	return nil
}

// ValidateTeamMemWritePath reports whether absolutePath resolves (symlinks) inside <autoMem>/team/.
func ValidateTeamMemWritePath(absolutePath, autoMemDir string) error {
	autoBase, err := filepath.Abs(filepath.Clean(strings.TrimSpace(autoMemDir)))
	if err != nil {
		return err
	}
	if autoBase == "" || autoBase == "." {
		return fmt.Errorf("%w: empty auto memory dir", ErrTeamMemPathEscape)
	}
	teamRoot := filepath.Join(autoBase, TeamMemSubdir)
	p, err := filepath.Abs(absolutePath)
	if err != nil {
		return err
	}
	p = filepath.Clean(p)
	// File may not exist yet: canonicalize via its parent so /var vs /private/var (macOS) matches teamRoot.
	dir := filepath.Dir(p)
	dirEval, derr := filepath.EvalSymlinks(dir)
	if derr != nil {
		dirEval = dir
	}
	p = filepath.Join(dirEval, filepath.Base(p))

	teamEval, err := filepath.EvalSymlinks(teamRoot)
	if err != nil {
		teamEval = teamRoot
	}
	rel, err := filepath.Rel(teamEval, p)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTeamMemPathEscape, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ErrTeamMemPathEscape
	}
	return nil
}

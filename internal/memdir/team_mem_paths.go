package memdir

// Corresponds to restored-src/src/memdir/teamMemPaths.ts plus team memory secret scan / write guard.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
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

// SanitizeTeamMemPathKey mirrors teamMemPaths.ts sanitizePathKey (order: null, URL decode, NFKC drift, backslash, absolute).
func SanitizeTeamMemPathKey(key string) error {
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("%w: null byte", ErrTeamMemPathKey)
	}
	decoded := key
	if d, err := url.PathUnescape(key); err == nil {
		decoded = d
	}
	if decoded != key && (strings.Contains(decoded, "..") || strings.Contains(decoded, "/") || strings.Contains(decoded, "\\")) {
		return fmt.Errorf("%w: url-encoded traversal", ErrTeamMemPathKey)
	}
	normalized := norm.NFKC.String(key)
	if normalized != key && (strings.Contains(normalized, "..") || strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") || strings.ContainsRune(normalized, 0)) {
		return fmt.Errorf("%w: unicode-normalized traversal", ErrTeamMemPathKey)
	}
	if strings.Contains(key, "\\") {
		return fmt.Errorf("%w: backslash", ErrTeamMemPathKey)
	}
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("%w: absolute path", ErrTeamMemPathKey)
	}
	return nil
}

// ValidateTeamMemWritePath reports whether absolutePath is safe for team writes (teamMemPaths.ts validateTeamMemWritePath).
func ValidateTeamMemWritePath(absolutePath, autoMemDir string) error {
	_, err := ValidateTeamMemWritePathFull(absolutePath, autoMemDir)
	return err
}

// PathTraversalError mirrors teamMemPaths.ts PathTraversalError (PSR M22186 / M22187).
type PathTraversalError struct {
	Msg string
}

func (e *PathTraversalError) Error() string {
	if e == nil {
		return "memdir: path traversal"
	}
	return "memdir: " + e.Msg
}

func pathErrno(err error) syscall.Errno {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		if n, ok := pe.Err.(syscall.Errno); ok {
			return n
		}
	}
	return 0
}

// RealpathDeepestExisting mirrors teamMemPaths.ts realpathDeepestExisting: resolve symlinks for the
// deepest existing ancestor, then rejoin non-existing tail.
func RealpathDeepestExisting(absPath string) (string, error) {
	cur, err := filepath.Abs(filepath.Clean(absPath))
	if err != nil {
		return "", err
	}
	orig := cur
	var tail []string
	for {
		parent := filepath.Dir(cur)
		realCurrent, err := filepath.EvalSymlinks(cur)
		if err == nil {
			out := realCurrent
			for i := len(tail) - 1; i >= 0; i-- {
				out = filepath.Join(out, tail[i])
			}
			return filepath.Clean(out), nil
		}
		if parent == cur {
			return orig, nil
		}
		errno := pathErrno(err)
		switch {
		case errors.Is(err, fs.ErrNotExist) || errno == syscall.ENOENT:
			fi, err2 := os.Lstat(cur)
			if err2 == nil && fi.Mode()&os.ModeSymlink != 0 {
				return "", &PathTraversalError{Msg: fmt.Sprintf("dangling symlink detected (target does not exist): %q", cur)}
			}
		case errno == syscall.ELOOP:
			return "", &PathTraversalError{Msg: fmt.Sprintf("symlink loop detected in path: %q", cur)}
		case errno == syscall.EACCES || errno == syscall.EPERM || errno == syscall.EIO:
			return "", &PathTraversalError{Msg: fmt.Sprintf("cannot verify path containment (%v): %q", err, cur)}
		}
		seg, ok := childSegment(parent, cur)
		if !ok {
			return orig, nil
		}
		tail = append(tail, seg)
		cur = parent
	}
}

func childSegment(parent, full string) (string, bool) {
	rel, err := filepath.Rel(parent, full)
	if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		return rel, true
	}
	s := strings.TrimPrefix(full, parent)
	s = strings.Trim(s, string(filepath.Separator))
	if s == "" {
		return "", false
	}
	return s, true
}

// IsRealPathWithinTeamDir reports whether realCandidate is under the real team directory (teamMemPaths.ts isRealPathWithinTeamDir).
func IsRealPathWithinTeamDir(realCandidate string, autoMemDir string) (bool, error) {
	autoBase := filepath.Clean(strings.TrimSpace(autoMemDir))
	teamRoot := filepath.Join(autoBase, TeamMemSubdir)
	realTeam, err := filepath.EvalSymlinks(teamRoot)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
			return true, nil
		}
		return false, err
	}
	realTeam = filepath.Clean(realTeam)
	rc := filepath.Clean(realCandidate)
	if rc == realTeam {
		return true, nil
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(rc, realTeam+sep), nil
}

func teamDirPrefix(autoBase string) string {
	return filepath.Join(autoBase, TeamMemSubdir) + string(filepath.Separator)
}

func underTeamDirFirstPass(resolvedPath, autoBase string) bool {
	td := teamDirPrefix(autoBase)
	if strings.HasPrefix(resolvedPath, td) {
		return true
	}
	teamOnly := filepath.Join(autoBase, TeamMemSubdir)
	return resolvedPath == teamOnly
}

// TeamMemPathResolved reports whether filePath is under <autoMem>/team/ after Abs+Clean (teamMemPaths.ts isTeamMemPath resolve pass).
func TeamMemPathResolved(filePath, autoMemDir string) bool {
	p, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return false
	}
	autoBase, err := filepath.Abs(filepath.Clean(strings.TrimSpace(autoMemDir)))
	if err != nil {
		return false
	}
	return underTeamDirFirstPass(p, autoBase)
}

// ValidateTeamMemWritePathFull mirrors teamMemPaths.ts validateTeamMemWritePath (two-pass + symlink containment).
func ValidateTeamMemWritePathFull(filePath, autoMemDir string) (resolvedPath string, err error) {
	if strings.ContainsRune(filePath, 0) {
		return "", &PathTraversalError{Msg: fmt.Sprintf("null byte in path: %q", filePath)}
	}
	autoBase, err := filepath.Abs(filepath.Clean(strings.TrimSpace(autoMemDir)))
	if err != nil {
		return "", err
	}
	resolvedPath, err = filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	if !underTeamDirFirstPass(resolvedPath, autoBase) {
		return "", &PathTraversalError{Msg: fmt.Sprintf("path escapes team memory directory: %q", filePath)}
	}
	realPath, err := RealpathDeepestExisting(resolvedPath)
	if err != nil {
		return "", err
	}
	ok, err := IsRealPathWithinTeamDir(realPath, autoBase)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", &PathTraversalError{Msg: fmt.Sprintf("path escapes team memory directory via symlink: %q", filePath)}
	}
	return resolvedPath, nil
}

// ValidateTeamMemKey mirrors teamMemPaths.ts validateTeamMemKey (sanitize key + join + two-pass).
func ValidateTeamMemKey(relativeKey, autoMemDir string) (resolvedPath string, err error) {
	if err := SanitizeTeamMemPathKey(relativeKey); err != nil {
		return "", err
	}
	autoBase, err := filepath.Abs(filepath.Clean(strings.TrimSpace(autoMemDir)))
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(autoBase, TeamMemSubdir, filepath.FromSlash(relativeKey))
	resolvedPath, err = filepath.Abs(filepath.Clean(fullPath))
	if err != nil {
		return "", err
	}
	if !underTeamDirFirstPass(resolvedPath, autoBase) {
		return "", &PathTraversalError{Msg: fmt.Sprintf("key escapes team memory directory: %q", relativeKey)}
	}
	realPath, err := RealpathDeepestExisting(resolvedPath)
	if err != nil {
		return "", err
	}
	ok, err := IsRealPathWithinTeamDir(realPath, autoBase)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", &PathTraversalError{Msg: fmt.Sprintf("key escapes team memory directory via symlink: %q", relativeKey)}
	}
	return resolvedPath, nil
}

// IsTeamMemFileActive is teamMemPaths.ts isTeamMemFile when teamMemoryEnabled is true.
func IsTeamMemFileActive(filePath, autoMemDir string, teamMemoryEnabled bool) bool {
	if !teamMemoryEnabled {
		return false
	}
	return TeamMemPathResolved(filePath, autoMemDir)
}

// SecretMatch records a gitleaks-style rule hit (secretScanner.ts); matched text is never stored.
type SecretMatch struct {
	RuleID string
	Label  string
}

type compiledSecretRule struct {
	id string
	re *regexp.Regexp
}

var (
	secretRulesOnce sync.Once
	secretRules     []compiledSecretRule
)

// ScanTeamMemorySecrets returns one match per rule that fires (deduped by rule id), mirroring secretScanner.scanForSecrets.
func ScanTeamMemorySecrets(content string) []SecretMatch {
	secretRulesOnce.Do(initTeamMemSecretRules)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	var out []SecretMatch
	seen := map[string]struct{}{}
	for _, r := range secretRules {
		if _, ok := seen[r.id]; ok {
			continue
		}
		if r.re.MatchString(content) {
			seen[r.id] = struct{}{}
			out = append(out, SecretMatch{RuleID: r.id, Label: secretRuleIDToLabel(r.id)})
		}
	}
	return out
}

func initTeamMemSecretRules() {
	antPfx := strings.Join([]string{"sk", "ant", "api"}, "-")
	bt := "`"
	sfx := "(?:[" + bt + `'";\s]|\n|\r|\\[nr]|$)`

	raw := []struct{ id, src string }{
		{"aws-access-token", `\b((?:A3T[A-Z0-9]|AKIA|ASIA|ABIA|ACCA)[A-Z2-7]{16})\b`},
		{"gcp-api-key", `\b(AIza[\w-]{35})` + sfx},
		{"digitalocean-pat", `\b(dop_v1_[a-f0-9]{64})` + sfx},
		{"digitalocean-access-token", `\b(doo_v1_[a-f0-9]{64})` + sfx},
		{"anthropic-api-key", `\b(` + antPfx + `03-[a-zA-Z0-9_\-]{93}AA)` + sfx},
		{"anthropic-admin-api-key", `\b(sk-ant-admin01-[a-zA-Z0-9_\-]{93}AA)` + sfx},
		{"huggingface-access-token", `\b(hf_[a-zA-Z]{34})` + sfx},
		{"github-pat", `ghp_[0-9a-zA-Z]{36}`},
		{"github-fine-grained-pat", `github_pat_\w{82}`},
		{"github-app-token", `(?:ghu|ghs)_[0-9a-zA-Z]{36}`},
		{"github-oauth", `gho_[0-9a-zA-Z]{36}`},
		{"github-refresh-token", `ghr_[0-9a-zA-Z]{36}`},
		{"gitlab-pat", `glpat-[\w-]{20}`},
		{"gitlab-deploy-token", `gldt-[0-9a-zA-Z_\-]{20}`},
		{"slack-bot-token", `xoxb-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`},
		{"npm-access-token", `\b(npm_[a-zA-Z0-9]{36})` + sfx},
		{"databricks-api-token", `\b(dapi[a-f0-9]{32}(?:-\d)?)` + sfx},
		{"pulumi-api-token", `\b(pul-[a-f0-9]{40})` + sfx},
		{"stripe-access-token", `\b((?:sk|rk)_(?:test|live|prod)_[a-zA-Z0-9]{10,99})` + sfx},
		{"shopify-access-token", `shpat_[a-fA-F0-9]{32}`},
		{"shopify-shared-secret", `shpss_[a-fA-F0-9]{32}`},
		{"twilio-api-key", `SK[0-9a-fA-F]{32}`},
		{"private-key", `(?is)-----BEGIN[ A-Z0-9_-]{0,100}PRIVATE KEY(?: BLOCK)?-----[\s\S-]{64,}?-----END[ A-Z0-9_-]{0,100}PRIVATE KEY(?: BLOCK)?-----`},
	}
	for _, r := range raw {
		secretRules = append(secretRules, compiledSecretRule{id: r.id, re: regexp.MustCompile(r.src)})
	}
}

func secretRuleIDToLabel(ruleID string) string {
	parts := strings.Split(ruleID, "-")
	special := map[string]string{
		"aws": "AWS", "gcp": "GCP", "api": "API", "pat": "PAT", "ad": "AD",
		"tf": "TF", "oauth": "OAuth", "npm": "NPM", "pypi": "PyPI", "jwt": "JWT",
		"github": "GitHub", "gitlab": "GitLab", "openai": "OpenAI",
		"digitalocean": "DigitalOcean", "huggingface": "HuggingFace",
		"hashicorp": "HashiCorp", "sendgrid": "SendGrid",
	}
	for i, p := range parts {
		if s, ok := special[p]; ok {
			parts[i] = s
			continue
		}
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// TeamMemSecretGuardRunner blocks Write/Edit when content matches teamMemSecretScan (teamMemSecretGuard.ts).
type TeamMemSecretGuardRunner struct {
	Inner      querydeps.ToolRunner
	AutoMemDir string
	Enabled    bool
}

// RunTool implements querydeps.ToolRunner.
func (w *TeamMemSecretGuardRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if w.Inner == nil {
		return nil, querydeps.ErrNoToolRunner
	}
	if w.Enabled && w.AutoMemDir != "" {
		if fp, content, ok := toolWritePathAndContent(name, inputJSON); ok && content != "" {
			if abs, err := filepath.Abs(fp); err == nil && IsTeamMemPathUnderAutoMem(abs, w.AutoMemDir+string(filepath.Separator)) {
				if err := ValidateTeamMemWritePath(abs, w.AutoMemDir); err != nil {
					return nil, err
				}
				if matches := ScanTeamMemorySecrets(content); len(matches) > 0 {
					var labels []string
					for _, m := range matches {
						labels = append(labels, m.Label)
					}
					return nil, fmt.Errorf(
						"memdir: content contains potential secrets (%s) and cannot be written to team memory. "+
							"Team memory is shared with all repository collaborators. Remove the sensitive content and try again",
						strings.Join(labels, ", "),
					)
				}
			}
		}
	}
	return w.Inner.RunTool(ctx, name, inputJSON)
}

func toolWritePathAndContent(toolName string, inputJSON []byte) (filePath string, content string, ok bool) {
	n := strings.ToLower(strings.TrimSpace(toolName))
	if n != "write" && n != "edit" {
		return "", "", false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(inputJSON, &m); err != nil {
		return "", "", false
	}
	fp := jsonStringField(m["file_path"])
	if fp == "" {
		return "", "", false
	}
	switch n {
	case "write":
		return fp, jsonStringField(m["content"]), true
	case "edit":
		oldS := jsonStringField(m["old_string"])
		newS := jsonStringField(m["new_string"])
		return fp, oldS + "\n" + newS, true
	default:
		return "", "", false
	}
}

func jsonStringField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

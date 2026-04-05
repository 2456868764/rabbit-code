package globtool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

type globCtxKey struct{}

// GlobContext mirrors GlobTool.ts call inputs (globLimits, permission filtering, ignore globs).
type GlobContext struct {
	// MaxResults caps matches (TS globLimits.maxResults, default 100).
	MaxResults int
	// DenyRead if non-nil, paths for which DenyRead(abs) is true are dropped.
	DenyRead func(absPath string) bool
	// IgnoreGlobs are passed to ripgrep as repeated --glob !pattern (TS getFileReadIgnorePatterns subset).
	IgnoreGlobs []string
}

// WithGlobContext attaches *GlobContext for Glob.Run.
func WithGlobContext(ctx context.Context, gc *GlobContext) context.Context {
	if gc == nil {
		return ctx
	}
	return context.WithValue(ctx, globCtxKey{}, gc)
}

// GlobContextFrom returns *GlobContext or nil.
func GlobContextFrom(ctx context.Context) *GlobContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(globCtxKey{}).(*GlobContext)
	return v
}

// Glob implements tools.Tool (GlobTool.ts).
type Glob struct{}

// New returns a Glob tool.
func New() *Glob { return &Glob{} }

func (g *Glob) Name() string { return GlobToolName }

func (g *Glob) Aliases() []string { return nil }

type globInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type globOutput struct {
	DurationMs int64    `json:"durationMs"`
	NumFiles   int      `json:"numFiles"`
	Filenames  []string `json:"filenames"`
	Truncated  bool     `json:"truncated"`
}

var globSpecialRE = regexp.MustCompile(`[*?[{]`)

func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
}

// extractGlobBaseDirectory mirrors utils/glob.ts extractGlobBaseDirectory.
func extractGlobBaseDirectory(pattern string) (baseDir, relativePattern string) {
	loc := globSpecialRE.FindStringIndex(pattern)
	if loc == nil {
		return filepath.Dir(pattern), filepath.Base(pattern)
	}
	staticPrefix := pattern[:loc[0]]
	lastSep := -1
	for i := len(staticPrefix) - 1; i >= 0; i-- {
		c := staticPrefix[i]
		if c == '/' || c == filepath.Separator {
			lastSep = i
			break
		}
	}
	if lastSep == -1 {
		return "", pattern
	}
	baseDir = staticPrefix[:lastSep]
	relativePattern = pattern[lastSep+1:]
	if baseDir == "" && lastSep == 0 {
		baseDir = string(filepath.Separator)
	}
	if runtime.GOOS == "windows" && len(baseDir) == 2 && baseDir[1] == ':' {
		baseDir += string(filepath.Separator)
	}
	return baseDir, relativePattern
}

func envGlobTruthyOrDefaultTrue(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return true
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func globTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_GLOB_TIMEOUT_SECONDS")); v != "" {
		sec, err := strconv.Atoi(v)
		if err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return 60 * time.Second
	}
	return 20 * time.Second
}

func parseGlobInputJSON(b []byte) (globInput, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var in globInput
	if err := dec.Decode(&in); err != nil {
		return globInput{}, err
	}
	if dec.More() {
		return globInput{}, errors.New("globtool: invalid json: extra data after input object")
	}
	return in, nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func toRelativePathDisplay(abs string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return abs
	}
	rel, err := filepath.Rel(cwd, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return abs
	}
	return rel
}

func (g *Glob) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	start := time.Now()

	in, err := parseGlobInputJSON(inputJSON)
	if err != nil {
		return nil, fmt.Errorf("globtool: invalid json: %w", err)
	}
	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		return nil, errors.New("globtool: missing pattern")
	}

	var searchDir string
	if strings.TrimSpace(in.Path) != "" {
		abs, err := filereadtool.ExpandPath(strings.TrimSpace(in.Path))
		if err != nil {
			return nil, fmt.Errorf("globtool: path: %w", err)
		}
		if !isUncPath(abs) {
			fi, statErr := os.Stat(abs)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					msg := fmt.Sprintf("Directory does not exist: %s. %s %s.", strings.TrimSpace(in.Path), fileedittool.FileNotFoundCwdNote, mustGetwd())
					if sug, ok := fileedittool.SuggestPathUnderCwd(abs); ok {
						msg += fmt.Sprintf(" Did you mean %s?", sug)
					}
					return nil, errors.New(msg)
				}
				return nil, statErr
			}
			if !fi.IsDir() {
				return nil, fmt.Errorf("Path is not a directory: %s", strings.TrimSpace(in.Path))
			}
		}
		searchDir = abs
	} else {
		searchDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	searchDir, err = filepath.Abs(searchDir)
	if err != nil {
		return nil, err
	}

	searchPattern := pattern
	if filepath.IsAbs(pattern) {
		bd, rel := extractGlobBaseDirectory(pattern)
		if bd != "" {
			bdClean, err := filepath.Abs(bd)
			if err != nil {
				return nil, err
			}
			searchDir = bdClean
			searchPattern = rel
		}
	}

	gc := GlobContextFrom(ctx)
	limit := 100
	if gc != nil && gc.MaxResults > 0 {
		limit = gc.MaxResults
	}

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		return nil, errors.New("globtool: ripgrep (rg) not found in PATH; install from https://github.com/BurntSushi/ripgrep")
	}

	allAbs, err := ripgrepListFiles(ctx, rgPath, searchDir, searchPattern, gc)
	if err != nil {
		return nil, err
	}

	if gc != nil && gc.DenyRead != nil {
		filtered := allAbs[:0]
		for _, p := range allAbs {
			if !gc.DenyRead(p) {
				filtered = append(filtered, p)
			}
		}
		allAbs = filtered
	}

	truncated := len(allAbs) > limit
	files := allAbs
	if truncated {
		files = allAbs[:limit]
	}

	display := make([]string, 0, len(files))
	for _, p := range files {
		display = append(display, toRelativePathDisplay(p))
	}

	out := globOutput{
		DurationMs: time.Since(start).Milliseconds(),
		NumFiles:   len(display),
		Filenames:  display,
		Truncated:  truncated,
	}
	return json.Marshal(out)
}

func ripgrepListFiles(ctx context.Context, rgPath, searchDir, searchPattern string, gc *GlobContext) ([]string, error) {
	timeout := globTimeout()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{
		"--files",
		"--glob", searchPattern,
		"--sort=modified",
	}
	if envGlobTruthyOrDefaultTrue("CLAUDE_CODE_GLOB_NO_IGNORE") {
		args = append(args, "--no-ignore")
	}
	if envGlobTruthyOrDefaultTrue("CLAUDE_CODE_GLOB_HIDDEN") {
		args = append(args, "--hidden")
	}
	if gc != nil {
		for _, ign := range gc.IgnoreGlobs {
			ign = strings.TrimSpace(ign)
			if ign == "" {
				continue
			}
			args = append(args, "--glob", "!"+ign)
		}
	}
	args = append(args, searchDir)

	cmd := exec.CommandContext(runCtx, rgPath, args...)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("globtool: ripgrep timed out after %v", timeout)
		}
		if len(stderr.Bytes()) > 0 {
			return nil, fmt.Errorf("globtool: ripgrep: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("globtool: ripgrep: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var abs string
		if filepath.IsAbs(line) {
			abs = filepath.Clean(line)
		} else {
			abs = filepath.Clean(filepath.Join(searchDir, line))
		}
		out = append(out, abs)
	}
	return out, nil
}

var _ tools.Tool = (*Glob)(nil)

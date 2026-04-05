package greptool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
)

// Upstream: GrepTool.ts DEFAULT_HEAD_LIMIT, maxResultSizeChars, VCS_DIRECTORIES_TO_EXCLUDE.
const (
	defaultHeadLimit   = 250
	maxResultSizeChars = 20_000
)

var vcsDirs = []string{".git", ".svn", ".hg", ".bzr", ".jj", ".sl"}

var errRipgrepEagain = errors.New("greptool: ripgrep EAGAIN")

// Grep implements tools.Tool (GrepTool.ts).
type Grep struct{}

// New returns a Grep tool.
func New() *Grep { return &Grep{} }

func (g *Grep) Name() string { return GrepToolName }

func (g *Grep) Aliases() []string { return nil }

type grepInput struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path,omitempty"`
	Glob        string `json:"glob,omitempty"`
	OutputMode  string `json:"output_mode,omitempty"`
	ContextB    *int   `json:"-B,omitempty"`
	ContextA    *int   `json:"-A,omitempty"`
	ContextC    *int   `json:"-C,omitempty"`
	Context     *int   `json:"context,omitempty"`
	ShowLineNum *bool  `json:"-n,omitempty"`
	CaseFold    *bool  `json:"-i,omitempty"`
	Type        string `json:"type,omitempty"`
	HeadLimit   *int   `json:"head_limit,omitempty"`
	Offset      *int   `json:"offset,omitempty"`
	Multiline   *bool  `json:"multiline,omitempty"`
}

type grepOutput struct {
	Mode          string   `json:"mode,omitempty"`
	NumFiles      int      `json:"numFiles"`
	Filenames     []string `json:"filenames"`
	Content       string   `json:"content,omitempty"`
	NumLines      int      `json:"numLines,omitempty"`
	NumMatches    int      `json:"numMatches,omitempty"`
	AppliedLimit  *int     `json:"appliedLimit,omitempty"`
	AppliedOffset *int     `json:"appliedOffset,omitempty"`
}

func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
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

func grepTimeout() time.Duration {
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

func parseGrepInputJSON(b []byte) (grepInput, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var in grepInput
	if err := dec.Decode(&in); err != nil {
		return grepInput{}, err
	}
	if dec.More() {
		return grepInput{}, errors.New("greptool: invalid json: extra data after input object")
	}
	return in, nil
}

// splitGlobPatterns mirrors GrepTool.ts glob splitting (commas / spaces, brace groups preserved).
func splitGlobPatterns(glob string) []string {
	glob = strings.TrimSpace(glob)
	if glob == "" {
		return nil
	}
	raw := strings.Fields(glob)
	var out []string
	for _, rawPattern := range raw {
		if strings.Contains(rawPattern, "{") && strings.Contains(rawPattern, "}") {
			out = append(out, rawPattern)
			continue
		}
		for _, p := range strings.Split(rawPattern, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func showLineNumbers(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func boolVal(v *bool) bool {
	return v != nil && *v
}

func coalesceOffset(v *int) int {
	if v == nil || *v < 0 {
		return 0
	}
	return *v
}

func applyHeadLimit[T any](items []T, headLimit *int, offset int) (out []T, appliedLimit *int, appliedOffset *int) {
	var offOut *int
	if offset > 0 {
		offOut = new(int)
		*offOut = offset
	}
	if headLimit != nil && *headLimit == 0 {
		if offset >= len(items) {
			return nil, nil, offOut
		}
		return items[offset:], nil, offOut
	}
	limit := defaultHeadLimit
	if headLimit != nil {
		limit = *headLimit
	}
	off := offset
	if off > len(items) {
		off = len(items)
	}
	sliced := items[off:]
	if len(sliced) <= limit {
		return sliced, nil, offOut
	}
	al := limit
	return sliced[:limit], &al, offOut
}

func splitFirstRgPathColon(line string) (pathPart, rest string) {
	runes := []rune(line)
	if len(runes) >= 2 && runes[1] == ':' && unicode.IsLetter(runes[0]) {
		for i := 2; i < len(runes); i++ {
			if runes[i] == ':' {
				return string(runes[:i]), string(runes[i:])
			}
		}
		return line, ""
	}
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return line, ""
	}
	return line[:i], line[i:]
}

func resolvePathLine(line, searchRoot string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	if filepath.IsAbs(line) {
		return filepath.Clean(line)
	}
	return filepath.Clean(filepath.Join(searchRoot, line))
}

func splitNonEmptyLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimRight(p, "\r")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (g *Grep) runRipgrep(ctx context.Context, rgPath, target string, args []string) ([]string, error) {
	timeout := grepTimeout()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	try := func(threadArg []string) ([]string, error) {
		full := append(append(append([]string(nil), threadArg...), args...), target)
		cmd := exec.CommandContext(runCtx, rgPath, full...)
		cmd.Env = os.Environ()
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		errStr := stderr.String()
		rawOut := strings.TrimSpace(stdout.String())
		lines := splitNonEmptyLines(rawOut)

		if runCtx.Err() == context.DeadlineExceeded {
			if len(lines) == 0 {
				return nil, fmt.Errorf("greptool: ripgrep timed out after %v", timeout)
			}
			if len(lines) > 1 {
				lines = lines[:len(lines)-1]
			} else {
				lines = nil
			}
			return lines, nil
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				return nil, nil
			}
		}
		if err != nil {
			if strings.Contains(errStr, "os error 11") || strings.Contains(errStr, "Resource temporarily unavailable") {
				return nil, errRipgrepEagain
			}
			if strings.TrimSpace(errStr) != "" {
				return nil, fmt.Errorf("greptool: ripgrep: %w: %s", err, strings.TrimSpace(errStr))
			}
			return nil, fmt.Errorf("greptool: ripgrep: %w", err)
		}
		return lines, nil
	}

	lines, err := try(nil)
	if errors.Is(err, errRipgrepEagain) {
		lines, err = try([]string{"-j", "1"})
	}
	return lines, err
}

func appendIgnoreGlobs(args []string, gc *GrepContext, cwd string) []string {
	if gc == nil {
		return args
	}
	for _, ign := range gc.IgnoreGlobs {
		ign = strings.TrimSpace(ign)
		if ign == "" {
			continue
		}
		if strings.HasPrefix(ign, "!") {
			args = append(args, "--glob", ign)
		} else {
			args = append(args, "--glob", "!"+ign)
		}
	}
	if len(gc.FileReadDenyPatternsByRoot) > 0 {
		for _, g := range globtool.NormalizeFileReadDenyPatternsToSearchDir(gc.FileReadDenyPatternsByRoot, cwd) {
			args = append(args, "--glob", g)
		}
	}
	return args
}

func buildRipgrepArgs(in grepInput, outputMode string, cwd string, gc *GrepContext, rgPath string, ctx context.Context, absSearch string) []string {
	args := []string{"--hidden"}
	for _, dir := range vcsDirs {
		args = append(args, "--glob", "!"+dir)
	}
	args = append(args, "--max-columns", "500")

	if boolVal(in.Multiline) {
		args = append(args, "-U", "--multiline-dotall")
	}
	if boolVal(in.CaseFold) {
		args = append(args, "-i")
	}
	switch outputMode {
	case "files_with_matches":
		args = append(args, "-l")
	case "count":
		args = append(args, "-c")
	}

	if outputMode == "content" && showLineNumbers(in.ShowLineNum) {
		args = append(args, "-n")
	}

	if outputMode == "content" {
		if in.Context != nil {
			args = append(args, "-C", strconv.Itoa(*in.Context))
		} else if in.ContextC != nil {
			args = append(args, "-C", strconv.Itoa(*in.ContextC))
		} else {
			if in.ContextB != nil {
				args = append(args, "-B", strconv.Itoa(*in.ContextB))
			}
			if in.ContextA != nil {
				args = append(args, "-A", strconv.Itoa(*in.ContextA))
			}
		}
	}

	pattern := strings.TrimSpace(in.Pattern)
	if strings.HasPrefix(pattern, "-") {
		args = append(args, "-e", pattern)
	} else {
		args = append(args, pattern)
	}

	if t := strings.TrimSpace(in.Type); t != "" {
		args = append(args, "--type", t)
	}
	for _, gp := range splitGlobPatterns(in.Glob) {
		args = append(args, "--glob", gp)
	}

	args = appendIgnoreGlobs(args, gc, cwd)
	for _, ex := range globtool.GlobExclusionsForPluginCache(ctx, rgPath, absSearch) {
		args = append(args, "--glob", ex)
	}
	return args
}

func (g *Grep) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	in, err := parseGrepInputJSON(inputJSON)
	if err != nil {
		return nil, fmt.Errorf("greptool: invalid json: %w", err)
	}
	pattern := strings.TrimSpace(in.Pattern)
	if pattern == "" {
		return nil, errors.New("greptool: missing pattern")
	}

	outputMode := strings.TrimSpace(in.OutputMode)
	if outputMode == "" {
		outputMode = "files_with_matches"
	}
	switch outputMode {
	case "content", "files_with_matches", "count":
	default:
		return nil, fmt.Errorf("greptool: invalid output_mode %q", in.OutputMode)
	}

	cwd, err := filepath.Abs(mustGetwd())
	if err != nil {
		return nil, err
	}

	var absSearch string
	if strings.TrimSpace(in.Path) != "" {
		absSearch, err = filereadtool.ExpandPath(strings.TrimSpace(in.Path))
		if err != nil {
			return nil, fmt.Errorf("greptool: path: %w", err)
		}
		if !isUncPath(absSearch) {
			fi, statErr := os.Stat(absSearch)
			if statErr != nil {
				if os.IsNotExist(statErr) {
					msg := fmt.Sprintf("Path does not exist: %s. %s %s.", strings.TrimSpace(in.Path), fileedittool.FileNotFoundCwdNote, mustGetwd())
					if sug, ok := fileedittool.SuggestPathUnderCwd(absSearch); ok {
						msg += fmt.Sprintf(" Did you mean %s?", sug)
					}
					return nil, errors.New(msg)
				}
				return nil, statErr
			}
			_ = fi
		}
	} else {
		absSearch = cwd
	}
	absSearch, err = filepath.Abs(absSearch)
	if err != nil {
		return nil, err
	}

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		return nil, errors.New("greptool: ripgrep (rg) not found in PATH; install from https://github.com/BurntSushi/ripgrep")
	}

	gc := GrepContextFrom(ctx)
	args := buildRipgrepArgs(in, outputMode, cwd, gc, rgPath, ctx, absSearch)

	rawLines, err := g.runRipgrep(ctx, rgPath, absSearch, args)
	if err != nil {
		return nil, err
	}

	offset := coalesceOffset(in.Offset)

	switch outputMode {
	case "content":
		lim, al, ao := applyHeadLimit(rawLines, in.HeadLimit, offset)
		final := make([]string, 0, len(lim))
		for _, line := range lim {
			fp, rest := splitFirstRgPathColon(line)
			if fp != "" && rest != "" {
				absP := resolvePathLine(fp, absSearch)
				final = append(final, toRelativePathDisplay(absP)+rest)
			} else {
				final = append(final, line)
			}
		}
		content := strings.Join(final, "\n")
		content = truncateContent(content)
		out := grepOutput{
			Mode:          "content",
			NumFiles:      0,
			Filenames:     []string{},
			Content:       content,
			NumLines:      len(final),
			AppliedLimit:  al,
			AppliedOffset: ao,
		}
		return json.Marshal(out)

	case "count":
		lim, al, ao := applyHeadLimit(rawLines, in.HeadLimit, offset)
		final := make([]string, 0, len(lim))
		total := 0
		files := 0
		for _, line := range lim {
			fp, countRest := splitCountLine(line)
			if fp == "" {
				final = append(final, line)
				continue
			}
			absP := resolvePathLine(fp, absSearch)
			rel := toRelativePathDisplay(absP)
			final = append(final, rel+countRest)
			nStr := strings.TrimPrefix(countRest, ":")
			nStr = strings.TrimSpace(nStr)
			if n, err := strconv.Atoi(nStr); err == nil {
				total += n
				files++
			}
		}
		content := strings.Join(final, "\n")
		content = truncateContent(content)
		out := grepOutput{
			Mode:          "count",
			NumFiles:      files,
			Filenames:     []string{},
			Content:       content,
			NumMatches:    total,
			AppliedLimit:  al,
			AppliedOffset: ao,
		}
		return json.Marshal(out)

	default: // files_with_matches
		paths := make([]string, 0, len(rawLines))
		for _, line := range rawLines {
			p := resolvePathLine(line, absSearch)
			if p == "" {
				continue
			}
			if gc != nil && gc.DenyRead != nil && gc.DenyRead(p) {
				continue
			}
			paths = append(paths, p)
		}
		type scored struct {
			path  string
			mtime int64
		}
		sc := make([]scored, 0, len(paths))
		for _, p := range paths {
			var mt int64
			if fi, err := os.Stat(p); err == nil {
				mt = fi.ModTime().UnixNano()
			}
			sc = append(sc, scored{path: p, mtime: mt})
		}
		sort.Slice(sc, func(i, j int) bool {
			if sc[i].mtime != sc[j].mtime {
				return sc[i].mtime > sc[j].mtime
			}
			return sc[i].path < sc[j].path
		})
		ordered := make([]string, len(sc))
		for i := range sc {
			ordered[i] = sc[i].path
		}
		finalPaths, al, ao := applyHeadLimit(ordered, in.HeadLimit, offset)
		relNames := make([]string, len(finalPaths))
		for i, p := range finalPaths {
			relNames[i] = toRelativePathDisplay(p)
		}
		out := grepOutput{
			Mode:          "files_with_matches",
			NumFiles:      len(relNames),
			Filenames:     relNames,
			AppliedLimit:  al,
			AppliedOffset: ao,
		}
		return json.Marshal(out)
	}
}

func splitCountLine(line string) (pathPart, countWithColon string) {
	i := strings.LastIndex(line, ":")
	if i <= 0 {
		return line, ""
	}
	return line[:i], line[i:]
}

func truncateContent(s string) string {
	if len(s) <= maxResultSizeChars {
		return s
	}
	return s[:maxResultSizeChars] + "\n\n[Output truncated]"
}

var _ tools.Tool = (*Grep)(nil)

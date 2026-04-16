package bashtool

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

var (
	reProcessSubstitutionPath = regexp.MustCompile(`>>\s*>\s*\(|>\s*>\s*\(|<\s*\(`)
	reDangerousRmRoot         = regexp.MustCompile(`(?:^|[;&|\n])\s*rm\s+(?:[^\n;|&]*?\s/(?:\s|$|[;&|]))`)
)

// PathCommand mirrors pathValidation.ts PathCommand union type.
type PathCommand = string

// FileOperationType mirrors pathValidation.ts / utils/permissions/pathValidation.ts FileOperationType.
type FileOperationType string

const (
	FileOperationRead   FileOperationType = "read"
	FileOperationWrite  FileOperationType = "write"
	FileOperationCreate FileOperationType = "create"
)

// COMMAND_OPERATION_TYPE mirrors pathValidation.ts COMMAND_OPERATION_TYPE.
var COMMAND_OPERATION_TYPE = map[string]FileOperationType{
	"cd": FileOperationRead, "ls": FileOperationRead, "find": FileOperationRead,
	"mkdir": FileOperationCreate, "touch": FileOperationCreate,
	"rm": FileOperationWrite, "rmdir": FileOperationWrite,
	"mv": FileOperationWrite, "cp": FileOperationWrite,
	"cat": FileOperationRead, "head": FileOperationRead, "tail": FileOperationRead,
	"sort": FileOperationRead, "uniq": FileOperationRead, "wc": FileOperationRead,
	"cut": FileOperationRead, "paste": FileOperationRead, "column": FileOperationRead,
	"tr": FileOperationRead, "file": FileOperationRead, "stat": FileOperationRead,
	"diff": FileOperationRead, "awk": FileOperationRead, "strings": FileOperationRead,
	"hexdump": FileOperationRead, "od": FileOperationRead, "base64": FileOperationRead,
	"nl": FileOperationRead, "grep": FileOperationRead, "rg": FileOperationRead,
	"sed": FileOperationWrite, "git": FileOperationRead, "jq": FileOperationRead,
	"sha256sum": FileOperationRead, "sha1sum": FileOperationRead, "md5sum": FileOperationRead,
}

// filterOutFlags mirrors pathValidation.ts filterOutFlags (POSIX -- end-of-options support).
func filterOutFlags(args []string) []string {
	var result []string
	afterDD := false
	for _, a := range args {
		if afterDD {
			result = append(result, a)
		} else if a == "--" {
			afterDD = true
		} else if !strings.HasPrefix(a, "-") {
			result = append(result, a)
		}
	}
	return result
}

// parsePatternCommand mirrors pathValidation.ts parsePatternCommand (grep/rg-style: pattern then paths).
func parsePatternCommand(args []string, flagsWithArgs map[string]struct{}, defaults []string) []string {
	var paths []string
	patternFound := false
	afterDD := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !afterDD && arg == "--" {
			afterDD = true
			continue
		}
		if !afterDD && strings.HasPrefix(arg, "-") {
			flag := strings.SplitN(arg, "=", 2)[0]
			if flag == "-e" || flag == "--regexp" || flag == "-f" || flag == "--file" {
				patternFound = true
			}
			if _, ok := flagsWithArgs[flag]; ok && !strings.Contains(arg, "=") {
				i++
			}
			continue
		}
		if !patternFound {
			patternFound = true
			continue
		}
		paths = append(paths, arg)
	}
	if len(paths) > 0 {
		return paths
	}
	return defaults
}

// PATH_EXTRACTORS mirrors pathValidation.ts PATH_EXTRACTORS.
// Each entry returns the file paths (non-flag args) for the given command's argv.
var PATH_EXTRACTORS = map[string]func([]string) []string{
	"cd": func(args []string) []string {
		if len(args) == 0 {
			h, err := os.UserHomeDir()
			if err != nil {
				return nil
			}
			return []string{h}
		}
		return []string{strings.Join(args, " ")}
	},
	"ls": func(args []string) []string {
		paths := filterOutFlags(args)
		if len(paths) == 0 {
			return []string{"."}
		}
		return paths
	},
	"find": func(args []string) []string {
		pathFlags := map[string]struct{}{
			"-newer": {}, "-anewer": {}, "-cnewer": {}, "-mnewer": {},
			"-samefile": {}, "-path": {}, "-wholename": {},
			"-ilname": {}, "-lname": {}, "-ipath": {}, "-iwholename": {},
		}
		newerPat := regexp.MustCompile(`^-newer[acmBt][acmtB]$`)
		var paths []string
		foundNonGlobal := false
		afterDD := false
		for i := 0; i < len(args); i++ {
			arg := args[i]
			if afterDD {
				paths = append(paths, arg)
				continue
			}
			if arg == "--" {
				afterDD = true
				continue
			}
			if strings.HasPrefix(arg, "-") {
				if arg == "-H" || arg == "-L" || arg == "-P" {
					continue
				}
				foundNonGlobal = true
				if _, ok := pathFlags[arg]; ok || newerPat.MatchString(arg) {
					if i+1 < len(args) {
						paths = append(paths, args[i+1])
						i++
					}
				}
				continue
			}
			if !foundNonGlobal {
				paths = append(paths, arg)
			}
		}
		if len(paths) == 0 {
			return []string{"."}
		}
		return paths
	},
	"sed": func(args []string) []string {
		var paths []string
		skipNext := false
		scriptFound := false
		afterDD := false
		for i := 0; i < len(args); i++ {
			if skipNext {
				skipNext = false
				continue
			}
			arg := args[i]
			if !afterDD && arg == "--" {
				afterDD = true
				continue
			}
			if !afterDD && strings.HasPrefix(arg, "-") {
				if arg == "-f" || arg == "--file" {
					if i+1 < len(args) {
						paths = append(paths, args[i+1])
						skipNext = true
					}
					scriptFound = true
				} else if arg == "-e" || arg == "--expression" {
					skipNext = true
					scriptFound = true
				} else if strings.Contains(arg, "e") || strings.Contains(arg, "f") {
					scriptFound = true
				}
				continue
			}
			if !scriptFound {
				scriptFound = true
				continue
			}
			paths = append(paths, arg)
		}
		return paths
	},
	"jq": func(args []string) []string {
		flagsWithArgs := map[string]struct{}{
			"-e": {}, "--expression": {}, "-f": {}, "--from-file": {},
			"--arg": {}, "--argjson": {}, "--slurpfile": {}, "--rawfile": {},
			"--args": {}, "--jsonargs": {}, "-L": {}, "--library-path": {},
			"--indent": {}, "--tab": {},
		}
		var paths []string
		filterFound := false
		afterDD := false
		for i := 0; i < len(args); i++ {
			arg := args[i]
			if !afterDD && arg == "--" {
				afterDD = true
				continue
			}
			if !afterDD && strings.HasPrefix(arg, "-") {
				flag := strings.SplitN(arg, "=", 2)[0]
				if flag == "-e" || flag == "--expression" {
					filterFound = true
				}
				if _, ok := flagsWithArgs[flag]; ok && !strings.Contains(arg, "=") {
					i++
				}
				continue
			}
			if !filterFound {
				filterFound = true
				continue
			}
			paths = append(paths, arg)
		}
		return paths
	},
	"git": func(args []string) []string {
		if len(args) >= 1 && args[0] == "diff" {
			for _, a := range args[1:] {
				if a == "--no-index" {
					filePaths := filterOutFlags(args[1:])
					if len(filePaths) > 2 {
						return filePaths[:2]
					}
					return filePaths
				}
			}
		}
		return nil
	},
	"grep": func(args []string) []string {
		flags := map[string]struct{}{
			"-e": {}, "--regexp": {}, "-f": {}, "--file": {},
			"--exclude": {}, "--include": {}, "--exclude-dir": {}, "--include-dir": {},
			"-m": {}, "--max-count": {}, "-A": {}, "--after-context": {},
			"-B": {}, "--before-context": {}, "-C": {}, "--context": {},
		}
		paths := parsePatternCommand(args, flags, nil)
		if len(paths) == 0 {
			for _, a := range args {
				if a == "-r" || a == "-R" || a == "--recursive" {
					return []string{"."}
				}
			}
		}
		return paths
	},
	"rg": func(args []string) []string {
		flags := map[string]struct{}{
			"-e": {}, "--regexp": {}, "-f": {}, "--file": {},
			"-t": {}, "--type": {}, "-T": {}, "--type-not": {},
			"-g": {}, "--glob": {}, "-m": {}, "--max-count": {},
			"--max-depth": {}, "-r": {}, "--replace": {},
			"-A": {}, "--after-context": {}, "-B": {}, "--before-context": {},
			"-C": {}, "--context": {},
		}
		return parsePatternCommand(args, flags, []string{"."})
	},
	"tr": func(args []string) []string {
		hasDelete := false
		for _, a := range args {
			if a == "-d" || a == "--delete" ||
				(strings.HasPrefix(a, "-") && strings.Contains(a, "d")) {
				hasDelete = true
				break
			}
		}
		nonFlags := filterOutFlags(args)
		skip := 2
		if hasDelete {
			skip = 1
		}
		if len(nonFlags) > skip {
			return nonFlags[skip:]
		}
		return nil
	},
}

func init() {
	// All simple "filterOutFlags" commands
	simple := []string{
		"mkdir", "touch", "rm", "rmdir", "mv", "cp",
		"cat", "head", "tail", "sort", "uniq", "wc", "cut", "paste", "column",
		"file", "stat", "diff", "awk", "strings", "hexdump", "od", "base64", "nl",
		"sha256sum", "sha1sum", "md5sum",
	}
	for _, cmd := range simple {
		c := cmd
		PATH_EXTRACTORS[c] = func(args []string) []string {
			return filterOutFlags(args)
		}
	}
}

// TIMEOUT_FLAG_VALUE_RE mirrors pathValidation.ts TIMEOUT_FLAG_VALUE_RE.
var timeoutFlagValueRE = regexp.MustCompile(`^[A-Za-z0-9_.+-]+$`)

// skipTimeoutFlags mirrors pathValidation.ts skipTimeoutFlags.
func skipTimeoutFlags(a []string) int {
	i := 1
	for i < len(a) {
		arg := a[i]
		var next string
		if i+1 < len(a) {
			next = a[i+1]
		}
		switch {
		case arg == "--foreground" || arg == "--preserve-status" || arg == "--verbose":
			i++
		case regexp.MustCompile(`^--(?:kill-after|signal)=[A-Za-z0-9_.+-]+$`).MatchString(arg):
			i++
		case (arg == "--kill-after" || arg == "--signal") && next != "" && timeoutFlagValueRE.MatchString(next):
			i += 2
		case arg == "--":
			i++
			return i
		case strings.HasPrefix(arg, "--"):
			return -1
		case arg == "-v":
			i++
		case (arg == "-k" || arg == "-s") && next != "" && timeoutFlagValueRE.MatchString(next):
			i += 2
		case regexp.MustCompile(`^-[ks][A-Za-z0-9_.+-]+$`).MatchString(arg):
			i++
		case strings.HasPrefix(arg, "-"):
			return -1
		default:
			return i
		}
	}
	return i
}

// skipStdbufFlags mirrors pathValidation.ts skipStdbufFlags.
func skipStdbufFlags(a []string) int {
	i := 1
	for i < len(a) {
		arg := a[i]
		var next string
		if i+1 < len(a) {
			next = a[i+1]
		}
		switch {
		case regexp.MustCompile(`^-[ioe]$`).MatchString(arg) && next != "":
			i += 2
		case regexp.MustCompile(`^-[ioe].`).MatchString(arg):
			i++
		case regexp.MustCompile(`^--(input|output|error)=`).MatchString(arg):
			i++
		case strings.HasPrefix(arg, "-"):
			return -1
		default:
			goto done
		}
	}
done:
	if i > 1 && i < len(a) {
		return i
	}
	return -1
}

// skipEnvFlags mirrors pathValidation.ts skipEnvFlags.
func skipEnvFlags(a []string) int {
	i := 1
	for i < len(a) {
		arg := a[i]
		var next string
		if i+1 < len(a) {
			next = a[i+1]
		}
		switch {
		case strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-"):
			i++
		case arg == "-i" || arg == "-0" || arg == "-v":
			i++
		case arg == "-u" && next != "":
			i += 2
		case strings.HasPrefix(arg, "-"):
			return -1
		default:
			goto envDone
		}
	}
envDone:
	if i < len(a) {
		return i
	}
	return -1
}

var durationRE = regexp.MustCompile(`^\d+(?:\.\d+)?[smhd]?$`)

// StripWrappersFromArgv mirrors pathValidation.ts stripWrappersFromArgv (argv-level safe-wrapper stripping).
// Strips time/nohup/timeout/nice/stdbuf/env from argv, returning the wrapped command argv.
func StripWrappersFromArgv(argv []string) []string {
	a := argv
	for {
		if len(a) == 0 {
			return a
		}
		switch a[0] {
		case "time", "nohup":
			if len(a) > 1 && a[1] == "--" {
				a = a[2:]
			} else {
				a = a[1:]
			}
		case "timeout":
			i := skipTimeoutFlags(a)
			if i < 0 || i >= len(a) || !durationRE.MatchString(a[i]) {
				return a
			}
			a = a[i+1:]
		case "nice":
			if len(a) > 2 && a[1] == "-n" && regexp.MustCompile(`^-?\d+$`).MatchString(a[2]) {
				if len(a) > 3 && a[3] == "--" {
					a = a[4:]
				} else {
					a = a[3:]
				}
			} else if len(a) > 1 && regexp.MustCompile(`^-\d+$`).MatchString(a[1]) {
				if len(a) > 2 && a[2] == "--" {
					a = a[3:]
				} else {
					a = a[2:]
				}
			} else {
				if len(a) > 1 && a[1] == "--" {
					a = a[2:]
				} else {
					a = a[1:]
				}
			}
		case "stdbuf":
			i := skipStdbufFlags(a)
			if i < 0 {
				return a
			}
			a = a[i:]
		case "env":
			i := skipEnvFlags(a)
			if i < 0 {
				return a
			}
			a = a[i:]
		default:
			return a
		}
	}
}

// PathValidationOptions mirrors pathValidation.ts checkPathConstraints inputs (headless subset).
type PathValidationOptions struct {
	Cwd              string
	WorkdirRoot      string
	CheckDangerousRm bool
}

// CheckPathConstraints mirrors pathValidation.ts: process substitution, optional dangerous rm heuristic, optional workdir root for cd/ls paths.
func CheckPathConstraints(command string, opt PathValidationOptions) error {
	if strings.TrimSpace(command) == "" {
		return nil
	}
	if reProcessSubstitutionPath.MatchString(command) {
		return errors.New("bashtool: process substitution requires manual approval")
	}
	checkRm := opt.CheckDangerousRm
	if !checkRm {
		checkRm = true
	}
	if checkRm && reDangerousRmRoot.MatchString(command) {
		return errors.New("bashtool: rm involving absolute root paths requires manual approval")
	}
	if CommandHasAnyCd(command) {
		if compoundHasWriteOrRedirection(command) {
			return errors.New("bashtool: cd combined with writes or output redirection requires manual approval (path resolution)")
		}
	}
	root := strings.TrimSpace(opt.WorkdirRoot)
	if root == "" {
		root = features.BashWorkdirRoot()
	}
	if root == "" {
		return nil
	}
	cwd := opt.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}
	cwd = filepath.Clean(cwd)
	root = filepath.Clean(root)
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			if err := validateSubcommandPathsUnderRoot(strings.TrimSpace(sub), root, cwd); err != nil {
				return err
			}
		}
	}
	return nil
}

func compoundHasWriteOrRedirection(command string) bool {
	writeVerbs := map[string]struct{}{
		"mkdir": {}, "touch": {}, "cp": {}, "mv": {}, "rm": {}, "rmdir": {},
		"sed": {}, "tee": {}, "sh": {},
	}
	for _, seg := range shellSegments(command) {
		for _, sub := range SplitCommandDeprecated(seg) {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			if reOutputRedirectTarget.MatchString(sub) {
				return true
			}
			fields := strings.Fields(StripSafeWrappers(sub))
			if len(fields) == 0 {
				continue
			}
			if _, ok := writeVerbs[fields[0]]; ok {
				return true
			}
		}
	}
	return false
}

func expandTildePath(p string) string {
	p = strings.TrimSpace(strings.Trim(p, `"'`))
	if p == "~" || strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil || h == "" {
			return p
		}
		if p == "~" {
			return h
		}
		return filepath.Join(h, strings.TrimPrefix(p, "~/"))
	}
	return p
}

func pathUnderRoot(root, cwd, path string) bool {
	path = expandTildePath(path)
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(cwd, path))
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func validateSubcommandPathsUnderRoot(sub, root, cwd string) error {
	sub = StripSafeWrappers(sub)
	if sub == "" {
		return nil
	}
	fields := strings.Fields(sub)
	if len(fields) == 0 {
		return nil
	}
	verb := fields[0]
	extractor, ok := PATH_EXTRACTORS[verb]
	if !ok {
		return nil
	}
	paths := extractor(fields[1:])
	for _, p := range paths {
		if p == "" || strings.HasPrefix(p, "-") {
			continue
		}
		p = expandTildePath(p)
		if !pathUnderRoot(root, cwd, p) {
			return fmt.Errorf("bashtool: %s path %q is outside workdir root %q", verb, p, root)
		}
	}
	return nil
}

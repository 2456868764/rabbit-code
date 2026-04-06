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
	switch verb {
	case "cd":
		args := fields[1:]
		if len(args) == 0 {
			return nil
		}
		target := strings.Join(args, " ")
		target = expandTildePath(target)
		if !pathUnderRoot(root, cwd, target) {
			return fmt.Errorf("bashtool: cd target %q is outside workdir root %q", target, root)
		}
	case "ls":
		paths := filterOutFlagArgs(fields[1:])
		if len(paths) == 0 {
			paths = []string{"."}
		}
		for _, p := range paths {
			if strings.HasPrefix(p, "-") {
				continue
			}
			if !pathUnderRoot(root, cwd, p) {
				return fmt.Errorf("bashtool: ls path %q is outside workdir root %q", p, root)
			}
		}
	case "cat", "head", "tail":
		for _, p := range filterOutFlagArgs(fields[1:]) {
			if strings.HasPrefix(p, "-") {
				continue
			}
			if !pathUnderRoot(root, cwd, p) {
				return fmt.Errorf("bashtool: %s path %q is outside workdir root %q", verb, p, root)
			}
		}
	}
	return nil
}

func filterOutFlagArgs(args []string) []string {
	var out []string
	afterDD := false
	for _, a := range args {
		if afterDD {
			out = append(out, a)
			continue
		}
		if a == "--" {
			afterDD = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	return out
}

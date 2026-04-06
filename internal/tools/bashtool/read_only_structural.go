package bashtool

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/2456868764/rabbit-code/internal/features"
)

// CheckReadOnlyStructuralConstraints mirrors readOnlyValidation.ts checkReadOnlyConstraints security gates
// that do not require full COMMAND_ALLOWLIST (bare repo, git-internal writes, UNC, sandbox cwd, shell-quote bug, control chars).
func CheckReadOnlyStructuralConstraints(command string, cwd string) error {
	if ContainsVulnerableUncPath(command) {
		return errors.New("bashtool: command contains Windows UNC patterns that may be unsafe for auto read-only approval")
	}
	if r := BashReadOnlySecurityRejectReason(command); r != "" {
		return fmt.Errorf("bashtool: %s", r)
	}
	if !CommandHasAnyGit(command) {
		return nil
	}
	cleanCwd := filepath.Clean(cwd)
	if IsBareGitExploitLayout(cleanCwd) {
		return errors.New("bashtool: git in a bare or broken-git-directory layout requires manual approval")
	}
	if CommandWritesToGitInternalPaths(command) {
		return errors.New("bashtool: command creates git-internal paths and runs git (hook escape risk)")
	}
	if features.BashSandboxPolicyEnabled() {
		orig := features.BashOriginalWorkdir()
		if orig != "" {
			if filepath.Clean(orig) != cleanCwd {
				return errors.New("bashtool: git outside original workdir when sandbox policy is enabled requires approval")
			}
		}
	}
	return nil
}

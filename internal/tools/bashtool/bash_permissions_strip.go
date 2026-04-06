package bashtool

import (
	"os"
	"regexp"
	"strings"
)

// bashSafeEnvVarNames mirrors bashPermissions.ts SAFE_ENV_VARS (strip allowlist for permission matching).
var bashSafeEnvVarNames = map[string]struct{}{
	"GOEXPERIMENT": {}, "GOOS": {}, "GOARCH": {}, "CGO_ENABLED": {}, "GO111MODULE": {},
	"RUST_BACKTRACE": {}, "RUST_LOG": {},
	"NODE_ENV": {},
	"PYTHONUNBUFFERED": {}, "PYTHONDONTWRITEBYTECODE": {},
	"PYTEST_DISABLE_PLUGIN_AUTOLOAD": {}, "PYTEST_DEBUG": {},
	"ANTHROPIC_API_KEY": {},
	"LANG": {}, "LANGUAGE": {}, "LC_ALL": {}, "LC_CTYPE": {}, "LC_TIME": {}, "CHARSET": {},
	"TERM": {}, "COLORTERM": {}, "NO_COLOR": {}, "FORCE_COLOR": {}, "TZ": {},
	"LS_COLORS": {}, "LSCOLORS": {}, "GREP_COLOR": {}, "GREP_COLORS": {}, "GCC_COLORS": {},
	"TIME_STYLE": {}, "BLOCK_SIZE": {}, "BLOCKSIZE": {},
}

// bashAntOnlySafeEnvVarNames mirrors bashPermissions.ts ANT_ONLY_SAFE_ENV_VARS.
var bashAntOnlySafeEnvVarNames = map[string]struct{}{
	"KUBECONFIG": {}, "DOCKER_HOST": {},
	"AWS_PROFILE": {}, "CLOUDSDK_CORE_PROJECT": {}, "CLUSTER": {},
	"COO_CLUSTER": {}, "COO_CLUSTER_NAME": {}, "COO_NAMESPACE": {}, "COO_LAUNCH_YAML_DRY_RUN": {},
	"SKIP_NODE_VERSION_CHECK": {}, "EXPECTTEST_ACCEPT": {}, "CI": {}, "GIT_LFS_SKIP_SMUDGE": {},
	"CUDA_VISIBLE_DEVICES": {}, "JAX_PLATFORMS": {},
	"COLUMNS": {}, "TMUX": {},
	"POSTGRESQL_VERSION": {}, "FIRESTORE_EMULATOR_HOST": {}, "HARNESS_QUIET": {},
	"TEST_CROSSCHECK_LISTS_MATCH_UPDATE": {}, "DBT_PER_DEVELOPER_ENVIRONMENTS": {}, "STATSIG_FORD_DB_CHECKS": {},
	"ANT_ENVIRONMENT": {}, "ANT_SERVICE": {}, "MONOREPO_ROOT_DIR": {},
	"PYENV_VERSION": {},
	"PGPASSWORD": {}, "GH_TOKEN": {}, "GROWTHBOOK_API_KEY": {},
}

var (
	bashEnvVarStripRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=([A-Za-z0-9_./:-]+)[ \t]+`)
	// Simplified from bashPermissions.ts SAFE_WRAPPER_PATTERNS (timeout/time/nice/stdbuf/nohup).
	bashWrapperRes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^timeout[ \t]+(?:(?:--(?:foreground|preserve-status|verbose)|--(?:kill-after|signal)=[A-Za-z0-9_.+-]+|--(?:kill-after|signal)[ \t]+[A-Za-z0-9_.+-]+|-v|-[ks][ \t]+[A-Za-z0-9_.+-]+|-[ks][A-Za-z0-9_.+-]+)[ \t]+)*(?:--[ \t]+)?\d+(?:\.\d+)?[smhd]?[ \t]+`),
		regexp.MustCompile(`(?i)^time[ \t]+(?:--[ \t]+)?`),
		regexp.MustCompile(`(?i)^nice(?:[ \t]+-n[ \t]+-?\d+|[ \t]+-\d+)?[ \t]+(?:--[ \t]+)?`),
		regexp.MustCompile(`(?i)^stdbuf(?:[ \t]+-[ioe][LN0-9]+)+[ \t]+(?:--[ \t]+)?`),
		regexp.MustCompile(`(?i)^nohup[ \t]+(?:--[ \t]+)?`),
	}
)

func stripCommentLinesForPermissions(command string) string {
	lines := strings.Split(command, "\n")
	var nonComment []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") {
			nonComment = append(nonComment, line)
		}
	}
	if len(nonComment) == 0 {
		return command
	}
	return strings.Join(nonComment, "\n")
}

func envVarAllowedForStrip(name string) bool {
	if _, ok := bashSafeEnvVarNames[name]; ok {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("USER_TYPE")), "ant") {
		_, ok := bashAntOnlySafeEnvVarNames[name]
		return ok
	}
	return false
}

// StripSafeWrappers mirrors bashPermissions.ts stripSafeWrappers (Phase1 env + Phase2 wrappers; horizontal whitespace only).
func StripSafeWrappers(command string) string {
	stripped := command
	// Phase 1: comments + safe env vars
	for prev := ""; stripped != prev; {
		prev = stripped
		stripped = stripCommentLinesForPermissions(stripped)
		if m := bashEnvVarStripRE.FindStringSubmatchIndex(stripped); m != nil {
			name := stripped[m[2]:m[3]]
			if envVarAllowedForStrip(name) {
				stripped = stripped[m[1]:]
			} else {
				break
			}
		}
	}
	// Phase 2: wrappers only (no env)
	for prev := ""; stripped != prev; {
		prev = stripped
		stripped = stripCommentLinesForPermissions(stripped)
		for _, re := range bashWrapperRes {
			stripped = re.ReplaceAllString(stripped, "")
		}
	}
	return strings.TrimSpace(stripped)
}

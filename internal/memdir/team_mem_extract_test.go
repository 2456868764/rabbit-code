package memdir

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestTeamMemDirFromAutoMemDir(t *testing.T) {
	auto := filepath.Clean("/home/u/.claude/projects/foo/memory") + string(filepath.Separator)
	got := TeamMemDirFromAutoMemDir(auto)
	want := filepath.Join(filepath.Clean("/home/u/.claude/projects/foo/memory"), "team") + string(filepath.Separator)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if p := TeamMemEntrypointFromAutoMemDir(auto); filepath.Base(p) != EntrypointName || !strings.Contains(p, TeamMemSubdir) {
		t.Fatalf("entrypoint %q", p)
	}
}

func TestIsTeamMemPathUnderAutoMem(t *testing.T) {
	auto := filepath.Clean("/m/proj/memory") + string(filepath.Separator)
	teamFile := filepath.Join(filepath.Clean("/m/proj/memory"), "team", "x.md")
	if !IsTeamMemPathUnderAutoMem(teamFile, auto) {
		t.Fatal("expected under team")
	}
	priv := filepath.Join(filepath.Clean("/m/proj/memory"), "user.md")
	if IsTeamMemPathUnderAutoMem(priv, auto) {
		t.Fatal("private root file is not team")
	}
}

func TestSanitizeTeamMemPathKey(t *testing.T) {
	if err := SanitizeTeamMemPathKey("notes/foo.md"); err != nil {
		t.Fatal(err)
	}
	if SanitizeTeamMemPathKey("/abs") == nil {
		t.Fatal("want error")
	}
	if SanitizeTeamMemPathKey(`a\b`) == nil {
		t.Fatal("want error")
	}
	if SanitizeTeamMemPathKey("%2e%2e%2ffoo") == nil {
		t.Fatal("want error")
	}
}

func TestBuildExtractCombinedPrompt_fallsBackWhenTeamOff(t *testing.T) {
	t.Setenv(features.EnvTeamMem, "")
	t.Setenv(features.EnvDisableAutoMemory, "")
	auto := BuildExtractAutoOnlyPrompt(8, "", false)
	combo := BuildExtractCombinedPrompt(8, "", false, nil)
	if auto != combo {
		t.Fatal("expected same prompt when team memory off")
	}
}

func TestBuildExtractCombinedPrompt_teamSections(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvTeamMem, "1")
	p := BuildExtractCombinedPrompt(3, "x", false, nil)
	if !strings.Contains(p, "<scope>") || !strings.Contains(p, "team memories") {
		t.Fatalf("missing combined sections: %q…", truncate(p, 120))
	}
}

func TestBuildCombinedMemoryPrompt_shape(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	p := BuildCombinedMemoryPrompt(CombinedMemoryPromptOpts{
		AutoMemDir: "/a/memory/",
		TeamMemDir: "/a/memory/team/",
	})
	if !strings.Contains(p, "# Memory") || !strings.Contains(p, "Types of memory") {
		t.Fatal("expected headings")
	}
}

func TestBuildSearchingPastContextSection_gate(t *testing.T) {
	t.Setenv(features.EnvMemorySearchPastContext, "")
	if len(BuildSearchingPastContextSection("/mem/", "/proj", false)) != 0 {
		t.Fatal("expected empty when env off")
	}
	t.Setenv(features.EnvMemorySearchPastContext, "1")
	lines := BuildSearchingPastContextSection("/mem/", "/proj", false)
	if len(lines) == 0 || !strings.Contains(strings.Join(lines, "\n"), ".jsonl") {
		t.Fatalf("got %v", lines)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

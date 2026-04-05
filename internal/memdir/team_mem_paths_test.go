package memdir

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetTeamMemPath_alias(t *testing.T) {
	auto := filepath.Join(t.TempDir(), "mem") + string(filepath.Separator)
	if GetTeamMemPath(auto) != TeamMemDirFromAutoMemDir(auto) {
		t.Fatal()
	}
}

func TestIsTeamMemFile_alias(t *testing.T) {
	if IsTeamMemFile("/x", "/m", false) {
		t.Fatal()
	}
}

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

func TestValidateTeamMemWritePath(t *testing.T) {
	auto := filepath.Clean(t.TempDir())
	teamDir := filepath.Join(auto, TeamMemSubdir)
	if err := EnsureMemoryDirExists(teamDir); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(teamDir, "x.md")
	if err := ValidateTeamMemWritePath(good, auto); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(auto), "outside-team.md")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ValidateTeamMemWritePath(outside, auto) == nil {
		t.Fatal("expected escape error")
	}
}

func TestScanTeamMemorySecrets_githubPat(t *testing.T) {
	m := ScanTeamMemorySecrets(`ok ghp_012345678901234567890123456789012345 `)
	if len(m) != 1 || m[0].RuleID != "github-pat" {
		t.Fatalf("got %+v", m)
	}
}

func TestTeamMemSecretGuardRunner_blocksTeamWrite(t *testing.T) {
	auto := filepath.Clean(t.TempDir())
	teamFile := filepath.Join(auto, TeamMemSubdir, "nope.md")
	_ = EnsureMemoryDirExists(filepath.Dir(teamFile))
	inner := queryRunnerAlwaysOK{}
	g := &TeamMemSecretGuardRunner{Inner: inner, AutoMemDir: auto, Enabled: true}
	fp, _ := json.Marshal(teamFile)
	payload := []byte(`{"file_path":` + string(fp) + `,"content":"token ghp_012345678901234567890123456789012345"}`)
	_, err := g.RunTool(t.Context(), "Write", payload)
	if err == nil || !strings.Contains(err.Error(), "potential secrets") {
		t.Fatalf("got %v", err)
	}
}

type queryRunnerAlwaysOK struct{}

func (queryRunnerAlwaysOK) RunTool(context.Context, string, []byte) ([]byte, error) {
	return []byte(`{}`), nil
}

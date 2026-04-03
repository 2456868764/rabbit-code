package memdir

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestLoadMemorySystemPrompt_gates(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvMemorySystemPrompt, "0")
	_, ok := LoadMemorySystemPrompt(MemorySystemPromptInput{
		MemoryDir: "/tmp/mem",
		Merged:    map[string]interface{}{"autoMemoryEnabled": true},
	})
	if ok {
		t.Fatal("expected off when MEMORY_SYSTEM_PROMPT falsy")
	}
}

func TestLoadMemorySystemPrompt_autoOnlyShape(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvMemorySystemPrompt, "1")
	t.Setenv(features.EnvTeamMem, "")
	t.Setenv(features.EnvKairosDailyLogMemory, "")
	t.Setenv(features.EnvKairosActive, "")
	t.Setenv(features.EnvMemorySearchPastContext, "")
	dir := t.TempDir()
	s, ok := LoadMemorySystemPrompt(MemorySystemPromptInput{
		MemoryDir:   dir,
		ProjectRoot: dir,
		Merged:      map[string]interface{}{"autoMemoryEnabled": true},
	})
	if !ok || !strings.Contains(s, "# auto memory") {
		t.Fatalf("ok=%v head=%q", ok, truncate(s, 80))
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

package query_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
)

func TestNewDefaultToolRunner_Grep(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("no ripgrep")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "z.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := query.NewDefaultToolRunner()
	in := `{"pattern":"needle","path":` + string(mustJSON(t, dir)) + `}`
	out, err := tr.RunTool(context.Background(), greptool.GrepToolName, []byte(in))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		NumFiles int `json:"numFiles"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.NumFiles != 1 {
		t.Fatalf("got %s", string(out))
	}
}

func TestNewDefaultToolRunner_bashFallback(t *testing.T) {
	tr := query.NewDefaultToolRunner()
	out, err := tr.RunTool(context.Background(), "bash", []byte(`{"command":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "hi") && !strings.Contains(string(out), "stub") {
		t.Fatalf("unexpected: %s", out)
	}
}

func mustJSON(t *testing.T, s string) []byte {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

package query

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestBashStubToolRunner(t *testing.T) {
	var tr BashStubToolRunner
	out, err := tr.RunTool(context.Background(), "bash", []byte(`{"cmd":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != true || m["stub"] != "bash" {
		t.Fatalf("got %s", out)
	}
	if _, err := tr.RunTool(context.Background(), "other", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestBashExecToolRunner_disabledUsesStub(t *testing.T) {
	t.Setenv(features.EnvBashExec, "")
	var tr BashExecToolRunner
	out, err := tr.RunTool(context.Background(), "bash", []byte(`{"cmd":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["stub"] != "bash" {
		t.Fatalf("%s", out)
	}
}

func TestBashExecToolRunner_runsEcho(t *testing.T) {
	t.Setenv(features.EnvBashExec, "1")
	var tr BashExecToolRunner
	out, err := tr.RunTool(context.Background(), "bash", []byte(`{"cmd":"echo rabbit"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["ok"] != true {
		t.Fatalf("%s", out)
	}
	if s, _ := m["stdout"].(string); !strings.Contains(s, "rabbit") {
		t.Fatalf("stdout %q", s)
	}
}

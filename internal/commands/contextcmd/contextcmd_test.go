package contextcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
)

func TestRun_usageNoArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run(nil, strings.NewReader(""), &out, &errBuf)
	if code != 2 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errBuf.String(), "break-cache") {
		t.Fatalf("stderr: %q", errBuf.String())
	}
}

func TestRun_help(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"help"}, strings.NewReader(""), &out, &errBuf)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("code=%d err=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "report-md") {
		t.Fatalf("stdout: %q", out.String())
	}
}

func TestRun_breakCache(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"break-cache"}, strings.NewReader(""), &out, &errBuf)
	if code != 0 || errBuf.Len() != 0 {
		t.Fatalf("code=%d err=%q", code, errBuf.String())
	}
	var m map[string]string
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["kind"] != "break_cache_command" || m["phase"] != "submit" {
		t.Fatalf("%v", m)
	}
}

func TestRun_report_minimalTranscript(t *testing.T) {
	in := strings.NewReader(`[{"role":"user","content":"hello"}]`)
	var out, errBuf bytes.Buffer
	code := Run([]string{"report", "-model", "claude-3-5-haiku-20241022"}, in, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errBuf.String())
	}
	var r query.HeadlessContextReport
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.TranscriptBytes == 0 {
		t.Fatal("expected transcript bytes")
	}
	if r.ContextWindowTokens <= 0 {
		t.Fatal("expected positive context window")
	}
}

func TestRun_unknownSubcommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"nope"}, strings.NewReader(""), &out, &errBuf)
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestRun_report_md_markdown(t *testing.T) {
	in := strings.NewReader(`[{"role":"user","content":"hello"}]`)
	var out, errBuf bytes.Buffer
	code := Run([]string{"report-md", "-model", "claude-3-5-haiku-20241022"}, in, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errBuf.String())
	}
	s := out.String()
	if !strings.Contains(s, "## Context Usage") || !strings.Contains(s, "claude-3-5-haiku") {
		t.Fatalf("markdown: %q", s)
	}
	if !strings.Contains(s, "Estimated usage by category") {
		t.Fatal("missing category table")
	}
}

func TestRun_report_md_microcompactFlag(t *testing.T) {
	in := strings.NewReader(`[{"role":"user","content":"x"}]`)
	var out, errBuf bytes.Buffer
	code := Run([]string{"report-md", "-microcompact", "-model", "claude-3-5-haiku-20241022"}, in, &out, &errBuf)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "## Context Usage") {
		t.Fatal(out.String())
	}
}

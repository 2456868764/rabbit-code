package bashtool_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/tools"
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestBash_implementsTool(t *testing.T) {
	var _ tools.Tool = bashtool.New()
}

func TestBash_NameAndAliases(t *testing.T) {
	b := bashtool.New()
	if b.Name() != bashtool.BashToolName {
		t.Fatal(b.Name())
	}
	if len(b.Aliases()) != 1 || b.Aliases()[0] != "bash" {
		t.Fatal(b.Aliases())
	}
}

func TestBash_strictJSONUnknownField(t *testing.T) {
	_, err := bashtool.New().Run(context.Background(), []byte(`{"command":"echo x","extra":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got %v", err)
	}
}

func TestBash_missingCommand(t *testing.T) {
	_, err := bashtool.New().Run(context.Background(), []byte(`{"command":"   "}`))
	if err == nil || !strings.Contains(err.Error(), "missing command") {
		t.Fatalf("got %v", err)
	}
}

func TestBash_execDisabledEmptyOutput(t *testing.T) {
	t.Setenv(features.EnvBashExec, "")
	out, err := bashtool.New().Run(context.Background(), []byte(`{"command":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]any
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o["stdout"] != "" || o["stderr"] != "" || o["interrupted"] != false {
		t.Fatalf("%v", o)
	}
}

func TestBash_execEcho(t *testing.T) {
	t.Setenv(features.EnvBashExec, "1")
	out, err := bashtool.New().Run(context.Background(), []byte(`{"command":"echo rabbit-code"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Stdout string `json:"stdout"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(o.Stdout, "rabbit-code") {
		t.Fatalf("%q", o.Stdout)
	}
}

func TestMapBashToolResultForMessagesAPI(t *testing.T) {
	s := bashtool.MapBashToolResultForMessagesAPI([]byte(`{"stdout":"a\n","stderr":"e","interrupted":false}`))
	if !strings.Contains(s, "a") || !strings.Contains(s, "e") {
		t.Fatalf("%q", s)
	}
	s2 := bashtool.MapBashToolResultForMessagesAPI([]byte(`{"stdout":"","stderr":"","interrupted":true}`))
	if !strings.Contains(s2, "aborted") {
		t.Fatalf("%q", s2)
	}
}

func TestNormalizeLegacyBashInput(t *testing.T) {
	b := bashtool.NormalizeLegacyBashInput([]byte(`{"cmd":"ls"}`))
	if !strings.Contains(string(b), `"command"`) || strings.Contains(string(b), `"cmd"`) {
		t.Fatalf("%s", b)
	}
}

func TestBash_monitorBlocksSleep2Plus(t *testing.T) {
	t.Setenv(features.EnvMonitorTool, "1")
	t.Setenv(features.EnvBashExec, "1")
	_, err := bashtool.New().Run(context.Background(), []byte(`{"command":"sleep 2"}`))
	if err == nil || !strings.Contains(err.Error(), "Blocked:") || !strings.Contains(err.Error(), "Monitor tool") {
		t.Fatalf("got %v", err)
	}
}

func TestBash_monitorAllowsSubTwoSecondSleep(t *testing.T) {
	t.Setenv(features.EnvMonitorTool, "1")
	t.Setenv(features.EnvBashExec, "1")
	out, err := bashtool.New().Run(context.Background(), []byte(`{"command":"sleep 0"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"interrupted"`) {
		t.Fatalf("%s", out)
	}
}

func TestBash_runInBackground(t *testing.T) {
	t.Setenv(features.EnvBashExec, "1")
	t.Setenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS", "")
	t.Setenv("RABBIT_CODE_DISABLE_BACKGROUND_TASKS", "")
	out, err := bashtool.New().Run(context.Background(), []byte(`{"command":"echo bgdone","run_in_background":true}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		BackgroundTaskID         string `json:"backgroundTaskId"`
		BackgroundTaskOutputPath string `json:"backgroundTaskOutputPath"`
		Stdout                   string `json:"stdout"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.BackgroundTaskID == "" || o.BackgroundTaskOutputPath == "" {
		t.Fatalf("missing bg fields: %s", out)
	}
	if o.Stdout != "" {
		t.Fatalf("stdout should be empty for bg spawn: %q", o.Stdout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := bashtool.WaitBackgroundTask(ctx, o.BackgroundTaskID); err != nil {
		t.Fatal(err)
	}
}

func TestBash_testCommandFalseSemantic(t *testing.T) {
	t.Setenv(features.EnvBashExec, "1")
	out, err := bashtool.New().Run(context.Background(), []byte(`{"command":"test 1 -eq 2"}`))
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Stderr                   string  `json:"stderr"`
		ReturnCodeInterpretation *string `json:"returnCodeInterpretation"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		t.Fatal(err)
	}
	if o.ReturnCodeInterpretation == nil || !strings.Contains(*o.ReturnCodeInterpretation, "false") {
		t.Fatalf("interpretation=%v out=%s", o.ReturnCodeInterpretation, out)
	}
}

func TestMapBash_backgroundAndInterpretation(t *testing.T) {
	s := bashtool.MapBashToolResultForMessagesAPI([]byte(`{
	  "stdout":"",
	  "stderr":"",
	  "interrupted":false,
	  "backgroundTaskId":"tid",
	  "backgroundTaskOutputPath":"/tmp/x.out",
	  "returnCodeInterpretation":"No matches found"
	}`))
	if !strings.Contains(s, "No matches found") || !strings.Contains(s, "tid") || !strings.Contains(s, "/tmp/x.out") {
		t.Fatalf("%q", s)
	}
}

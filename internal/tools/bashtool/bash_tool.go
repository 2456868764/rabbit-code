package bashtool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

// blockedSleepFollowup mirrors BashTool.tsx validateInput message after "Blocked: …" (errorCode 10).
const blockedSleepFollowup = " Run blocking commands in the background with run_in_background: true — you'll get a completion notification when done. For streaming events (watching logs, polling APIs), use the Monitor tool. If you genuinely need a delay (rate limiting, deliberate pacing), keep it under 2 seconds."

// Bash implements tools.Tool for BashTool.tsx (Run: parse, timeout, exec, background, semantics).
type Bash struct{}

// New returns a Bash tool.
func New() *Bash { return &Bash{} }

func (b *Bash) Name() string { return BashToolName }

// Aliases includes lowercase "bash" for callers that use API-style names.
func (b *Bash) Aliases() []string { return []string{"bash"} }

type bashInputWithBG struct {
	Command                   string `json:"command"`
	Timeout                   *int   `json:"timeout,omitempty"`
	Description               string `json:"description,omitempty"`
	RunInBackground           *bool  `json:"run_in_background,omitempty"`
	DangerouslyDisableSandbox *bool  `json:"dangerouslyDisableSandbox,omitempty"`
}

type bashInputNoBG struct {
	Command                   string `json:"command"`
	Timeout                   *int   `json:"timeout,omitempty"`
	Description               string `json:"description,omitempty"`
	DangerouslyDisableSandbox *bool  `json:"dangerouslyDisableSandbox,omitempty"`
}

type bashOutput struct {
	Stdout                    string  `json:"stdout"`
	Stderr                    string  `json:"stderr"`
	Interrupted               bool    `json:"interrupted"`
	DangerouslyDisableSandbox *bool   `json:"dangerouslyDisableSandbox,omitempty"`
	BackgroundTaskID          string  `json:"backgroundTaskId,omitempty"`
	BackgroundTaskOutputPath  string  `json:"backgroundTaskOutputPath,omitempty"`
	ReturnCodeInterpretation  *string `json:"returnCodeInterpretation,omitempty"`
	NoOutputExpected          *bool   `json:"noOutputExpected,omitempty"`
}

func backgroundTasksDisabled() bool {
	return envTruthy("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS") || envTruthy("RABBIT_CODE_DISABLE_BACKGROUND_TASKS")
}

func envTruthy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseBashInputJSON(b []byte) (bashInputWithBG, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var zero bashInputWithBG
	if backgroundTasksDisabled() {
		var in bashInputNoBG
		if err := dec.Decode(&in); err != nil {
			return zero, err
		}
		if dec.More() {
			return zero, errors.New("bashtool: invalid json: extra data after input object")
		}
		return bashInputWithBG{
			Command:                   in.Command,
			Timeout:                   in.Timeout,
			Description:               in.Description,
			DangerouslyDisableSandbox: in.DangerouslyDisableSandbox,
		}, nil
	}
	var in bashInputWithBG
	if err := dec.Decode(&in); err != nil {
		return zero, err
	}
	if dec.More() {
		return zero, errors.New("bashtool: invalid json: extra data after input object")
	}
	return in, nil
}

// NormalizeLegacyBashInput maps {"cmd":...} to {"command":...} for Phase 5 BashExecToolRunner compatibility.
func NormalizeLegacyBashInput(inputJSON []byte) []byte {
	var m map[string]any
	if err := json.Unmarshal(inputJSON, &m); err != nil || m == nil {
		return inputJSON
	}
	if _, ok := m["command"]; !ok {
		if v, ok := m["cmd"]; ok {
			m["command"] = v
			delete(m, "cmd")
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return inputJSON
	}
	return out
}

func applyDangerouslyDisableSandbox(out *bashOutput, in *bool) {
	if in != nil {
		v := *in
		out.DangerouslyDisableSandbox = &v
	}
}

// Run implements tools.Tool.
func (b *Bash) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	in, err := parseBashInputJSON(inputJSON)
	if err != nil {
		return nil, fmt.Errorf("bashtool: invalid json: %w", err)
	}
	cmdStr := strings.TrimSpace(in.Command)
	if cmdStr == "" {
		return nil, errors.New("bashtool: missing command")
	}
	if strings.ContainsRune(cmdStr, 0) {
		return nil, errors.New("bashtool: null byte in command")
	}

	runInBG := in.RunInBackground != nil && *in.RunInBackground

	if features.MonitorToolEnabled() && !backgroundTasksDisabled() && !runInBG {
		if hint := DetectBlockedSleepPattern(cmdStr); hint != "" {
			return nil, fmt.Errorf("Blocked: %s.%s", hint, blockedSleepFollowup)
		}
	}

	ms := DefaultBashTimeoutMs()
	if in.Timeout != nil && *in.Timeout > 0 {
		ms = *in.Timeout
		if ms > MaxBashTimeoutMs() {
			ms = MaxBashTimeoutMs()
		}
	}

	if !features.BashExecEnabled() {
		out := bashOutput{Stdout: "", Stderr: "", Interrupted: false}
		applyDangerouslyDisableSandbox(&out, in.DangerouslyDisableSandbox)
		return json.Marshal(out)
	}

	// BashTool.tsx: run_in_background only when background tasks enabled; otherwise foreground.
	if runInBG && !backgroundTasksDisabled() {
		tid, path, err := startBackgroundCommand(cmdStr, ms)
		if err != nil {
			return nil, err
		}
		out := bashOutput{
			Stdout:                   "",
			Stderr:                   "",
			Interrupted:              false,
			BackgroundTaskID:         tid,
			BackgroundTaskOutputPath: path,
		}
		applyDangerouslyDisableSandbox(&out, in.DangerouslyDisableSandbox)
		silent := IsSilentBashCommand(cmdStr)
		out.NoOutputExpected = &silent
		return json.Marshal(out)
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(cctx, "sh", "-c", cmdStr)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	exitCode := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	outStdout := stdout.String()
	outStderr := stderr.String()
	isErr, semMsg := InterpretCommandResult(cmdStr, exitCode, outStdout, outStderr)
	interrupted := errors.Is(cctx.Err(), context.DeadlineExceeded)

	if isErr && !interrupted {
		if outStderr == "" && runErr != nil {
			outStderr = runErr.Error()
		}
	}

	var retInterp *string
	if semMsg != "" {
		s := semMsg
		retInterp = &s
	}

	noExp := IsSilentBashCommand(cmdStr)
	out := bashOutput{
		Stdout:                   outStdout,
		Stderr:                   outStderr,
		Interrupted:              interrupted,
		ReturnCodeInterpretation: retInterp,
		NoOutputExpected:         &noExp,
	}
	applyDangerouslyDisableSandbox(&out, in.DangerouslyDisableSandbox)
	return json.Marshal(out)
}

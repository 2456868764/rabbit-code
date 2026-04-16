package bashtool

import (
	"strings"
	"testing"
)

// sed_validation.go gap tests

func TestIsLinePrintingCommand_exported(t *testing.T) {
	exprs := []string{"1p", "2,5p"}
	if !IsLinePrintingCommand("sed -n '1p'", []string{"1p"}) {
		t.Fatal("expected sed -n 1p to be a line printing command")
	}
	if IsLinePrintingCommand("sed 's/a/b/'", exprs) {
		t.Fatal("sed s/// should not be line printing")
	}
}

func TestCheckSedConstraints_safeRead(t *testing.T) {
	if CheckSedConstraints("sed -n '1p' file.txt", ModeDefault) != "" {
		t.Fatal("expected passthrough for safe read sed")
	}
}

func TestCheckSedConstraints_dangerousExec(t *testing.T) {
	if CheckSedConstraints("sed '1e cat /etc/passwd' file.txt", ModeDefault) == "" {
		t.Fatal("expected rejection for sed execute command")
	}
}

func TestCheckSedConstraints_acceptEditsAllowsInPlace(t *testing.T) {
	if CheckSedConstraints("sed -i 's/foo/bar/' file.txt", ModeAcceptEdits) != "" {
		t.Fatal("expected acceptEdits to allow sed -i substitution")
	}
}

func TestCheckSedConstraints_skipNonSed(t *testing.T) {
	if CheckSedConstraints("grep foo bar", ModeDefault) != "" {
		t.Fatal("expected passthrough for non-sed command")
	}
}

// sed_edit_parser.go gap test

func TestIsSedInPlaceEdit(t *testing.T) {
	if !IsSedInPlaceEdit("sed -i 's/foo/bar/' file.txt") {
		t.Fatal("expected sed -i s/// to be in-place edit")
	}
	if IsSedInPlaceEdit("sed -n '1p' file.txt") {
		t.Fatal("expected sed -n (no -i) to not be in-place edit")
	}
	if IsSedInPlaceEdit("echo hello") {
		t.Fatal("expected non-sed to not be in-place edit")
	}
}

// mode_validation.go gap test

func TestGetAutoAllowedCommands_acceptEdits(t *testing.T) {
	cmds := GetAutoAllowedCommands(ModeAcceptEdits)
	if len(cmds) == 0 {
		t.Fatal("expected non-empty list for acceptEdits")
	}
	found := false
	for _, c := range cmds {
		if c == "sed" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'sed' in acceptEdits auto-allowed commands")
	}
}

func TestGetAutoAllowedCommands_otherModes(t *testing.T) {
	for _, m := range []PermissionMode{ModeDefault, ModeDontAsk, ModeBypassPermissions} {
		if len(GetAutoAllowedCommands(m)) != 0 {
			t.Fatalf("expected empty auto-allowed for mode %q", m)
		}
	}
}

// command_semantics.go gap test — CommandSemantic type exported

func TestCommandSemantic_typeUsable(t *testing.T) {
	var sem CommandSemantic = func(exitCode int, stdout, stderr string) (bool, string) {
		return exitCode != 0, ""
	}
	isErr, _ := sem(1, "", "")
	if !isErr {
		t.Fatal("expected isError=true for exit 1")
	}
}

// utils_bash_tool.go gap tests

func TestIsImageOutput(t *testing.T) {
	if !IsImageOutput("data:image/png;base64,abc123") {
		t.Fatal("expected image data URI detected")
	}
	if IsImageOutput("hello world") {
		t.Fatal("expected non-image rejected")
	}
}

func TestParseDataUri(t *testing.T) {
	r := ParseDataUri("data:image/png;base64,abc123")
	if r == nil || r.MediaType != "image/png" || r.Data != "abc123" {
		t.Fatalf("unexpected parse result: %v", r)
	}
	if ParseDataUri("not a data uri") != nil {
		t.Fatal("expected nil for non data uri")
	}
}

func TestFormatOutput_noTruncation(t *testing.T) {
	result := FormatOutput("line1\nline2\nline3", -1)
	if result.TotalLines != 3 {
		t.Fatalf("expected 3 lines, got %d", result.TotalLines)
	}
	if result.IsImage {
		t.Fatal("expected IsImage=false")
	}
}

func TestFormatOutput_truncation(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	result := FormatOutput(content, 6)
	if !strings.Contains(result.TruncatedContent, "truncated") {
		t.Fatal("expected truncation marker in output")
	}
	if result.TotalLines != 5 {
		t.Fatalf("expected 5 total lines, got %d", result.TotalLines)
	}
}

func TestFormatOutput_image(t *testing.T) {
	result := FormatOutput("data:image/png;base64,abc", -1)
	if !result.IsImage {
		t.Fatal("expected IsImage=true for data URI")
	}
	if result.TotalLines != 1 {
		t.Fatal("expected 1 line for image")
	}
}

func TestStdErrAppendShellResetMessage(t *testing.T) {
	msg := StdErrAppendShellResetMessage("error output\n", "/home/user/project")
	if !strings.Contains(msg, "Shell cwd was reset to /home/user/project") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestCreateContentSummary(t *testing.T) {
	blocks := []ContentBlock{
		{Type: "image"},
		{Type: "text", Text: "hello world"},
	}
	summary := CreateContentSummary(blocks)
	if !strings.Contains(summary, "1 image") {
		t.Fatalf("expected '1 image' in summary, got: %q", summary)
	}
	if !strings.Contains(summary, "1 text block") {
		t.Fatalf("expected '1 text block' in summary, got: %q", summary)
	}
	if !strings.Contains(summary, "hello world") {
		t.Fatalf("expected text content in summary, got: %q", summary)
	}
}

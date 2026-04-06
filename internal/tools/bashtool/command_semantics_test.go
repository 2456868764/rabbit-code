package bashtool_test

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestInterpretCommandResult_grepNoMatch(t *testing.T) {
	isErr, msg := bashtool.InterpretCommandResult("grep foo /dev/null", 1, "", "")
	if isErr || msg != "No matches found" {
		t.Fatalf("isErr=%v msg=%q", isErr, msg)
	}
}

func TestInterpretCommandResult_grepError(t *testing.T) {
	isErr, _ := bashtool.InterpretCommandResult("grep", 2, "", "")
	if !isErr {
		t.Fatal("exit 2 should be error")
	}
}

package bashtool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestSplitCommandDeprecated_sleepChain(t *testing.T) {
	parts := bashtool.SplitCommandDeprecated("sleep 5 && echo hi")
	if len(parts) != 2 || parts[0] != "sleep 5" || parts[1] != "echo hi" {
		t.Fatalf("got %#v", parts)
	}
}

func TestSplitCommandDeprecated_semicolon(t *testing.T) {
	parts := bashtool.SplitCommandDeprecated("true; false")
	if len(parts) != 2 || parts[0] != "true" || parts[1] != "false" {
		t.Fatalf("got %#v", parts)
	}
}

func TestSplitCommandWithOperators_redirectSkip(t *testing.T) {
	// grep x > out → operator stream still lists grep before > for IsSearchOrRead
	toks := bashtool.SplitCommandWithOperators("grep x > out")
	if !strings.Contains(strings.Join(toks, " "), "grep") {
		t.Fatalf("got %#v", toks)
	}
}

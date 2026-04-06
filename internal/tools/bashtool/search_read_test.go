package bashtool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

func TestIsSearchOrReadBashCommand_grep(t *testing.T) {
	k := bashtool.IsSearchOrReadBashCommand("grep -r foo .")
	if !k.IsSearch || k.IsRead || k.IsList {
		t.Fatalf("%+v", k)
	}
}

func TestIsSearchOrReadBashCommand_lsAndEcho(t *testing.T) {
	k := bashtool.IsSearchOrReadBashCommand("ls . && echo ---")
	if !k.IsList || k.IsSearch {
		t.Fatalf("%+v", k)
	}
}

func TestIsSearchOrReadBashCommand_neutralOnly(t *testing.T) {
	k := bashtool.IsSearchOrReadBashCommand("echo hello")
	if k.IsSearch || k.IsRead || k.IsList {
		t.Fatalf("%+v", k)
	}
}

func TestDetectBlockedSleepPattern(t *testing.T) {
	if bashtool.DetectBlockedSleepPattern("sleep 1") != "" {
		t.Fatal("sub-2s should pass")
	}
	if s := bashtool.DetectBlockedSleepPattern("sleep 2"); s == "" || !strings.Contains(s, "standalone") {
		t.Fatalf("got %q", s)
	}
	if s := bashtool.DetectBlockedSleepPattern("sleep 5 && npm test"); s == "" || !strings.Contains(s, "followed by") {
		t.Fatalf("got %q", s)
	}
}

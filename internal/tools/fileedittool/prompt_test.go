package fileedittool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
)

func TestGetEditToolPrompt_upstreamShape(t *testing.T) {
	p := fileedittool.GetEditToolPrompt()
	if !strings.Contains(p, filereadtool.FileReadToolName) {
		t.Fatal("missing Read tool pre-read instruction")
	}
	if !strings.Contains(p, "old_string") {
		t.Fatal("missing old_string mention")
	}
	if !strings.Contains(p, "replace_all") {
		t.Fatal("missing replace_all mention")
	}
}

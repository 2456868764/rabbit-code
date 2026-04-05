package filereadtool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func TestRenderReadToolPrompt_upstreamShape(t *testing.T) {
	p := filereadtool.RenderReadToolPrompt(filereadtool.DefaultFileReadingLimits(), true)
	if !strings.Contains(p, "2000") {
		snip := p
		if len(snip) > 200 {
			snip = snip[:200]
		}
		t.Fatalf("prompt should include default line cap: %s", snip)
	}
	if !strings.Contains(p, bashtool.BashToolName) {
		t.Fatal("missing Bash tool reference")
	}
	if !strings.Contains(p, "Jupyter notebooks") {
		t.Fatal("missing notebook line")
	}
}

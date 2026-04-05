package filewritetool_test

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
)

func TestGetWriteToolDescription_upstreamShape(t *testing.T) {
	p := filewritetool.GetWriteToolDescription()
	if !strings.Contains(p, filereadtool.FileReadToolName) {
		t.Fatal("missing Read pre-read instruction")
	}
	if !strings.Contains(p, "Edit tool") {
		t.Fatal("missing Edit preference line")
	}
}

package engine

import (
	"strings"

	"github.com/2456868764/rabbit-code/internal/memdir"
	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
)

func (e *Engine) refreshMemorySystemPromptForAssistant() {
	mem := strings.TrimSpace(e.memdirMemoryDir)
	if mem == "" {
		e.setAnthropicSystemPrompt("")
		return
	}
	text, ok := memdir.LoadMemorySystemPrompt(memdir.MemorySystemPromptInput{
		MemoryDir:   mem,
		ProjectRoot: e.memdirProjectRoot,
		Merged:      e.initialSettings,
	})
	if !ok {
		e.setAnthropicSystemPrompt("")
		return
	}
	e.setAnthropicSystemPrompt(text)
}

func (e *Engine) setAnthropicSystemPrompt(s string) {
	for _, x := range []any{e.deps.Turn, e.deps.Assistant} {
		if a, ok := x.(*anthropic.AnthropicAssistant); ok && a != nil {
			a.SystemPrompt = s
		}
	}
}

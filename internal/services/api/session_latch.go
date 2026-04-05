package anthropic

import (
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// EnvRabbitSessionLatchedBetas seeds session-stable anthropic-beta tokens (comma-separated) once per client (TS session latch analogue).
const EnvRabbitSessionLatchedBetas = "RABBIT_CODE_SESSION_LATCHED_BETAS"

// LatchSessionBeta appends a beta name to the client's session latch list (deduplicated). Merged into anthropic-beta on each request.
func (c *Client) LatchSessionBeta(name string) {
	name = strings.TrimSpace(name)
	if name == "" || c == nil {
		return
	}
	c.sessionLatchMu.Lock()
	defer c.sessionLatchMu.Unlock()
	c.sessionLatchedBetas = AppendBetaUnique(c.sessionLatchedBetas, name)
}

// LatchFastModeHeader latches fast-mode beta on the session (claude.ts fastModeHeaderLatched).
func (c *Client) LatchFastModeHeader() { c.LatchSessionBeta(BetaFastMode) }

// LatchAFKHeader marks AFK beta as session-latched; it is merged only when TranscriptClassifierEnabled, AgenticQuery, and 1P provider hold (claude.ts afkHeaderLatched).
func (c *Client) LatchAFKHeader() {
	if c == nil {
		return
	}
	c.sessionLatchMu.Lock()
	c.afkHeaderLatched = true
	c.sessionLatchMu.Unlock()
}

// LatchCacheEditingHeader latches cache-editing header for the session; it is merged only for repl_main_thread on 1P (claude.ts cacheEditingHeaderLatched + querySource gate).
func (c *Client) LatchCacheEditingHeader() {
	if c == nil {
		return
	}
	c.sessionLatchMu.Lock()
	c.cacheEditingHeaderLatched = true
	c.sessionLatchMu.Unlock()
}

// LatchThinkingClear sets the session latch so context-management uses clear-all thinking for the remainder of the session (thinkingClearLatched).
func (c *Client) LatchThinkingClear() {
	if c == nil {
		return
	}
	c.sessionLatchMu.Lock()
	c.thinkingClearLatched = true
	c.sessionLatchMu.Unlock()
}

func (c *Client) ThinkingClearLatched() bool {
	if c == nil {
		return false
	}
	c.sessionLatchMu.Lock()
	defer c.sessionLatchMu.Unlock()
	return c.thinkingClearLatched
}

func (c *Client) loadEnvSessionLatchesLocked() {
	if c == nil || c.envSessionLatchesLoaded {
		return
	}
	c.envSessionLatchesLoaded = true
	s := strings.TrimSpace(os.Getenv(EnvRabbitSessionLatchedBetas))
	if s == "" {
		return
	}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			c.sessionLatchedBetas = AppendBetaUnique(c.sessionLatchedBetas, p)
		}
	}
}

// mergeSessionLatchedBetas appends latched betas and conditional AFK beta into the header value.
func (c *Client) mergeSessionLatchedBetas(header string, pol Policy) string {
	if c == nil {
		return header
	}
	c.sessionLatchMu.Lock()
	c.loadEnvSessionLatchesLocked()
	betas := append([]string(nil), c.sessionLatchedBetas...)
	afk := c.afkHeaderLatched
	cacheEdit := c.cacheEditingHeaderLatched
	c.sessionLatchMu.Unlock()

	out := header
	for _, b := range betas {
		out = MergeBetaHeaderAppend(out, b)
	}
	if afk && features.TranscriptClassifierEnabled() && pol.AgenticQuery && c.Provider == ProviderAnthropic {
		out = MergeBetaHeaderAppend(out, BetaAFKMode)
	}
	if cacheEdit && pol.QuerySource == QuerySourceReplMainThread && c.Provider == ProviderAnthropic {
		out = MergeBetaHeaderAppend(out, BetaPromptCachingScope)
	}
	return out
}

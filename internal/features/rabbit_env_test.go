package features

import (
	"os"
	"testing"
)

func TestOAuthBetaAppendNames(t *testing.T) {
	t.Setenv(EnvOAuthBetaAppend, " a , b ")
	got := OAuthBetaAppendNames()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}

func TestNativeAttestationRequestHeader(t *testing.T) {
	t.Setenv(EnvNativeAttestation, "1")
	t.Setenv(EnvNativeAttestationHeader, "X-Test-Attest")
	t.Setenv(EnvNativeAttestationValue, "token")
	n, v, ok := NativeAttestationRequestHeader()
	if !ok || n != "X-Test-Attest" || v != "token" {
		t.Fatalf("%q %q %v", n, v, ok)
	}
}

func TestStrictForeground529Enabled(t *testing.T) {
	t.Setenv(EnvStrictForeground529, "1")
	if !StrictForeground529Enabled() {
		t.Fatal()
	}
	t.Setenv(EnvStrictForeground529, "")
	if StrictForeground529Enabled() {
		t.Fatal()
	}
}

func TestAttributionHeaderPromptEnabled(t *testing.T) {
	t.Setenv(EnvAttributionHeader, "0")
	if AttributionHeaderPromptEnabled() {
		t.Fatal()
	}
	t.Setenv(EnvAttributionHeader, "1")
	if !AttributionHeaderPromptEnabled() {
		t.Fatal()
	}
}

func TestAntiDistillationFakeToolsInBody(t *testing.T) {
	t.Setenv(EnvAntiDistillation, "")
	t.Setenv(EnvAntiDistillationFakeTools, "1")
	if AntiDistillationFakeToolsInBody() {
		t.Fatal("CC off")
	}
	t.Setenv(EnvAntiDistillation, "1")
	if !AntiDistillationFakeToolsInBody() {
		t.Fatal("both on")
	}
}

func TestDisableKeepAliveOnECONNRESETEnabled(t *testing.T) {
	t.Setenv(EnvDisableKeepAliveOnECONNRESET, "1")
	if !DisableKeepAliveOnECONNRESETEnabled() {
		t.Fatal()
	}
	t.Setenv(EnvDisableKeepAliveOnECONNRESET, "")
	if DisableKeepAliveOnECONNRESETEnabled() {
		t.Fatal()
	}
}

func TestAntiDistillationRequestHeader(t *testing.T) {
	t.Setenv(EnvAntiDistillation, "1")
	t.Setenv(EnvAntiDistillationHeader, "X-Custom-AD")
	t.Setenv(EnvAntiDistillationValue, "yes")
	n, v, ok := AntiDistillationRequestHeader()
	if !ok || n != "X-Custom-AD" || v != "yes" {
		t.Fatalf("%q %q %v", n, v, ok)
	}
}

func TestHeadlessQueryEnv_defaultsOff(t *testing.T) {
	t.Setenv(EnvTokenBudget, "")
	t.Setenv(EnvReactiveCompact, "")
	if TokenBudgetEnabled() || ReactiveCompactEnabled() {
		t.Fatal("expected off")
	}
	if TokenBudgetMaxInputBytes() != 0 {
		t.Fatalf("got %d", TokenBudgetMaxInputBytes())
	}
}

func TestTokenBudgetMaxInputBytes_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	t.Setenv(EnvTokenBudgetMaxInputBytes, "")
	if TokenBudgetMaxInputBytes() != 4_000_000 {
		t.Fatalf("default %d", TokenBudgetMaxInputBytes())
	}
	t.Setenv(EnvTokenBudgetMaxInputBytes, "100")
	if TokenBudgetMaxInputBytes() != 100 {
		t.Fatalf("got %d", TokenBudgetMaxInputBytes())
	}
}

func TestTokenBudgetMaxInputTokens_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	if TokenBudgetMaxInputTokens() != 0 {
		t.Fatalf("unset want 0 got %d", TokenBudgetMaxInputTokens())
	}
	t.Setenv(EnvTokenBudgetMaxInputTokens, "500")
	if TokenBudgetMaxInputTokens() != 500 {
		t.Fatalf("got %d", TokenBudgetMaxInputTokens())
	}
}

func TestSubmitTokenEstimateMode(t *testing.T) {
	t.Setenv(EnvTokenBudget, "")
	if SubmitTokenEstimateMode() != "bytes4" {
		t.Fatal("off should bytes4")
	}
	t.Setenv(EnvTokenBudget, "1")
	t.Setenv(EnvTokenSubmitEstimateMode, "")
	if SubmitTokenEstimateMode() != "bytes4" {
		t.Fatal()
	}
	t.Setenv(EnvTokenSubmitEstimateMode, "structured")
	if SubmitTokenEstimateMode() != "structured" {
		t.Fatal()
	}
	t.Setenv(EnvTokenSubmitEstimateMode, "api")
	if SubmitTokenEstimateMode() != "api" {
		t.Fatal()
	}
}

func TestMemdirRelevanceMode(t *testing.T) {
	t.Setenv(EnvMemdirRelevanceMode, "")
	if MemdirRelevanceMode() != "heuristic" {
		t.Fatalf("%q", MemdirRelevanceMode())
	}
	t.Setenv(EnvMemdirRelevanceMode, "llm")
	if MemdirRelevanceMode() != "llm" {
		t.Fatalf("%q", MemdirRelevanceMode())
	}
	t.Setenv(EnvMemdirRelevanceMode, "side_query")
	if MemdirRelevanceMode() != "llm" {
		t.Fatalf("%q", MemdirRelevanceMode())
	}
}

func TestMemdirStrictLLM(t *testing.T) {
	t.Setenv(EnvMemdirStrictLLM, "")
	if MemdirStrictLLM() {
		t.Fatal()
	}
	t.Setenv(EnvMemdirStrictLLM, "1")
	if !MemdirStrictLLM() {
		t.Fatal()
	}
}

func TestMemdirMemoryDirFromEnv(t *testing.T) {
	t.Setenv(EnvMemdirMemoryDir, "")
	if MemdirMemoryDirFromEnv() != "" {
		t.Fatal()
	}
	t.Setenv(EnvMemdirMemoryDir, "  /tmp/memdir-x  ")
	if MemdirMemoryDirFromEnv() != "/tmp/memdir-x" {
		t.Fatalf("%q", MemdirMemoryDirFromEnv())
	}
}

func TestAutoMemdirFromProject(t *testing.T) {
	t.Setenv(EnvAutoMemdir, "")
	if AutoMemdirFromProject() {
		t.Fatal()
	}
	t.Setenv(EnvAutoMemdir, "yes")
	if !AutoMemdirFromProject() {
		t.Fatal()
	}
}

func TestAutoMemoryEnabled(t *testing.T) {
	clear := func() {
		for _, k := range []string{
			EnvDisableAutoMemory, EnvClaudeDisableAutoMemory,
			EnvSimple, EnvClaudeSimple,
			EnvRemote, EnvClaudeRemote,
			EnvRemoteMemoryDir, EnvClaudeRemoteMemoryDir,
		} {
			_ = os.Unsetenv(k)
		}
	}
	clear()
	if !AutoMemoryEnabled() {
		t.Fatal("default on")
	}
	t.Setenv(EnvDisableAutoMemory, "1")
	if AutoMemoryEnabled() {
		t.Fatal("disable rabbit")
	}
	clear()
	t.Setenv(EnvClaudeDisableAutoMemory, "true")
	if AutoMemoryEnabled() {
		t.Fatal("disable claude")
	}
	clear()
	t.Setenv(EnvDisableAutoMemory, "0")
	if !AutoMemoryEnabled() {
		t.Fatal("explicit off should re-enable when truthy chain cleared")
	}
	clear()
	t.Setenv(EnvSimple, "1")
	if AutoMemoryEnabled() {
		t.Fatal("simple off")
	}
	clear()
	t.Setenv(EnvRemote, "1")
	if AutoMemoryEnabled() {
		t.Fatal("remote without memory dir off")
	}
	clear()
	t.Setenv(EnvRemote, "1")
	t.Setenv(EnvRemoteMemoryDir, "/tmp/x")
	if !AutoMemoryEnabled() {
		t.Fatal("remote with memory dir on")
	}
}

func TestTokenBudgetMaxAttachmentBytes_whenEnabled(t *testing.T) {
	t.Setenv(EnvTokenBudget, "1")
	if TokenBudgetMaxAttachmentBytes() != 0 {
		t.Fatal()
	}
	t.Setenv(EnvTokenBudgetMaxAttachmentBytes, "99")
	if TokenBudgetMaxAttachmentBytes() != 99 {
		t.Fatal()
	}
}

func TestPromptCacheBreakSuggestCompactEnabled(t *testing.T) {
	t.Setenv(EnvPromptCacheBreakSuggestCompact, "1")
	if !PromptCacheBreakSuggestCompactEnabled() {
		t.Fatal()
	}
}

func TestPromptCacheBreakTrimResendEnabled(t *testing.T) {
	t.Setenv(EnvPromptCacheBreak, "")
	if PromptCacheBreakTrimResendEnabled() {
		t.Fatal("detection off => trim off")
	}
	t.Setenv(EnvPromptCacheBreak, "1")
	t.Setenv(EnvPromptCacheBreakTrimResend, "")
	if !PromptCacheBreakTrimResendEnabled() {
		t.Fatal("default on when detection on")
	}
	t.Setenv(EnvPromptCacheBreakTrimResend, "0")
	if PromptCacheBreakTrimResendEnabled() {
		t.Fatal("explicit off")
	}
}

func TestPromptCacheBreakAutoCompactEnabled(t *testing.T) {
	t.Setenv(EnvPromptCacheBreak, "1")
	t.Setenv(EnvPromptCacheBreakAutoCompact, "")
	if PromptCacheBreakAutoCompactEnabled() {
		t.Fatal("default off")
	}
	t.Setenv(EnvPromptCacheBreakAutoCompact, "1")
	if !PromptCacheBreakAutoCompactEnabled() {
		t.Fatal("explicit on")
	}
}

func TestSessionRestoreEnabled(t *testing.T) {
	t.Setenv(EnvSessionRestore, "true")
	if !SessionRestoreEnabled() {
		t.Fatal()
	}
}

func TestBashExecEnabled(t *testing.T) {
	t.Setenv(EnvBashExec, "true")
	if !BashExecEnabled() {
		t.Fatal()
	}
}

func TestSnipCompactEnabled(t *testing.T) {
	t.Setenv(EnvSnipCompact, "true")
	if !SnipCompactEnabled() {
		t.Fatal()
	}
}

func TestReactiveCompactMinTranscriptBytes(t *testing.T) {
	t.Setenv(EnvReactiveCompact, "1")
	t.Setenv(EnvReactiveCompactMinBytes, "50")
	if ReactiveCompactMinTranscriptBytes() != 50 {
		t.Fatal()
	}
}

func TestReactiveCompactMinEstimatedTokens(t *testing.T) {
	t.Setenv(EnvReactiveCompact, "1")
	if ReactiveCompactMinEstimatedTokens() != 0 {
		t.Fatal()
	}
	t.Setenv(EnvReactiveCompactMinTokens, "100")
	if ReactiveCompactMinEstimatedTokens() != 100 {
		t.Fatal()
	}
}

func TestHistorySnipThresholds(t *testing.T) {
	t.Setenv(EnvHistorySnip, "true")
	t.Setenv(EnvHistorySnipMaxBytes, "99")
	if HistorySnipMaxBytes() != 99 {
		t.Fatal()
	}
}

func TestSnipCompactThresholds(t *testing.T) {
	t.Setenv(EnvSnipCompact, "true")
	t.Setenv(EnvSnipCompactMaxBytes, "100")
	if SnipCompactMaxBytes() != 100 {
		t.Fatal()
	}
	t.Setenv(EnvHistorySnip, "")
	if HistorySnipMaxBytes() != 0 {
		t.Fatal("history snip should be off")
	}
}

func TestTemplateMarkdownDir(t *testing.T) {
	t.Setenv(EnvTemplates, "1")
	t.Setenv(EnvTemplateDir, "/tmp/tpl")
	if TemplateMarkdownDir() != "/tmp/tpl" {
		t.Fatal()
	}
}

func TestTemplateNames(t *testing.T) {
	t.Setenv(EnvTemplates, "true")
	t.Setenv(EnvTemplateNames, " a , b ")
	n := TemplateNames()
	if len(n) != 2 || n[0] != "a" || n[1] != "b" {
		t.Fatalf("%#v", n)
	}
}

func TestIsAutoCompactEnabled_envChain(t *testing.T) {
	t.Setenv(EnvDisableCompact, "")
	t.Setenv(EnvDisableAutoCompact, "")
	t.Setenv(EnvAutoCompact, "")
	if !IsAutoCompactEnabled() {
		t.Fatal("default on")
	}
	t.Setenv(EnvAutoCompact, "0")
	if IsAutoCompactEnabled() {
		t.Fatal("user off")
	}
	t.Setenv(EnvAutoCompact, "1")
	t.Setenv(EnvDisableAutoCompact, "true")
	if IsAutoCompactEnabled() {
		t.Fatal("disable auto")
	}
	t.Setenv(EnvDisableAutoCompact, "")
	t.Setenv(EnvDisableCompact, "1")
	if IsAutoCompactEnabled() {
		t.Fatal("disable compact")
	}
}

func TestApplyAutoCompactWindowCap(t *testing.T) {
	t.Setenv(EnvAutoCompactWindow, "")
	if ApplyAutoCompactWindowCap(200_000) != 200_000 {
		t.Fatal()
	}
	t.Setenv(EnvAutoCompactWindow, "50000")
	if ApplyAutoCompactWindowCap(200_000) != 50_000 {
		t.Fatal()
	}
	if ApplyAutoCompactWindowCap(40_000) != 40_000 {
		t.Fatal("cap should not raise window")
	}
}

func TestContextWindowTokensForModel(t *testing.T) {
	t.Setenv(EnvContextWindowTokens, "")
	t.Setenv(EnvAutoCompactWindow, "")
	if ContextWindowTokensForModel("claude-foo") != 200_000 {
		t.Fatalf("default 200k got %d", ContextWindowTokensForModel("claude-foo"))
	}
	if ContextWindowTokensForModel("opus-1m-extra") != 1_000_000 {
		t.Fatalf("1m hint got %d", ContextWindowTokensForModel("opus-1m-extra"))
	}
	t.Setenv(EnvContextWindowTokens, "12345")
	if ContextWindowTokensForModel("x") != 12_345 {
		t.Fatal()
	}
}

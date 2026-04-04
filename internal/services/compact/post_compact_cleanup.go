package compact

import (
	"context"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// IsMainThreadPostCompactSource mirrors postCompactCleanup.ts isMainThreadCompact (main-thread-only cache resets).
func IsMainThreadPostCompactSource(querySource string) bool {
	s := strings.TrimSpace(querySource)
	if s == "" {
		return true
	}
	if s == "sdk" {
		return true
	}
	return strings.HasPrefix(s, "repl_main_thread")
}

// PostCompactCleanupHooks carries optional callbacks for runPostCompactCleanup (postCompactCleanup.ts).
// Nil fields are skipped. Implementations live at the app/engine boundary; compact must not import query/session stores.
type PostCompactCleanupHooks struct {
	ResetContextCollapse      func()
	ClearUserContextCache     func()
	ResetMemoryFilesCache     func(reason string)
	ClearSystemPromptSections func()
	ClearClassifierApprovals  func()
	ClearSpeculativeChecks    func()
	ClearBetaTracingState     func()
	SweepFileContentCache     func()
	ClearSessionMessagesCache func()
}

// RunPostCompactCleanup mirrors services/compact/postCompactCleanup.ts runPostCompactCleanup:
// reset microcompact state, then optional cache clears gated like upstream (CONTEXT_COLLAPSE + main thread, etc.).
//
// When hooks is nil, only ResetMicrocompactStateIfAny(buf) runs (TS always calls resetMicrocompactState first).
func RunPostCompactCleanup(ctx context.Context, querySource string, buf *MicrocompactEditBuffer, hooks *PostCompactCleanupHooks) {
	_ = ctx
	ResetMicrocompactStateIfAny(buf)
	if hooks == nil {
		return
	}
	main := IsMainThreadPostCompactSource(querySource)
	if features.ContextCollapseEnabled() && main && hooks.ResetContextCollapse != nil {
		hooks.ResetContextCollapse()
	}
	if main {
		if hooks.ClearUserContextCache != nil {
			hooks.ClearUserContextCache()
		}
		if hooks.ResetMemoryFilesCache != nil {
			hooks.ResetMemoryFilesCache("compact")
		}
	}
	if hooks.ClearSystemPromptSections != nil {
		hooks.ClearSystemPromptSections()
	}
	if hooks.ClearClassifierApprovals != nil {
		hooks.ClearClassifierApprovals()
	}
	if hooks.ClearSpeculativeChecks != nil {
		hooks.ClearSpeculativeChecks()
	}
	if hooks.ClearBetaTracingState != nil {
		hooks.ClearBetaTracingState()
	}
	if features.CommitAttributionEnabled() && hooks.SweepFileContentCache != nil {
		hooks.SweepFileContentCache()
	}
	if hooks.ClearSessionMessagesCache != nil {
		hooks.ClearSessionMessagesCache()
	}
}

// ResetMicrocompactStateIfAny calls ResetMicrocompactState when b is non-nil (runPostCompactCleanup first step).
func ResetMicrocompactStateIfAny(b *MicrocompactEditBuffer) {
	if b != nil {
		b.ResetMicrocompactState()
	}
}

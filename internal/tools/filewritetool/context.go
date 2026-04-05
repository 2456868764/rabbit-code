package filewritetool

import (
	"context"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

type writeCtxKey struct{}

// WriteContext carries optional hooks mirroring FileWriteTool.ts ToolUseContext / services (LSP, VSCode, fileHistory, gitDiff, team memory).
// Nil func fields are no-ops, matching upstream when features or services are off.
type WriteContext struct {
	// DenyEdit returns true if path must be rejected (TS matchingRuleForInput ... 'edit' 'deny').
	DenyEdit func(absPath string) bool
	// ReadFileState same map as Read dedup (required for updates to existing files).
	ReadFileState *filereadtool.ReadFileStateMap
	// CheckTeamMemSecrets mirrors checkTeamMemSecrets (TEAMMEM); return non-empty to block the write with that message.
	CheckTeamMemSecrets func(absPath, content string) string
	// BeforeFileEdited mirrors diagnosticTracker.beforeFileEdited.
	BeforeFileEdited func(absPath string)
	// AfterWrite mirrors LSP changeFile/saveFile + notifyVscodeFileUpdated (host implements or leaves nil).
	AfterWrite func(absPath, oldContent, newContent string)
	// FileHistoryTrack mirrors fileHistoryTrackEdit(updateFileHistoryState, path, parentUUID); nil skips.
	FileHistoryTrack func(absPath, parentMessageUUID string)
	// ParentMessageUUID optional parent tool message id for file history.
	ParentMessageUUID string
	// QuartzLanternEnabled mirrors getFeatureValue_CACHED_MAY_BE_STALE('tengu_quartz_lantern', false) for gitDiff attachment.
	QuartzLanternEnabled func() bool
	// FetchGitDiff mirrors fetchSingleFileGitDiff when remote+flag; return nil to omit gitDiff.
	FetchGitDiff func(absPath string) (map[string]any, error)
	// FileEncodingMetadata optional override for readFileSyncWithMetadata (encoding + lineEndings); when ok is false, sniff from disk bytes like TS.
	FileEncodingMetadata func(absPath string) (encoding string, lineEndings LineEndingType, ok bool)
}

// WithWriteContext attaches *WriteContext. Prefer filereadtool.WithRunContext(ReadFileState) when only Read state is needed.
func WithWriteContext(ctx context.Context, w *WriteContext) context.Context {
	if w == nil {
		return ctx
	}
	return context.WithValue(ctx, writeCtxKey{}, w)
}

// WriteContextFrom returns *WriteContext or nil.
func WriteContextFrom(ctx context.Context) *WriteContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(writeCtxKey{}).(*WriteContext)
	return v
}

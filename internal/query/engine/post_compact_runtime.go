package engine

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// postCompactReadEntry mirrors compact.ts readFileState map values (content + recency).
type postCompactReadEntry struct {
	Content string
	At      int64 // Unix nano; sort desc for restore order
}

// SetPostCompactWorkspaceDir sets cwd for relative displayPath in file restore attachments (optional).
func (e *Engine) SetPostCompactWorkspaceDir(dir string) {
	if e == nil {
		return
	}
	e.postCompactWorkspaceDir = strings.TrimSpace(dir)
}

// SetPostCompactPlan sets plan file path and content for BuildPlanFileReferenceAttachmentMessageJSON after compact.
func (e *Engine) SetPostCompactPlan(planFilePath, planContent string) {
	if e == nil {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.postCompactPlanPath = strings.TrimSpace(planFilePath)
	e.postCompactPlanContent = planContent
}

// SetPostCompactPlanMode mirrors plan mode in createPlanModeAttachmentIfNeeded.
func (e *Engine) SetPostCompactPlanMode(on bool) {
	if e == nil {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.postCompactPlanMode = on
}

// AddPostCompactInvokedSkill appends a skill row (most-recent-first ordering is host responsibility).
func (e *Engine) AddPostCompactInvokedSkill(sk compact.PostCompactSkillEntry) {
	if e == nil {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.postCompactSkills = append(e.postCompactSkills, sk)
}

// AppendPostCompactDeltaAttachment appends pre-built attachment message JSON (deferred tools / MCP / listing deltas from host).
func (e *Engine) AppendPostCompactDeltaAttachment(msg json.RawMessage) {
	if e == nil || len(bytesTrimSpaceJSON(msg)) == 0 {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.postCompactDeltaAttach = append(e.postCompactDeltaAttach, json.RawMessage(append([]byte(nil), msg...)))
}

// RecordPostCompactFileRead records file content for post-compact restore (host may call instead of relying on Read tool observer).
func (e *Engine) RecordPostCompactFileRead(canonicalPath, content string) {
	if e == nil {
		return
	}
	p := filepath.Clean(strings.TrimSpace(canonicalPath))
	if p == "" || p == "." {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.putPostCompactReadUnlocked(p, content)
}

func (e *Engine) putPostCompactReadUnlocked(path, content string) {
	if e.postCompactReads == nil {
		e.postCompactReads = make(map[string]postCompactReadEntry)
	}
	e.postCompactReads[path] = postCompactReadEntry{Content: content, At: time.Now().UnixNano()}
}

func (e *Engine) recordPostCompactReadTool(name string, inputJSON, result []byte) {
	if e == nil || name != filereadtool.FileReadToolName {
		return
	}
	body := strings.TrimSpace(string(result))
	if body == "" || strings.HasPrefix(body, filereadtool.FileUnchangedStub) {
		return
	}
	fp := filepath.Clean(strings.TrimSpace(compact.ToolInputFilePathFromJSON(inputJSON)))
	if fp == "" || fp == "." {
		return
	}
	e.postCompactMu.Lock()
	defer e.postCompactMu.Unlock()
	e.putPostCompactReadUnlocked(fp, string(result))
}

// PostCompactAttachmentsForNextTranscript implements compact.ts postCompactFileAttachments ordering subset:
// file restores (from snapshot of read state, then clear), plan, plan_mode, invoked_skills, host deltas.
// preservedPaths come from CollectReadToolFilePathsFromTranscriptJSON(transcriptBefore).
func (e *Engine) PostCompactAttachmentsForNextTranscript(ctx context.Context, transcriptBefore []byte, rawAssistantSummary string) ([]json.RawMessage, error) {
	_ = ctx
	_ = rawAssistantSummary
	if e == nil {
		return nil, nil
	}
	preserved, err := compact.CollectReadToolFilePathsFromTranscriptJSON(transcriptBefore)
	if err != nil {
		return nil, err
	}

	e.postCompactMu.Lock()
	snap := make(map[string]postCompactReadEntry, len(e.postCompactReads))
	for k, v := range e.postCompactReads {
		snap[k] = v
	}
	e.postCompactReads = make(map[string]postCompactReadEntry)
	planPath := strings.TrimSpace(e.postCompactPlanPath)
	planContent := e.postCompactPlanContent
	planMode := e.postCompactPlanMode
	skills := append([]compact.PostCompactSkillEntry(nil), e.postCompactSkills...)
	deltas := append([]json.RawMessage(nil), e.postCompactDeltaAttach...)
	e.postCompactDeltaAttach = nil
	wsDir := e.postCompactWorkspaceDir
	subAgent := strings.TrimSpace(e.agentID) != ""
	e.postCompactMu.Unlock()

	type pair struct {
		path string
		ent  postCompactReadEntry
	}
	var pairs []pair
	for p, ent := range snap {
		pairs = append(pairs, pair{path: p, ent: ent})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].ent.At > pairs[j].ent.At })
	if len(pairs) > compact.PostCompactMaxFilesToRestore {
		pairs = pairs[:compact.PostCompactMaxFilesToRestore]
	}

	planKey := filepath.Clean(planPath)
	if planKey == "." {
		planKey = ""
	}

	var fileMsgs []json.RawMessage
	for _, pr := range pairs {
		if _, skip := preserved[pr.path]; skip {
			continue
		}
		if planKey != "" && pr.path == planKey {
			continue
		}
		disp := postCompactDisplayPath(wsDir, pr.path)
		body := compact.TruncateFileContentForPostCompact(pr.ent.Content, compact.PostCompactMaxTokensPerFile)
		truncated := body != pr.ent.Content
		raw, err := compact.BuildFileRestoreAttachmentMessageJSON(pr.path, disp, body, truncated)
		if err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			fileMsgs = append(fileMsgs, raw)
		}
	}
	fileMsgs = compact.FilterAttachmentMessagesByRoughTokenBudget(fileMsgs, compact.PostCompactTokenBudget)

	var out []json.RawMessage
	out = append(out, fileMsgs...)

	if raw, err := compact.BuildPlanFileReferenceAttachmentMessageJSON(planPath, planContent); err != nil {
		return nil, err
	} else if len(raw) > 0 {
		out = append(out, raw)
	}

	if planMode {
		exists := strings.TrimSpace(planContent) != ""
		raw, err := compact.BuildPlanModeAttachmentMessageJSON("full", subAgent, planPath, exists)
		if err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			out = append(out, raw)
		}
	}

	if raw, err := compact.BuildInvokedSkillsAttachmentMessageJSON(skills); err != nil {
		return nil, err
	} else if len(raw) > 0 {
		out = append(out, raw)
	}

	out = append(out, deltas...)
	return out, nil
}

// AttachPostCompactToStreamingConfig merges PostCompactAttachmentsForNextTranscript into cfg.PostCompactAttachmentsJSON (prepend engine attachments).
func (e *Engine) AttachPostCompactToStreamingConfig(cfg *query.StreamingCompactExecutorConfig) {
	if e == nil || cfg == nil {
		return
	}
	prev := cfg.PostCompactAttachmentsJSON
	cfg.PostCompactAttachmentsJSON = func(ctx context.Context, tr []byte, raw string) ([]json.RawMessage, error) {
		mine, err := e.PostCompactAttachmentsForNextTranscript(ctx, tr, raw)
		if err != nil {
			return nil, err
		}
		if prev == nil {
			return mine, nil
		}
		rest, err := prev(ctx, tr, raw)
		if err != nil {
			return nil, err
		}
		return append(append([]json.RawMessage(nil), mine...), rest...), nil
	}
}

func postCompactDisplayPath(workspaceDir, absPath string) string {
	ws := filepath.Clean(strings.TrimSpace(workspaceDir))
	if ws == "" || ws == "." {
		return absPath
	}
	rel, err := filepath.Rel(ws, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return absPath
	}
	return rel
}

func bytesTrimSpaceJSON(m json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(m)))
}

package compact

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// POST_COMPACT_* budgets: compact.go. Marker text matches compact.ts (truncateToTokens).
const (
	skillTruncationMarkerPostCompact = "\n\n[... skill content truncated for compaction; use Read on the skill path if you need the full text]"
	fileTruncationMarkerPostCompact  = "\n\n[... truncated for post-compact restore; use Read for the full file]"
)

// PostCompactSkillEntry is one skill row for BuildInvokedSkillsAttachmentMessageJSON (invokedAt ordering is host responsibility).
type PostCompactSkillEntry struct {
	Name    string
	Path    string
	Content string
}

func randomUUIDv4Attachment() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// TruncateSkillContentForPostCompact mirrors compact.ts truncateToTokens (head + marker; ~4 chars/token).
// TruncateFileContentForPostCompact caps file body by rough tokens (head + marker), for post-compact file restore attachments.
func TruncateFileContentForPostCompact(content string, maxTokens int) string {
	if maxTokens <= 0 {
		return content
	}
	if RoughTokenCountEstimationBytes(content) <= maxTokens {
		return content
	}
	charBudget := maxTokens*4 - len(fileTruncationMarkerPostCompact)
	if charBudget < 0 {
		charBudget = 0
	}
	if charBudget >= len(content) {
		return content
	}
	return content[:charBudget] + fileTruncationMarkerPostCompact
}

func TruncateSkillContentForPostCompact(content string, maxTokens int) string {
	if maxTokens <= 0 {
		return content
	}
	if RoughTokenCountEstimationBytes(content) <= maxTokens {
		return content
	}
	charBudget := maxTokens*4 - len(skillTruncationMarkerPostCompact)
	if charBudget < 0 {
		charBudget = 0
	}
	if charBudget >= len(content) {
		return content
	}
	return content[:charBudget] + skillTruncationMarkerPostCompact
}

// CreateAttachmentMessageJSON mirrors attachments.ts createAttachmentMessage: { type, uuid, timestamp, attachment }.
func CreateAttachmentMessageJSON(attachment map[string]interface{}) (json.RawMessage, error) {
	if attachment == nil {
		return nil, fmt.Errorf("compact: nil attachment payload")
	}
	env := map[string]interface{}{
		"type":       "attachment",
		"uuid":       randomUUIDv4Attachment(),
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"attachment": attachment,
	}
	return json.Marshal(env)
}

// BuildPlanFileReferenceAttachmentMessageJSON mirrors createPlanAttachmentIfNeeded. Empty planContent → nil, nil (skip).
func BuildPlanFileReferenceAttachmentMessageJSON(planFilePath, planContent string) (json.RawMessage, error) {
	if strings.TrimSpace(planContent) == "" {
		return nil, nil
	}
	att := map[string]interface{}{
		"type":         "plan_file_reference",
		"planFilePath": strings.TrimSpace(planFilePath),
		"planContent":  planContent,
	}
	return CreateAttachmentMessageJSON(att)
}

// BuildPlanModeAttachmentMessageJSON mirrors createPlanModeAttachmentIfNeeded (plan mode reminder after compact).
func BuildPlanModeAttachmentMessageJSON(reminderType string, isSubAgent bool, planFilePath string, planExists bool) (json.RawMessage, error) {
	rt := strings.TrimSpace(reminderType)
	if rt == "" {
		rt = "full"
	}
	att := map[string]interface{}{
		"type":         "plan_mode",
		"reminderType": rt,
		"isSubAgent":   isSubAgent,
		"planFilePath": strings.TrimSpace(planFilePath),
		"planExists":   planExists,
	}
	return CreateAttachmentMessageJSON(att)
}

// BuildInvokedSkillsAttachmentMessageJSON mirrors createSkillAttachmentIfNeeded: per-skill truncate, global budget, most-recent-first slice from host.
func BuildInvokedSkillsAttachmentMessageJSON(skills []PostCompactSkillEntry) (json.RawMessage, error) {
	if len(skills) == 0 {
		return nil, nil
	}
	used := 0
	var rows []map[string]interface{}
	for _, sk := range skills {
		content := TruncateSkillContentForPostCompact(sk.Content, PostCompactMaxTokensPerSkill)
		tok := RoughTokenCountEstimationBytes(content)
		if used+tok > PostCompactSkillsTokenBudget {
			continue
		}
		used += tok
		rows = append(rows, map[string]interface{}{
			"name":    sk.Name,
			"path":    sk.Path,
			"content": content,
		})
	}
	if len(rows) == 0 {
		return nil, nil
	}
	att := map[string]interface{}{
		"type":   "invoked_skills",
		"skills": rows,
	}
	return CreateAttachmentMessageJSON(att)
}

// BuildFileRestoreAttachmentMessageJSON builds attachment payload type "file" (attachments.ts generateFileAttachment success shape) wrapped via CreateAttachmentMessageJSON.
func BuildFileRestoreAttachmentMessageJSON(filename, displayPath, content string, truncated bool) (json.RawMessage, error) {
	att := map[string]interface{}{
		"type":        "file",
		"filename":    filename,
		"content":     content,
		"displayPath": displayPath,
	}
	if truncated {
		att["truncated"] = true
	}
	return CreateAttachmentMessageJSON(att)
}

// BuildTaskStatusAttachmentMessageJSON mirrors one createAsyncAgentAttachmentsIfNeeded attachment (local_agent row).
func BuildTaskStatusAttachmentMessageJSON(taskID, description, status, outputFilePath string, deltaSummary interface{}) (json.RawMessage, error) {
	att := map[string]interface{}{
		"type":           "task_status",
		"taskId":         strings.TrimSpace(taskID),
		"taskType":       "local_agent",
		"description":    description,
		"status":         status,
		"deltaSummary":   deltaSummary,
		"outputFilePath": strings.TrimSpace(outputFilePath),
	}
	return CreateAttachmentMessageJSON(att)
}

// FilterAttachmentMessagesByRoughTokenBudget mirrors createPostCompactFileAttachments result filtering: keep prefix until cumulative rough tokens exceed budget.
func FilterAttachmentMessagesByRoughTokenBudget(messages []json.RawMessage, budget int) []json.RawMessage {
	if budget <= 0 || len(messages) == 0 {
		return nil
	}
	used := 0
	var out []json.RawMessage
	for _, m := range messages {
		if len(bytesTrimSpaceRaw(m)) == 0 {
			continue
		}
		tok := RoughTokenCountEstimationBytes(string(m))
		if used+tok > budget {
			break
		}
		used += tok
		out = append(out, m)
	}
	return out
}

func bytesTrimSpaceRaw(m json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(m)))
}

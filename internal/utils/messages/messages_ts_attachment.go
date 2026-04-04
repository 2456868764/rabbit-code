// normalizeAttachmentForAPI parity (src/utils/messages.ts) — attachment map[string]any → user TSMsg slices.
package messages

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// FormatTeammateMailboxMessagesForAPI optionally overrides DefaultFormatTeammateMailboxMessagesForAPI.
var FormatTeammateMailboxMessagesForAPI func(messages []any) string

const teammateMessageXMLTag = "teammate-message"

// DefaultFormatTeammateMailboxMessagesForAPI mirrors TS formatTeammateMessages (utils/teammateMailbox.ts).
func DefaultFormatTeammateMailboxMessagesForAPI(messages []any) string {
	if len(messages) == 0 {
		return ""
	}
	var parts []string
	for _, raw := range messages {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		from := strField(m, "from")
		text := strField(m, "text")
		color := strField(m, "color")
		summary := strField(m, "summary")
		var colorAttr, summaryAttr string
		if color != "" {
			colorAttr = fmt.Sprintf(` color="%s"`, color)
		}
		if summary != "" {
			summaryAttr = fmt.Sprintf(` summary="%s"`, summary)
		}
		parts = append(parts, fmt.Sprintf(`<%s teammate_id="%s"%s%s>
%s
</%s>`, teammateMessageXMLTag, from, colorAttr, summaryAttr, text, teammateMessageXMLTag))
	}
	return strings.Join(parts, "\n\n")
}

// LogNormalizeAttachmentUnknownType mirrors TS logAntError for unknown attachment types; if nil and RABBIT_ATTACHMENT_UNKNOWN_LOG=1, uses log.Printf.
var LogNormalizeAttachmentUnknownType func(attachmentType string, attachment map[string]any)

// LogMCPResourceNoDisplayable mirrors TS logMCPDebug when an MCP resource has no displayable text/binary summary.
var LogMCPResourceNoDisplayable func(server, uri string)

func invokeLogMCPResourceNoDisplayable(server, uri string) {
	if LogMCPResourceNoDisplayable != nil {
		LogMCPResourceNoDisplayable(server, uri)
	}
	if os.Getenv("RABBIT_MCP_RESOURCE_DEBUG") == "1" {
		log.Printf("messages: mcp_resource no displayable content server=%q uri=%q", server, uri)
	}
}

// NormalizeAttachmentForAPI mirrors TS normalizeAttachmentForAPI(attachment).
func NormalizeAttachmentForAPI(attachment map[string]any) ([]TSMsg, error) {
	ty, _ := attachment["type"].(string)

	if os.Getenv("RABBIT_AGENT_SWARMS") == "1" {
		switch ty {
		case "teammate_mailbox":
			raw, _ := attachment["messages"].([]any)
			var body string
			if FormatTeammateMailboxMessagesForAPI != nil {
				body = FormatTeammateMailboxMessagesForAPI(raw)
			} else {
				body = DefaultFormatTeammateMailboxMessagesForAPI(raw)
			}
			body = strings.TrimSpace(body)
			if body == "" && len(raw) > 0 {
				// Malformed entries: surface raw JSON for debugging (TS formatTeammateMessages skips non-objects).
				body = fmt.Sprintf("Teammate mailbox (%d message(s); unparseable or empty fields):\n%s",
					len(raw), tsJSONString(raw))
			}
			if body == "" {
				body = "[teammate_mailbox] empty messages"
			}
			return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
				Content: body,
				IsMeta:  true,
			})}, nil
		case "team_context":
			team, _ := attachment["teamName"].(string)
			agent, _ := attachment["agentName"].(string)
			cfg, _ := attachment["teamConfigPath"].(string)
			tasks, _ := attachment["taskListPath"].(string)
			// TS: raw <system-reminder> in content; no wrapMessagesInSystemReminder.
			return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
				Content: fmt.Sprintf(`<system-reminder>
# Team Coordination

You are a teammate in team "%s".

**Your Identity:**
- Name: %s

**Team Resources:**
- Team config: %s
- Task list: %s

**Team Leader:** The team lead's name is "team-lead". Send updates and completion notifications to them.

Read the team config to discover your teammates' names. Check the task list periodically. Create new tasks when work should be divided. Mark tasks resolved when complete.

**IMPORTANT:** Always refer to teammates by their NAME (e.g., "team-lead", "analyzer", "researcher"), never by UUID. When messaging, use the name directly:

`+"```"+`json
{
  "to": "team-lead",
  "message": "Your message here",
  "summary": "Brief 5-10 word preview"
}
`+"```"+`
</system-reminder>`,
					team, agent, cfg, tasks),
				IsMeta: true,
			})}, nil
		}
	}

	if os.Getenv("RABBIT_EXPERIMENTAL_SKILL_SEARCH") == "1" && ty == "skill_discovery" {
		skills, _ := attachment["skills"].([]any)
		if len(skills) == 0 {
			return nil, nil
		}
		var lines []string
		for _, s := range skills {
			sm, _ := s.(map[string]any)
			name, _ := sm["name"].(string)
			desc, _ := sm["description"].(string)
			lines = append(lines, fmt.Sprintf("- %s: %s", name, desc))
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"Skills relevant to your task:\n\n%s\n\nThese skills encode project-specific conventions. Invoke via Skill(\"<name>\") for complete instructions.",
				strings.Join(lines, "\n"),
			),
			IsMeta: true,
		})}), nil
	}

	switch ty {
	case "directory":
		path, _ := attachment["path"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{
			metaToolUseMessage(ToolNameBash, map[string]any{"command": fmt.Sprintf("ls %q", path), "description": fmt.Sprintf("Lists files in %s", path)}),
			BashToolResultMetaMessage(ToolNameBash, attachment),
		}), nil

	case "edited_text_file":
		fn, _ := attachment["filename"].(string)
		snip, _ := attachment["snippet"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"Note: %s was modified, either by the user or by a linter. This change was intentional, so make sure to take it into account as you proceed (ie. don't revert it unless the user asks you to). Don't tell the user this, since they are already aware. Here are the relevant changes (shown with line numbers):\n%s",
				fn, snip,
			),
			IsMeta: true,
		})}), nil

	case "file":
		return fileAttachmentMessages(attachment)

	case "compact_file_reference":
		fn, _ := attachment["filename"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"Note: %s was read before the last conversation was summarized, but the contents are too large to include. Use %s tool if you need to access it.",
				fn, ToolNameRead,
			),
			IsMeta: true,
		})}), nil

	case "pdf_reference":
		fn, _ := attachment["filename"].(string)
		pages, _ := attachment["pageCount"].(float64)
		sizeF, _ := attachment["fileSize"].(float64)
		size := int(sizeF)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				`PDF file: %s (%.0f pages, %s). This PDF is too large to read all at once. You MUST use the %s tool with the pages parameter to read specific page ranges (e.g., pages: "1-5"). Do NOT call %s without the pages parameter or it will fail. Start by reading the first few pages to understand the structure, then read more as needed. Maximum 20 pages per request.`,
				fn, pages, formatFileSizeForAPI(size), ToolNameRead, ToolNameRead,
			),
			IsMeta: true,
		})}), nil

	case "selected_lines_in_ide":
		fn, _ := attachment["filename"].(string)
		ls, _ := attachment["lineStart"].(float64)
		le, _ := attachment["lineEnd"].(float64)
		cont, _ := attachment["content"].(string)
		if len(cont) > 2000 {
			cont = cont[:2000] + "\n... (truncated)"
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"The user selected the lines %.0f to %.0f from %s:\n%s\n\nThis may or may not be related to the current task.",
				ls, le, fn, cont,
			),
			IsMeta: true,
		})}), nil

	case "opened_file_in_ide":
		fn, _ := attachment["filename"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("The user opened the file %s in the IDE. This may or may not be related to the current task.", fn),
			IsMeta:  true,
		})}), nil

	case "plan_file_reference":
		pfp, _ := attachment["planFilePath"].(string)
		pc, _ := attachment["planContent"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"A plan file exists from plan mode at: %s\n\nPlan contents:\n\n%s\n\nIf this plan is relevant to the current work and not already complete, continue working on it.",
				pfp, pc,
			),
			IsMeta: true,
		})}), nil

	case "invoked_skills":
		skills, _ := attachment["skills"].([]any)
		if len(skills) == 0 {
			return nil, nil
		}
		var parts []string
		for _, s := range skills {
			sm, _ := s.(map[string]any)
			parts = append(parts, fmt.Sprintf("### Skill: %s\nPath: %s\n\n%s",
				strField(sm, "name"), strField(sm, "path"), strField(sm, "content")))
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "The following skills were invoked in this session. Continue to follow these guidelines:\n\n" + strings.Join(parts, "\n\n---\n\n"),
			IsMeta:  true,
		})}), nil

	case "todo_reminder":
		items, _ := attachment["content"].([]any)
		var lines []string
		for i, it := range items {
			tm, _ := it.(map[string]any)
			lines = append(lines, fmt.Sprintf("%d. [%s] %s", i+1, strField(tm, "status"), strField(tm, "content")))
		}
		msg := "The TodoWrite tool hasn't been used recently. If you're working on tasks that would benefit from tracking progress, consider using the TodoWrite tool to track progress. Also consider cleaning up the todo list if has become stale and no longer matches what you are working on. Only use it if it's relevant to the current work. This is just a gentle reminder - ignore if not applicable. Make sure that you NEVER mention this reminder to the user\n"
		if len(lines) > 0 {
			msg += "\n\nHere are the existing contents of your todo list:\n\n[" + strings.Join(lines, "\n") + "]"
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: msg, IsMeta: true})}), nil

	case "task_reminder":
		if os.Getenv("RABBIT_TODO_V2") != "1" {
			return nil, nil
		}
		items, _ := attachment["content"].([]any)
		var lines []string
		for _, it := range items {
			tm, _ := it.(map[string]any)
			lines = append(lines, fmt.Sprintf("#%s. [%s] %s", strField(tm, "id"), strField(tm, "status"), strField(tm, "subject")))
		}
		msg := fmt.Sprintf("The task tools haven't been used recently. If you're working on tasks that would benefit from tracking progress, consider using %s to add new tasks and %s to update task status (set to in_progress when starting, completed when done). Also consider cleaning up the task list if it has become stale. Only use these if relevant to the current work. This is just a gentle reminder - ignore if not applicable. Make sure that you NEVER mention this reminder to the user\n",
			ToolNameTaskCreate, ToolNameTaskUpdate)
		if len(lines) > 0 {
			msg += "\n\nHere are the existing tasks:\n\n" + strings.Join(lines, "\n")
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: msg, IsMeta: true})}), nil

	case "nested_memory":
		nc, _ := attachment["content"].(map[string]any)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("Contents of %s:\n\n%s", strField(nc, "path"), strField(nc, "content")),
			IsMeta:  true,
		})}), nil

	case "relevant_memories":
		memories, _ := attachment["memories"].([]any)
		var out []TSMsg
		for _, m := range memories {
			mm, _ := m.(map[string]any)
			header := strField(mm, "header")
			if header == "" {
				p := strField(mm, "path")
				header = MemoryHeader(p, int64FromAny(mm["mtimeMs"]))
			}
			out = append(out, CreateUserMessage(CreateUserMessageOpts{
				Content: header + "\n\n" + strField(mm, "content"),
				IsMeta:  true,
			}))
		}
		return WrapMessagesInSystemReminder(out), nil

	case "dynamic_skill":
		return nil, nil

	case "skill_listing":
		c, _ := attachment["content"].(string)
		if c == "" {
			return nil, nil
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "The following skills are available for use with the Skill tool:\n\n" + c,
			IsMeta:  true,
		})}), nil

	case "queued_command":
		var origin map[string]any
		if o, ok := attachment["origin"].(map[string]any); ok {
			origin = o
		} else if cm, _ := attachment["commandMode"].(string); cm == "task-notification" {
			origin = map[string]any{"kind": "task-notification"}
		}
		meta := len(origin) > 0 || truthy(attachment["isMeta"])
		opts := CreateUserMessageOpts{Origin: origin}
		if meta {
			opts.IsMeta = true
		}
		if u, ok := attachment["source_uuid"].(string); ok {
			opts.UUID = u
		}
		if p, ok := attachment["prompt"].([]any); ok {
			var texts []string
			var images []any
			for _, it := range p {
				b, ok := it.(map[string]any)
				if !ok {
					continue
				}
				if strField(b, "type") == "text" {
					texts = append(texts, strField(b, "text"))
				} else {
					images = append(images, b)
				}
			}
			textJoined := strings.Join(texts, "\n")
			blocks := []any{map[string]any{"type": "text", "text": WrapCommandText(textJoined, origin)}}
			blocks = append(blocks, images...)
			opts.Content = blocks
			return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(opts)}), nil
		}
		pr, _ := attachment["prompt"].(string)
		opts.Content = WrapCommandText(pr, origin)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(opts)}), nil

	case "output_style":
		style, _ := attachment["style"].(string)
		style = strings.TrimSpace(style)
		if style == "" || style == "default" {
			style = settingsFallbackOutputStyle()
			style = strings.TrimSpace(style)
		}
		if style == "" || style == "default" {
			return nil, nil
		}
		name := outputStyleDisplayName(style)
		if name == "" {
			return nil, nil
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("%s output style is active. Remember to follow the specific guidelines for this style.", name),
			IsMeta:  true,
		})}), nil

	case "diagnostics":
		files, _ := attachment["files"].([]any)
		if len(files) == 0 {
			return nil, nil
		}
		summary := FormatDiagnosticsSummary(files)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "<new-diagnostics>The following new diagnostic issues were detected:\n\n" + summary + "</new-diagnostics>",
			IsMeta:  true,
		})}), nil

	case "plan_mode":
		return planModeMessages(attachment), nil

	case "plan_mode_reentry":
		pfp, _ := attachment["planFilePath"].(string)
		content := fmt.Sprintf(`## Re-entering Plan Mode

You are returning to plan mode after having previously exited it. A plan file exists at %s from your previous planning session.

**Before proceeding with any new planning, you should:**
1. Read the existing plan file to understand what was previously planned
2. Evaluate the user's current request against that plan
3. Decide how to proceed:
   - **Different task**: If the user's request is for a different task—even if it's similar or related—start fresh by overwriting the existing plan
   - **Same task, continuing**: If this is explicitly a continuation or refinement of the exact same task, modify the existing plan while cleaning up outdated or irrelevant sections
4. Continue on with the plan process and most importantly you should always edit the plan file one way or the other before calling %s

Treat this as a fresh planning session. Do not assume the existing plan is relevant without evaluating it first.`, pfp, ToolNameExitPlanModeV2)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})}), nil

	case "plan_mode_exit":
		exists, _ := attachment["planExists"].(bool)
		pfp, _ := attachment["planFilePath"].(string)
		ref := ""
		if exists {
			ref = fmt.Sprintf(" The plan file is located at %s if you need to reference it.", pfp)
		}
		content := "## Exited Plan Mode\n\nYou have exited plan mode. You can now make edits, run tools, and take actions." + ref
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})}), nil

	case "auto_mode":
		return autoModeMessages(attachment), nil

	case "auto_mode_exit":
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "## Exited Auto Mode\n\nYou have exited auto mode. The user may now want to interact more directly. You should ask clarifying questions when the approach is ambiguous rather than making assumptions.",
			IsMeta:  true,
		})}), nil

	case "critical_system_reminder":
		c, _ := attachment["content"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: c, IsMeta: true})}), nil

	case "mcp_resource":
		srv, _ := attachment["server"].(string)
		uri, _ := attachment["uri"].(string)
		cont, _ := attachment["content"].(map[string]any)
		var blocks []any
		arr, hasContents := cont["contents"].([]any)
		if cont == nil || !hasContents || len(arr) == 0 {
			return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
				Content: fmt.Sprintf(`<mcp-resource server=%q uri=%q>(No content)</mcp-resource>`, srv, uri),
				IsMeta:  true,
			})}), nil
		}
		for _, item := range arr {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if tx, ok := im["text"].(string); ok {
				blocks = append(blocks, map[string]any{"type": "text", "text": "Full contents of resource:"})
				blocks = append(blocks, map[string]any{"type": "text", "text": tx})
				blocks = append(blocks, map[string]any{"type": "text", "text": "Do NOT read this resource again unless you think it may have changed, since you already have the full contents."})
			} else if _, ok := im["blob"]; ok {
				mt := "application/octet-stream"
				if m, ok := im["mimeType"].(string); ok {
					mt = m
				}
				blocks = append(blocks, map[string]any{"type": "text", "text": "[Binary content: " + mt + "]"})
			}
		}
		if len(blocks) > 0 {
			return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: blocks, IsMeta: true})}), nil
		}
		invokeLogMCPResourceNoDisplayable(srv, uri)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(`<mcp-resource server=%q uri=%q>(No displayable content)</mcp-resource>`, srv, uri),
			IsMeta:  true,
		})}), nil

	case "agent_mention":
		at, _ := attachment["agentType"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(`The user has expressed a desire to invoke the agent %q. Please invoke the agent appropriately, passing in the required context to it. `, at),
			IsMeta:  true,
		})}), nil

	case "task_status":
		return taskStatusMessages(attachment), nil

	case "async_hook_response":
		resp, _ := attachment["response"].(map[string]any)
		var msgs []TSMsg
		if sm, _ := resp["systemMessage"].(string); sm != "" {
			msgs = append(msgs, CreateUserMessage(CreateUserMessageOpts{Content: sm, IsMeta: true}))
		}
		if hso, ok := resp["hookSpecificOutput"].(map[string]any); ok {
			if ac, ok := hso["additionalContext"].(string); ok && ac != "" {
				msgs = append(msgs, CreateUserMessage(CreateUserMessageOpts{Content: ac, IsMeta: true}))
			}
		}
		return WrapMessagesInSystemReminder(msgs), nil

	case "token_usage":
		used, _ := attachment["used"].(float64)
		total, _ := attachment["total"].(float64)
		rem, _ := attachment["remaining"].(float64)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("Token usage: %s/%s; %s remaining",
				formatAttachmentNumberForTemplate(used), formatAttachmentNumberForTemplate(total), formatAttachmentNumberForTemplate(rem))),
			IsMeta: true,
		})}, nil

	case "budget_usd":
		used, _ := attachment["used"].(float64)
		total, _ := attachment["total"].(float64)
		rem, _ := attachment["remaining"].(float64)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("USD budget: $%s/$%s; $%s remaining",
				formatAttachmentUSDNumber(used), formatAttachmentUSDNumber(total), formatAttachmentUSDNumber(rem))),
			IsMeta: true,
		})}, nil

	case "output_token_usage":
		turn, _ := attachment["turn"].(float64)
		sess, _ := attachment["session"].(float64)
		turnText := formatNumberCompact(turn)
		if b, ok := attachmentNonNilFloat64(attachment, "budget"); ok {
			turnText = formatNumberCompact(turn) + " / " + formatNumberCompact(b)
		}
		// TS: \u2014 and \u00b7
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("Output tokens \u2014 turn: %s \u00b7 session: %s", turnText, formatNumberCompact(sess))),
			IsMeta:  true,
		})}, nil

	case "hook_blocking_error":
		hn, _ := attachment["hookName"].(string)
		be, _ := attachment["blockingError"].(map[string]any)
		cmd, _ := be["command"].(string)
		err, _ := be["blockingError"].(string)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf(`%s hook blocking error from command: "%s": %s`, hn, cmd, err)),
			IsMeta:  true,
		})}, nil

	case "hook_success":
		he, _ := attachment["hookEvent"].(string)
		if he != "SessionStart" && he != "UserPromptSubmit" {
			return nil, nil
		}
		c, _ := attachment["content"].(string)
		if c == "" {
			return nil, nil
		}
		hn, _ := attachment["hookName"].(string)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("%s hook success: %s", hn, c)),
			IsMeta:  true,
		})}, nil

	case "hook_additional_context":
		lines, _ := attachment["content"].([]any)
		if len(lines) == 0 {
			return nil, nil
		}
		var ss []string
		for _, l := range lines {
			ss = append(ss, fmt.Sprint(l))
		}
		hn, _ := attachment["hookName"].(string)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("%s hook additional context: %s", hn, strings.Join(ss, "\n"))),
			IsMeta:  true,
		})}, nil

	case "hook_stopped_continuation":
		hn, _ := attachment["hookName"].(string)
		m, _ := attachment["message"].(string)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: WrapInSystemReminder(fmt.Sprintf("%s hook stopped continuation: %s", hn, m)),
			IsMeta:  true,
		})}, nil

	case "compaction_reminder":
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "Auto-compact is enabled. When the context window is nearly full, older messages will be automatically summarized so you can continue working seamlessly. There is no need to stop or rush \u2014 you have unlimited context through automatic compaction.",
			IsMeta:  true,
		})}), nil

	case "context_efficiency":
		if os.Getenv("RABBIT_HISTORY_SNIP") == "1" && IsSnipRuntimeEnabled() {
			return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
				Content: SnipNudgeText(),
				IsMeta:  true,
			})}), nil
		}
		return nil, nil

	case "date_change":
		d, _ := attachment["newDate"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("The date has changed. Today's date is now %s. DO NOT mention this to the user explicitly because they are already aware.", d),
			IsMeta:  true,
		})}), nil

	case "ultrathink_effort":
		lv, _ := attachment["level"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf("The user has requested reasoning effort level: %s. Apply this to the current turn.", lv),
			IsMeta:  true,
		})}), nil

	case "deferred_tools_delta":
		added, _ := attachment["addedLines"].([]any)
		removed, _ := attachment["removedNames"].([]any)
		var parts []string
		if len(added) > 0 {
			var al []string
			for _, x := range added {
				al = append(al, fmt.Sprint(x))
			}
			parts = append(parts, "The following deferred tools are now available via ToolSearch:\n"+strings.Join(al, "\n"))
		}
		if len(removed) > 0 {
			var rl []string
			for _, x := range removed {
				rl = append(rl, fmt.Sprint(x))
			}
			parts = append(parts, "The following deferred tools are no longer available (their MCP server disconnected). Do not search for them — ToolSearch will return no match:\n"+strings.Join(rl, "\n"))
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: strings.Join(parts, "\n\n"),
			IsMeta:  true,
		})}), nil

	case "agent_listing_delta":
		added, _ := attachment["addedLines"].([]any)
		removed, _ := attachment["removedTypes"].([]any)
		initial, _ := attachment["isInitial"].(bool)
		showConc, _ := attachment["showConcurrencyNote"].(bool)
		var parts []string
		if len(added) > 0 {
			var al []string
			for _, x := range added {
				al = append(al, fmt.Sprint(x))
			}
			hdr := "New agent types are now available for the Agent tool:"
			if initial {
				hdr = "Available agent types for the Agent tool:"
			}
			parts = append(parts, hdr+"\n"+strings.Join(al, "\n"))
		}
		if len(removed) > 0 {
			var rl []string
			for _, x := range removed {
				rl = append(rl, "- "+fmt.Sprint(x))
			}
			parts = append(parts, "The following agent types are no longer available:\n"+strings.Join(rl, "\n"))
		}
		if initial && showConc {
			parts = append(parts, "Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses.")
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: strings.Join(parts, "\n\n"),
			IsMeta:  true,
		})}), nil

	case "mcp_instructions_delta":
		added, _ := attachment["addedBlocks"].([]any)
		removed, _ := attachment["removedNames"].([]any)
		var parts []string
		if len(added) > 0 {
			var ab []string
			for _, x := range added {
				ab = append(ab, fmt.Sprint(x))
			}
			parts = append(parts, "# MCP Server Instructions\n\nThe following MCP servers have provided instructions for how to use their tools and resources:\n\n"+strings.Join(ab, "\n\n"))
		}
		if len(removed) > 0 {
			var rn []string
			for _, x := range removed {
				rn = append(rn, fmt.Sprint(x))
			}
			parts = append(parts, "The following MCP servers have disconnected. Their instructions above no longer apply:\n"+strings.Join(rn, "\n"))
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: strings.Join(parts, "\n\n"),
			IsMeta:  true,
		})}), nil

	case "companion_intro":
		name, _ := attachment["name"].(string)
		species, _ := attachment["species"].(string)
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: CompanionIntroText(name, species),
			IsMeta:  true,
		})}), nil

	case "verify_plan_reminder":
		toolName := ""
		if os.Getenv("CLAUDE_CODE_VERIFY_PLAN") == "true" {
			toolName = "VerifyPlanExecution"
		}
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(`You have completed implementing the plan. Please call the %q tool directly (NOT the %s tool or an agent) to verify that all plan items were completed correctly.`, toolName, ToolNameAgent),
			IsMeta:  true,
		})}), nil

	case "already_read_file", "command_permissions", "edited_image_file", "hook_cancelled",
		"hook_error_during_execution", "hook_non_blocking_error", "hook_system_message",
		"structured_output", "hook_permission_decision":
		return nil, nil
	}

	legacy := map[string]struct{}{
		"autocheckpointing": {}, "background_task_status": {}, "todo": {}, "task_progress": {}, "ultramemory": {},
		// TS comments / DCE-gated types that may still appear in old transcripts
		"max_turns_reached": {}, "bagel_console": {},
	}
	if _, ok := legacy[ty]; ok {
		return nil, nil
	}

	logNormalizeAttachmentUnknown(ty, attachment)
	return nil, nil
}

func logNormalizeAttachmentUnknown(ty string, attachment map[string]any) {
	if LogNormalizeAttachmentUnknownType != nil {
		LogNormalizeAttachmentUnknownType(ty, attachment)
		return
	}
	if os.Getenv("RABBIT_ANT_UNKNOWN_ATTACHMENT") == "1" {
		log.Printf("messages: normalizeAttachmentForAPI unknown attachment type %q", ty)
		return
	}
	if os.Getenv("RABBIT_ATTACHMENT_UNKNOWN_LOG") == "1" {
		log.Printf("messages: normalizeAttachmentForAPI unknown attachment type %q", ty)
	}
}

// fileNested returns fc["file"] when present (FileReadTool output shape in TS).
func fileNested(fc map[string]any) (map[string]any, bool) {
	f, ok := fc["file"].(map[string]any)
	return f, ok
}

// fileReadTextBody mirrors TS text branch: prefer file.content, then legacy top-level text.
func fileReadTextBody(fc map[string]any) string {
	if f, ok := fileNested(fc); ok {
		if c, ok := f["content"].(string); ok && c != "" {
			return c
		}
	}
	if t, ok := fc["text"].(string); ok {
		return t
	}
	return ""
}

// fileAttachmentMessages mirrors TS normalizeAttachmentForAPI 'file' (image / text / notebook / pdf / parts / file_unchanged).
func fileAttachmentMessages(attachment map[string]any) ([]TSMsg, error) {
	fn, _ := attachment["filename"].(string)
	fc, ok := attachment["content"].(map[string]any)
	if !ok {
		return nil, nil
	}
	ft, _ := fc["type"].(string)
	truncated, _ := attachment["truncated"].(bool)

	msgs := []TSMsg{
		metaToolUseMessage(ToolNameRead, map[string]any{"file_path": fn}),
	}
	switch ft {
	case "image":
		imgBlock := fileReadImageBlock(fc)
		msgs = append(msgs, CreateUserMessage(CreateUserMessageOpts{
			IsMeta:  true,
			Content: []any{imgBlock},
		}))
	case "text":
		txt := fileReadTextToolResultString(fc)
		msgs = append(msgs, metaToolResultMessage(ToolNameRead, txt))
	case "notebook":
		blocks := notebookMapCellsToToolResultBlocksOpts(fc, NotebookCellsOpts{
			Filename:            fn,
			IncludeLargeOutputs: false,
		})
		msgs = append(msgs, notebookReadToolResultMessage(ToolNameRead, blocks))
	case "pdf":
		body := fileReadPDFResultBody(fc)
		msgs = append(msgs, metaToolResultMessage(ToolNameRead, body))
	case "parts":
		body := fileReadPartsResultBody(fc)
		msgs = append(msgs, metaToolResultMessage(ToolNameRead, body))
	case "file_unchanged":
		// TS FileReadTool FILE_UNCHANGED_STUB
		msgs = append(msgs, metaToolResultMessage(ToolNameRead,
			"File unchanged since last read. The content from the earlier Read tool_result in this conversation is still current — refer to that instead of re-reading."))
	default:
		msgs = append(msgs, metaToolResultMessage(ToolNameRead, tsJSONString(fc)))
	}
	msgs = WrapMessagesInSystemReminder(msgs)
	if ft == "text" && truncated {
		msgs = append(msgs, CreateUserMessage(CreateUserMessageOpts{
			Content: fmt.Sprintf(
				"Note: The file %s was too large and has been truncated to the first %d lines. Don't tell the user about this truncation. Use %s to read more of the file if you need.",
				fn, MaxLinesToRead, ToolNameRead,
			),
			IsMeta: true,
		}))
	}
	return msgs, nil
}

func fileReadImageBlock(fc map[string]any) map[string]any {
	// TS mapToolResult image: source base64 + media_type from file.base64 / file.type
	if f, ok := fileNested(fc); ok {
		b64, ok := f["base64"].(string)
		if ok && b64 != "" {
			mt, _ := f["type"].(string)
			if mt == "" {
				mt = "image/jpeg"
			}
			return map[string]any{
				"type": "image",
				"source": map[string]any{
					"type":       "base64",
					"data":       b64,
					"media_type": mt,
				},
			}
		}
	}
	imgBlock := map[string]any{"type": "image"}
	if src, ok := fc["source"].(map[string]any); ok {
		imgBlock["source"] = src
		return imgBlock
	}
	for k, v := range fc {
		if k != "type" {
			imgBlock[k] = v
		}
	}
	return imgBlock
}

// NotebookCellsOpts mirrors TS readNotebook(includeLargeOutputs) and path for the jq hint.
type NotebookCellsOpts struct {
	Filename            string
	IncludeLargeOutputs bool
}

// notebookMapCellsToToolResultBlocks mirrors TS mapNotebookCellsToToolResult (utils/notebook.ts):
// flatten cells → text/image blocks, merge adjacent text blocks. Default: include large outputs (tests / single-cell).
func notebookMapCellsToToolResultBlocks(fc map[string]any) []map[string]any {
	return notebookMapCellsToToolResultBlocksOpts(fc, NotebookCellsOpts{IncludeLargeOutputs: true})
}

func notebookMapCellsToToolResultBlocksOpts(fc map[string]any, opts NotebookCellsOpts) []map[string]any {
	cells := notebookExtractCells(fc)
	if len(cells) == 0 {
		return []map[string]any{}
	}
	var flat []map[string]any
	for i, raw := range cells {
		cm, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		flat = append(flat, notebookCellToBlocksOpts(cm, i, opts)...)
	}
	return mergeAdjacentNotebookTextBlocks(flat)
}

func notebookExtractCells(fc map[string]any) []any {
	if f, ok := fileNested(fc); ok {
		if cells, ok := f["cells"].([]any); ok {
			return cells
		}
	}
	if cells, ok := fc["cells"].([]any); ok {
		return cells
	}
	return nil
}

func notebookSourceString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	arr, ok := v.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, it := range arr {
		if s, ok := it.(string); ok {
			b.WriteString(s)
		} else {
			b.WriteString(fmt.Sprint(it))
		}
	}
	return b.String()
}

func notebookCellContentBlock(cellType, source, cellID, language string) map[string]any {
	var meta []string
	if cellType != "code" {
		meta = append(meta, fmt.Sprintf("<cell_type>%s</cell_type>", cellType))
	}
	if cellType == "code" && language != "" && language != "python" {
		meta = append(meta, fmt.Sprintf("<language>%s</language>", language))
	}
	inner := strings.Join(meta, "") + source
	text := fmt.Sprintf(`<cell id="%s">%s</cell id="%s">`, cellID, inner, cellID)
	return map[string]any{"type": "text", "text": text}
}

func notebookTruncateApproxOutputText(s string) string {
	maxU := notebookCellOutputTruncateBytes
	if jsStringUTF16Len(s) <= maxU {
		return s
	}
	prefix, suffix := truncateJSStringToMaxUTF16(s, maxU)
	extraLines := strings.Count(suffix, "\n") + 1
	return prefix + fmt.Sprintf("\n\n... [%d lines truncated] ...", extraLines)
}

func notebookJupyterWhitespaceStrip(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
}

// notebookExtractImageFromJupyterData mirrors TS extractImage (display_data / execute_result).
func notebookExtractImageFromJupyterData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	if s, ok := data["image/png"].(string); ok && strings.TrimSpace(s) != "" {
		return map[string]any{
			"image_data": notebookJupyterWhitespaceStrip(s),
			"media_type": "image/png",
		}
	}
	if s, ok := data["image/jpeg"].(string); ok && strings.TrimSpace(s) != "" {
		return map[string]any{
			"image_data": notebookJupyterWhitespaceStrip(s),
			"media_type": "image/jpeg",
		}
	}
	return nil
}

func notebookJoinOutputText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []any:
		var b strings.Builder
		for _, it := range x {
			if s, ok := it.(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	default:
		return fmt.Sprint(x)
	}
}

// notebookFormatOutputTruncated mirrors TS formatOutput (BashTool utils) for non-image text.
func notebookFormatOutputTruncated(content string) string {
	s := strings.TrimSpace(content)
	if bashDataURIRe.MatchString(s) {
		return content
	}
	maxLen := bashMaxOutputLength()
	if jsStringUTF16Len(content) <= maxLen {
		return content
	}
	truncatedPart, rest := truncateJSStringToMaxUTF16(content, maxLen)
	remainingLines := strings.Count(rest, "\n") + 1
	return fmt.Sprintf("%s\n\n... [%d lines truncated] ...", truncatedPart, remainingLines)
}

// notebookProcessRawOutput mirrors TS processOutput for raw Jupyter cell.outputs[].
func notebookProcessRawOutput(om map[string]any) map[string]any {
	ot, _ := om["output_type"].(string)
	switch ot {
	case "stream":
		return map[string]any{
			"output_type": ot,
			"text":        notebookFormatOutputTruncated(notebookJoinOutputText(om["text"])),
		}
	case "execute_result", "display_data":
		data, _ := om["data"].(map[string]any)
		text := ""
		if data != nil {
			text = notebookFormatOutputTruncated(notebookJoinOutputText(data["text/plain"]))
		}
		out := map[string]any{"output_type": ot, "text": text}
		if img := notebookExtractImageFromJupyterData(data); img != nil {
			out["image"] = img
		}
		return out
	case "error":
		ename, _ := om["ename"].(string)
		evalue, _ := om["evalue"].(string)
		var lines []string
		if arr, ok := om["traceback"].([]any); ok {
			for _, line := range arr {
				if s, ok := line.(string); ok {
					lines = append(lines, s)
				}
			}
		}
		raw := fmt.Sprintf("%s: %s\n%s", ename, evalue, strings.Join(lines, "\n"))
		return map[string]any{"output_type": ot, "text": notebookFormatOutputTruncated(raw)}
	default:
		return om
	}
}

func notebookIsRawJupyterOutput(om map[string]any) bool {
	_, ok := om["output_type"].(string)
	return ok
}

func notebookOutputBlocks(om map[string]any) []map[string]any {
	var blocks []map[string]any
	if t := strField(om, "text"); t != "" {
		t = notebookTruncateApproxOutputText(t)
		blocks = append(blocks, map[string]any{"type": "text", "text": "\n" + t})
	}
	if im, ok := om["image"].(map[string]any); ok {
		data := strField(im, "image_data")
		mt := strField(im, "media_type")
		if data != "" {
			if mt == "" {
				mt = "image/png"
			}
			blocks = append(blocks, map[string]any{
				"type": "image",
				"source": map[string]any{
					"type":       "base64",
					"data":       data,
					"media_type": mt,
				},
			})
		}
	}
	return blocks
}

func notebookCellToBlocks(m map[string]any, index int) []map[string]any {
	return notebookCellToBlocksOpts(m, index, NotebookCellsOpts{IncludeLargeOutputs: true})
}

func notebookProcessedOutputsTooLarge(outputs []map[string]any) bool {
	total := 0
	for _, o := range outputs {
		total += jsStringUTF16Len(strField(o, "text"))
		if im, ok := o["image"].(map[string]any); ok {
			total += jsStringUTF16Len(strField(im, "image_data"))
		}
		if total > notebookCellOutputTruncateBytes {
			return true
		}
	}
	return false
}

func notebookLargeCellOutputHint(notebookPath string, cellIndex int) string {
	if strings.TrimSpace(notebookPath) == "" {
		return fmt.Sprintf("Outputs are too large to include. Use %s with: cat <notebook_path> | jq '.cells[%d].outputs'", ToolNameBash, cellIndex)
	}
	return fmt.Sprintf("Outputs are too large to include. Use %s with: cat %q | jq '.cells[%d].outputs'", ToolNameBash, notebookPath, cellIndex)
}

func notebookCellToBlocksOpts(m map[string]any, index int, opts NotebookCellsOpts) []map[string]any {
	cellType := strField(m, "cellType")
	if cellType == "" {
		cellType = strField(m, "cell_type")
	}
	source := notebookSourceString(m["source"])
	cellID := strField(m, "cell_id")
	if cellID == "" {
		cellID = strField(m, "id")
	}
	if cellID == "" {
		cellID = fmt.Sprintf("cell-%d", index)
	}
	language := strField(m, "language")

	out := []map[string]any{notebookCellContentBlock(cellType, source, cellID, language)}
	if arr, ok := m["outputs"].([]any); ok {
		var processed []map[string]any
		for _, o := range arr {
			om, ok := o.(map[string]any)
			if !ok {
				continue
			}
			if notebookIsRawJupyterOutput(om) {
				om = notebookProcessRawOutput(om)
			}
			processed = append(processed, om)
		}
		if !opts.IncludeLargeOutputs && notebookProcessedOutputsTooLarge(processed) {
			single := map[string]any{
				"output_type": "stream",
				"text":        notebookLargeCellOutputHint(opts.Filename, index),
			}
			out = append(out, notebookOutputBlocks(single)...)
		} else {
			for _, om := range processed {
				out = append(out, notebookOutputBlocks(om)...)
			}
		}
	}
	return out
}

func mergeAdjacentNotebookTextBlocks(blocks []map[string]any) []map[string]any {
	if len(blocks) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(blocks))
	for _, curr := range blocks {
		ct, _ := curr["type"].(string)
		if ct == "text" && len(out) > 0 {
			prev := out[len(out)-1]
			if pt, _ := prev["type"].(string); pt == "text" {
				ptPrev, _ := prev["text"].(string)
				ptCurr, _ := curr["text"].(string)
				prev["text"] = ptPrev + "\n" + ptCurr
				continue
			}
		}
		out = append(out, curr)
	}
	return out
}

// notebookReadToolResultMessage mirrors TS createToolResultMessage(FileReadTool, notebookOutput):
// with any image block, content is the block array only; otherwise "Result of calling …" + jsonStringify(blocks).
func notebookReadToolResultMessage(toolName string, blocks []map[string]any) TSMsg {
	hasImage := false
	for _, b := range blocks {
		if strField(b, "type") == "image" {
			hasImage = true
			break
		}
	}
	if hasImage {
		arr := make([]any, len(blocks))
		for i := range blocks {
			arr[i] = blocks[i]
		}
		return CreateUserMessage(CreateUserMessageOpts{IsMeta: true, Content: arr})
	}
	body := fmt.Sprintf("Result of calling the %s tool:\n%s", toolName, tsJSONString(blocks))
	return CreateUserMessage(CreateUserMessageOpts{IsMeta: true, Content: body})
}

func fileReadPDFResultBody(fc map[string]any) string {
	// TS: content string `PDF file read: path (size)`
	if f, ok := fileNested(fc); ok {
		path, _ := f["filePath"].(string)
		if path == "" {
			path, _ = f["file_path"].(string)
		}
		sz := intFromAny(f["originalSize"])
		if path != "" && sz > 0 {
			return fmt.Sprintf("PDF file read: %s (%s)", path, formatFileSizeForAPI(sz))
		}
	}
	return tsJSONString(fc)
}

func fileReadPartsResultBody(fc map[string]any) string {
	if f, ok := fileNested(fc); ok {
		cnt := intFromAny(f["count"])
		path, _ := f["filePath"].(string)
		if path == "" {
			path, _ = f["file_path"].(string)
		}
		sz := intFromAny(f["originalSize"])
		if cnt > 0 && path != "" {
			// TS FileReadTool mapToolResult: always "page(s)"
			return fmt.Sprintf("PDF pages extracted: %d page(s) from %s (%s)", cnt, path, formatFileSizeForAPI(sz))
		}
	}
	return tsJSONString(fc)
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func metaToolUseMessage(toolName string, input map[string]any) TSMsg {
	return CreateUserMessage(CreateUserMessageOpts{
		Content: fmt.Sprintf("Called the %s tool with the following input: %s", toolName, tsJSONString(input)),
		IsMeta:  true,
	})
}

func metaToolResultMessage(toolName, result string) (msg TSMsg) {
	defer func() {
		if recover() != nil {
			msg = CreateUserMessage(CreateUserMessageOpts{
				Content: fmt.Sprintf("Result of calling the %s tool: Error", toolName),
				IsMeta:  true,
			})
		}
	}()
	return CreateUserMessage(CreateUserMessageOpts{
		Content: fmt.Sprintf("Result of calling the %s tool:\n%s", toolName, result),
		IsMeta:  true,
	})
}

func outputStyleName(style string) string {
	switch style {
	case "Explanatory":
		return "Explanatory"
	case "Learning":
		return "Learning"
	default:
		return ""
	}
}

// ExtraOutputStyleNames maps attachment style keys to display names (TS plugin/custom output styles).
var ExtraOutputStyleNames map[string]string

func outputStyleDisplayName(style string) string {
	if n, ok := outputStyleNameFromEnv(style); ok {
		return n
	}
	if n, ok := outputStyleNameFromConfigFile(style); ok {
		return n
	}
	if n, ok := outputStyleNameFromScanDirs(style); ok {
		return n
	}
	if n, ok := outputStyleNameFromPlugins(style); ok {
		return n
	}
	if ExtraOutputStyleNames != nil {
		if n, ok := ExtraOutputStyleNames[style]; ok && strings.TrimSpace(n) != "" {
			return n
		}
	}
	return outputStyleName(style)
}

func planModeMessages(att map[string]any) []TSMsg {
	if truthy(att["isSubAgent"]) {
		return planModeSubAgent(att)
	}
	if rt, _ := att["reminderType"].(string); rt == "sparse" {
		return planModeSparse(att)
	}
	if os.Getenv("RABBIT_PLAN_MODE_INTERVIEW") == "1" {
		return planModeInterview(att)
	}
	return planModeV2Default(att)
}

func planModeSparse(att map[string]any) []TSMsg {
	pfp, _ := att["planFilePath"].(string)
	workflow := "Follow 5-phase workflow."
	if os.Getenv("RABBIT_PLAN_MODE_INTERVIEW") == "1" {
		workflow = "Follow iterative workflow: explore codebase, interview user, write to plan incrementally."
	}
	content := fmt.Sprintf(
		"Plan mode still active (see full instructions earlier in conversation). Read-only except plan file (%s). %s End turns with %s (for clarifications) or %s (for plan approval). Never ask about plan approval via text or AskUserQuestion.",
		pfp, workflow, ToolNameAskUserQuestion, ToolNameExitPlanModeV2,
	)
	return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})})
}

func planModeSubAgent(att map[string]any) []TSMsg {
	pfp, _ := att["planFilePath"].(string)
	exists, _ := att["planExists"].(bool)
	planInfo := fmt.Sprintf("No plan file exists yet. You should create your plan at %s using the %s tool if you need to.", pfp, ToolNameWrite)
	if exists {
		planInfo = fmt.Sprintf("A plan file already exists at %s. You can read it and make incremental edits using the %s tool if you need to.", pfp, ToolNameEdit)
	}
	content := fmt.Sprintf(`Plan mode is active. The user indicated that they do not want you to execute yet -- you MUST NOT make any edits, run any non-readonly tools (including changing configs or making commits), or otherwise make any changes to the system. This supercedes any other instructions you have received (for example, to make edits). Instead, you should:

## Plan File Info:
%s
You should build your plan incrementally by writing to or editing this file. NOTE that this is the only file you are allowed to edit - other than this you are only allowed to take READ-ONLY actions.
Answer the user's query comprehensively, using the %s tool if you need to ask the user clarifying questions.`,
		planInfo, ToolNameAskUserQuestion)
	return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})})
}

func planModeInterview(att map[string]any) []TSMsg {
	pfp, _ := att["planFilePath"].(string)
	exists, _ := att["planExists"].(bool)
	planInfo := fmt.Sprintf("No plan file exists yet. You should create your plan at %s using the %s tool.", pfp, ToolNameWrite)
	if exists {
		planInfo = fmt.Sprintf("A plan file already exists at %s. You can read it and make incremental edits using the %s tool.", pfp, ToolNameEdit)
	}
	content := fmt.Sprintf(`Plan mode is active. The user indicated that they do not want you to execute yet -- you MUST NOT make any edits (with the exception of the plan file mentioned below), run any non-readonly tools (including changing configs or making commits), or otherwise make any changes to the system. This supercedes any other instructions you have received.

## Plan File Info:
%s

## Iterative Planning Workflow

You are pair-planning with the user. Explore the code to build context, ask the user questions when you hit decisions you can't make alone, and write your findings into the plan file as you go. The plan file (above) is the ONLY file you may edit — it starts as a rough skeleton and gradually becomes the final plan.

### The Loop

Repeat this cycle until the plan is complete:

1. **Explore** — Use %s, %s, %s to read code.
2. **Update the plan file** — After each discovery, immediately capture what you learned.
3. **Ask the user** — When you hit an ambiguity, use %s.

### When to Converge

Your plan is ready when you've addressed all ambiguities. Call %s when the plan is ready for approval.`,
		planInfo, ToolNameRead, ToolNameGlob, ToolNameGrep, ToolNameAskUserQuestion, ToolNameExitPlanModeV2)
	return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})})
}

func planModeV2Default(att map[string]any) []TSMsg {
	pfp, _ := att["planFilePath"].(string)
	exists, _ := att["planExists"].(bool)
	planInfo := fmt.Sprintf("No plan file exists yet. You should create your plan at %s using the %s tool.", pfp, ToolNameWrite)
	if exists {
		planInfo = fmt.Sprintf("A plan file already exists at %s. You can read it and make incremental edits using the %s tool.", pfp, ToolNameEdit)
	}
	explore := "Explore"
	plan := "Plan"
	exploreN := 3
	agentN := 1
	content := fmt.Sprintf(`Plan mode is active. The user indicated that they do not want you to execute yet -- you MUST NOT make any edits (with the exception of the plan file mentioned below), run any non-readonly tools (including changing configs or making commits), or otherwise make any changes to the system. This supercedes any other instructions you have received.

## Plan File Info:
%s
You should build your plan incrementally by writing to or editing this file. NOTE that this is the only file you are allowed to edit - other than this you are only allowed to take READ-ONLY actions.

## Plan Workflow

### Phase 1: Initial Understanding
Goal: Gain a comprehensive understanding of the user's request. Critical: in this phase you should only use the %s subagent type.

2. **Launch up to %d %s agents IN PARALLEL** (single message, multiple tool calls) to efficiently explore the codebase.

### Phase 2: Design
Launch %s agent(s) to design the implementation. You can launch up to %d agent(s) in parallel.

### Phase 3: Review
Use %s to clarify any remaining questions with the user.

%s

### Phase 5: Call %s
At the very end of your turn, call %s to indicate you are done planning. Your turn should only end with either %s or %s.`,
		planInfo, explore, exploreN, explore, plan, agentN, ToolNameAskUserQuestion, PLAN_PHASE4_CONTROL, ToolNameExitPlanModeV2, ToolNameExitPlanModeV2, ToolNameAskUserQuestion, ToolNameExitPlanModeV2)
	return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: content, IsMeta: true})})
}

func autoModeMessages(att map[string]any) []TSMsg {
	rt, _ := att["reminderType"].(string)
	if rt == "sparse" {
		return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{
			Content: "Auto mode still active (see full instructions earlier in conversation). Execute autonomously, minimize interruptions, prefer action over planning.",
			IsMeta:  true,
		})})
	}
	full := `## Auto Mode Active

Auto mode is active. The user chose continuous, autonomous execution. You should:

1. **Execute immediately** — Start implementing right away. Make reasonable assumptions and proceed on low-risk work.
2. **Minimize interruptions** — Prefer making reasonable assumptions over asking questions for routine decisions.
3. **Prefer action over planning** — Do not enter plan mode unless the user explicitly asks. When in doubt, start coding.
4. **Expect course corrections** — The user may provide suggestions or course corrections at any point; treat those as normal input.
5. **Do not take overly destructive actions** — Auto mode is not a license to destroy. Anything that deletes data or modifies shared or production systems still needs explicit user confirmation.
6. **Avoid data exfiltration** — Post even routine messages to chat platforms or work tickets only if the user has directed you to.`
	return WrapMessagesInSystemReminder([]TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: full, IsMeta: true})})
}

func taskStatusMessages(att map[string]any) []TSMsg {
	st, _ := att["status"].(string)
	desc, _ := att["description"].(string)
	tid, _ := att["taskId"].(string)
	tt, _ := att["taskType"].(string)
	delta, _ := att["deltaSummary"].(string)
	outPath, _ := att["outputFilePath"].(string)

	if st == "killed" {
		s := fmt.Sprintf(`Task %q (%s) was stopped by the user.`, desc, tid)
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: WrapInSystemReminder(s), IsMeta: true})}
	}
	if st == "running" {
		parts := []string{fmt.Sprintf(`Background agent %q (%s) is still running.`, desc, tid)}
		if delta != "" {
			parts = append(parts, "Progress: "+delta)
		}
		if outPath != "" {
			parts = append(parts, fmt.Sprintf("Do NOT spawn a duplicate. You will be notified when it completes. You can read partial output at %s or send it a message with %s.", outPath, ToolNameSendMessage))
		} else {
			parts = append(parts, fmt.Sprintf("Do NOT spawn a duplicate. You will be notified when it completes. You can check its progress with the %s tool or send it a message with %s.", ToolNameTaskOutput, ToolNameSendMessage))
		}
		return []TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: WrapInSystemReminder(strings.Join(parts, " ")), IsMeta: true})}
	}
	display := st
	parts := []string{fmt.Sprintf("Task %s (type: %s) (status: %s) (description: %s)", tid, tt, display, desc)}
	if delta != "" {
		parts = append(parts, "Delta: "+delta)
	}
	if outPath != "" {
		parts = append(parts, "Read the output file to retrieve the result: "+outPath)
	} else {
		parts = append(parts, fmt.Sprintf("You can check its output using the %s tool.", ToolNameTaskOutput))
	}
	return []TSMsg{CreateUserMessage(CreateUserMessageOpts{Content: WrapInSystemReminder(strings.Join(parts, " ")), IsMeta: true})}
}

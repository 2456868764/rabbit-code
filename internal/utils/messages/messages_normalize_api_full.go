// Full TS normalizeMessagesForAPI parity on map-shaped messages (see src/utils/messages.ts).
//
// Parity baseline path: src/utils/messages.ts (upstream Claude Code). Intentional diffs: no analytics
// logEvent; stream event union follows the ported snapshot in this repo.
package messages

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

const (
	toolReferenceTurnBoundary = "Tool loaded."

	apiPDFMaxPages   = 100
	pdfTargetRawSize = 20 * 1024 * 1024
)

func formatFileSizeForAPI(sizeInBytes int) string {
	kb := float64(sizeInBytes) / 1024
	if kb < 1 {
		return strconv.Itoa(sizeInBytes) + " bytes"
	}
	if kb < 1024 {
		return trimTrailingZeroDecimal(kb) + "KB"
	}
	mb := kb / 1024
	if mb < 1024 {
		return trimTrailingZeroDecimal(mb) + "MB"
	}
	gb := mb / 1024
	return trimTrailingZeroDecimal(gb) + "GB"
}

func trimTrailingZeroDecimal(x float64) string {
	s := strconv.FormatFloat(x, 'f', 1, 64)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "0"), ".")
	return s
}

func isNonInteractiveSessionForAPI() bool {
	return os.Getenv("RABBIT_NON_INTERACTIVE") == "1"
}

func pdfTooLargeErrorMessageGo() string {
	limits := "max " + strconv.Itoa(apiPDFMaxPages) + " pages, " + formatFileSizeForAPI(pdfTargetRawSize)
	if isNonInteractiveSessionForAPI() {
		return "PDF too large (" + limits + "). Try reading the file a different way (e.g., extract text with pdftotext)."
	}
	return "PDF too large (" + limits + "). Double press esc to go back and try again, or use pdftotext to convert to text first."
}

func pdfPasswordProtectedErrorMessageGo() string {
	if isNonInteractiveSessionForAPI() {
		return "PDF is password protected. Try using a CLI tool to extract or convert the PDF."
	}
	return "PDF is password protected. Please double press esc to edit your message and try again."
}

func pdfInvalidErrorMessageGo() string {
	if isNonInteractiveSessionForAPI() {
		return "The PDF file was not valid. Try converting it to text first (e.g., pdftotext)."
	}
	return "The PDF file was not valid. Double press esc to go back and try again with a different file."
}

func imageTooLargeErrorMessageGo() string {
	if isNonInteractiveSessionForAPI() {
		return "Image was too large. Try resizing the image or using a different approach."
	}
	return "Image was too large. Double press esc to go back and try again with a smaller image."
}

func requestTooLargeErrorMessageGo() string {
	limits := "max " + formatFileSizeForAPI(pdfTargetRawSize)
	if isNonInteractiveSessionForAPI() {
		return "Request too large (" + limits + "). Try with a smaller file."
	}
	return "Request too large (" + limits + "). Double press esc to go back and try again with a smaller file."
}

// apiErrorTextToStripTypes maps exact synthetic API error first-line text → block types to strip from preceding meta user.
func apiErrorTextToStripTypes() map[string]map[string]struct{} {
	add := func(m map[string]map[string]struct{}, key string, types ...string) {
		set := make(map[string]struct{})
		for _, t := range types {
			set[t] = struct{}{}
		}
		m[key] = set
	}
	out := make(map[string]map[string]struct{})
	add(out, pdfTooLargeErrorMessageGo(), "document")
	add(out, pdfPasswordProtectedErrorMessageGo(), "document")
	add(out, pdfInvalidErrorMessageGo(), "document")
	add(out, imageTooLargeErrorMessageGo(), "image")
	add(out, requestTooLargeErrorMessageGo(), "document", "image")
	return out
}

func isSyntheticApiErrorMessageMap(m map[string]any) bool {
	t, _ := m["type"].(string)
	if t != "assistant" || !truthy(m["isApiErrorMessage"]) {
		return false
	}
	inner, ok := m["message"].(map[string]any)
	if !ok {
		return false
	}
	model, _ := inner["model"].(string)
	return model == SyntheticModel
}

func syntheticApiErrorFirstText(msg map[string]any) string {
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return ""
	}
	arr, ok := inner["content"].([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	first, ok := arr[0].(map[string]any)
	if !ok || strField(first, "type") != "text" {
		return ""
	}
	return strField(first, "text")
}

func mergeStripTargets(into map[string]map[string]struct{}, uuid string, types map[string]struct{}) {
	if uuid == "" {
		return
	}
	if into[uuid] == nil {
		into[uuid] = map[string]struct{}{}
	}
	for t := range types {
		into[uuid][t] = struct{}{}
	}
}

func buildStripTargetsMap(reordered []map[string]any) map[string]map[string]struct{} {
	errMap := apiErrorTextToStripTypes()
	stripTargets := make(map[string]map[string]struct{})
	for i := 0; i < len(reordered); i++ {
		msg := reordered[i]
		if !isSyntheticApiErrorMessageMap(msg) {
			continue
		}
		errText := syntheticApiErrorFirstText(msg)
		if errText == "" {
			continue
		}
		blockTypes, ok := errMap[errText]
		if !ok {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			candidate := reordered[j]
			ct, _ := candidate["type"].(string)
			if ct == "user" && truthy(candidate["isMeta"]) {
				uuid, _ := candidate["uuid"].(string)
				mergeStripTargets(stripTargets, uuid, blockTypes)
				break
			}
			if isSyntheticApiErrorMessageMap(candidate) {
				continue
			}
			break
		}
	}
	return stripTargets
}

func isToolReferenceBlockMap(c map[string]any) bool {
	return strField(c, "type") == "tool_reference"
}

// StripUnavailableToolReferencesFromUserMessageMap mirrors TS stripUnavailableToolReferencesFromUserMessage.
func StripUnavailableToolReferencesFromUserMessageMap(msg map[string]any, availableToolNames map[string]struct{}) map[string]any {
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return msg
	}
	arr, ok := inner["content"].([]any)
	if !ok {
		return msg
	}
	hasUnavailable := false
	for _, it := range arr {
		b, ok := it.(map[string]any)
		if !ok || strField(b, "type") != "tool_result" {
			continue
		}
		innerC, ok := b["content"].([]any)
		if !ok {
			continue
		}
		for _, c := range innerC {
			cm, ok := c.(map[string]any)
			if !ok || !isToolReferenceBlockMap(cm) {
				continue
			}
			raw, _ := cm["tool_name"].(string)
			if raw == "" {
				continue
			}
			canon := NormalizeLegacyToolName(raw)
			if _, ok := availableToolNames[canon]; !ok {
				hasUnavailable = true
				break
			}
		}
		if hasUnavailable {
			break
		}
	}
	if !hasUnavailable {
		return msg
	}
	out := cloneMapJSON(msg)
	oinner, _ := out["message"].(map[string]any)
	oarr, _ := oinner["content"].([]any)
	newArr := make([]any, 0, len(oarr))
	for _, it := range oarr {
		b, ok := it.(map[string]any)
		if !ok || strField(b, "type") != "tool_result" {
			newArr = append(newArr, it)
			continue
		}
		innerC, ok := b["content"].([]any)
		if !ok {
			newArr = append(newArr, b)
			continue
		}
		filtered := make([]any, 0, len(innerC))
		for _, c := range innerC {
			cm, ok := c.(map[string]any)
			if !ok || !isToolReferenceBlockMap(cm) {
				filtered = append(filtered, c)
				continue
			}
			raw, _ := cm["tool_name"].(string)
			if raw == "" {
				filtered = append(filtered, c)
				continue
			}
			canon := NormalizeLegacyToolName(raw)
			if _, ok := availableToolNames[canon]; ok {
				filtered = append(filtered, c)
			}
		}
		nb := cloneMapJSON(b)
		if len(filtered) == 0 {
			nb["content"] = []any{map[string]any{"type": "text", "text": "[Tool references removed - tools no longer available]"}}
		} else {
			nb["content"] = filtered
		}
		newArr = append(newArr, nb)
	}
	oinner["content"] = newArr
	return out
}

func userMessageContentArrayHasToolReference(arr []any) bool {
	for _, it := range arr {
		b, ok := it.(map[string]any)
		if !ok || strField(b, "type") != "tool_result" {
			continue
		}
		innerC, ok := b["content"].([]any)
		if !ok {
			continue
		}
		for _, c := range innerC {
			cm, ok := c.(map[string]any)
			if ok && isToolReferenceBlockMap(cm) {
				return true
			}
		}
	}
	return false
}

func tenguToolRefDeferEnabled() bool {
	return os.Getenv("RABBIT_TENGU_TOOLREF_DEFER") == "1"
}

func maybeInjectToolReferenceTurnBoundary(u map[string]any) map[string]any {
	inner, ok := u["message"].(map[string]any)
	if !ok {
		return u
	}
	arr, ok := inner["content"].([]any)
	if !ok || len(arr) == 0 {
		return u
	}
	for _, it := range arr {
		b, ok := it.(map[string]any)
		if ok && strField(b, "type") == "text" {
			if strings.HasPrefix(strField(b, "text"), toolReferenceTurnBoundary) {
				return u
			}
		}
	}
	if !userMessageContentArrayHasToolReference(arr) {
		return u
	}
	out := cloneMapJSON(u)
	oinner, _ := out["message"].(map[string]any)
	oarr, _ := oinner["content"].([]any)
	oinner["content"] = append(append([]any{}, oarr...), map[string]any{"type": "text", "text": toolReferenceTurnBoundary})
	return out
}

func stripUserMetaBlockTypes(u map[string]any, typesToStrip map[string]struct{}) map[string]any {
	if len(typesToStrip) == 0 {
		return u
	}
	inner, ok := u["message"].(map[string]any)
	if !ok {
		return u
	}
	arr, ok := inner["content"].([]any)
	if !ok {
		return u
	}
	var filtered []any
	for _, it := range arr {
		b, ok := it.(map[string]any)
		if !ok {
			filtered = append(filtered, it)
			continue
		}
		bt := strField(b, "type")
		if _, strip := typesToStrip[bt]; strip {
			continue
		}
		filtered = append(filtered, b)
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == len(arr) {
		return u
	}
	out := cloneMapJSON(u)
	oinner, _ := out["message"].(map[string]any)
	oinner["content"] = filtered
	return out
}

func relocateToolReferenceSiblingsGeneric(messages []map[string]any) []map[string]any {
	result := append([]map[string]any(nil), messages...)
	for i := 0; i < len(result); i++ {
		msg := result[i]
		t, _ := msg["type"].(string)
		if t != "user" {
			continue
		}
		inner, ok := msg["message"].(map[string]any)
		if !ok {
			continue
		}
		arr, ok := inner["content"].([]any)
		if !ok {
			continue
		}
		if !userMessageContentArrayHasToolReference(arr) {
			continue
		}
		var textSiblings, nonText []any
		for _, it := range arr {
			b, ok := it.(map[string]any)
			if ok && strField(b, "type") == "text" {
				textSiblings = append(textSiblings, it)
			} else {
				nonText = append(nonText, it)
			}
		}
		if len(textSiblings) == 0 {
			continue
		}
		targetIdx := -1
		for j := i + 1; j < len(result); j++ {
			cand := result[j]
			ct, _ := cand["type"].(string)
			if ct != "user" {
				continue
			}
			cInner, ok := cand["message"].(map[string]any)
			if !ok {
				continue
			}
			cc, ok := cInner["content"].([]any)
			if !ok {
				continue
			}
			hasTR := false
			for _, it := range cc {
				b, ok := it.(map[string]any)
				if ok && strField(b, "type") == "tool_result" {
					hasTR = true
					break
				}
			}
			if !hasTR {
				continue
			}
			if userMessageContentArrayHasToolReference(cc) {
				continue
			}
			targetIdx = j
			break
		}
		if targetIdx == -1 {
			continue
		}
		nm := cloneMapJSON(msg)
		nInner, _ := nm["message"].(map[string]any)
		nInner["content"] = nonText
		result[i] = nm

		tm := cloneMapJSON(result[targetIdx])
		tInner, _ := tm["message"].(map[string]any)
		tc, _ := tInner["content"].([]any)
		tInner["content"] = append(append([]any{}, tc...), textSiblings...)
		result[targetIdx] = tm
	}
	return result
}

func sanitizeErrorToolResultContentGeneric(messages []map[string]any) []map[string]any {
	out := make([]map[string]any, len(messages))
	copy(out, messages)
	for i, msg := range out {
		t, _ := msg["type"].(string)
		if t != "user" {
			continue
		}
		inner, ok := msg["message"].(map[string]any)
		if !ok {
			continue
		}
		arr, ok := inner["content"].([]any)
		if !ok {
			continue
		}
		changed := false
		newArr := make([]any, len(arr))
		copy(newArr, arr)
		for j, it := range arr {
			b, ok := it.(map[string]any)
			if !ok || strField(b, "type") != "tool_result" || !truthy(b["is_error"]) {
				continue
			}
			trContent, ok := b["content"].([]any)
			if !ok {
				continue
			}
			allText := true
			for _, c := range trContent {
				cm, ok := c.(map[string]any)
				if !ok || strField(cm, "type") != "text" {
					allText = false
					break
				}
			}
			if allText {
				continue
			}
			changed = true
			var texts []string
			for _, c := range trContent {
				cm, ok := c.(map[string]any)
				if ok && strField(cm, "type") == "text" {
					texts = append(texts, strField(cm, "text"))
				}
			}
			nb := cloneMapJSON(b)
			if len(texts) > 0 {
				nb["content"] = []any{map[string]any{"type": "text", "text": strings.Join(texts, "\n\n")}}
			} else {
				nb["content"] = []any{}
			}
			newArr[j] = nb
		}
		if !changed {
			continue
		}
		nm := cloneMapJSON(msg)
		nInner, _ := nm["message"].(map[string]any)
		nInner["content"] = newArr
		out[i] = nm
	}
	return out
}

func ensureSystemReminderWrapUserMap(msg map[string]any) map[string]any {
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return msg
	}
	switch c := inner["content"].(type) {
	case string:
		if strings.HasPrefix(c, "<system-reminder>") {
			return msg
		}
		out := cloneMapJSON(msg)
		oinner, _ := out["message"].(map[string]any)
		oinner["content"] = WrapInSystemReminder(c)
		return out
	case []any:
		changed := false
		newArr := make([]any, len(c))
		for i, it := range c {
			b, ok := it.(map[string]any)
			if !ok || strField(b, "type") != "text" {
				newArr[i] = it
				continue
			}
			tx := strField(b, "text")
			if strings.HasPrefix(tx, "<system-reminder>") {
				newArr[i] = it
				continue
			}
			changed = true
			nb := cloneMapJSON(b)
			nb["text"] = WrapInSystemReminder(tx)
			newArr[i] = nb
		}
		if !changed {
			return msg
		}
		out := cloneMapJSON(msg)
		oinner, _ := out["message"].(map[string]any)
		oinner["content"] = newArr
		return out
	default:
		return msg
	}
}

func parseToolUseInput(block map[string]any) map[string]any {
	inp := block["input"]
	if s, ok := inp.(string); ok {
		var parsed any
		_ = json.Unmarshal([]byte(s), &parsed)
		if m, ok := parsed.(map[string]any); ok {
			return m
		}
		if parsed == nil {
			return map[string]any{}
		}
		return map[string]any{"value": parsed}
	}
	if m, ok := inp.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func normalizeAssistantForAPIMap(msg map[string]any, cfg NormalizeMessagesForAPIConfig) map[string]any {
	out := cloneMapJSON(msg)
	inner, ok := out["message"].(map[string]any)
	if !ok {
		return out
	}
	arr, ok := inner["content"].([]any)
	if !ok {
		return out
	}
	newContent := make([]any, len(arr))
	for i, it := range arr {
		b, ok := it.(map[string]any)
		if !ok || strField(b, "type") != "tool_use" {
			newContent[i] = it
			continue
		}
		block := cloneMapJSON(b)
		rawName, _ := block["name"].(string)
		canon := ResolveCanonicalToolName(rawName, cfg.Tools)
		block["name"] = canon
		inp := parseToolUseInput(block)
		inp = NormalizeToolInputForAPIMap(canon, inp)
		block["input"] = inp
		if cfg.NormalizeToolUseBlock != nil {
			block = cfg.NormalizeToolUseBlock(block)
		}
		if cfg.ToolSearchEnabled {
			newContent[i] = block
		} else {
			newContent[i] = map[string]any{
				"type":  "tool_use",
				"id":    block["id"],
				"name":  block["name"],
				"input": block["input"],
			}
		}
	}
	inner["content"] = newContent
	return out
}

func appendMessageTagToUserMessageMap(msg map[string]any) map[string]any {
	if truthy(msg["isMeta"]) {
		return msg
	}
	uuid, _ := msg["uuid"].(string)
	tag := "\n[id:" + DeriveShortMessageId(uuid) + "]"
	inner, ok := msg["message"].(map[string]any)
	if !ok {
		return msg
	}
	switch c := inner["content"].(type) {
	case string:
		out := cloneMapJSON(msg)
		oinner, _ := out["message"].(map[string]any)
		oinner["content"] = c + tag
		return out
	case []any:
		if len(c) == 0 {
			return msg
		}
		lastText := -1
		for i := len(c) - 1; i >= 0; i-- {
			b, ok := c[i].(map[string]any)
			if ok && strField(b, "type") == "text" {
				lastText = i
				break
			}
		}
		if lastText < 0 {
			return msg
		}
		out := cloneMapJSON(msg)
		oinner, _ := out["message"].(map[string]any)
		oc, _ := oinner["content"].([]any)
		nc := append([]any{}, oc...)
		tb, _ := nc[lastText].(map[string]any)
		ntb := cloneMapJSON(tb)
		ntb["text"] = strField(tb, "text") + tag
		nc[lastText] = ntb
		oinner["content"] = nc
		return out
	default:
		return msg
	}
}

func mapsToTSMsgSlice(msgs []map[string]any) []TSMsg {
	out := make([]TSMsg, len(msgs))
	for i := range msgs {
		out[i] = TSMsg(msgs[i])
	}
	return out
}

func tsMsgSliceToMaps(msgs []TSMsg) []map[string]any {
	out := make([]map[string]any, len(msgs))
	for i := range msgs {
		out[i] = map[string]any(msgs[i])
	}
	return out
}

// normalizeMessagesForAPIComplete is full TS normalizeMessagesForAPI on map-shaped messages.
func normalizeMessagesForAPIComplete(msgs []map[string]any, cfg NormalizeMessagesForAPIConfig) ([]map[string]any, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	ordered := ReorderAttachmentsForAPIGeneric(msgs)
	var reordered []map[string]any
	for _, m := range ordered {
		t, _ := m["type"].(string)
		if (t == "user" || t == "assistant") && truthy(m["isVirtual"]) {
			continue
		}
		reordered = append(reordered, m)
	}
	stripTargets := buildStripTargetsMap(reordered)

	var result []map[string]any
	for _, message := range reordered {
		t, _ := message["type"].(string)
		switch t {
		case "progress":
			continue
		case "system":
			if !IsSystemLocalCommandMessageMap(message) {
				continue
			}
			um := systemLocalCommandToUserMap(message)
			result = mergeOrAppendUser(result, um)
		case "user":
			u := cloneMapJSON(message)
			if !cfg.ToolSearchEnabled {
				u = StripToolReferenceBlocksFromUserMessageMap(u)
			} else {
				u = StripUnavailableToolReferencesFromUserMessageMap(u, cfg.AvailableToolNames)
			}
			uuid, _ := u["uuid"].(string)
			if typesToStrip := stripTargets[uuid]; len(typesToStrip) > 0 && truthy(u["isMeta"]) {
				u = stripUserMetaBlockTypes(u, typesToStrip)
				if u == nil {
					continue
				}
			}
			if !tenguToolRefDeferEnabled() {
				u = maybeInjectToolReferenceTurnBoundary(u)
			}
			result = mergeOrAppendUser(result, u)
		case "assistant":
			if isSyntheticApiErrorMessageMap(message) {
				continue
			}
			a := normalizeAssistantForAPIMap(cloneMapJSON(message), cfg)
			id := assistantMessageID(a)
			merged := false
			for i := len(result) - 1; i >= 0; i-- {
				prev := result[i]
				pt, _ := prev["type"].(string)
				if pt != "assistant" && !isToolResultUserMessage(prev) {
					break
				}
				if pt == "assistant" && assistantMessageID(prev) == id && id != "" {
					result[i] = MergeAssistantMessagesMap(prev, a)
					merged = true
					break
				}
			}
			if !merged {
				result = append(result, a)
			}
		case "attachment":
			if cfg.NormalizeAttachment == nil {
				return nil, ErrAttachmentNeedsNormalizer
			}
			att, _ := message["attachment"].(map[string]any)
			expanded, err := cfg.NormalizeAttachment(att)
			if err != nil {
				return nil, err
			}
			chain := append([]map[string]any(nil), expanded...)
			if tenguChairSermonEnabled() {
				for i := range chain {
					chain[i] = ensureSystemReminderWrapUserMap(cloneMapJSON(chain[i]))
				}
			}
			if len(result) > 0 {
				last := result[len(result)-1]
				if lt, _ := last["type"].(string); lt == "user" {
					acc := last
					for _, ex := range chain {
						acc = MergeUserMessagesAndToolResultsMap(acc, cloneMapJSON(ex))
					}
					result[len(result)-1] = acc
					continue
				}
			}
			for _, ex := range chain {
				result = append(result, cloneMapJSON(ex))
			}
		default:
			continue
		}
	}

	relocated := result
	if tenguToolRefDeferEnabled() {
		relocated = relocateToolReferenceSiblingsGeneric(result)
	}

	ts := mapsToTSMsgSlice(relocated)
	ts = FilterOrphanedThinkingOnlyMessages(ts)
	ts = FilterTrailingThinkingFromLastAssistant(ts)
	ts = FilterWhitespaceOnlyAssistantMessages(ts)
	ts = EnsureNonEmptyAssistantContent(ts)
	if tenguChairSermonEnabled() {
		merged := MergeAdjacentUserMessagesGeneric(tsMsgSliceToMaps(ts))
		ts = mapsToTSMsgSlice(merged)
		ts = SmooshSystemReminderSiblings(ts)
	}
	outMaps := sanitizeErrorToolResultContentGeneric(tsMsgSliceToMaps(ts))
	if shouldAppendSnipMessageTags() {
		for i := range outMaps {
			if t, _ := outMaps[i]["type"].(string); t == "user" {
				outMaps[i] = appendMessageTagToUserMessageMap(outMaps[i])
			}
		}
	}
	if err := ValidateImagesForAPIMap(outMaps); err != nil {
		return nil, err
	}
	return outMaps, nil
}

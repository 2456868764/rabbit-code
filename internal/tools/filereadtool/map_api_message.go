package filereadtool

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MapReadResultOptions configures transcript mapping (optional; zero value is valid).
type MapReadResultOptions struct {
	// MainLoopModel is used like TS getMainLoopModel() for ShouldIncludeCyberMitigation on text reads.
	MainLoopModel string
}

// MapReadResultForMessagesAPI maps FileRead JSON output to Messages API shapes matching
// FileReadTool.ts mapToolResultToToolResultBlockParam plus supplemental user messages
// (document block for full PDF; image blocks for parts), as in toolExecution newMessages.
//
// toolResultContent is either a string or a []any of content blocks (image Read, notebook).
// supplemental is a slice of user-message contents: each element is the full content array
// for one follow-up user message (meta in TS). On recoverable errors (e.g. parts dir read),
// returns a text tool_result only and no supplemental blocks.
func MapReadResultForMessagesAPI(resultJSON []byte, opt MapReadResultOptions) (toolResultContent any, supplemental [][]any) {
	var root map[string]any
	if err := json.Unmarshal(resultJSON, &root); err != nil {
		return string(resultJSON), nil
	}
	typ, _ := root["type"].(string)
	switch typ {
	case "text":
		return mapReadTextToolResultAPI(root, opt)
	case "notebook":
		return mapReadNotebookToolResultAPI(root)
	case "file_unchanged":
		return mapReadFileUnchangedToolResultAPI()
	case "image":
		return mapReadImageToolResultAPI(root)
	case "pdf":
		return mapReadPDFToolResultAPI(root)
	case "parts":
		return mapReadPartsToolResultAPI(root)
	default:
		return string(resultJSON), nil
	}
}

func mapReadImageToolResultAPI(root map[string]any) (any, [][]any) {
	file, _ := root["file"].(map[string]any)
	if file == nil {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	b64, _ := file["base64"].(string)
	mt, _ := file["type"].(string)
	if b64 == "" || mt == "" {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	block := map[string]any{
		"type": "image",
		"source": map[string]any{
			"type":       "base64",
			"data":       b64,
			"media_type": mt,
		},
	}
	return []any{block}, nil
}

func mapReadPDFToolResultAPI(root map[string]any) (any, [][]any) {
	file, _ := root["file"].(map[string]any)
	if file == nil {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	fp, _ := file["filePath"].(string)
	b64, _ := file["base64"].(string)
	orig := jsonNumberToInt64(file["originalSize"])
	summary := fmt.Sprintf("PDF file read: %s (%s)", fp, FormatFileSize(orig))
	if b64 == "" {
		return summary, nil
	}
	doc := map[string]any{
		"type": "document",
		"source": map[string]any{
			"type":       "base64",
			"media_type": "application/pdf",
			"data":       b64,
		},
	}
	return summary, [][]any{{doc}}
}

func mapReadPartsToolResultAPI(root map[string]any) (any, [][]any) {
	file, _ := root["file"].(map[string]any)
	if file == nil {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	fp, _ := file["filePath"].(string)
	outDir, _ := file["outputDir"].(string)
	nPages := jsonNumberToInt64(file["count"])
	orig := jsonNumberToInt64(file["originalSize"])
	summary := fmt.Sprintf("PDF pages extracted: %d page(s) from %s (%s)", nPages, fp, FormatFileSize(orig))
	if outDir == "" {
		return summary, nil
	}
	blocks, err := PartsOutputDirToImageContentBlocks(outDir)
	if err != nil || len(blocks) == 0 {
		return summary, nil
	}
	return summary, [][]any{blocks}
}

func jsonNumberToInt64(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int:
		return int64(x)
	case int64:
		return x
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
}

// PartsOutputDirToImageContentBlocks reads sorted *.jpg from a pdftoppm output directory
// and returns Messages API image blocks (JPEG, base64), after ResizeJPEGBytesForAPI.
func PartsOutputDirToImageContentBlocks(outDir string) ([]any, error) {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var blocks []any
	for _, n := range names {
		path := filepath.Join(outDir, n)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		processed, err := ResizeJPEGBytesForAPI(raw)
		if err != nil {
			processed = raw
		}
		b64 := base64.StdEncoding.EncodeToString(processed)
		blocks = append(blocks, map[string]any{
			"type": "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": "image/jpeg",
				"data":       b64,
			},
		})
	}
	return blocks, nil
}

func mapReadFileUnchangedToolResultAPI() (any, [][]any) {
	return FileUnchangedStub, nil
}

func mapReadTextToolResultAPI(root map[string]any, opt MapReadResultOptions) (any, [][]any) {
	file, _ := root["file"].(map[string]any)
	if file == nil {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	content, _ := file["content"].(string)
	startLine := int(jsonNumberToInt64(file["startLine"]))
	if startLine < 1 {
		startLine = 1
	}
	totalLines := int(jsonNumberToInt64(file["totalLines"]))

	if content != "" {
		s := AddLineNumbers(content, startLine)
		if ShouldIncludeCyberMitigation(opt.MainLoopModel) {
			s += CyberRiskMitigationReminder
		}
		return s, nil
	}
	if totalLines == 0 {
		return `<system-reminder>Warning: the file exists but the contents are empty.</system-reminder>`, nil
	}
	return fmt.Sprintf(`<system-reminder>Warning: the file exists but is shorter than the provided offset (%d). The file has %d lines.</system-reminder>`, startLine, totalLines), nil
}

func mapReadNotebookToolResultAPI(root map[string]any) (any, [][]any) {
	file, _ := root["file"].(map[string]any)
	if file == nil {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	cellsRaw, ok := file["cells"].([]any)
	if !ok || len(cellsRaw) == 0 {
		b, _ := json.Marshal(root)
		return string(b), nil
	}
	blocks := notebookCellsToMergedContentBlocks(cellsRaw)
	return blocks, nil
}

func notebookCellMainTextBlock(cell map[string]any) map[string]any {
	cellType, _ := cell["cellType"].(string)
	cellID, _ := cell["cell_id"].(string)
	source, _ := cell["source"].(string)
	var meta strings.Builder
	if cellType != "code" {
		fmt.Fprintf(&meta, "<cell_type>%s</cell_type>", cellType)
	}
	if cellType == "code" {
		if lang, ok := cell["language"].(string); ok && lang != "" && strings.ToLower(lang) != "python" {
			fmt.Fprintf(&meta, "<language>%s</language>", lang)
		}
	}
	text := fmt.Sprintf(`<cell id="%s">%s%s</cell id="%s">`, cellID, meta.String(), source, cellID)
	return map[string]any{"type": "text", "text": text}
}

func notebookOutputsToBlocks(outputs []any) []any {
	var out []any
	for _, o := range outputs {
		om, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if txt, ok := om["text"].(string); ok && txt != "" {
			out = append(out, map[string]any{"type": "text", "text": "\n" + txt})
		}
		if img, ok := om["image"].(map[string]any); ok {
			data, _ := img["image_data"].(string)
			mt, _ := img["media_type"].(string)
			if data != "" {
				out = append(out, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"data":       data,
						"media_type": mt,
					},
				})
			}
		}
	}
	return out
}

func mergeAdjacentTextToolBlocks(blocks []any) []any {
	if len(blocks) == 0 {
		return blocks
	}
	acc := make([]any, 0, len(blocks))
	for _, curr := range blocks {
		cm, ok := curr.(map[string]any)
		if !ok || cm["type"] != "text" {
			acc = append(acc, curr)
			continue
		}
		if len(acc) == 0 {
			acc = append(acc, curr)
			continue
		}
		prev := acc[len(acc)-1]
		pm, ok := prev.(map[string]any)
		if !ok || pm["type"] != "text" {
			acc = append(acc, curr)
			continue
		}
		pt, _ := pm["text"].(string)
		ct, _ := cm["text"].(string)
		pm["text"] = pt + "\n" + ct
	}
	return acc
}

func notebookCellsToMergedContentBlocks(cells []any) []any {
	var all []any
	for _, c := range cells {
		cell, ok := c.(map[string]any)
		if !ok {
			continue
		}
		all = append(all, notebookCellMainTextBlock(cell))
		if outs, ok := cell["outputs"].([]any); ok {
			all = append(all, notebookOutputsToBlocks(outs)...)
		}
	}
	return mergeAdjacentTextToolBlocks(all)
}

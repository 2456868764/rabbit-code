package filereadtool

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

const (
	notebookLargeOutputThreshold = 10000
	notebookTruncateMaxChars     = 30000
)

type rawNotebook struct {
	Cells    []json.RawMessage `json:"cells"`
	Metadata struct {
		LanguageInfo struct {
			Name string `json:"name"`
		} `json:"language_info"`
	} `json:"metadata"`
}

type rawCell struct {
	CellType       string            `json:"cell_type"`
	Source         json.RawMessage   `json:"source"`
	Outputs        []json.RawMessage `json:"outputs"`
	ExecutionCount *int              `json:"execution_count"`
	ID             string            `json:"id"`
}

// ReadNotebook mirrors utils/notebook.ts readNotebook (all cells, no cellId filter).
func ReadNotebook(notebookPath string) ([]any, error) {
	data, err := os.ReadFile(notebookPath)
	if err != nil {
		return nil, err
	}
	var nb rawNotebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return nil, fmt.Errorf("notebook json: %w", err)
	}
	lang := nb.Metadata.LanguageInfo.Name
	if lang == "" {
		lang = "python"
	}
	out := make([]any, 0, len(nb.Cells))
	for i, raw := range nb.Cells {
		var cell rawCell
		if err := json.Unmarshal(raw, &cell); err != nil {
			return nil, fmt.Errorf("notebook cell %d: %w", i, err)
		}
		out = append(out, processNotebookCell(&cell, i, lang, false))
	}
	return out, nil
}

func cellSourceString(src json.RawMessage) string {
	if len(src) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(src, &s); err == nil {
		return s
	}
	var parts []string
	if err := json.Unmarshal(src, &parts); err == nil {
		return strings.Join(parts, "")
	}
	return string(src)
}

func formatNotebookOutputText(text string) string {
	if len(text) <= notebookTruncateMaxChars {
		return text
	}
	head := text[:notebookTruncateMaxChars]
	rest := text[notebookTruncateMaxChars:]
	remainingLines := strings.Count(rest, "\n") + 1
	return fmt.Sprintf("%s\n\n... [%d lines truncated] ...", head, remainingLines)
}

func notebookOutputsTooLarge(outputs []any) bool {
	n := 0
	for _, o := range outputs {
		m, ok := o.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["text"].(string); t != "" {
			n += len(t)
		}
		if im, ok := m["image"].(map[string]any); ok {
			if s, _ := im["image_data"].(string); s != "" {
				n += len(s)
			}
		}
		if n > notebookLargeOutputThreshold {
			return true
		}
	}
	return false
}

func extractNotebookImage(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	if s, ok := data["image/png"].(string); ok && s != "" {
		return map[string]any{
			"image_data": strings.ReplaceAll(s, " ", ""),
			"media_type": "image/png",
		}
	}
	if s, ok := data["image/jpeg"].(string); ok && s != "" {
		return map[string]any{
			"image_data": strings.ReplaceAll(s, " ", ""),
			"media_type": "image/jpeg",
		}
	}
	return nil
}

func processNotebookOutput(raw []byte) any {
	var o struct {
		OutputType string          `json:"output_type"`
		Text       json.RawMessage `json:"text"`
		Data       map[string]any  `json:"data"`
		Ename      string          `json:"ename"`
		Evalue     string          `json:"evalue"`
		Traceback  []string        `json:"traceback"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return map[string]any{"output_type": "stream", "text": ""}
	}
	switch o.OutputType {
	case "stream":
		return map[string]any{
			"output_type": o.OutputType,
			"text":        formatNotebookOutputText(processNotebookOutputText(o.Text)),
		}
	case "execute_result", "display_data":
		var plain string
		if o.Data != nil {
			switch v := o.Data["text/plain"].(type) {
			case string:
				plain = v
			case []any:
				for _, x := range v {
					if s, ok := x.(string); ok {
						plain += s
					}
				}
			}
		}
		out := map[string]any{
			"output_type": o.OutputType,
			"text":        formatNotebookOutputText(processNotebookOutputTextFromJSON(o.Text, plain)),
		}
		if img := extractNotebookImage(o.Data); img != nil {
			out["image"] = img
		}
		return out
	case "error":
		tb := strings.Join(o.Traceback, "\n")
		msg := fmt.Sprintf("%s: %s\n%s", o.Ename, o.Evalue, tb)
		return map[string]any{
			"output_type": o.OutputType,
			"text":        formatNotebookOutputText(msg),
		}
	default:
		return map[string]any{"output_type": o.OutputType, "text": ""}
	}
}

func processNotebookOutputText(rawText json.RawMessage) string {
	if len(rawText) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(rawText, &s); err == nil {
		return s
	}
	var parts []string
	if err := json.Unmarshal(rawText, &parts); err == nil {
		return strings.Join(parts, "")
	}
	return string(rawText)
}

func processNotebookOutputTextFromJSON(textField json.RawMessage, plain string) string {
	if plain != "" {
		return plain
	}
	return processNotebookOutputText(textField)
}

func processNotebookCell(cell *rawCell, index int, codeLanguage string, includeLargeOutputs bool) map[string]any {
	cellID := cell.ID
	if cellID == "" {
		cellID = fmt.Sprintf("cell-%d", index)
	}
	src := cellSourceString(cell.Source)
	out := map[string]any{
		"cellType": cell.CellType,
		"source":   src,
		"cell_id":  cellID,
	}
	if cell.CellType == "code" && cell.ExecutionCount != nil {
		out["execution_count"] = *cell.ExecutionCount
	}
	if cell.CellType == "code" {
		out["language"] = codeLanguage
	}
	if cell.CellType == "code" && len(cell.Outputs) > 0 {
		proc := make([]any, 0, len(cell.Outputs))
		for _, oraw := range cell.Outputs {
			proc = append(proc, processNotebookOutput(oraw))
		}
		if !includeLargeOutputs && notebookOutputsTooLarge(proc) {
			out["outputs"] = []any{map[string]any{
				"output_type": "stream",
				"text": fmt.Sprintf("Outputs are too large to include. Use %s with: cat <notebook_path> | jq '.cells[%d].outputs'",
					bashtool.BashToolName, index),
			}}
		} else {
			out["outputs"] = proc
		}
	}
	return out
}

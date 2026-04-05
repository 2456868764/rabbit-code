package notebookedittool

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
)

// NotebookEdit implements tools.Tool for NotebookEditTool.ts.
type NotebookEdit struct{}

// New returns a NotebookEdit tool.
func New() *NotebookEdit { return &NotebookEdit{} }

func (n *NotebookEdit) Name() string { return NotebookEditToolName }

func (n *NotebookEdit) Aliases() []string { return nil }

type notebookEditInput struct {
	NotebookPath string  `json:"notebook_path"`
	CellID       *string `json:"cell_id,omitempty"`
	NewSource    string  `json:"new_source"`
	CellType     *string `json:"cell_type,omitempty"`
	EditMode     string  `json:"edit_mode,omitempty"`
}

func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
}

func modTimeMillis(fi os.FileInfo) int64 {
	return fi.ModTime().UnixMilli()
}

func readFileStateFromCtx(ctx context.Context) *filereadtool.ReadFileStateMap {
	if w := filewritetool.WriteContextFrom(ctx); w != nil && w.ReadFileState != nil {
		return w.ReadFileState
	}
	if rc := filereadtool.RunContextFrom(ctx); rc != nil && rc.ReadFileState != nil {
		return rc.ReadFileState
	}
	return nil
}

func denyEditFromCtx(ctx context.Context) func(string) bool {
	if w := filewritetool.WriteContextFrom(ctx); w != nil && w.DenyEdit != nil {
		return w.DenyEdit
	}
	if rc := filereadtool.RunContextFrom(ctx); rc != nil && rc.DenyRead != nil {
		return rc.DenyRead
	}
	return nil
}

func numberToFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	default:
		return 0
	}
}

func notebookRequiresCellIDs(nb map[string]any) bool {
	maj := numberToFloat(nb["nbformat"])
	min := numberToFloat(nb["nbformat_minor"])
	return maj > 4 || (maj == 4 && min >= 5)
}

func cellStringID(cell map[string]any) string {
	if cell == nil {
		return ""
	}
	s, _ := cell["id"].(string)
	return s
}

func cellsSlice(nb map[string]any) ([]any, error) {
	raw, ok := nb["cells"]
	if !ok {
		return nil, errors.New("notebook missing cells")
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, errors.New("notebook cells is not an array")
	}
	return arr, nil
}

func findCellIndexByID(cells []any, id string) int {
	for i, c := range cells {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if cellStringID(m) == id {
			return i
		}
	}
	return -1
}

func randomNotebookCellID() string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	var b [13]byte
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			b[i] = 'x'
			continue
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b[:])
}

func marshalNotebookOutput(o map[string]any) ([]byte, error) {
	return json.Marshal(o)
}

// Run implements tools.Tool (NotebookEditTool.ts validateInput + call).
func (n *NotebookEdit) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var in notebookEditInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("notebookedittool: invalid json: %w", err)
	}
	path := strings.TrimSpace(in.NotebookPath)
	if path == "" {
		return nil, errors.New("notebookedittool: missing notebook_path")
	}

	abs, err := filereadtool.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("notebookedittool: path: %w", err)
	}

	editMode := strings.TrimSpace(in.EditMode)
	if editMode == "" {
		editMode = "replace"
	}
	if editMode != "replace" && editMode != "insert" && editMode != "delete" {
		return nil, errors.New("Edit mode must be replace, insert, or delete.")
	}
	if editMode == "insert" && in.CellType == nil {
		return nil, errors.New("Cell type is required when using edit_mode=insert.")
	}

	deny := denyEditFromCtx(ctx)
	if deny != nil && deny(abs) {
		return nil, errors.New("File is in a directory that is denied by your permission settings.")
	}

	wc := filewritetool.WriteContextFrom(ctx)
	if wc != nil && wc.CheckTeamMemSecrets != nil {
		if msg := wc.CheckTeamMemSecrets(abs, in.NewSource); msg != "" {
			return nil, errors.New(msg)
		}
	}

	st := readFileStateFromCtx(ctx)
	unc := isUncPath(abs)
	if !unc {
		if !strings.EqualFold(filepath.Ext(abs), ".ipynb") {
			return nil, errors.New("File must be a Jupyter notebook (.ipynb file). For editing other file types, use the FileEdit tool.")
		}
		if st == nil {
			return nil, errors.New("File has not been read yet. Read it first before writing to it.")
		}
		ent, ok := st.Get(abs)
		if !ok || ent.IsPartialView {
			return nil, errors.New("File has not been read yet. Read it first before writing to it.")
		}

		fi, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("Notebook file does not exist.")
			}
			return nil, err
		}
		if modTimeMillis(fi) > ent.Timestamp {
			return nil, errors.New("File has been modified since read, either by the user or by a linter. Read it again before attempting to write it.")
		}
	}

	originalNormalized, enc, endings, err := filewritetool.ReadNormalizedFileWithContext(abs, wc)
	if err != nil {
		return nil, err
	}

	var nb map[string]any
	dec := json.NewDecoder(strings.NewReader(originalNormalized))
	dec.UseNumber()
	if err := dec.Decode(&nb); err != nil {
		return nil, errors.New("Notebook is not valid JSON.")
	}
	cells, err := cellsSlice(nb)
	if err != nil {
		return nil, err
	}

	var cellIDArg string
	if in.CellID != nil {
		cellIDArg = strings.TrimSpace(*in.CellID)
	}
	if cellIDArg == "" && editMode != "insert" {
		return nil, errors.New("Cell ID must be specified when not inserting a new cell.")
	}

	if cellIDArg != "" {
		idx := findCellIndexByID(cells, cellIDArg)
		if idx < 0 {
			if parsed, ok := ParseCellId(cellIDArg); ok {
				if parsed < 0 || parsed >= len(cells) {
					return nil, fmt.Errorf("Cell with index %d does not exist in notebook.", parsed)
				}
			} else {
				return nil, fmt.Errorf("Cell with ID %q not found in notebook.", cellIDArg)
			}
		}
	}

	if wc != nil && wc.BeforeFileEdited != nil {
		wc.BeforeFileEdited(abs)
	}
	if wc != nil && wc.FileHistoryTrack != nil {
		wc.FileHistoryTrack(abs, wc.ParentMessageUUID)
	}

	out, softErr := n.applyEdit(abs, originalNormalized, enc, endings, nb, cells, in, editMode, wc, st)
	if softErr != nil {
		return nil, softErr
	}
	return out, nil
}

func (n *NotebookEdit) applyEdit(
	abs, originalNormalized, enc string, endings filewritetool.LineEndingType,
	nb map[string]any, cells []any,
	in notebookEditInput, originalEditMode string,
	wc *filewritetool.WriteContext,
	st *filereadtool.ReadFileStateMap,
) ([]byte, error) {
	editMode := originalEditMode
	var cellIDArg string
	if in.CellID != nil {
		cellIDArg = strings.TrimSpace(*in.CellID)
	}

	cellIndex := 0
	if cellIDArg == "" {
		cellIndex = 0
	} else {
		cellIndex = findCellIndexByID(cells, cellIDArg)
		if cellIndex < 0 {
			if parsed, ok := ParseCellId(cellIDArg); ok {
				cellIndex = parsed
			}
		}
		if originalEditMode == "insert" {
			cellIndex++
		}
	}

	if editMode == "replace" && cellIndex == len(cells) {
		editMode = "insert"
		if in.CellType == nil {
			t := "code"
			in.CellType = &t
		}
	}

	lang := "python"
	if meta, ok := nb["metadata"].(map[string]any); ok {
		if li, ok := meta["language_info"].(map[string]any); ok {
			if name, ok := li["name"].(string); ok && name != "" {
				lang = name
			}
		}
	}

	cellTypeOut := "code"
	if in.CellType != nil && strings.TrimSpace(*in.CellType) != "" {
		cellTypeOut = strings.TrimSpace(*in.CellType)
	}

	failOut := func(errMsg string) ([]byte, error) {
		cid := ""
		if in.CellID != nil {
			cid = strings.TrimSpace(*in.CellID)
		}
		o := map[string]any{
			"new_source":    in.NewSource,
			"cell_type":     cellTypeOut,
			"language":      "python",
			"edit_mode":     "replace",
			"cell_id":       cid,
			"error":         errMsg,
			"notebook_path": abs,
			"original_file": "",
			"updated_file":  "",
		}
		return marshalNotebookOutput(o)
	}

	needsID := notebookRequiresCellIDs(nb)
	var newCellID string
	if needsID && editMode == "insert" {
		newCellID = randomNotebookCellID()
	}

	switch editMode {
	case "delete":
		if cellIndex < 0 || cellIndex >= len(cells) {
			return failOut(fmt.Sprintf("Cell index %d is out of range.", cellIndex))
		}
		cells = append(cells[:cellIndex], cells[cellIndex+1:]...)
		nb["cells"] = cells
	case "insert":
		ct := cellTypeOut
		if in.CellType != nil {
			ct = strings.TrimSpace(*in.CellType)
		}
		if ct != "code" && ct != "markdown" {
			return nil, errors.New("cell_type must be code or markdown.")
		}
		var newCell map[string]any
		if ct == "markdown" {
			newCell = map[string]any{
				"cell_type": "markdown",
				"metadata":  map[string]any{},
				"source":    in.NewSource,
			}
		} else {
			newCell = map[string]any{
				"cell_type":       "code",
				"metadata":        map[string]any{},
				"source":          in.NewSource,
				"execution_count": nil,
				"outputs":         []any{},
			}
		}
		if newCellID != "" {
			newCell["id"] = newCellID
		}
		if cellIndex < 0 || cellIndex > len(cells) {
			return failOut(fmt.Sprintf("Cell index %d is out of range for insert.", cellIndex))
		}
		cells = append(cells[:cellIndex], append([]any{newCell}, cells[cellIndex:]...)...)
		nb["cells"] = cells
		cellTypeOut = ct
	default:
		if cellIndex < 0 || cellIndex >= len(cells) {
			return failOut(fmt.Sprintf("Cell index %d is out of range.", cellIndex))
		}
		target, ok := cells[cellIndex].(map[string]any)
		if !ok {
			return failOut("Invalid cell structure.")
		}
		target["source"] = in.NewSource
		if ct, ok := target["cell_type"].(string); ok && ct == "code" {
			target["execution_count"] = nil
			target["outputs"] = []any{}
		}
		if in.CellType != nil {
			ct := strings.TrimSpace(*in.CellType)
			if ct == "code" || ct == "markdown" {
				target["cell_type"] = ct
				if ct == "code" {
					target["execution_count"] = nil
					target["outputs"] = []any{}
				}
			}
		}
		if t, ok := target["cell_type"].(string); ok {
			cellTypeOut = t
		}
	}

	updated, err := json.MarshalIndent(nb, "", " ")
	if err != nil {
		return failOut(err.Error())
	}
	updatedStr := string(updated)
	toWrite := updatedStr
	if endings == filewritetool.LineEndingCRLF {
		toWrite = filewritetool.ApplyCRLFLineEndings(updatedStr)
	}
	outBytes, err := filewritetool.EncodeTextToFileBytes(toWrite, enc)
	if err != nil {
		b, e2 := failOut(err.Error())
		if e2 != nil {
			return nil, err
		}
		return b, nil
	}
	if err := os.WriteFile(abs, outBytes, 0o644); err != nil {
		b, e2 := failOut(err.Error())
		if e2 != nil {
			return nil, err
		}
		return b, nil
	}

	if wc != nil && wc.AfterWrite != nil {
		wc.AfterWrite(abs, originalNormalized, updatedStr)
	}

	fi2, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if st != nil {
		st.Set(abs, filereadtool.ReadFileStateEntry{
			Content:       updatedStr,
			Timestamp:     modTimeMillis(fi2),
			Offset:        nil,
			Limit:         nil,
			IsPartialView: false,
		})
	}

	outCellID := cellIDArg
	if editMode == "insert" {
		outCellID = newCellID
	}

	o := map[string]any{
		"new_source":    in.NewSource,
		"cell_type":     cellTypeOut,
		"language":      lang,
		"edit_mode":     editMode,
		"notebook_path": abs,
		"original_file": originalNormalized,
		"updated_file":  updatedStr,
		"error":         "",
		"cell_id":       outCellID,
	}
	return marshalNotebookOutput(o)
}

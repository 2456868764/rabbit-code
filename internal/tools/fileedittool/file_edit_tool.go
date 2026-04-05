package fileedittool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/notebookedittool"
)

// FileEdit implements tools.Tool for Edit (FileEditTool.ts).
type FileEdit struct{}

// New returns a FileEdit tool.
func New() *FileEdit { return &FileEdit{} }

func (f *FileEdit) Name() string { return FileEditToolName }

func (f *FileEdit) Aliases() []string { return nil }

type editInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
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

func simulateReplace(file, old, new string, replaceAll bool) string {
	if replaceAll {
		return strings.ReplaceAll(file, old, new)
	}
	return strings.Replace(file, old, new, 1)
}

func criticalEditStaleness(abs string, st *filereadtool.ReadFileStateMap, diskNormalized string, mtimeMs int64) error {
	lastRead, ok := st.Get(abs)
	if !ok {
		return errors.New(filewritetool.ErrFileUnexpectedlyModified)
	}
	if mtimeMs > lastRead.Timestamp {
		full := lastRead.Offset == nil && lastRead.Limit == nil
		if !full || diskNormalized != lastRead.Content {
			return errors.New(filewritetool.ErrFileUnexpectedlyModified)
		}
	}
	return nil
}

func validateEditExistingReadState(abs string, st *filereadtool.ReadFileStateMap, fileContentNormalized string) error {
	if st == nil {
		return errors.New("File has not been read yet. Read it first before writing to it.")
	}
	ent, ok := st.Get(abs)
	if !ok || ent.IsPartialView {
		return errors.New("File has not been read yet. Read it first before writing to it.")
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if modTimeMillis(fi) > ent.Timestamp {
		full := ent.Offset == nil && ent.Limit == nil
		if !(full && fileContentNormalized == ent.Content) {
			return errors.New(filewritetool.ErrFileModifiedSinceRead)
		}
	}
	return nil
}

func maybeGitDiff(ctx context.Context, abs string) map[string]any {
	if !filewritetoolEnvRemote() {
		return nil
	}
	w := filewritetool.WriteContextFrom(ctx)
	if w == nil || w.FetchGitDiff == nil {
		return nil
	}
	if w.QuartzLanternEnabled == nil || !w.QuartzLanternEnabled() {
		return nil
	}
	d, err := w.FetchGitDiff(abs)
	if err != nil || d == nil {
		return nil
	}
	return d
}

func filewritetoolEnvRemote() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_REMOTE")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// Run implements tools.Tool.
func (f *FileEdit) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var in editInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("fileedittool: invalid json: %w", err)
	}
	path := strings.TrimSpace(in.FilePath)
	if path == "" {
		return nil, errors.New("fileedittool: missing file_path")
	}
	abs, err := filereadtool.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("fileedittool: path: %w", err)
	}

	wc := filewritetool.WriteContextFrom(ctx)
	if wc != nil && wc.CheckTeamMemSecrets != nil {
		if msg := wc.CheckTeamMemSecrets(abs, in.NewString); msg != "" {
			return nil, errors.New(msg)
		}
	}

	if in.OldString == in.NewString {
		return nil, errors.New("No changes to make: old_string and new_string are exactly the same.")
	}

	deny := denyEditFromCtx(ctx)
	if deny != nil && deny(abs) {
		return nil, errors.New("File is in a directory that is denied by your permission settings.")
	}

	st := readFileStateFromCtx(ctx)

	if !isUncPath(abs) {
		fi, statErr := os.Stat(abs)
		if statErr == nil && !fi.IsDir() && fi.Size() > MaxEditFileSize {
			return nil, fmt.Errorf("File is too large to edit (%s). Maximum editable file size is %s.",
				FormatFileSize(fi.Size()), FormatFileSize(MaxEditFileSize))
		}
	}

	if isUncPath(abs) {
		return nil, errors.New("fileedittool: Edit on UNC paths is not supported in this build.")
	}

	var fileExists bool
	var norm string

	fi, statErr := os.Stat(abs)
	if statErr != nil && os.IsNotExist(statErr) {
		fileExists = false
	} else if statErr != nil {
		return nil, statErr
	} else if fi.IsDir() {
		return nil, fmt.Errorf("fileedittool: path is a directory: %s", abs)
	} else {
		fileExists = true
		var rerr error
		norm, _, _, rerr = filewritetool.ReadNormalizedFileWithContext(abs, wc)
		if rerr != nil {
			return nil, rerr
		}
	}

	if !fileExists {
		if in.OldString == "" {
			// create path — validated; proceed to call
		} else {
			msg := fmt.Sprintf("File does not exist. %s %s.", FileNotFoundCwdNote, mustGetwd())
			if sim := FindSimilarFile(abs); sim != "" {
				msg += fmt.Sprintf(" Did you mean %s?", sim)
			}
			return nil, errors.New(msg)
		}
	} else {
		if in.OldString == "" {
			if strings.TrimSpace(norm) != "" {
				return nil, errors.New("Cannot create new file - file already exists.")
			}
		} else {
			if strings.EqualFold(filepath.Ext(abs), ".ipynb") {
				return nil, fmt.Errorf("File is a Jupyter Notebook. Use the %s to edit this file.", notebookedittool.NotebookEditToolName)
			}
			if err := validateEditExistingReadState(abs, st, norm); err != nil {
				return nil, err
			}
			act := FindActualString(norm, in.OldString)
			if act == "" && in.OldString != "" {
				return nil, fmt.Errorf("String to replace not found in file.\nString: %s", in.OldString)
			}
			matches := strings.Count(norm, act)
			if in.OldString != "" && matches > 1 && !in.ReplaceAll {
				return nil, fmt.Errorf("Found %d matches of the string to replace, but replace_all is false. To replace all occurrences, set replace_all to true. To replace only one occurrence, please provide more context to uniquely identify the instance.\nString: %s", matches, in.OldString)
			}
			after := simulateReplace(norm, act, in.NewString, in.ReplaceAll)
			if err := validateSettingsFileEdit(abs, norm, after); err != nil {
				return nil, err
			}
		}
	}

	if wc != nil && wc.BeforeFileEdited != nil {
		wc.BeforeFileEdited(abs)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}

	if wc != nil && wc.FileHistoryTrack != nil {
		wc.FileHistoryTrack(abs, wc.ParentMessageUUID)
	}

	var originalContents string
	var enc string
	var endings filewritetool.LineEndingType
	var existsNow bool

	fi2, err := os.Stat(abs)
	if err != nil && os.IsNotExist(err) {
		existsNow = false
		originalContents = ""
		enc = filewritetool.EncUTF8
		endings = filewritetool.LineEndingLF
	} else if err != nil {
		return nil, err
	} else if fi2.IsDir() {
		return nil, fmt.Errorf("fileedittool: path is a directory: %s", abs)
	} else {
		existsNow = true
		var rerr error
		originalContents, enc, endings, rerr = filewritetool.ReadNormalizedFileWithContext(abs, wc)
		if rerr != nil {
			return nil, rerr
		}
		if st == nil {
			return nil, errors.New(filewritetool.ErrFileUnexpectedlyModified)
		}
		fi3, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		if err := criticalEditStaleness(abs, st, originalContents, modTimeMillis(fi3)); err != nil {
			return nil, err
		}
	}

	actualOld := FindActualString(originalContents, in.OldString)
	if actualOld == "" {
		actualOld = in.OldString
	}
	actualNew := PreserveQuoteStyle(in.OldString, actualOld, in.NewString)

	updated := ApplyEditToFile(originalContents, actualOld, actualNew, in.ReplaceAll)

	if existsNow && updated == originalContents {
		return nil, errors.New("String not found in file. Failed to apply edit.")
	}

	patch := filewritetool.GetPatchForDisplay(path, originalContents, updated)
	if patch == nil {
		patch = []map[string]any{}
	}

	toWrite := updated
	if endings == filewritetool.LineEndingCRLF {
		toWrite = filewritetool.ApplyCRLFLineEndings(updated)
	}
	outBytes, err := filewritetool.EncodeTextToFileBytes(toWrite, enc)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, outBytes, 0o644); err != nil {
		return nil, err
	}

	if wc != nil && wc.AfterWrite != nil {
		wc.AfterWrite(abs, originalContents, updated)
	}

	fi4, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if st != nil {
		st.Set(abs, filereadtool.ReadFileStateEntry{
			Content:       updated,
			Timestamp:     modTimeMillis(fi4),
			Offset:        nil,
			Limit:         nil,
			IsPartialView: false,
		})
	}

	um := false
	if wc != nil && wc.UserModified != nil {
		um = *wc.UserModified
	}

	out := map[string]any{
		"filePath":        path,
		"oldString":       actualOld,
		"newString":       in.NewString,
		"originalFile":    originalContents,
		"structuredPatch": patch,
		"userModified":    um,
		"replaceAll":      in.ReplaceAll,
	}
	if gd := maybeGitDiff(ctx, abs); gd != nil {
		out["gitDiff"] = gd
	}

	return json.Marshal(out)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

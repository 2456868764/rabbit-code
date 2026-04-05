package filewritetool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

// FileWrite implements tools.Tool for Write (FileWriteTool.ts).
type FileWrite struct{}

// New returns a FileWrite tool.
func New() *FileWrite { return &FileWrite{} }

func (f *FileWrite) Name() string { return FileWriteToolName }

func (f *FileWrite) Aliases() []string { return nil }

type writeInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
}

func readFileStateFromCtx(ctx context.Context) *filereadtool.ReadFileStateMap {
	if w := WriteContextFrom(ctx); w != nil && w.ReadFileState != nil {
		return w.ReadFileState
	}
	if rc := filereadtool.RunContextFrom(ctx); rc != nil && rc.ReadFileState != nil {
		return rc.ReadFileState
	}
	return nil
}

func denyEditFromCtx(ctx context.Context) func(string) bool {
	if w := WriteContextFrom(ctx); w != nil && w.DenyEdit != nil {
		return w.DenyEdit
	}
	if rc := filereadtool.RunContextFrom(ctx); rc != nil && rc.DenyRead != nil {
		return rc.DenyRead
	}
	return nil
}

func modTimeMillis(fi os.FileInfo) int64 {
	return fi.ModTime().UnixMilli()
}

// validateWriteInputExisting mirrors FileWriteTool.validateInput when the path exists (non-UNC): strict mtime vs read timestamp (no content fallback).
func validateWriteInputExisting(abs string, st *filereadtool.ReadFileStateMap) error {
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
		return errors.New(ErrFileModifiedSinceRead)
	}
	return nil
}

// criticalWriteStaleness mirrors FileWriteTool.call: after re-stat/read, mtime newer than cache → full-read content match or FILE_UNEXPECTEDLY_MODIFIED_ERROR.
func criticalWriteStaleness(abs string, st *filereadtool.ReadFileStateMap, diskContent string, mtimeMs int64) error {
	lastRead, ok := st.Get(abs)
	if !ok {
		return errors.New(ErrFileUnexpectedlyModified)
	}
	if mtimeMs > lastRead.Timestamp {
		full := lastRead.Offset == nil && lastRead.Limit == nil
		disk := normalizeNewlines(diskContent)
		if !full || disk != lastRead.Content {
			return errors.New(ErrFileUnexpectedlyModified)
		}
	}
	return nil
}

func maybeGitDiff(ctx context.Context, abs string) map[string]any {
	if !envTruthy("CLAUDE_CODE_REMOTE") {
		return nil
	}
	w := WriteContextFrom(ctx)
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

// Run implements tools.Tool.
func (f *FileWrite) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var in writeInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("filewritetool: invalid json: %w", err)
	}
	path := strings.TrimSpace(in.FilePath)
	if path == "" {
		return nil, errors.New("filewritetool: missing file_path")
	}
	abs, err := filereadtool.ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("filewritetool: path: %w", err)
	}

	wc := WriteContextFrom(ctx)
	if wc != nil && wc.CheckTeamMemSecrets != nil {
		if msg := wc.CheckTeamMemSecrets(abs, in.Content); msg != "" {
			return nil, errors.New(msg)
		}
	}

	deny := denyEditFromCtx(ctx)
	if deny != nil && deny(abs) {
		return nil, errors.New("File is in a directory that is denied by your permission settings.")
	}

	st := readFileStateFromCtx(ctx)

	if !isUncPath(abs) {
		fi, statErr := os.Stat(abs)
		if statErr != nil && !os.IsNotExist(statErr) {
			return nil, statErr
		}
		if statErr == nil && !fi.IsDir() {
			if err := validateWriteInputExisting(abs, st); err != nil {
				return nil, err
			}
		}
	}

	if wc != nil && wc.BeforeFileEdited != nil {
		wc.BeforeFileEdited(abs)
	}

	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	if wc != nil && wc.FileHistoryTrack != nil {
		wc.FileHistoryTrack(abs, wc.ParentMessageUUID)
	}

	writeEnc, writeLe := EncUTF8, LineEndingLF
	var hadFile bool
	var diskDecoded string
	prevBytes, readErr := os.ReadFile(abs)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return nil, readErr
		}
	} else {
		hadFile = true
		var err error
		writeEnc, writeLe, diskDecoded, err = resolveWriteEncoding(abs, prevBytes, true, wc)
		if err != nil {
			return nil, err
		}
	}

	if hadFile && !isUncPath(abs) {
		if st == nil {
			return nil, errors.New(ErrFileUnexpectedlyModified)
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}
		if err := criticalWriteStaleness(abs, st, diskDecoded, modTimeMillis(fi)); err != nil {
			return nil, err
		}
	}

	toWrite := in.Content
	if writeLe == LineEndingCRLF {
		toWrite = ApplyCRLFLineEndings(in.Content)
	}
	outBytes, err := EncodeTextToFileBytes(toWrite, writeEnc)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, outBytes, 0o644); err != nil {
		return nil, err
	}

	oldNorm := ""
	if hadFile {
		oldNorm = normalizeNewlines(diskDecoded)
	}
	if wc != nil && wc.AfterWrite != nil {
		wc.AfterWrite(abs, oldNorm, in.Content)
	}

	fi2, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	mtimeMs := modTimeMillis(fi2)
	if st != nil {
		st.Set(abs, filereadtool.ReadFileStateEntry{
			Content:       in.Content,
			Timestamp:     mtimeMs,
			Offset:        nil,
			Limit:         nil,
			IsPartialView: false,
		})
	}

	var out map[string]any
	if hadFile {
		patch := GetPatchForDisplay(path, oldNorm, in.Content)
		if patch == nil {
			patch = []map[string]any{}
		}
		out = map[string]any{
			"type":            "update",
			"filePath":        path,
			"content":         in.Content,
			"structuredPatch": patch,
			"originalFile":    oldNorm,
		}
	} else {
		out = map[string]any{
			"type":            "create",
			"filePath":        path,
			"content":         in.Content,
			"structuredPatch": []any{},
			"originalFile":    nil,
		}
	}

	if gd := maybeGitDiff(ctx, abs); gd != nil {
		out["gitDiff"] = gd
	}

	return json.Marshal(out)
}

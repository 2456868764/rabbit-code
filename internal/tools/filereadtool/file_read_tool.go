package filereadtool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
)

// FileRead implements tools.Tool for the Read tool (FileReadTool.ts call / output shapes).
type FileRead struct {
	Limits *FileReadingLimits
}

// New returns a FileRead tool with default limits.
func New() *FileRead {
	return &FileRead{}
}

func (f *FileRead) limits() FileReadingLimits {
	if f != nil && f.Limits != nil {
		return *f.Limits
	}
	return DefaultFileReadingLimits()
}

// Name implements tools.Tool.
func (f *FileRead) Name() string { return FileReadToolName }

// Aliases implements tools.Tool.
func (f *FileRead) Aliases() []string { return nil }

type readInput struct {
	FilePath string  `json:"file_path"`
	Offset   *int    `json:"offset"`
	Limit    *int    `json:"limit"`
	Pages    *string `json:"pages,omitempty"`
}

func parseReadInputJSON(b []byte) (readInput, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var in readInput
	if err := dec.Decode(&in); err != nil {
		return readInput{}, err
	}
	if dec.More() {
		return readInput{}, errors.New("filereadtool: invalid json: extra data after input object")
	}
	return in, nil
}

// Run implements tools.Tool (full FileReadTool callInner parity: text, notebook, image, PDF).
func (f *FileRead) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	in, err := parseReadInputJSON(inputJSON)
	if err != nil {
		return nil, fmt.Errorf("filereadtool: invalid json: %w", err)
	}
	path := strings.TrimSpace(in.FilePath)
	if path == "" {
		return nil, errors.New("filereadtool: missing file_path")
	}

	var pagesArg *string
	if in.Pages != nil && strings.TrimSpace(*in.Pages) != "" {
		p := strings.TrimSpace(*in.Pages)
		pagesArg = &p
	}

	rc := RunContextFrom(ctx)
	if err := ValidateReadInput(path, pagesArg, rc); err != nil {
		return nil, err
	}

	abs, err := ExpandPath(path)
	if err != nil {
		return nil, fmt.Errorf("filereadtool: path: %w", err)
	}

	lim := f.limits()
	if rc != nil {
		if rc.MaxSizeBytes != nil && *rc.MaxSizeBytes > 0 {
			lim.MaxSizeBytes = *rc.MaxSizeBytes
		}
		if rc.MaxTokens != nil && *rc.MaxTokens > 0 {
			lim.MaxTokens = *rc.MaxTokens
		}
	}

	out, err := f.runResolved(ctx, path, abs, abs, &in, lim, rc)
	if err != nil && os.IsNotExist(err) {
		if alt := AlternateScreenshotPath(abs); alt != "" {
			if _, stErr := os.Stat(alt); stErr == nil {
				out, err = f.runResolved(ctx, path, abs, alt, &in, lim, rc)
			}
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("File does not exist: %s", path)
		}
		return nil, err
	}
	return out, nil
}

func readDedupExtOK(ext string) bool {
	return !isImageExt(ext) && !IsPDFExtension(ext)
}

func (f *FileRead) runResolved(ctx context.Context, userPath, cacheKey, resolved string, in *readInput, lim FileReadingLimits, rc *RunContext) ([]byte, error) {
	ext := ExtFromPath(resolved)

	if rc != nil && !rc.DisableReadDedup && rc.ReadFileState != nil && readDedupExtOK(ext) {
		if f.readDedupHit(rc, cacheKey, resolved, in) {
			stub := map[string]any{
				"type": "file_unchanged",
				"file": map[string]any{"filePath": userPath},
			}
			return json.Marshal(stub)
		}
	}

	switch {
	case ext == "ipynb":
		return f.runNotebook(ctx, userPath, cacheKey, resolved, in, lim, rc)
	case isImageExt(ext):
		return f.runImage(userPath, resolved, in, lim, rc)
	case IsPDFExtension(ext):
		return f.runPDF(ctx, userPath, cacheKey, resolved, in, lim, rc)
	default:
		return f.runText(userPath, cacheKey, resolved, in, lim, rc)
	}
}

func inputOffset(in *readInput) int {
	if in.Offset != nil {
		return *in.Offset
	}
	return 1
}

func (f *FileRead) readDedupHit(rc *RunContext, cacheKey, resolved string, in *readInput) bool {
	off := inputOffset(in)
	ent, ok := rc.ReadFileState.Get(cacheKey)
	if !ok || ent.IsPartialView || ent.Offset == nil {
		return false
	}
	if *ent.Offset != off {
		return false
	}
	if (ent.Limit == nil) != (in.Limit == nil) {
		return false
	}
	if ent.Limit != nil && in.Limit != nil && *ent.Limit != *in.Limit {
		return false
	}
	st, err := os.Stat(resolved)
	if err != nil {
		return false
	}
	return st.ModTime().UnixMilli() == ent.Timestamp
}

func countFn(rc *RunContext) func(string) (int, error) {
	if rc == nil {
		return nil
	}
	return rc.CountTokens
}

func (f *FileRead) runNotebook(_ context.Context, userPath, cacheKey, resolved string, in *readInput, lim FileReadingLimits, rc *RunContext) ([]byte, error) {
	cells, err := ReadNotebook(resolved)
	if err != nil {
		return nil, err
	}
	cellsJSON, err := json.Marshal(cells)
	if err != nil {
		return nil, err
	}
	if len(cellsJSON) > lim.MaxSizeBytes {
		return nil, fmt.Errorf("Notebook content (%s) exceeds maximum allowed size (%s). Use %s with jq to read specific portions (see FileReadTool.ts).",
			FormatFileSize(int64(len(cellsJSON))), FormatFileSize(int64(lim.MaxSizeBytes)), bashtool.BashToolName)
	}
	if err := ValidateContentTokens(string(cellsJSON), "ipynb", lim.MaxTokens, countFn(rc)); err != nil {
		return nil, err
	}
	if rc != nil && rc.ReadFileState != nil {
		off := inputOffset(in)
		st, _ := os.Stat(resolved)
		ts := int64(0)
		if st != nil {
			ts = st.ModTime().UnixMilli()
		}
		rc.ReadFileState.Set(cacheKey, ReadFileStateEntry{
			Content:       string(cellsJSON),
			Timestamp:     ts,
			Offset:        intPtr(off),
			Limit:         in.Limit,
			IsPartialView: false,
		})
	}
	out := map[string]any{
		"type": "notebook",
		"file": map[string]any{
			"filePath": userPath,
			"cells":    cells,
		},
	}
	return json.Marshal(out)
}

func (f *FileRead) runImage(_ string, resolved string, in *readInput, lim FileReadingLimits, _ *RunContext) ([]byte, error) {
	_ = in
	data, err := ReadImageWithTokenBudget(resolved, lim.MaxTokens)
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (f *FileRead) runPDF(ctx context.Context, userPath, cacheKey, resolved string, in *readInput, _ FileReadingLimits, rc *RunContext) ([]byte, error) {
	if in.Pages != nil && strings.TrimSpace(*in.Pages) != "" {
		first, last, ok := ParsePDFPageRange(strings.TrimSpace(*in.Pages))
		if !ok {
			return nil, fmt.Errorf(`Invalid pages parameter: %q`, *in.Pages)
		}
		from, to := PdftoppmPageRange(first, last)
		parts, err := ExtractPDFPages(ctx, resolved, userPath, from, to)
		if err != nil {
			return nil, err
		}
		return json.Marshal(parts)
	}

	pc, err := GetPDFPageCount(resolved)
	if err == nil && pc > PDFAtMentionInlineThreshold {
		return nil, fmt.Errorf("This PDF has %d pages, which is too many to read at once. Use the pages parameter to read specific page ranges (e.g., pages: \"1-5\"). Maximum %d pages per request.",
			pc, PDFMaxPagesPerRead)
	}

	st, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	shouldExtract := !IsPDFSupported(rcStringModel(rc)) || st.Size() > PDFExtractSizeThreshold
	if shouldExtract {
		_, _ = ExtractPDFPages(ctx, resolved, userPath, nil, nil)
	}

	if !IsPDFSupported(rcStringModel(rc)) {
		return nil, fmt.Errorf("Reading full PDFs is not supported with this model. Use a newer model (Sonnet 3.5 v2 or later), or use the pages parameter to read specific page ranges (e.g., pages: \"1-5\", maximum %d pages per request). Page extraction requires poppler-utils: install with `brew install poppler` on macOS or `apt-get install poppler-utils` on Debian/Ubuntu.",
			PDFMaxPagesPerRead)
	}

	pdfData, err := readPDFFile(resolved, userPath)
	if err != nil {
		return nil, err
	}
	_ = cacheKey
	return json.Marshal(pdfData)
}

func rcStringModel(rc *RunContext) string {
	if rc == nil {
		return ""
	}
	return rc.MainLoopModel
}

func (f *FileRead) runText(userPath, cacheKey, resolved string, in *readInput, lim FileReadingLimits, rc *RunContext) ([]byte, error) {
	off := inputOffset(in)
	if off < 0 {
		return nil, errors.New("filereadtool: offset must be non-negative")
	}
	lineOffset := off - 1
	if off == 0 {
		lineOffset = 0
	}
	if in.Limit != nil && *in.Limit <= 0 {
		return nil, errors.New("filereadtool: limit must be positive")
	}

	var maxBytes *int
	if in.Limit == nil {
		mb := lim.MaxSizeBytes
		maxBytes = &mb
	}

	rr, err := ReadFileInRange(resolved, lineOffset, in.Limit, maxBytes)
	if err != nil {
		if e, ok := err.(*FileTooLargeError); ok {
			return nil, e
		}
		return nil, err
	}

	ext := ExtFromPath(resolved)
	if err := ValidateContentTokens(rr.Content, ext, lim.MaxTokens, countFn(rc)); err != nil {
		return nil, err
	}

	if rc != nil && rc.ReadFileState != nil {
		rc.ReadFileState.Set(cacheKey, ReadFileStateEntry{
			Content:       rr.Content,
			Timestamp:     rr.MtimeMs,
			Offset:        intPtr(off),
			Limit:         in.Limit,
			IsPartialView: rr.TruncatedByBytes,
		})
	}

	data := map[string]any{
		"type": "text",
		"file": map[string]any{
			"filePath":   userPath,
			"content":    rr.Content,
			"numLines":   rr.LineCount,
			"startLine":  off,
			"totalLines": rr.TotalLines,
		},
	}
	return json.Marshal(data)
}

func intPtr(i int) *int {
	p := i
	return &p
}

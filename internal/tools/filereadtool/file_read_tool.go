package filereadtool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// blockedDevicePaths mirrors FileReadTool.ts BLOCKED_DEVICE_PATHS (path-only, no I/O).
var blockedDevicePaths = map[string]struct{}{
	"/dev/zero": {}, "/dev/random": {}, "/dev/urandom": {}, "/dev/full": {},
	"/dev/stdin": {}, "/dev/tty": {}, "/dev/console": {},
	"/dev/stdout": {}, "/dev/stderr": {},
	"/dev/fd/0": {}, "/dev/fd/1": {}, "/dev/fd/2": {},
}

var imageExtensions = map[string]struct{}{
	"png": {}, "jpg": {}, "jpeg": {}, "gif": {}, "webp": {},
}

// binaryExtensions is a subset of constants/files.ts BINARY_EXTENSIONS (non-image, non-PDF handled separately).
var binaryExtensions = map[string]struct{}{
	"bmp": {}, "ico": {}, "tiff": {}, "tif": {},
	"mp4": {}, "mov": {}, "zip": {}, "tar": {}, "gz": {},
	"exe": {}, "dll": {}, "so": {}, "dylib": {}, "bin": {},
	"doc": {}, "docx": {}, "xls": {}, "xlsx": {},
	"woff": {}, "woff2": {}, "ttf": {}, "otf": {},
	"pyc": {}, "class": {}, "wasm": {},
}

// FileRead implements tools.Tool for the Read tool (FileReadTool.ts call() text path subset).
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

// readInput mirrors FileReadTool strictObject input (pages optional; PDF deferred).
type readInput struct {
	FilePath string `json:"file_path"`
	Offset   *int   `json:"offset"`
	Limit    *int   `json:"limit"`
	Pages    string `json:"pages,omitempty"`
}

// Run implements tools.Tool — text files only; PDF/images/notebooks deferred.
func (f *FileRead) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var in readInput
	if err := json.Unmarshal(inputJSON, &in); err != nil {
		return nil, fmt.Errorf("filereadtool: invalid json: %w", err)
	}
	path := strings.TrimSpace(in.FilePath)
	if path == "" {
		return nil, errors.New("filereadtool: missing file_path")
	}
	if strings.TrimSpace(in.Pages) != "" {
		return nil, errors.New("filereadtool: pages is only valid for PDF (deferred)")
	}
	abs, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("filereadtool: path: %w", err)
	}
	if isBlockedDevicePath(abs) {
		return nil, fmt.Errorf("filereadtool: cannot read %q: device file would block or produce infinite output", path)
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
	if ext == "pdf" {
		return nil, errors.New("filereadtool: PDF read deferred")
	}
	if _, img := imageExtensions[ext]; img {
		return nil, ErrImageProcessorDeferred
	}
	if _, bin := binaryExtensions[ext]; bin {
		return nil, fmt.Errorf("filereadtool: cannot read binary file type .%s", ext)
	}

	lim := f.limits()
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("filereadtool: file not found: %s", path)
		}
		return nil, fmt.Errorf("filereadtool: stat: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("filereadtool: path is a directory: %s", path)
	}
	if fi.Size() > int64(lim.MaxSizeBytes) {
		return nil, fmt.Errorf("filereadtool: file size %d exceeds max %d bytes; use offset and limit", fi.Size(), lim.MaxSizeBytes)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("filereadtool: read: %w", err)
	}
	if looksBinary(data) {
		return nil, errors.New("filereadtool: file appears to be binary")
	}

	text := normalizeNewlines(string(data))
	lines := splitLines(text)
	total := len(lines)

	offset := 1
	if in.Offset != nil {
		if *in.Offset < 0 {
			return nil, errors.New("filereadtool: offset must be non-negative")
		}
		offset = *in.Offset
		if offset == 0 {
			return nil, errors.New("filereadtool: offset must be >= 1 (1-based lines)")
		}
	}
	var limit *int
	if in.Limit != nil {
		if *in.Limit <= 0 {
			return nil, errors.New("filereadtool: limit must be positive")
		}
		limit = in.Limit
	}
	if total > 0 && offset > total {
		return nil, fmt.Errorf("filereadtool: offset %d beyond file (%d lines)", offset, total)
	}

	content, startLine, numLines, totalOut := sliceLinesFrom(lines, offset, limit)
	out := map[string]any{
		"type": "text",
		"file": map[string]any{
			"filePath":   path,
			"content":    content,
			"numLines":   numLines,
			"startLine":  startLine,
			"totalLines": totalOut,
		},
	}
	return json.Marshal(out)
}

func expandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(h, strings.TrimPrefix(p, "~/"))
	}
	return filepath.Abs(p)
}

func isBlockedDevicePath(p string) bool {
	if _, ok := blockedDevicePaths[p]; ok {
		return true
	}
	if strings.HasPrefix(p, "/proc/") {
		if strings.HasSuffix(p, "/fd/0") || strings.HasSuffix(p, "/fd/1") || strings.HasSuffix(p, "/fd/2") {
			return true
		}
	}
	return false
}

func looksBinary(b []byte) bool {
	n := 512
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if b[i] == 0 {
			return true
		}
	}
	return false
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func sliceLinesFrom(lines []string, offset1 int, limit *int) (content string, startLine, numLines, total int) {
	total = len(lines)
	if total == 0 {
		return "", 1, 0, 0
	}
	start := offset1 - 1
	if start < 0 || start >= total {
		return "", offset1, 0, total
	}
	end := total
	if limit != nil {
		end = start + *limit
		if end > total {
			end = total
		}
	}
	chunk := lines[start:end]
	content = strings.Join(chunk, "\n")
	if len(chunk) > 0 && end < total {
		content += "\n"
	}
	return content, offset1, len(chunk), total
}

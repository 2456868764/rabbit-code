package filereadtool

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// FastPathMaxSize mirrors readFileInRange.ts FAST_PATH_MAX_SIZE.
const FastPathMaxSize = 10 * 1024 * 1024

// ReadFileRangeResult mirrors readFileInRange.ts ReadFileRangeResult.
type ReadFileRangeResult struct {
	Content          string
	LineCount        int
	TotalLines       int
	TotalBytes       int
	ReadBytes        int
	MtimeMs          int64
	TruncatedByBytes bool
}

// FileTooLargeError mirrors readFileInRange.ts FileTooLargeError.
type FileTooLargeError struct {
	SizeInBytes  int64
	MaxSizeBytes int64
}

func (e *FileTooLargeError) Error() string {
	return fmt.Sprintf("File content (%s) exceeds maximum allowed size (%s). Use offset and limit parameters to read specific portions of the file, or search for specific content instead of reading the whole file.",
		FormatFileSize(e.SizeInBytes), FormatFileSize(e.MaxSizeBytes))
}

// ReadFileInRange mirrors utils/readFileInRange.ts (fast path for regular files < 10 MiB).
// lineOffset is 0-based first line index (TS passes offset-1). maxLines nil = all remaining lines.
func ReadFileInRange(path string, lineOffset int, maxLines *int, maxBytes *int) (ReadFileRangeResult, error) {
	st, err := os.Stat(path)
	if err != nil {
		return ReadFileRangeResult{}, err
	}
	if st.IsDir() {
		return ReadFileRangeResult{}, fmt.Errorf("EISDIR: illegal operation on a directory, read '%s'", path)
	}
	if maxBytes != nil && st.Size() > int64(*maxBytes) {
		return ReadFileRangeResult{}, &FileTooLargeError{SizeInBytes: st.Size(), MaxSizeBytes: int64(*maxBytes)}
	}
	if !st.Mode().IsRegular() || st.Size() >= FastPathMaxSize {
		return readFileInRangeStreaming(path, st.ModTime().UnixMilli(), lineOffset, maxLines, maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ReadFileRangeResult{}, err
	}
	raw := string(data)
	return readFileInRangeFast(raw, st.ModTime().UnixMilli(), lineOffset, maxLines, nil), nil
}

func readFileInRangeFast(raw string, mtimeMs int64, offset int, maxLines *int, truncateAtBytes *int) ReadFileRangeResult {
	endLine := int(^uint(0) >> 1)
	if maxLines != nil {
		endLine = offset + *maxLines
	}
	if len(raw) > 0 {
		r, sz := utf8.DecodeRuneInString(raw)
		if r == '\ufeff' {
			raw = raw[sz:]
		}
	}
	var selected []string
	lineIndex := 0
	startPos := 0
	selectedBytes := 0
	truncated := false

	tryPush := func(line string) bool {
		if truncateAtBytes != nil {
			sep := 0
			if len(selected) > 0 {
				sep = 1
			}
			next := selectedBytes + sep + utf8ByteLen(line)
			if next > *truncateAtBytes {
				truncated = true
				return false
			}
			selectedBytes = next
		}
		selected = append(selected, line)
		return true
	}

	for {
		newlinePos := indexByteFrom(raw, '\n', startPos)
		if newlinePos < 0 {
			break
		}
		if lineIndex >= offset && lineIndex < endLine && !truncated {
			line := raw[startPos:newlinePos]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if !tryPush(line) {
				break
			}
		}
		lineIndex++
		startPos = newlinePos + 1
	}
	if lineIndex >= offset && lineIndex < endLine && !truncated {
		line := raw[startPos:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		tryPush(line)
	}
	lineIndex++

	content := strings.Join(selected, "\n")
	tb := utf8ByteLen(raw)
	rb := utf8ByteLen(content)
	return ReadFileRangeResult{
		Content:          content,
		LineCount:        len(selected),
		TotalLines:       lineIndex,
		TotalBytes:       tb,
		ReadBytes:        rb,
		MtimeMs:          mtimeMs,
		TruncatedByBytes: truncated,
	}
}

func indexByteFrom(s string, c byte, from int) int {
	if from >= len(s) {
		return -1
	}
	i := strings.IndexByte(s[from:], c)
	if i < 0 {
		return -1
	}
	return from + i
}

func utf8ByteLen(s string) int {
	return len(s)
}

// readFileInRangeStreaming handles large/non-regular files (readFileInRange.ts streaming path subset).
func readFileInRangeStreaming(path string, mtimeMs int64, lineOffset int, maxLines *int, maxBytes *int) (ReadFileRangeResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return ReadFileRangeResult{}, err
	}
	defer f.Close()

	endLine := int(^uint(0) >> 1)
	if maxLines != nil {
		endLine = lineOffset + *maxLines
	}

	sc := bufio.NewScanner(f)
	const maxLine = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, maxLine)

	lineIndex := 0
	var selected []string
	totalBytesRead := int64(0)
	truncated := false

	for sc.Scan() {
		lineB := sc.Bytes()
		totalBytesRead += int64(len(lineB)) + 1
		if maxBytes != nil && totalBytesRead > int64(*maxBytes) {
			return ReadFileRangeResult{}, &FileTooLargeError{SizeInBytes: totalBytesRead, MaxSizeBytes: int64(*maxBytes)}
		}
		line := string(lineB)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if lineIndex >= lineOffset && lineIndex < endLine && !truncated {
			selected = append(selected, line)
		}
		lineIndex++
	}
	if err := sc.Err(); err != nil {
		return ReadFileRangeResult{}, err
	}

	content := strings.Join(selected, "\n")
	return ReadFileRangeResult{
		Content:          content,
		LineCount:        len(selected),
		TotalLines:       lineIndex,
		TotalBytes:       int(totalBytesRead),
		ReadBytes:        utf8ByteLen(content),
		MtimeMs:          mtimeMs,
		TruncatedByBytes: truncated,
	}, nil
}

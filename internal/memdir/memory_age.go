package memdir

// Corresponds to restored-src/src/memdir/memoryAge.ts and attachment-header / session-fragment helpers.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const msPerDay = 24 * 60 * 60 * 1000

// MemoryAgeDaysAt returns whole days from mtime to now, floored; future mtimes clamp to 0 (memoryAge.ts).
func MemoryAgeDaysAt(mtimeMs int64, now time.Time) int {
	delta := now.UnixMilli() - mtimeMs
	if delta < 0 {
		return 0
	}
	return int(delta / msPerDay)
}

// MemoryAgeDays uses time.Now() (memoryAgeDays).
func MemoryAgeDays(mtimeMs int64) int {
	return MemoryAgeDaysAt(mtimeMs, time.Now())
}

// MemoryAgeAt returns "today", "yesterday", or "N days ago" (memoryAge).
func MemoryAgeAt(mtimeMs int64, now time.Time) string {
	d := MemoryAgeDaysAt(mtimeMs, now)
	switch d {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", d)
	}
}

// MemoryAge uses time.Now().
func MemoryAge(mtimeMs int64) string {
	return MemoryAgeAt(mtimeMs, time.Now())
}

// MemoryFreshnessTextAt returns staleness caveat for memories older than 1 day; else "" (memoryFreshnessText).
func MemoryFreshnessTextAt(mtimeMs int64, now time.Time) string {
	d := MemoryAgeDaysAt(mtimeMs, now)
	if d <= 1 {
		return ""
	}
	return fmt.Sprintf(
		"This memory is %d days old. Memories are point-in-time observations, not live state — "+
			"claims about code behavior or file:line citations may be outdated. "+
			"Verify against current code before asserting as fact.",
		d,
	)
}

// MemoryFreshnessText uses time.Now().
func MemoryFreshnessText(mtimeMs int64) string {
	return MemoryFreshnessTextAt(mtimeMs, time.Now())
}

// MemoryFreshnessNoteAt wraps MemoryFreshnessTextAt in <system-reminder> when non-empty (memoryFreshnessNote).
func MemoryFreshnessNoteAt(mtimeMs int64, now time.Time) string {
	text := MemoryFreshnessTextAt(mtimeMs, now)
	if text == "" {
		return ""
	}
	return "<system-reminder>" + text + "</system-reminder>\n"
}

// MemoryFreshnessNote uses time.Now().
func MemoryFreshnessNote(mtimeMs int64) string {
	return MemoryFreshnessNoteAt(mtimeMs, time.Now())
}

// MemoryAttachmentHeaderAt mirrors attachments.ts memoryHeader(path, mtimeMs).
func MemoryAttachmentHeaderAt(path string, mtimeMs int64, now time.Time) string {
	staleness := MemoryFreshnessTextAt(mtimeMs, now)
	if staleness != "" {
		return fmt.Sprintf("%s\n\nMemory: %s:", staleness, path)
	}
	return fmt.Sprintf("Memory (saved %s): %s:", MemoryAgeAt(mtimeMs, now), path)
}

// MemoryAttachmentHeader uses time.Now().
func MemoryAttachmentHeader(path string, mtimeMs int64) string {
	return MemoryAttachmentHeaderAt(path, mtimeMs, time.Now())
}

// SessionFragmentsFromPaths reads each path and returns trimmed text fragments plus total raw byte length.
func SessionFragmentsFromPaths(paths []string) ([]string, int, error) {
	var frags []string
	var raw int
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, 0, err
		}
		raw += len(b)
		frags = append(frags, strings.TrimSpace(string(b)))
	}
	return frags, raw, nil
}

// SessionFragmentsFromPathsWithAttachmentHeadersAt wraps each file with MemoryAttachmentHeaderAt(absPath, mtime, now).
func SessionFragmentsFromPathsWithAttachmentHeadersAt(paths []string, now time.Time) ([]string, int, error) {
	var frags []string
	var totalRaw int
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, 0, err
		}
		fi, err := os.Stat(p)
		if err != nil {
			return nil, 0, err
		}
		abs := p
		if ap, err := filepath.Abs(p); err == nil {
			abs = ap
		}
		mtime := fi.ModTime().UnixMilli()
		header := MemoryAttachmentHeaderAt(abs, mtime, now)
		body := strings.TrimSpace(string(b))
		frag := header + "\n\n" + body
		frags = append(frags, frag)
		totalRaw += len(frag)
	}
	return frags, totalRaw, nil
}

// SessionFragmentsFromPathsWithAttachmentHeaders uses time.Now() for freshness text.
func SessionFragmentsFromPathsWithAttachmentHeaders(paths []string) ([]string, int, error) {
	return SessionFragmentsFromPathsWithAttachmentHeadersAt(paths, time.Now())
}

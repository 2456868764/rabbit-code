package memdir

import (
	"bytes"
	"os"
	"time"
)

// SessionFragmentsFromPaths reads each path as UTF-8 text and returns non-empty trimmed fragments.
// Empty files are skipped. totalRawBytes is the sum of on-disk file sizes (before trim) for attachment-style budgets.
func SessionFragmentsFromPaths(paths []string) (fragments []string, totalRawBytes int, err error) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, 0, err
		}
		totalRawBytes += len(b)
		s := string(bytes.TrimSpace(b))
		if s != "" {
			fragments = append(fragments, s)
		}
	}
	return fragments, totalRawBytes, nil
}

// SessionFragmentsFromPathsWithAttachmentHeaders prepends attachments.ts memoryHeader + body per path.
// injectRawBytes is the total UTF-8 size of emitted fragments (headers + trimmed body).
func SessionFragmentsFromPathsWithAttachmentHeaders(paths []string) (fragments []string, injectRawBytes int, err error) {
	return SessionFragmentsFromPathsWithAttachmentHeadersAt(paths, time.Now())
}

// SessionFragmentsFromPathsWithAttachmentHeadersAt is SessionFragmentsFromPathsWithAttachmentHeaders with a fixed clock (tests).
func SessionFragmentsFromPathsWithAttachmentHeadersAt(paths []string, now time.Time) (fragments []string, injectRawBytes int, err error) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		st, err := os.Stat(p)
		if err != nil {
			return nil, 0, err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, 0, err
		}
		body := string(bytes.TrimSpace(b))
		if body == "" {
			continue
		}
		mtimeMs := st.ModTime().UnixMilli()
		hdr := MemoryAttachmentHeaderAt(p, mtimeMs, now)
		frag := hdr + "\n\n" + body
		fragments = append(fragments, frag)
		injectRawBytes += len(frag)
	}
	return fragments, injectRawBytes, nil
}

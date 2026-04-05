package filewritetool

import (
	"os"
	"strings"
)

// ReadNormalizedFileWithContext mirrors readFileSyncWithMetadata: normalized LF content, encoding, lineEndings.
func ReadNormalizedFileWithContext(abs string, wc *WriteContext) (normalized string, enc string, le LineEndingType, err error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", EncUTF8, LineEndingLF, err
	}
	var raw string
	enc, le, raw, err = resolveWriteEncoding(abs, b, true, wc)
	if err != nil {
		return "", EncUTF8, LineEndingLF, err
	}
	normalized = strings.ReplaceAll(raw, "\r\n", "\n")
	return normalized, enc, le, nil
}

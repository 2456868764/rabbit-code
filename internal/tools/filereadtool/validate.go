package filereadtool

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateReadInput mirrors FileReadTool.ts validateInput (no I/O except path string ops).
func ValidateReadInput(filePath string, pages *string, rc *RunContext) error {
	if pages != nil && strings.TrimSpace(*pages) != "" {
		first, last, ok := ParsePDFPageRange(*pages)
		if !ok {
			return fmt.Errorf(`Invalid pages parameter: %q. Use formats like "1-5", "3", or "10-20". Pages are 1-indexed.`, *pages)
		}
		w := PDFPageRangeWidth(first, last)
		if last < 0 {
			w = PDFMaxPagesPerRead + 1
		}
		if w > PDFMaxPagesPerRead {
			return fmt.Errorf(`Page range %q exceeds maximum of %d pages per request. Please use a smaller range.`, *pages, PDFMaxPagesPerRead)
		}
	}

	abs, err := ExpandPath(filePath)
	if err != nil {
		return err
	}

	if rc != nil && rc.DenyRead != nil && rc.DenyRead(abs) {
		return fmt.Errorf("File is in a directory that is denied by your permission settings.")
	}

	if isUncPath(abs) {
		return nil
	}

	ext := filepath.Ext(abs)
	if HasBinaryExtension(abs) && !IsPDFExtension(ext) && !isImageExt(strings.TrimPrefix(strings.ToLower(ext), ".")) {
		return fmt.Errorf("This tool cannot read binary files. The file appears to be a binary %s file. Please use appropriate tools for binary file analysis.", ext)
	}

	if isBlockedDevicePath(abs) {
		return fmt.Errorf("Cannot read '%s': this device file would block or produce infinite output.", filePath)
	}
	return nil
}

func isUncPath(p string) bool {
	return strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//")
}

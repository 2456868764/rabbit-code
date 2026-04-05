package filereadtool

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

var (
	rePDFStderrPassword = regexp.MustCompile(`(?i)password`)
	rePDFStderrCorrupt  = regexp.MustCompile(`(?i)damaged|corrupt|invalid`)
)

var (
	pdfPopplerOnce sync.Once
	pdfPopplerOK   bool
)

func isPdftoppmAvailable() bool {
	pdfPopplerOnce.Do(func() {
		p, err := exec.LookPath("pdftoppm")
		pdfPopplerOK = err == nil && p != ""
	})
	return pdfPopplerOK
}

// PDFReadResult is the successful readPDF payload (type + file) as maps for JSON.
func readPDFFile(resolvedPath, displayPath string) (map[string]any, error) {
	st, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, err
	}
	if st.Size() == 0 {
		return nil, fmt.Errorf("PDF file is empty: %s", resolvedPath)
	}
	if st.Size() > PDFTargetRawSize {
		return nil, fmt.Errorf("PDF file exceeds maximum allowed size of %s.", FormatFileSize(PDFTargetRawSize))
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}
	if len(data) < 5 || !strings.HasPrefix(string(data[:5]), "%PDF-") {
		return nil, fmt.Errorf("File is not a valid PDF (missing %%PDF- header): %s", resolvedPath)
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	return map[string]any{
		"type": "pdf",
		"file": map[string]any{
			"filePath":     displayPath,
			"base64":       b64,
			"originalSize": st.Size(),
		},
	}, nil
}

// GetPDFPageCount uses pdfcpu (mirrors getPDFPageCount when pdfinfo absent).
func GetPDFPageCount(path string) (int, error) {
	return api.PageCountFile(path)
}

// PdftoppmPageRange maps parsePDFPageRange output to pdftoppm -f/-l (last < 1 means open-ended).
func PdftoppmPageRange(firstPage, lastPage int) (from, to *int) {
	if firstPage >= 1 {
		from = new(int)
		*from = firstPage
	}
	if lastPage >= 1 {
		to = new(int)
		*to = lastPage
	}
	return from, to
}

// ExtractPDFPages runs pdftoppm into a new temp directory; returns TS-shaped "parts" payload.
func ExtractPDFPages(ctx context.Context, resolvedPath, displayPath string, fromPage, toPage *int) (map[string]any, error) {
	st, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("not a file: %s", resolvedPath)
	}
	if st.Size() == 0 {
		return nil, fmt.Errorf("PDF file is empty: %s", resolvedPath)
	}
	if st.Size() > PDFMaxExtractSize {
		return nil, fmt.Errorf("PDF file exceeds maximum allowed size for text extraction (%s).", FormatFileSize(PDFMaxExtractSize))
	}
	if !isPdftoppmAvailable() {
		return nil, fmt.Errorf("pdftoppm is not installed. Install poppler-utils (e.g. `brew install poppler` or `apt-get install poppler-utils`) to enable PDF page rendering.")
	}
	outDir, err := os.MkdirTemp("", "rabbit-pdf-*")
	if err != nil {
		return nil, err
	}
	prefix := filepath.Join(outDir, "page")
	args := []string{"-jpeg", "-r", "100"}
	if fromPage != nil && *fromPage > 0 {
		args = append(args, "-f", fmt.Sprintf("%d", *fromPage))
	}
	if toPage != nil && *toPage > 0 {
		args = append(args, "-l", fmt.Sprintf("%d", *toPage))
	}
	args = append(args, resolvedPath, prefix)

	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "pdftoppm", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(outDir)
		s := stderr.String()
		switch {
		case rePDFStderrPassword.MatchString(s):
			return nil, fmt.Errorf("PDF is password-protected. Please provide an unprotected version.")
		case rePDFStderrCorrupt.MatchString(s):
			return nil, fmt.Errorf("PDF file is corrupted or invalid.")
		default:
			return nil, fmt.Errorf("pdftoppm failed: %s", strings.TrimSpace(s))
		}
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		_ = os.RemoveAll(outDir)
		return nil, err
	}
	n := 0
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") {
			n++
		}
	}
	if n == 0 {
		_ = os.RemoveAll(outDir)
		return nil, fmt.Errorf("pdftoppm produced no output pages. The PDF may be invalid.")
	}
	return map[string]any{
		"type": "parts",
		"file": map[string]any{
			"filePath":     displayPath,
			"originalSize": st.Size(),
			"count":        n,
			"outputDir":    outDir,
		},
	}, nil
}

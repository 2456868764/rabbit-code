package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// MascotPNG returns the rabbit-code mascot as PNG bytes.
// It prefers, in order: RABBIT_CODE_MASCOT_PATH, then rabbit.png under common working-directory
// layouts (module root assets/, monorepo rabbit-code/assets/), then next to the executable,
// then a program-generated fallback when no file is found.
func MascotPNG() ([]byte, error) {
	if p := strings.TrimSpace(os.Getenv("RABBIT_CODE_MASCOT_PATH")); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("mascot: RABBIT_CODE_MASCOT_PATH: %w", err)
		}
		if !isLikelyPNG(b) {
			return nil, fmt.Errorf("mascot: RABBIT_CODE_MASCOT_PATH is not a PNG")
		}
		return b, nil
	}
	for _, p := range mascotSearchPaths() {
		b, err := os.ReadFile(p)
		if err != nil || !isLikelyPNG(b) {
			continue
		}
		return b, nil
	}
	var buf limitedWriter
	if err := encodeMascotPNG(&buf, 256); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isLikelyPNG(b []byte) bool {
	return len(b) >= len(pngMagic) && bytes.HasPrefix(b, pngMagic)
}

func mascotSearchPaths() []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "assets", "rabbit.png"))
		add(filepath.Join(wd, "rabbit-code", "assets", "rabbit.png"))
	}
	if exe, err := os.Executable(); err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		dir := filepath.Dir(exe)
		add(filepath.Join(dir, "assets", "rabbit.png"))
		add(filepath.Join(dir, "..", "assets", "rabbit.png"))
	}
	return out
}

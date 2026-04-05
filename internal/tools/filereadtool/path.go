package filereadtool

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var blockedDevicePaths = map[string]struct{}{
	"/dev/zero": {}, "/dev/random": {}, "/dev/urandom": {}, "/dev/full": {},
	"/dev/stdin": {}, "/dev/tty": {}, "/dev/console": {},
	"/dev/stdout": {}, "/dev/stderr": {},
	"/dev/fd/0": {}, "/dev/fd/1": {}, "/dev/fd/2": {},
}

// ExpandPath mirrors utils/path.ts expandPath subset: trim, ~, Abs.
func ExpandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", os.ErrInvalid
	}
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

// Narrow no-break space U+202F before AM/PM (macOS screenshot filenames).
var amPmScreenshot = regexp.MustCompile(`^(.+)([ \x{202F}])(AM|PM)(\.png)$`)

// AlternateScreenshotPath mirrors getAlternateScreenshotPath.
func AlternateScreenshotPath(filePath string) string {
	base := filepath.Base(filePath)
	m := amPmScreenshot.FindStringSubmatch(base)
	if m == nil {
		return ""
	}
	prefix, space, ampm, ext := m[1], m[2], m[3], m[4]
	var altSpace string
	if space == " " {
		altSpace = "\u202f"
	} else {
		altSpace = " "
	}
	newBase := prefix + altSpace + ampm + ext
	return filepath.Join(filepath.Dir(filePath), newBase)
}

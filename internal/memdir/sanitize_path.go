package memdir

import (
	"strconv"
	"strings"
)

// MaxSanitizedLength matches sessionStoragePortable.ts MAX_SANITIZED_LENGTH (room for hash suffix).
const MaxSanitizedLength = 200

// djb2Hash matches utils/hash.ts djb2Hash for ASCII; non-ASCII uses Unicode code points (TS uses UTF-16 code units).
func djb2Hash(s string) int32 {
	var hash int32
	for _, r := range s {
		hash = ((hash << 5) - hash + int32(r)) | 0
	}
	return hash
}

// SanitizePath makes a single filesystem path component safe (sessionStoragePortable.ts sanitizePath).
func SanitizePath(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	sanitized := b.String()
	if len(sanitized) <= MaxSanitizedLength {
		return sanitized
	}
	h := djb2Hash(name)
	if h < 0 {
		h = -h
	}
	suffix := strconv.FormatInt(int64(h), 36)
	return sanitized[:MaxSanitizedLength] + "-" + suffix
}

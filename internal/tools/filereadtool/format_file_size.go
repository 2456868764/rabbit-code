package filereadtool

import "fmt"

// FormatFileSize mirrors utils/format.ts formatFileSize (decimal SI for tool errors).
func FormatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	return fmt.Sprintf("%.1f GB", mb/1024)
}

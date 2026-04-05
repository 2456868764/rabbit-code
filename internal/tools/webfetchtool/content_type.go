package webfetchtool

import "strings"

// isBinaryContentType mirrors mcpOutputStorage.ts isBinaryContentType.
func isBinaryContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	mt := strings.TrimSpace(strings.Split(contentType, ";")[0])
	mt = strings.ToLower(mt)
	if strings.HasPrefix(mt, "text/") {
		return false
	}
	if strings.HasSuffix(mt, "+json") || mt == "application/json" {
		return false
	}
	if strings.HasSuffix(mt, "+xml") || mt == "application/xml" {
		return false
	}
	if strings.HasPrefix(mt, "application/javascript") {
		return false
	}
	if mt == "application/x-www-form-urlencoded" {
		return false
	}
	return true
}

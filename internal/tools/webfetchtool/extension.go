package webfetchtool

import "strings"

// extensionForMimeType mirrors mcpOutputStorage.ts extensionForMimeType.
func extensionForMimeType(mimeType string) string {
	if mimeType == "" {
		return "bin"
	}
	mt := strings.TrimSpace(strings.Split(mimeType, ";")[0])
	mt = strings.ToLower(mt)
	switch mt {
	case "application/pdf":
		return "pdf"
	case "application/json":
		return "json"
	case "text/csv":
		return "csv"
	case "text/plain":
		return "txt"
	case "text/html":
		return "html"
	case "text/markdown":
		return "md"
	case "application/zip":
		return "zip"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "pptx"
	case "application/msword":
		return "doc"
	case "application/vnd.ms-excel":
		return "xls"
	case "audio/mpeg":
		return "mp3"
	case "audio/wav":
		return "wav"
	case "audio/ogg":
		return "ogg"
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/svg+xml":
		return "svg"
	default:
		return "bin"
	}
}

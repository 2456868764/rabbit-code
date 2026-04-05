package filereadtool

import (
	"strings"
)

// binaryExtensionsDot mirrors constants/files.ts BINARY_EXTENSIONS (keys include leading dot).
var binaryExtensionsDot = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".ico": {}, ".webp": {}, ".tiff": {}, ".tif": {},
	".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {}, ".webm": {}, ".wmv": {}, ".flv": {}, ".m4v": {}, ".mpeg": {}, ".mpg": {},
	".mp3": {}, ".wav": {}, ".ogg": {}, ".flac": {}, ".aac": {}, ".m4a": {}, ".wma": {}, ".aiff": {}, ".opus": {},
	".zip": {}, ".tar": {}, ".gz": {}, ".bz2": {}, ".7z": {}, ".rar": {}, ".xz": {}, ".z": {}, ".tgz": {}, ".iso": {},
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".bin": {}, ".o": {}, ".a": {}, ".obj": {}, ".lib": {}, ".app": {}, ".msi": {}, ".deb": {}, ".rpm": {},
	".pdf": {}, ".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {}, ".ppt": {}, ".pptx": {}, ".odt": {}, ".ods": {}, ".odp": {},
	".ttf": {}, ".otf": {}, ".woff": {}, ".woff2": {}, ".eot": {},
	".pyc": {}, ".pyo": {}, ".class": {}, ".jar": {}, ".war": {}, ".ear": {}, ".node": {}, ".wasm": {}, ".rlib": {},
	".sqlite": {}, ".sqlite3": {}, ".db": {}, ".mdb": {}, ".idx": {},
	".psd": {}, ".ai": {}, ".eps": {}, ".sketch": {}, ".fig": {}, ".xd": {}, ".blend": {}, ".3ds": {}, ".max": {},
	".swf": {}, ".fla": {},
	".lockb": {}, ".dat": {}, ".data": {},
}

// HasBinaryExtension mirrors constants/files.ts hasBinaryExtension.
func HasBinaryExtension(filePath string) bool {
	i := strings.LastIndex(filePath, ".")
	if i < 0 {
		return false
	}
	ext := strings.ToLower(filePath[i:])
	_, ok := binaryExtensionsDot[ext]
	return ok
}

// ImageExtensions mirrors FileReadTool.ts IMAGE_EXTENSIONS (no dot keys).
var ImageExtensions = map[string]struct{}{
	"png": {}, "jpg": {}, "jpeg": {}, "gif": {}, "webp": {},
}

func isImageExt(ext string) bool {
	_, ok := ImageExtensions[strings.ToLower(ext)]
	return ok
}

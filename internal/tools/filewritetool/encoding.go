package filewritetool

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// LineEndingType mirrors fileRead.ts LineEndingType.
type LineEndingType string

const (
	LineEndingLF   LineEndingType = "LF"
	LineEndingCRLF LineEndingType = "CRLF"
)

// EncUTF8 / EncUTF16LE match Node BufferEncoding for readFileSyncWithMetadata / writeTextContent.
const (
	EncUTF8    = "utf8"
	EncUTF16LE = "utf16le"
)

const sniffLen = 4096

// DetectEncodingFromPrefix mirrors fileRead.ts detectEncodingForResolvedPath (first bytes only).
func DetectEncodingFromPrefix(b []byte) string {
	if len(b) == 0 {
		return EncUTF8
	}
	if len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe {
		return EncUTF16LE
	}
	if len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		return EncUTF8
	}
	return EncUTF8
}

// DetectLineEndingsForString mirrors fileRead.ts detectLineEndingsForString on a UTF-8 string
// (decoded file text), using at most maxRunes runes from the start (approx. TS 4096 code units).
func DetectLineEndingsForString(s string, maxRunes int) LineEndingType {
	if maxRunes <= 0 {
		return LineEndingLF
	}
	var head strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes {
			break
		}
		head.WriteRune(r)
		n++
	}
	content := head.String()
	crlfCount, lfCount := 0, 0
	for i := 0; i < len(content); i++ {
		if content[i] != '\n' {
			continue
		}
		if i > 0 && content[i-1] == '\r' {
			crlfCount++
		} else {
			lfCount++
		}
	}
	if crlfCount > lfCount {
		return LineEndingCRLF
	}
	return LineEndingLF
}

func decodeFileBytesToUTF8(b []byte, enc string) (string, error) {
	switch enc {
	case EncUTF16LE:
		raw := b
		if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
			raw = raw[2:]
		}
		if len(raw)%2 != 0 {
			return "", errors.New("filewritetool: invalid utf16le length")
		}
		u := make([]uint16, len(raw)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(raw[i*2:])
		}
		return string(utf16.Decode(u)), nil
	case EncUTF8:
		if bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}) {
			b = b[3:]
		}
		if !utf8.Valid(b) {
			return "", errors.New("filewritetool: invalid utf8")
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("filewritetool: unsupported encoding %q", enc)
	}
}

// NormalizeCRLFToLF mirrors readFileSyncWithMetadata: raw.replaceAll('\r\n', '\n').
func NormalizeCRLFToLF(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// ApplyCRLFLineEndings mirrors file.ts writeTextContent when endings === 'CRLF'.
func ApplyCRLFLineEndings(content string) string {
	s := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Join(strings.Split(s, "\n"), "\r\n")
}

func encodeUTF8ToBytes(s string) ([]byte, error) {
	if !utf8.ValidString(s) {
		return nil, errors.New("filewritetool: content is not valid utf-8")
	}
	return []byte(s), nil
}

func encodeUTF16LEToBytes(s string) ([]byte, error) {
	runes := []rune(s)
	u := utf16.Encode(runes)
	out := make([]byte, 2+len(u)*2)
	out[0], out[1] = 0xff, 0xfe
	for i, v := range u {
		binary.LittleEndian.PutUint16(out[2+i*2:], v)
	}
	return out, nil
}

// EncodeTextToFileBytes encodes UTF-8 text for writeTextContent (utf8 / utf16le).
func EncodeTextToFileBytes(content string, enc string) ([]byte, error) {
	switch enc {
	case EncUTF8:
		return encodeUTF8ToBytes(content)
	case EncUTF16LE:
		return encodeUTF16LEToBytes(content)
	default:
		return nil, fmt.Errorf("filewritetool: unsupported encoding %q", enc)
	}
}

// resolveWriteEncoding picks encoding and line endings for writeTextContent parity (fileRead.ts + file.ts).
// When hadFile is false, returns utf-8 + LF. diskDecoded is UTF-8 text from disk (before CRLF→LF normalization).
func resolveWriteEncoding(abs string, prevBytes []byte, hadFile bool, wc *WriteContext) (enc string, le LineEndingType, diskDecoded string, err error) {
	if !hadFile {
		return EncUTF8, LineEndingLF, "", nil
	}
	if wc != nil && wc.FileEncodingMetadata != nil {
		if e, l, ok := wc.FileEncodingMetadata(abs); ok && e != "" {
			diskDecoded, err = decodeFileBytesToUTF8(prevBytes, e)
			if err != nil {
				return "", LineEndingLF, "", err
			}
			return e, l, diskDecoded, nil
		}
	}
	enc = DetectEncodingFromPrefix(prevBytes)
	diskDecoded, err = decodeFileBytesToUTF8(prevBytes, enc)
	if err != nil {
		return "", LineEndingLF, "", err
	}
	le = DetectLineEndingsForString(diskDecoded, sniffLen)
	return enc, le, diskDecoded, nil
}

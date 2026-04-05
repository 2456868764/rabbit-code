package filereadtool

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxImageFileReadBytes = 50 * 1024 * 1024

func detectImageFormat(buf []byte) (ext string, mediaType string) {
	if len(buf) < 12 {
		return "png", "image/png"
	}
	switch {
	case len(buf) >= 3 && buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF:
		return "jpeg", "image/jpeg"
	case len(buf) >= 8 && string(buf[0:8]) == "\x89PNG\r\n\x1a\n":
		return "png", "image/png"
	case len(buf) >= 6 && (string(buf[0:6]) == "GIF87a" || string(buf[0:6]) == "GIF89a"):
		return "gif", "image/gif"
	case len(buf) >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP":
		return "webp", "image/webp"
	default:
		return "png", "image/png"
	}
}

func decodeImage(buf []byte) (image.Image, string, error) {
	r := bytes.NewReader(buf)
	m, _, err := image.Decode(r)
	if err != nil {
		return nil, "", err
	}
	_, mt := detectImageFormat(buf)
	return m, mt, nil
}

func resizeToMaxDim(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}
	scale := float64(maxDim) / float64(max(w, h))
	nw := int(math.Max(1, math.Round(float64(w)*scale)))
	nh := int(math.Max(1, math.Round(float64(h)*scale)))
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func encodeImageJPEG(m image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, m, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeImagePNG(m image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeImageGIF(m image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := gif.Encode(&buf, m, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageResponseMap(b64, mediaType string, originalSize int, ow, oh, dw, dh int) map[string]any {
	f := map[string]any{
		"base64":       b64,
		"type":         mediaType,
		"originalSize": originalSize,
	}
	if ow > 0 && oh > 0 {
		f["dimensions"] = map[string]any{
			"originalWidth": ow, "originalHeight": oh,
			"displayWidth": dw, "displayHeight": dh,
		}
	}
	return map[string]any{
		"type": "image",
		"file": f,
	}
}

func estimatedTokensFromBase64(b64Len int) int {
	return int(math.Ceil(float64(b64Len) * 0.125))
}

// ReadImageWithTokenBudget mirrors FileReadTool.ts readImageWithTokenBudget (single read, resize, token cap).
func ReadImageWithTokenBudget(path string, maxTokens int) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lim := maxImageFileReadBytes
	if ImageTargetRawSize > 0 && ImageTargetRawSize < lim {
		lim = ImageTargetRawSize
	}
	buf, err := io.ReadAll(io.LimitReader(f, int64(lim+1)))
	if err != nil {
		return nil, err
	}
	if len(buf) > lim {
		return nil, fmt.Errorf("image file exceeds maximum read size (%d bytes)", lim)
	}
	if len(buf) == 0 {
		return nil, fmt.Errorf("image file is empty: %s", path)
	}
	origSize := len(buf)
	m, mediaType, err := decodeImage(buf)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	b := m.Bounds()
	ow, oh := b.Dx(), b.Dy()

	resized := resizeToMaxDim(m, ImageResizeMaxDim)
	rb := resized.Bounds()
	dw, dh := rb.Dx(), rb.Dy()

	var outBytes []byte
	var outType string
	switch {
	case strings.Contains(mediaType, "png"):
		outBytes, err = encodeImagePNG(resized)
		outType = "image/png"
	case strings.Contains(mediaType, "gif"):
		outBytes, err = encodeImageGIF(resized)
		outType = "image/gif"
	default:
		outBytes, err = encodeImageJPEG(resized, 85)
		outType = "image/jpeg"
	}
	if err != nil {
		outBytes = buf
		_, outType = detectImageFormat(buf)
	}
	b64 := base64.StdEncoding.EncodeToString(outBytes)
	if maxTokens <= 0 {
		maxTokens = DefaultFileReadingLimits().MaxTokens
	}
	if estimatedTokensFromBase64(len(b64)) <= maxTokens {
		return imageResponseMap(b64, outType, origSize, ow, oh, dw, dh), nil
	}

	// Aggressive downscale (mirrors sharp fallback: 400×400 JPEG q20).
	small := resizeToMaxDim(m, 400)
	sb := small.Bounds()
	sdw, sdh := sb.Dx(), sb.Dy()
	jpegBytes, jerr := encodeImageJPEG(small, 20)
	if jerr == nil {
		b64c := base64.StdEncoding.EncodeToString(jpegBytes)
		if estimatedTokensFromBase64(len(b64c)) <= maxTokens {
			return imageResponseMap(b64c, "image/jpeg", origSize, ow, oh, sdw, sdh), nil
		}
	}
	// Last resort: return whatever we had from first encode.
	return imageResponseMap(b64, outType, origSize, ow, oh, dw, dh), nil
}

// ResizeJPEGBytesForAPI decodes JPEG bytes, downscales to ImageResizeMaxDim, re-encodes as JPEG (for pdftoppm page thumbnails in Messages API).
func ResizeJPEGBytesForAPI(data []byte) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	out := resizeToMaxDim(img, ImageResizeMaxDim)
	return encodeImageJPEG(out, 85)
}

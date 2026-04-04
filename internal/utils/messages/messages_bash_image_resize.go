// Bash shell image handling: TS resizeShellImageOutput + maybeResizeAndDownsampleImageBuffer.
package messages

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	bashMaxShellImageFileBytes = 20 * 1024 * 1024 // TS MAX_IMAGE_FILE_SIZE
	apiImageMaxBase64Bytes     = 5 * 1024 * 1024  // TS API_IMAGE_MAX_BASE64_SIZE
	imageTargetRawBytes        = (apiImageMaxBase64Bytes * 3) / 4
	imageMaxDimension          = 2000 // TS IMAGE_MAX_WIDTH / HEIGHT
)

// bashResizeShellImageOutput mirrors TS resizeShellImageOutput: read optional outputFilePath,
// parse data URI, downscale / recompress to fit API limits. Returns ok=false when the image
// should not be sent as an image block (parse failure, file too large, decode error).
func bashResizeShellImageOutput(stdout string, src map[string]any) (out string, ok bool) {
	source := strings.TrimSpace(stdout)
	path := strings.TrimSpace(strField(src, "outputFilePath"))
	if path != "" {
		st, err := os.Stat(path)
		if err != nil {
			return stdout, false
		}
		if st.Size() > bashMaxShellImageFileBytes {
			return stdout, false
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return stdout, false
		}
		source = strings.TrimSpace(string(b))
	}
	_, b64, parsed := bashParseDataURI(source)
	if !parsed || b64 == "" {
		return stdout, false
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) == 0 {
		return stdout, false
	}
	outBuf, outMT, err := bashMaybeResizeImageBuffer(raw)
	if err != nil || len(outBuf) == 0 {
		return stdout, false
	}
	enc := base64.StdEncoding.EncodeToString(outBuf)
	ext := "jpeg"
	if strings.HasPrefix(outMT, "image/") {
		ext = strings.TrimPrefix(outMT, "image/")
		if ext == "jpg" {
			ext = "jpeg"
		}
	}
	return fmt.Sprintf("data:image/%s;base64,%s", ext, enc), true
}

func bashMaybeResizeImageBuffer(raw []byte) ([]byte, string, error) {
	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	b := img.Bounds()
	origW, origH := b.Dx(), b.Dy()
	if origW <= 0 || origH <= 0 {
		return nil, "", fmt.Errorf("empty bounds")
	}
	origSize := len(raw)
	outMT := imageFormatToMIME(format)
	isPng := format == "png"
	needsDim := origW > imageMaxDimension || origH > imageMaxDimension

	// Fits as-is (TS: dimensions + raw size under target)
	if !needsDim && origW <= imageMaxDimension && origH <= imageMaxDimension && origSize <= imageTargetRawBytes {
		return raw, outMT, nil
	}

	rgbaFull := imageToRGBA(img)

	// Dimensions OK but payload too large: PNG palette + best compression, then JPEG qualities (TS order).
	if !needsDim && origSize > imageTargetRawBytes {
		if isPng {
			if buf := bashTryPNGPaletteBest(img); len(buf) > 0 && len(buf) <= imageTargetRawBytes {
				return buf, "image/png", nil
			}
		}
		if buf := bashTryJPEGQualities(rgbaFull); buf != nil {
			return buf, "image/jpeg", nil
		}
	}

	// Dimension resize (TS fit inside)
	nw, nh := origW, origH
	if needsDim {
		scale := minFloat64(float64(imageMaxDimension)/float64(origW), float64(imageMaxDimension)/float64(origH))
		nw = max(1, int(float64(origW)*scale+0.5))
		nh = max(1, int(float64(origH)*scale+0.5))
	}
	rgbaScaled := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(rgbaScaled, rgbaScaled.Bounds(), img, b, xdraw.Over, nil)

	if isPng {
		if pngBytes := bashEncodePNGBytes(rgbaScaled, true); len(pngBytes) > 0 && len(pngBytes) <= imageTargetRawBytes {
			return pngBytes, "image/png", nil
		}
	}
	if buf := bashTryJPEGQualities(rgbaScaled); buf != nil {
		return buf, "image/jpeg", nil
	}

	// Further shrink dimensions (TS fallthrough)
	for nw > 256 && nh > 256 {
		nw = max(1, nw/2)
		nh = max(1, nh/2)
		rgbaScaled = image.NewRGBA(image.Rect(0, 0, nw, nh))
		xdraw.CatmullRom.Scale(rgbaScaled, rgbaScaled.Bounds(), img, b, xdraw.Over, nil)
		if buf := bashTryJPEGQualities(rgbaScaled); buf != nil {
			return buf, "image/jpeg", nil
		}
	}
	out := rgbaToBytesJPEG(rgbaScaled, 20)
	return out, "image/jpeg", nil
}

// bashTryPNGPaletteBest mirrors sharp png({ compressionLevel: 9, palette: true }) using Plan9 palette + Floyd–Steinberg.
func bashTryPNGPaletteBest(src image.Image) []byte {
	rgba := imageToRGBA(src)
	pm := image.NewPaletted(rgba.Bounds(), palette.Plan9)
	draw.FloydSteinberg.Draw(pm, rgba.Bounds(), rgba, rgba.Bounds().Min)
	return bashEncodePalettedPNG(pm)
}

func bashEncodePalettedPNG(pm *image.Paletted) []byte {
	var buf bytes.Buffer
	enc := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, pm); err != nil {
		return nil
	}
	return buf.Bytes()
}

func bashEncodePNGBytes(img image.Image, best bool) []byte {
	var buf bytes.Buffer
	cl := png.DefaultCompression
	if best {
		cl = png.BestCompression
	}
	enc := &png.Encoder{CompressionLevel: cl}
	if err := enc.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func bashTryJPEGQualities(rgba *image.RGBA) []byte {
	for _, q := range []int{80, 60, 40, 20} {
		out := rgbaToBytesJPEG(rgba, q)
		if len(out) <= imageTargetRawBytes {
			return out
		}
	}
	return nil
}

func imageFormatToMIME(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func imageToRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)
	return rgba
}

func rgbaToBytesJPEG(img *image.RGBA, quality int) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	return buf.Bytes()
}

// bashEffectiveImageStdout applies resize when isImage is set; clears image semantics on failure (TS parity).
func bashEffectiveImageStdout(src, att map[string]any) (stdout string, isImage bool) {
	stdout = bashResolvedStdout(src, att)
	isImage = truthy(src["isImage"])
	if !isImage {
		return stdout, false
	}
	if out, ok := bashResizeShellImageOutput(stdout, src); ok {
		return out, true
	}
	return stdout, false
}

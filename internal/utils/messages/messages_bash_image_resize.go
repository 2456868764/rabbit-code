// Bash shell image handling: TS resizeShellImageOutput + maybeResizeAndDownsampleImageBuffer (simplified).
package messages

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"strings"

	xdraw "golang.org/x/image/draw"
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
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, "", fmt.Errorf("empty bounds")
	}
	outMT := imageFormatToMIME(format)
	if w <= imageMaxDimension && h <= imageMaxDimension && len(raw) <= imageTargetRawBytes {
		return raw, outMT, nil
	}

	nw, nh := w, h
	if w > imageMaxDimension || h > imageMaxDimension {
		scale := minFloat64(float64(imageMaxDimension)/float64(w), float64(imageMaxDimension)/float64(h))
		nw = max(1, int(float64(w)*scale+0.5))
		nh = max(1, int(float64(h)*scale+0.5))
	}

	var rgba *image.RGBA
	if nw != w || nh != h {
		rgba = image.NewRGBA(image.Rect(0, 0, nw, nh))
		xdraw.CatmullRom.Scale(rgba, rgba.Bounds(), img, b, xdraw.Over, nil)
	} else {
		rgba = imageToRGBA(img)
	}

	out := rgbaToBytesJPEG(rgba, 80)
	if len(out) <= imageTargetRawBytes {
		return out, "image/jpeg", nil
	}
	for _, q := range []int{60, 40, 20} {
		out = rgbaToBytesJPEG(rgba, q)
		if len(out) <= imageTargetRawBytes {
			return out, "image/jpeg", nil
		}
	}
	// Dimension shrink then retry
	for nw > 256 && nh > 256 {
		nw = max(1, nw/2)
		nh = max(1, nh/2)
		rgba = image.NewRGBA(image.Rect(0, 0, nw, nh))
		xdraw.CatmullRom.Scale(rgba, rgba.Bounds(), img, b, xdraw.Over, nil)
		out = rgbaToBytesJPEG(rgba, 20)
		if len(out) <= imageTargetRawBytes {
			return out, "image/jpeg", nil
		}
	}
	return out, "image/jpeg", nil
}

func imageFormatToMIME(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
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

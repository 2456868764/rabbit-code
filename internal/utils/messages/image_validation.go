// validateImagesForAPI parity (src/utils/imageValidation.ts).
package messages

import (
	"fmt"
	"strings"
)

const apiImageMaxBase64Size = 5 * 1024 * 1024 // API_IMAGE_MAX_BASE64_SIZE

// OversizedImage mirrors TS OversizedImage.
type OversizedImage struct {
	Index int
	Size  int
}

// ImageSizeError mirrors TS ImageSizeError.
type ImageSizeError struct {
	Oversized []OversizedImage
	MaxSize   int
}

func (e *ImageSizeError) Error() string {
	if len(e.Oversized) == 1 {
		first := e.Oversized[0]
		return fmt.Sprintf(
			"Image base64 size (%s) exceeds API limit (%s). Please resize the image before sending.",
			formatFileSizeForAPI(first.Size), formatFileSizeForAPI(e.MaxSize),
		)
	}
	var parts []string
	for _, img := range e.Oversized {
		parts = append(parts, fmt.Sprintf("Image %d: %s", img.Index, formatFileSizeForAPI(img.Size)))
	}
	return fmt.Sprintf(
		"%d images exceed the API limit (%s): %s. Please resize these images before sending.",
		len(e.Oversized), formatFileSizeForAPI(e.MaxSize), strings.Join(parts, ", "),
	)
}

func isBase64ImageBlock(block map[string]any) bool {
	if strField(block, "type") != "image" {
		return false
	}
	src, ok := block["source"].(map[string]any)
	if !ok {
		return false
	}
	if strField(src, "type") != "base64" {
		return false
	}
	data, ok := src["data"].(string)
	return ok && data != ""
}

// ValidateImagesForAPIMap returns ImageSizeError if any user message image exceeds API base64 length limit.
func ValidateImagesForAPIMap(messages []map[string]any) error {
	var oversized []OversizedImage
	imageIndex := 0
	for _, msg := range messages {
		if strField(msg, "type") != "user" {
			continue
		}
		inner, ok := msg["message"].(map[string]any)
		if !ok {
			continue
		}
		content := inner["content"]
		if _, ok := content.(string); ok {
			continue
		}
		arr, ok := content.([]any)
		if !ok {
			continue
		}
		for _, raw := range arr {
			b, ok := raw.(map[string]any)
			if !ok || !isBase64ImageBlock(b) {
				continue
			}
			imageIndex++
			src, _ := b["source"].(map[string]any)
			data, _ := src["data"].(string)
			if len(data) > apiImageMaxBase64Size {
				oversized = append(oversized, OversizedImage{Index: imageIndex, Size: len(data)})
			}
		}
	}
	if len(oversized) > 0 {
		return &ImageSizeError{Oversized: oversized, MaxSize: apiImageMaxBase64Size}
	}
	return nil
}

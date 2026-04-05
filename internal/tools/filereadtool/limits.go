package filereadtool

import (
	"os"
	"strconv"
	"strings"
)

// MaxOutputSizeBytes mirrors utils/file.ts MAX_OUTPUT_SIZE (0.25 MiB) for pre-read cap.
const MaxOutputSizeBytes = 262144

// DefaultMaxOutputTokens mirrors limits.ts DEFAULT_MAX_OUTPUT_TOKENS (post-read token cap for text/notebook).
const DefaultMaxOutputTokens = 25000

// EnvMaxOutputTokens mirrors CLAUDE_CODE_FILE_READ_MAX_OUTPUT_TOKENS (limits.ts).
const EnvMaxOutputTokens = "RABBIT_CODE_FILE_READ_MAX_OUTPUT_TOKENS"

// FileReadingLimits mirrors limits.ts FileReadingLimits (GrowthBook fields deferred).
type FileReadingLimits struct {
	MaxTokens              int
	MaxSizeBytes           int
	IncludeMaxSizeInPrompt bool
	TargetedRangeNudge     bool
}

// DefaultFileReadingLimits returns defaults (limits.ts getDefaultFileReadingLimits without GrowthBook).
func DefaultFileReadingLimits() FileReadingLimits {
	maxTok := DefaultMaxOutputTokens
	if v := strings.TrimSpace(os.Getenv(EnvMaxOutputTokens)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxTok = n
		}
	}
	return FileReadingLimits{
		MaxTokens:    maxTok,
		MaxSizeBytes: MaxOutputSizeBytes,
	}
}

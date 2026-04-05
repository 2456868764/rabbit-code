package processuserinput

import (
	"strconv"
	"strings"
)

// MaxHookOutputLength mirrors processUserInput.ts MAX_HOOK_OUTPUT_LENGTH (UserPromptSubmit hook truncation).
const MaxHookOutputLength = 10000

// TruncateHookOutput mirrors applyTruncation (suffix … + notice when over cap).
func TruncateHookOutput(content string) string {
	if len(content) <= MaxHookOutputLength {
		return content
	}
	suffix := "… [output truncated - exceeded " + strconv.Itoa(MaxHookOutputLength) + " characters]"
	return content[:MaxHookOutputLength] + suffix
}

// PlainString returns trimmed string input or empty (headless submit body).
func PlainString(input string) string {
	return strings.TrimSpace(input)
}

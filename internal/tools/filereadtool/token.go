package filereadtool

import (
	"fmt"
	"unicode/utf8"
)

// MaxFileReadTokenExceededError mirrors FileReadTool.ts MaxFileReadTokenExceededError.
type MaxFileReadTokenExceededError struct {
	TokenCount int
	MaxTokens  int
}

func (e *MaxFileReadTokenExceededError) Error() string {
	return fmt.Sprintf("File content (%d tokens) exceeds maximum allowed tokens (%d). Use offset and limit parameters to read specific portions of the file, or search for specific content instead of reading the whole file.",
		e.TokenCount, e.MaxTokens)
}

// ValidateContentTokens mirrors validateContentTokens (rough estimate + optional API count).
func ValidateContentTokens(content string, ext string, maxTokens int, count func(string) (int, error)) error {
	effectiveMax := maxTokens
	if effectiveMax <= 0 {
		effectiveMax = DefaultFileReadingLimits().MaxTokens
	}
	rough := utf8.RuneCountInString(content) / 4
	if rough <= effectiveMax/4 {
		return nil
	}
	var effective int
	if count != nil {
		n, err := count(content)
		if err != nil {
			effective = rough
		} else {
			effective = n
		}
	} else {
		effective = rough
	}
	if effective > effectiveMax {
		return &MaxFileReadTokenExceededError{TokenCount: effective, MaxTokens: effectiveMax}
	}
	return nil
}

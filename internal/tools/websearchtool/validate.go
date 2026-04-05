package websearchtool

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// ValidateInputResult mirrors WebSearchTool.validateInput return shape (errorCode 1 missing query, 2 both domains).
type ValidateInputResult struct {
	Result    bool
	Message   string
	ErrorCode int
}

// ValidateInputTS mirrors validateInput() only (no zod min(2)).
func ValidateInputTS(in Input) ValidateInputResult {
	if len(in.Query) == 0 {
		return ValidateInputResult{Result: false, Message: "Error: Missing query", ErrorCode: 1}
	}
	if len(in.AllowedDomains) > 0 && len(in.BlockedDomains) > 0 {
		return ValidateInputResult{
			Result:    false,
			Message:   "Error: Cannot specify both allowed_domains and blocked_domains in the same request",
			ErrorCode: 2,
		}
	}
	return ValidateInputResult{Result: true}
}

// ZodQueryMinLength mirrors z.string().min(2) on the raw query string (JS .length ≈ utf8.RuneCount for BMP-heavy queries).
func ZodQueryMinLength(query string) error {
	if utf8.RuneCountInString(query) < 2 {
		return errors.New(ErrQueryZodMin)
	}
	return nil
}

// ValidateInput applies validateInput + zod min(2), then returns a single error for Run.
func ValidateInput(in Input) error {
	if r := ValidateInputTS(in); !r.Result {
		return fmt.Errorf("%s", r.Message)
	}
	return ZodQueryMinLength(in.Query)
}

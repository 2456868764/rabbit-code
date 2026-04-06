package bashtool

import (
	"encoding/json"
	"strings"

	"github.com/2456868764/rabbit-code/internal/extractbash"
)

// IsExtractReadOnlyBashInputJSON is true when command is allowed under extract-style read-only rules (readOnlyValidation subset).
func IsExtractReadOnlyBashInputJSON(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return true
	}
	b, err := json.Marshal(map[string]string{"command": cmd})
	if err != nil {
		return false
	}
	return extractbash.IsReadOnlyBashInputJSON(b)
}

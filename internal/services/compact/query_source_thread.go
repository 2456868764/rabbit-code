package compact

import "strings"

// IsMainThreadQuerySource mirrors microCompact.ts isMainThreadSource (repl_main_thread prefix for output styles).
func IsMainThreadQuerySource(querySource string) bool {
	s := strings.TrimSpace(querySource)
	return s == "" || strings.HasPrefix(s, "repl_main_thread")
}

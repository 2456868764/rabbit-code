// Package version holds build metadata (overridable via -ldflags).
package version

var (
	// Version is the semantic version; overridden at link time.
	Version = "0.0.0-dev"
	// Commit is the VCS revision; overridden at link time.
	Commit = "unknown"
)

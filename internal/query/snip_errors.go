package query

import "errors"

// ErrSnipReplayUUIDMapRequired is returned when a log entry carries removedUuids but ReplaySnipRemovals has no map to resolve them.
var ErrSnipReplayUUIDMapRequired = errors.New("query: snip entry uses removedUuids; use ReplaySnipRemovalsEx with SnipReplayOptions.UUIDToIndex")

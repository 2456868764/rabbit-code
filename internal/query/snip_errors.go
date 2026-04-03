package query

import "errors"

// ErrSnipReplayUUIDMapRequired is returned when a log entry carries removedUuids but ReplaySnipRemovals has no map to resolve them.
var ErrSnipReplayUUIDMapRequired = errors.New("query: snip entry uses removedUuids; use ReplaySnipRemovalsEx with SnipReplayOptions.UUIDToIndex")

// ErrSnipNoEmbeddedUUIDs means ReplaySnipRemovalsAuto could not find rabbit_message_uuid (or custom field) on messages.
var ErrSnipNoEmbeddedUUIDs = errors.New("query: no embedded message UUID fields for removedUuids replay")

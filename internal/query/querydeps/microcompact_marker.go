package querydeps

// MicrocompactAPIStateMarker is implemented by *compact.MicrocompactEditBuffer (services/compact) without importing that package here (avoids import cycles).
type MicrocompactAPIStateMarker interface {
	MarkToolsSentToAPIState()
}

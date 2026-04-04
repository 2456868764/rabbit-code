package querydeps_test

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestMicrocompactEditBuffer_satisfiesMarker(t *testing.T) {
	var buf compact.MicrocompactEditBuffer
	var m querydeps.MicrocompactAPIStateMarker = &buf
	m.MarkToolsSentToAPIState()
}

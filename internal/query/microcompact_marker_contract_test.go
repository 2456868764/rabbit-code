package query_test

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestMicrocompactEditBuffer_satisfiesMarker(t *testing.T) {
	var buf compact.MicrocompactEditBuffer
	var m query.MicrocompactAPIStateMarker = &buf
	m.MarkToolsSentToAPIState()
}

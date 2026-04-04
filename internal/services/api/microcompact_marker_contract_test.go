package anthropic_test

import (
	"testing"

	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestMicrocompactEditBuffer_satisfiesMarker(t *testing.T) {
	var buf compact.MicrocompactEditBuffer
	var m anthropic.MicrocompactAPIStateMarker = &buf
	m.MarkToolsSentToAPIState()
}

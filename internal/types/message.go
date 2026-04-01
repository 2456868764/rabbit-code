package types

// Message is one turn in a transcript (user, assistant, system, or internal progress).
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentPiece `json:"content"`
}

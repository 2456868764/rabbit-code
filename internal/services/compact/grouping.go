package compact

// ApiRoundMessage is the subset of upstream types/message used by groupMessagesByApiRound
// (restored-src/src/services/compact/grouping.ts): type === "assistant" and message.id boundaries.
type ApiRoundMessage struct {
	Type    string `json:"type"`
	Message struct {
		ID string `json:"id"`
	} `json:"message"`
}

// GroupMessagesByApiRound mirrors grouping.ts groupMessagesByApiRound: one group per API round-trip,
// split when a new assistant message starts (message.id differs from the previous assistant).
func GroupMessagesByApiRound(messages []ApiRoundMessage) [][]ApiRoundMessage {
	var groups [][]ApiRoundMessage
	var current []ApiRoundMessage
	var lastAssistantID *string

	for _, msg := range messages {
		isAssistant := msg.Type == "assistant"
		if isAssistant && len(current) > 0 {
			sameRound := lastAssistantID != nil && msg.Message.ID == *lastAssistantID
			if !sameRound {
				groups = append(groups, current)
				current = []ApiRoundMessage{msg}
			} else {
				current = append(current, msg)
			}
		} else {
			current = append(current, msg)
		}
		if isAssistant {
			id := msg.Message.ID
			lastAssistantID = new(string)
			*lastAssistantID = id
		}
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

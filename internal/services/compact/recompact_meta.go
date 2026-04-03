package compact

// RecompactionMeta mirrors autoCompact.ts RecompactionInfo fields used before compactConversation (headless subset).
type RecompactionMeta struct {
	IsRecompactionInChain     bool
	TurnsSincePreviousCompact int
	PreviousCompactTurnID     string
	AutoCompactThreshold      int
	QuerySource               string
}

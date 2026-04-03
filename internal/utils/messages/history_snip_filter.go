package messages

import "github.com/2456868764/rabbit-code/internal/types"

// StripHistorySnipPieces removes `history_snip` content blocks from each message (P5.F.10 scrollback / API prep).
// Messages whose content becomes empty are dropped.
func StripHistorySnipPieces(msgs []types.Message) []types.Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]types.Message, 0, len(msgs))
	for _, m := range msgs {
		var keep []types.ContentPiece
		for _, p := range m.Content {
			if p.Type == types.BlockTypeHistorySnip {
				continue
			}
			keep = append(keep, p)
		}
		if len(keep) == 0 {
			continue
		}
		m.Content = keep
		out = append(out, m)
	}
	return out
}

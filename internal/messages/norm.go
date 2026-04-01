package messages

import (
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/types"
)

// NormalizeOptions controls NormalizeForAPI (P3.2.1).
type NormalizeOptions struct {
	// StripNonAPI removes internal-only content blocks and progress messages.
	StripNonAPI bool
	// ConnectorToText converts connector_text blocks to text (P3.F.2).
	ConnectorToText bool
}

// DefaultNormalizeAPI returns options suitable for sending to the Messages API.
func DefaultNormalizeAPI() NormalizeOptions {
	return NormalizeOptions{
		StripNonAPI:     true,
		ConnectorToText: true,
	}
}

// NormalizeForAPI returns a copy of messages with internal fields stripped or mapped for API use.
func NormalizeForAPI(msgs []types.Message, opt NormalizeOptions) []types.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]types.Message, 0, len(msgs))
	for _, m := range msgs {
		if opt.StripNonAPI && m.Role == types.RoleProgress {
			continue
		}
		nm := types.Message{Role: m.Role, Content: normalizeContent(m.Content, opt)}
		if len(nm.Content) == 0 && opt.StripNonAPI {
			// drop empty turns after stripping (e.g. only had history_snip)
			continue
		}
		out = append(out, nm)
	}
	return out
}

func normalizeContent(in []types.ContentPiece, opt NormalizeOptions) []types.ContentPiece {
	if len(in) == 0 {
		return nil
	}
	out := make([]types.ContentPiece, 0, len(in))
	for _, c := range in {
		switch c.Type {
		case types.BlockTypeText, types.BlockTypeToolUse, types.BlockTypeToolResult:
			out = append(out, c)
		case types.BlockTypeConnectorText:
			if opt.ConnectorToText {
				out = append(out, types.ContentPiece{Type: types.BlockTypeText, Text: c.Text})
			} else {
				out = append(out, c)
			}
		case types.BlockTypeFileRef:
			if !opt.StripNonAPI {
				out = append(out, c)
			}
			// Stripped for API until Phase 4 maps to document blocks.
		case types.BlockTypeBoundary, types.BlockTypeTombstone, types.BlockTypeHistorySnip,
			types.BlockTypeCompactionReminder, types.BlockTypeKairosQueue, types.BlockTypeKairosChannel,
			types.BlockTypeKairosBrief, types.BlockTypeUDSInbox, types.BlockTypeProgress:
			if !opt.StripNonAPI {
				out = append(out, c)
			}
		default:
			// Unknown types: keep if not stripping, else drop for safety
			if !opt.StripNonAPI {
				out = append(out, c)
			}
		}
	}
	return out
}

// StripToolResultSignature is a no-op placeholder for signature stripping (PARITY with TS).
func StripToolResultSignature(raw json.RawMessage) json.RawMessage {
	return raw
}

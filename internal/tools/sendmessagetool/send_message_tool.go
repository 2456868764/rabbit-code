package sendmessagetool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// SendMessage implements tools.Tool for SendMessageTool.ts.
type SendMessage struct{}

// New returns a SendMessage tool.
func New() *SendMessage { return &SendMessage{} }

func (s *SendMessage) Name() string      { return SendMessageToolName }
func (s *SendMessage) Aliases() []string { return nil }

// --- Structured message types (discriminated union on "type") ---

// StructuredMessageType enumerates the type values for StructuredMessage.
type StructuredMessageType string

const (
	StructuredMessageShutdownRequest    StructuredMessageType = "shutdown_request"
	StructuredMessageShutdownResponse   StructuredMessageType = "shutdown_response"
	StructuredMessagePlanApprovalResponse StructuredMessageType = "plan_approval_response"
)

// StructuredMessage mirrors StructuredMessage discriminated union in SendMessageTool.ts.
type StructuredMessage struct {
	Type      StructuredMessageType `json:"type"`
	RequestID string                `json:"request_id,omitempty"`
	Approve   *bool                 `json:"approve,omitempty"`
	Reason    string                `json:"reason,omitempty"`
	Feedback  string                `json:"feedback,omitempty"`
}

// rawInput is used for JSON decoding; message can be string or StructuredMessage.
type rawInput struct {
	To      string          `json:"to"`
	Summary string          `json:"summary,omitempty"`
	Message json.RawMessage `json:"message"`
}

// Input mirrors SendMessageTool.ts Input.
type Input struct {
	To      string
	Summary string
	// Message is either a string or a StructuredMessage.
	// Only one of TextMessage/Structured will be non-zero.
	TextMessage string
	Structured  *StructuredMessage
}

// UnmarshalJSON implements custom unmarshalling for Input (message is string | StructuredMessage).
func unmarshalInput(data []byte) (Input, error) {
	var r rawInput
	if err := json.Unmarshal(data, &r); err != nil {
		return Input{}, err
	}
	in := Input{To: r.To, Summary: r.Summary}
	if len(r.Message) == 0 {
		return Input{}, fmt.Errorf("sendmessagetool: message is required")
	}
	// Try string first.
	var s string
	if err := json.Unmarshal(r.Message, &s); err == nil {
		in.TextMessage = s
		return in, nil
	}
	// Try StructuredMessage.
	var sm StructuredMessage
	if err := json.Unmarshal(r.Message, &sm); err != nil {
		return Input{}, fmt.Errorf("sendmessagetool: message must be a string or structured message object")
	}
	in.Structured = &sm
	return in, nil
}

// --- Output types ---

// MessageRouting mirrors MessageRouting in SendMessageTool.ts.
type MessageRouting struct {
	Sender      string `json:"sender"`
	SenderColor string `json:"senderColor,omitempty"`
	Target      string `json:"target"`
	TargetColor string `json:"targetColor,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Content     string `json:"content,omitempty"`
}

// MessageOutput mirrors MessageOutput in SendMessageTool.ts.
type MessageOutput struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Routing *MessageRouting `json:"routing,omitempty"`
}

// BroadcastOutput mirrors BroadcastOutput in SendMessageTool.ts.
type BroadcastOutput struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	Recipients []string        `json:"recipients"`
	Routing    *MessageRouting `json:"routing,omitempty"`
}

// RequestOutput mirrors RequestOutput in SendMessageTool.ts.
type RequestOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Target    string `json:"target"`
}

// ResponseOutput mirrors ResponseOutput in SendMessageTool.ts.
type ResponseOutput struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// --- RunContext ---

// RunContext carries message delivery callbacks.
// Set SendMsg to wire actual mailbox writing / in-process agent message queuing.
type RunContext struct {
	// SendMsg delivers the message and returns JSON-encoded output (one of the Output types).
	// When nil a stub result is returned.
	SendMsg func(ctx context.Context, in Input) ([]byte, error)
}

type runCtxKey struct{}

// WithRunContext attaches *RunContext for SendMessage.Run.
func WithRunContext(ctx context.Context, rc *RunContext) context.Context {
	if rc == nil {
		return ctx
	}
	return context.WithValue(ctx, runCtxKey{}, rc)
}

// RunContextFrom returns *RunContext or nil.
func RunContextFrom(ctx context.Context) *RunContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(runCtxKey{}).(*RunContext)
	return v
}

// Run implements tools.Tool.
// Input: { to, summary?, message: string | StructuredMessage }.
// Output: JSON-encoded MessageOutput | BroadcastOutput | RequestOutput | ResponseOutput.
func (s *SendMessage) Run(ctx context.Context, inputJSON []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Validate JSON structure first (no unknown fields at top level).
	dec := json.NewDecoder(bytes.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("sendmessagetool: invalid json: %w", err)
	}
	for k := range raw {
		switch k {
		case "to", "summary", "message":
		default:
			return nil, fmt.Errorf("sendmessagetool: unknown field %q", k)
		}
	}

	in, err := unmarshalInput(inputJSON)
	if err != nil {
		return nil, err
	}
	if in.To == "" {
		return nil, fmt.Errorf("sendmessagetool: to is required")
	}
	if in.TextMessage == "" && in.Structured == nil {
		return nil, fmt.Errorf("sendmessagetool: message is required")
	}

	rc := RunContextFrom(ctx)
	if rc != nil && rc.SendMsg != nil {
		return rc.SendMsg(ctx, in)
	}

	// Stub: return a generic success.
	out := MessageOutput{
		Success: true,
		Message: fmt.Sprintf("Message sent to %s.", in.To),
	}
	return json.Marshal(out)
}

// MapResultForMessagesAPI mirrors mapToolResultToToolResultBlockParam in SendMessageTool.ts.
// Returns the JSON-stringified output as the tool_result content.
func MapResultForMessagesAPI(outJSON []byte) string {
	return string(outJSON)
}

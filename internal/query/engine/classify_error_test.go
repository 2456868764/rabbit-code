package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/api"
)

func TestClassifyAnthropicError_wrapped(t *testing.T) {
	inner := &anthropic.APIError{Kind: anthropic.KindMaxOutputTokens, Status: 400, Msg: "too long"}
	err := fmt.Errorf("stream: %w", inner)
	k, rec := classifyAnthropicError(err)
	if k != string(anthropic.KindMaxOutputTokens) || !rec {
		t.Fatalf("%q %v", k, rec)
	}
}

func TestClassifyAnthropicError_plain(t *testing.T) {
	k, rec := classifyAnthropicError(errors.New("nope"))
	if k != "" || rec {
		t.Fatalf("%q %v", k, rec)
	}
}

func TestClassifyAnthropicError_blockingLimit(t *testing.T) {
	k, rec := classifyAnthropicError(query.ErrBlockingLimit)
	if k != "blocking_limit" || rec {
		t.Fatalf("%q %v", k, rec)
	}
}

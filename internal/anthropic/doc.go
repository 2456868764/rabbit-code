// Package anthropic implements the Anthropic Messages HTTP client surface aligned with
// claude-code-sourcemap/restored-src/src/services/api (streaming, retry, errors, auth, betas, preconnect).
//
// Backpressure (P4.1.2 / AC4-1c): StreamEvents uses a bounded channel; the SSE reader goroutine
// blocks on send when the buffer is full, so a slow consumer does not grow unbounded memory.
//
// Bedrock: betas in BEDROCK_EXTRA_PARAMS_HEADERS are JSON-encoded as "anthropic_beta" on the request
// body when using Client.SetBetaNames (HTTP anthropic-beta gets the remainder).
//
// Proxy (P4.3.3): HTTPTransportWithProxyFromEnv (and HTTPTransportWithProxyFromEnvAndRoots for Bootstrap TLS pool)
// as the base RoundTripper for NewTransportChain / NewClient.
// mTLS: HTTPTransportAPIOutbound / HTTPTransportAPIOutboundWithRoots load CLAUDE_CODE_CLIENT_CERT + CLAUDE_CODE_CLIENT_KEY when both are set.
// ReadAssistantStream: optional WithThinkingAccumulator / WithCompactionAccumulator / WithToolInputAccumulators (Client fields).
// Retry: Policy.StrictForeground529 mirrors withRetry.ts FOREGROUND_529_RETRY_SOURCES; RABBIT_CODE_STRICT_FOREGROUND_529 enables it in DefaultPolicy.
// Policy.InitialConsecutive529Errors mirrors withRetry.ts initialConsecutive529Errors (529 budget pre-seed).
package anthropic

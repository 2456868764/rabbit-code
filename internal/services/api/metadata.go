package anthropic

import (
	"encoding/json"
	"os"
	"strings"
)

// EnvClaudeCodeExtraMetadata mirrors process.env.CLAUDE_CODE_EXTRA_METADATA (claude.ts getAPIMetadata).
const EnvClaudeCodeExtraMetadata = "CLAUDE_CODE_EXTRA_METADATA"

// EnvRabbitDeviceID when set overrides file-backed device id (LoadOrCreateDeviceID / getOrCreateUserID parity).
const EnvRabbitDeviceID = "RABBIT_CODE_DEVICE_ID"

// EnvRabbitOAuthAccountUUID optional OAuth account UUID (upstream getOauthAccountInfo()?.accountUuid).
const EnvRabbitOAuthAccountUUID = "RABBIT_CODE_OAUTH_ACCOUNT_UUID"

// BuildMessagesAPIMetadata returns JSON {"user_id":"<stringified-inner-json>"} matching claude.ts getAPIMetadata.
// Inner object merges CLAUDE_CODE_EXTRA_METADATA first, then device_id, account_uuid, session_id (latter override).
func BuildMessagesAPIMetadata(c *Client) (json.RawMessage, error) {
	inner := make(map[string]any)
	if extra := strings.TrimSpace(os.Getenv(EnvClaudeCodeExtraMetadata)); extra != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(extra), &parsed); err == nil && parsed != nil {
			for k, v := range parsed {
				inner[k] = v
			}
		}
	}
	inner["device_id"] = LoadOrCreateDeviceID()
	inner["account_uuid"] = strings.TrimSpace(os.Getenv(EnvRabbitOAuthAccountUUID))
	sid := ""
	if c != nil {
		sid = strings.TrimSpace(c.SessionID)
	}
	inner["session_id"] = sid

	innerBytes, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	// user_id is a JSON string whose value is the stringified inner object (same as jsonStringify in TS).
	out, err := json.Marshal(map[string]any{"user_id": string(innerBytes)})
	if err != nil {
		return nil, err
	}
	return out, nil
}

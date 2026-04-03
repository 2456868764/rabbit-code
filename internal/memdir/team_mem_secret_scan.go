package memdir

import (
	"regexp"
	"strings"
	"sync"
)

// SecretMatch records a gitleaks-style rule hit (secretScanner.ts); matched text is never stored.
type SecretMatch struct {
	RuleID string
	Label  string
}

type compiledSecretRule struct {
	id string
	re *regexp.Regexp
}

var (
	secretRulesOnce sync.Once
	secretRules     []compiledSecretRule
)

// ScanTeamMemorySecrets returns one match per rule that fires (deduped by rule id), mirroring secretScanner.scanForSecrets.
func ScanTeamMemorySecrets(content string) []SecretMatch {
	secretRulesOnce.Do(initTeamMemSecretRules)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	var out []SecretMatch
	seen := map[string]struct{}{}
	for _, r := range secretRules {
		if _, ok := seen[r.id]; ok {
			continue
		}
		if r.re.MatchString(content) {
			seen[r.id] = struct{}{}
			out = append(out, SecretMatch{RuleID: r.id, Label: secretRuleIDToLabel(r.id)})
		}
	}
	return out
}

func initTeamMemSecretRules() {
	antPfx := strings.Join([]string{"sk", "ant", "api"}, "-")
	bt := "`"
	// Boundary suffix: backtick, quotes, whitespace, semicolon, or \n/\r escapes (secretScanner.ts).
	sfx := "(?:[" + bt + `'";\s]|\n|\r|\\[nr]|$)`

	raw := []struct{ id, src string }{
		{"aws-access-token", `\b((?:A3T[A-Z0-9]|AKIA|ASIA|ABIA|ACCA)[A-Z2-7]{16})\b`},
		{"gcp-api-key", `\b(AIza[\w-]{35})` + sfx},
		{"digitalocean-pat", `\b(dop_v1_[a-f0-9]{64})` + sfx},
		{"digitalocean-access-token", `\b(doo_v1_[a-f0-9]{64})` + sfx},
		{"anthropic-api-key", `\b(` + antPfx + `03-[a-zA-Z0-9_\-]{93}AA)` + sfx},
		{"anthropic-admin-api-key", `\b(sk-ant-admin01-[a-zA-Z0-9_\-]{93}AA)` + sfx},
		{"huggingface-access-token", `\b(hf_[a-zA-Z]{34})` + sfx},
		{"github-pat", `ghp_[0-9a-zA-Z]{36}`},
		{"github-fine-grained-pat", `github_pat_\w{82}`},
		{"github-app-token", `(?:ghu|ghs)_[0-9a-zA-Z]{36}`},
		{"github-oauth", `gho_[0-9a-zA-Z]{36}`},
		{"github-refresh-token", `ghr_[0-9a-zA-Z]{36}`},
		{"gitlab-pat", `glpat-[\w-]{20}`},
		{"gitlab-deploy-token", `gldt-[0-9a-zA-Z_\-]{20}`},
		{"slack-bot-token", `xoxb-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`},
		{"npm-access-token", `\b(npm_[a-zA-Z0-9]{36})` + sfx},
		{"databricks-api-token", `\b(dapi[a-f0-9]{32}(?:-\d)?)` + sfx},
		{"pulumi-api-token", `\b(pul-[a-f0-9]{40})` + sfx},
		{"stripe-access-token", `\b((?:sk|rk)_(?:test|live|prod)_[a-zA-Z0-9]{10,99})` + sfx},
		{"shopify-access-token", `shpat_[a-fA-F0-9]{32}`},
		{"shopify-shared-secret", `shpss_[a-fA-F0-9]{32}`},
		{"twilio-api-key", `SK[0-9a-fA-F]{32}`},
		{"private-key", `(?is)-----BEGIN[ A-Z0-9_-]{0,100}PRIVATE KEY(?: BLOCK)?-----[\s\S-]{64,}?-----END[ A-Z0-9_-]{0,100}PRIVATE KEY(?: BLOCK)?-----`},
	}
	for _, r := range raw {
		secretRules = append(secretRules, compiledSecretRule{id: r.id, re: regexp.MustCompile(r.src)})
	}
}

func secretRuleIDToLabel(ruleID string) string {
	parts := strings.Split(ruleID, "-")
	special := map[string]string{
		"aws": "AWS", "gcp": "GCP", "api": "API", "pat": "PAT", "ad": "AD",
		"tf": "TF", "oauth": "OAuth", "npm": "NPM", "pypi": "PyPI", "jwt": "JWT",
		"github": "GitHub", "gitlab": "GitLab", "openai": "OpenAI",
		"digitalocean": "DigitalOcean", "huggingface": "HuggingFace",
		"hashicorp": "HashiCorp", "sendgrid": "SendGrid",
	}
	for i, p := range parts {
		if s, ok := special[p]; ok {
			parts[i] = s
			continue
		}
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

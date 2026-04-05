package webfetchtool

import "strings"

// MakeSecondaryModelPrompt mirrors prompt.ts makeSecondaryModelPrompt (user message to Haiku).
func MakeSecondaryModelPrompt(markdownContent, prompt string, isPreapprovedDomain bool) string {
	var guidelines string
	if isPreapprovedDomain {
		guidelines = `Provide a concise response based on the content above. Include relevant details, code examples, and documentation excerpts as needed.`
	} else {
		guidelines = `Provide a concise response based only on the content above. In your response:
 - Enforce a strict 125-character maximum for quotes from any source document. Open Source Software is ok as long as we respect the license.
 - Use quotation marks for exact language from articles; any language outside of the quotation should never be word-for-word the same.
 - You are not a lawyer and never comment on the legality of your own prompts and responses.
 - Never produce or reproduce exact song lyrics.`
	}
	var b strings.Builder
	b.WriteString("\nWeb page content:\n---\n")
	b.WriteString(markdownContent)
	b.WriteString("\n---\n\n")
	b.WriteString(prompt)
	b.WriteString("\n\n")
	b.WriteString(guidelines)
	b.WriteString("\n")
	return b.String()
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	// Cut by bytes is OK for limit; avoid splitting UTF-8.
	for len(s) > max {
		s = s[:max]
		for len(s) > 0 && s[len(s)-1]&0xC0 == 0x80 {
			s = s[:len(s)-1]
		}
		break
	}
	return s
}

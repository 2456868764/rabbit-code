package webfetchtool

import (
	"regexp"
	"strings"
)

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTags   = regexp.MustCompile(`<[^>]+>`)
	reWS     = regexp.MustCompile(`[ \t\r\n]+`)
)

// htmlToPlainText is a lightweight HTML → readable text step (upstream uses Turndown).
func htmlToPlainText(html string) string {
	s := reScript.ReplaceAllString(html, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n").Replace(s)
	s = reTags.ReplaceAllString(s, " ")
	s = reWS.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

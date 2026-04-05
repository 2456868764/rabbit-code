package webfetchtool

import htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"

// htmlToMarkdown mirrors WebFetch Turndown path; falls back to htmlToPlainText in the caller on error.
func htmlToMarkdown(html string) (string, error) {
	return htmltomarkdown.ConvertString(html)
}

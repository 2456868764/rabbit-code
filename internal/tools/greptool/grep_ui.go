package greptool

import (
	"encoding/json"
	"strconv"
	"strings"
)

// MapGrepToolResultForMessagesAPI mirrors GrepTool.ts mapToolResultToToolResultBlockParam (headless string).
func MapGrepToolResultForMessagesAPI(out []byte) string {
	var o struct {
		Mode          string   `json:"mode"`
		NumFiles      int      `json:"numFiles"`
		Filenames     []string `json:"filenames"`
		Content       string   `json:"content"`
		NumMatches    int      `json:"numMatches"`
		AppliedLimit  *int     `json:"appliedLimit"`
		AppliedOffset *int     `json:"appliedOffset"`
	}
	if err := json.Unmarshal(out, &o); err != nil {
		return ""
	}
	mode := o.Mode
	if mode == "" {
		mode = "files_with_matches"
	}
	limitInfo := formatLimitInfo(o.AppliedLimit, o.AppliedOffset)

	switch mode {
	case "content":
		resultContent := o.Content
		if resultContent == "" {
			resultContent = "No matches found"
		}
		if limitInfo != "" {
			return resultContent + "\n\n[Showing results with pagination = " + limitInfo + "]"
		}
		return resultContent
	case "count":
		raw := o.Content
		if raw == "" {
			raw = "No matches found"
		}
		matches := o.NumMatches
		files := o.NumFiles
		summary := "\n\nFound " + strconv.Itoa(matches) + " total "
		if matches == 1 {
			summary += "occurrence"
		} else {
			summary += "occurrences"
		}
		summary += " across " + strconv.Itoa(files) + " "
		if files == 1 {
			summary += "file."
		} else {
			summary += "files."
		}
		if limitInfo != "" {
			summary += " with pagination = " + limitInfo
		}
		return raw + summary
	default: // files_with_matches
		if o.NumFiles == 0 {
			return "No files found"
		}
		plural := "files"
		if o.NumFiles == 1 {
			plural = "file"
		}
		line1 := "Found " + strconv.Itoa(o.NumFiles) + " " + plural
		if limitInfo != "" {
			line1 += " " + limitInfo
		}
		return line1 + "\n" + strings.Join(o.Filenames, "\n")
	}
}

func formatLimitInfo(appliedLimit, appliedOffset *int) string {
	var parts []string
	if appliedLimit != nil {
		parts = append(parts, "limit: "+strconv.Itoa(*appliedLimit))
	}
	if appliedOffset != nil && *appliedOffset > 0 {
		parts = append(parts, "offset: "+strconv.Itoa(*appliedOffset))
	}
	return strings.Join(parts, ", ")
}

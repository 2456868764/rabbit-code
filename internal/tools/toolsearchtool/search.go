package toolsearchtool

import (
	"regexp"
	"sort"
	"strings"
)

type parsedName struct {
	parts []string
	full  string
	isMcp bool
}

func parseToolName(name string) parsedName {
	if strings.HasPrefix(name, "mcp__") {
		without := strings.ToLower(strings.TrimPrefix(name, "mcp__"))
		var parts []string
		for _, seg := range strings.Split(without, "__") {
			for _, sub := range strings.Split(seg, "_") {
				if sub != "" {
					parts = append(parts, sub)
				}
			}
		}
		full := strings.ReplaceAll(strings.ReplaceAll(without, "__", " "), "_", " ")
		return parsedName{parts: parts, full: full, isMcp: true}
	}
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	spaced := re.ReplaceAllString(name, "$1 $2")
	spaced = strings.ReplaceAll(spaced, "_", " ")
	fields := strings.Fields(strings.ToLower(spaced))
	return parsedName{parts: fields, full: strings.Join(fields, " "), isMcp: false}
}

func compileTermPatterns(terms []string) map[string]*regexp.Regexp {
	m := make(map[string]*regexp.Regexp)
	for _, term := range terms {
		if _, ok := m[term]; ok {
			continue
		}
		m[term] = regexp.MustCompile(`\b` + regexp.QuoteMeta(term) + `\b`)
	}
	return m
}

func partExact(parts []string, term string) bool {
	for _, p := range parts {
		if p == term {
			return true
		}
	}
	return false
}

func partSub(parts []string, term string) bool {
	for _, p := range parts {
		if strings.Contains(p, term) {
			return true
		}
	}
	return false
}

func scoreOneTool(parsed parsedName, descNorm, hintNorm string, terms []string, patterns map[string]*regexp.Regexp) int {
	score := 0
	for _, term := range terms {
		pat := patterns[term]
		if partExact(parsed.parts, term) {
			if parsed.isMcp {
				score += 12
			} else {
				score += 10
			}
		} else if partSub(parsed.parts, term) {
			if parsed.isMcp {
				score += 6
			} else {
				score += 5
			}
		}
		if strings.Contains(parsed.full, term) && score == 0 {
			score += 3
		}
		if hintNorm != "" && pat != nil && pat.MatchString(hintNorm) {
			score += 4
		}
		if pat != nil && pat.MatchString(descNorm) {
			score += 2
		}
	}
	return score
}

func toolMatchesRequired(tool ToolEntry, full ToolEntry, required []string, patterns map[string]*regexp.Regexp) bool {
	parsed := parseToolName(tool.Name)
	desc := strings.ToLower(full.Description)
	hint := strings.ToLower(full.SearchHint)
	for _, term := range required {
		pat := patterns[term]
		ok := partExact(parsed.parts, term) || partSub(parsed.parts, term)
		if !ok && pat != nil {
			ok = pat.MatchString(desc) || (hint != "" && pat.MatchString(hint))
		}
		if !ok {
			return false
		}
	}
	return true
}

// searchToolsWithKeywords mirrors ToolSearchTool.ts searchToolsWithKeywords.
func searchToolsWithKeywords(query string, deferred, all []ToolEntry, maxResults int) []string {
	queryLower := strings.ToLower(strings.TrimSpace(query))
	byName := catalogByName(all)

	for _, t := range deferred {
		if strings.ToLower(t.Name) == queryLower {
			return []string{t.Name}
		}
	}
	for _, t := range all {
		if strings.ToLower(t.Name) == queryLower {
			return []string{t.Name}
		}
	}

	if strings.HasPrefix(queryLower, "mcp__") && len(queryLower) > 5 {
		var ms []string
		for _, t := range deferred {
			if strings.HasPrefix(strings.ToLower(t.Name), queryLower) {
				ms = append(ms, t.Name)
				if len(ms) >= maxResults {
					return ms
				}
			}
		}
		if len(ms) > 0 {
			return ms
		}
	}

	queryTerms := strings.Fields(queryLower)
	var required, optional []string
	for _, term := range queryTerms {
		if strings.HasPrefix(term, "+") && len(term) > 1 {
			required = append(required, term[1:])
		} else {
			optional = append(optional, term)
		}
	}
	allScoringTerms := queryTerms
	if len(required) > 0 {
		allScoringTerms = append(append([]string{}, required...), optional...)
	}
	termPatterns := compileTermPatterns(allScoringTerms)

	candidateTools := deferred
	if len(required) > 0 {
		var filtered []ToolEntry
		for _, tool := range deferred {
			full := byName[tool.Name]
			if full.Name == "" {
				full = tool
			}
			if toolMatchesRequired(tool, full, required, termPatterns) {
				filtered = append(filtered, tool)
			}
		}
		candidateTools = filtered
	}

	type scored struct {
		name  string
		score int
	}
	var ranked []scored
	for _, tool := range candidateTools {
		full := byName[tool.Name]
		if full.Name == "" {
			full = tool
		}
		parsed := parseToolName(tool.Name)
		desc := strings.ToLower(full.Description)
		hint := strings.ToLower(full.SearchHint)
		s := scoreOneTool(parsed, desc, hint, allScoringTerms, termPatterns)
		if s > 0 {
			ranked = append(ranked, scored{tool.Name, s})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].name < ranked[j].name
	})
	var out []string
	for i := 0; i < len(ranked) && len(out) < maxResults; i++ {
		out = append(out, ranked[i].name)
	}
	return out
}

var selectPrefixRe = regexp.MustCompile(`(?i)^select:(.+)$`)

func parseSelectQuery(query string) (names []string, ok bool) {
	m := selectPrefixRe.FindStringSubmatch(strings.TrimSpace(query))
	if m == nil {
		return nil, false
	}
	for _, p := range strings.Split(m[1], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names, len(names) > 0
}

// toolMatchesName mirrors Tool.ts toolMatchesName (exact primary name or alias).
func toolMatchesName(e ToolEntry, name string) bool {
	if e.Name == name {
		return true
	}
	for _, a := range e.Aliases {
		if a == name {
			return true
		}
	}
	return false
}

// findToolByName mirrors Tool.ts findToolByName on a slice of ToolEntry.
func findToolByName(entries []ToolEntry, name string) string {
	for _, e := range entries {
		if toolMatchesName(e, name) {
			return e.Name
		}
	}
	return ""
}

func resolveSelect(names []string, deferred, all []ToolEntry) (found, missing []string) {
	seen := make(map[string]struct{})
	for _, raw := range names {
		hit := findToolByName(deferred, raw)
		if hit == "" {
			hit = findToolByName(all, raw)
		}
		if hit == "" {
			missing = append(missing, raw)
			continue
		}
		if _, ok := seen[hit]; ok {
			continue
		}
		seen[hit] = struct{}{}
		found = append(found, hit)
	}
	return found, missing
}

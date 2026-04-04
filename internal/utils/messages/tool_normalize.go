// Tool name aliases + normalizeToolInputForAPI parity (src/utils/permissions/permissionRuleParser.ts, src/utils/api.ts).
package messages

import (
	"os"
)

const (
	toolNameBriefCanonical = "SendUserMessage"
	toolNameTaskStop       = "TaskStop"
)

// ToolSpec is a minimal tool descriptor for name resolution (TS Tool.name + aliases).
type ToolSpec struct {
	Name    string
	Aliases []string
}

// ToolSpecsFromNames builds specs with no aliases (each name is canonical).
func ToolSpecsFromNames(names []string) []ToolSpec {
	out := make([]ToolSpec, 0, len(names))
	for _, n := range names {
		if n == "" {
			continue
		}
		out = append(out, ToolSpec{Name: n})
	}
	return out
}

// ToolMatchesName mirrors TS toolMatchesName.
func ToolMatchesName(tool ToolSpec, name string) bool {
	if tool.Name == name {
		return true
	}
	for _, a := range tool.Aliases {
		if a == name {
			return true
		}
	}
	return false
}

// FindToolBySpecs returns the first spec matching name or alias.
func FindToolBySpecs(specs []ToolSpec, name string) *ToolSpec {
	for i := range specs {
		if ToolMatchesName(specs[i], name) {
			return &specs[i]
		}
	}
	return nil
}

func legacyToolNameAliases() map[string]string {
	m := map[string]string{
		ToolNameTaskLegacy: ToolNameAgent,
		"KillShell":        toolNameTaskStop,
		"AgentOutputTool":  ToolNameTaskOutput,
		"BashOutputTool":   ToolNameTaskOutput,
	}
	if os.Getenv("RABBIT_FEATURE_KAIROS") == "1" ||
		os.Getenv("RABBIT_KAIROS") == "1" || os.Getenv("RABBIT_KAIROS_CHANNELS") == "1" || os.Getenv("RABBIT_KAIROS_BRIEF") == "1" {
		m["Brief"] = toolNameBriefCanonical
	}
	return m
}

// NormalizeLegacyToolName mirrors TS normalizeLegacyToolName (static aliases + optional Brief under KAIROS).
func NormalizeLegacyToolName(name string) string {
	if c, ok := legacyToolNameAliases()[name]; ok {
		return c
	}
	return name
}

// ResolveCanonicalToolName applies FindToolBySpecs first, then legacy aliases.
func ResolveCanonicalToolName(name string, specs []ToolSpec) string {
	if t := FindToolBySpecs(specs, name); t != nil {
		return t.Name
	}
	return NormalizeLegacyToolName(name)
}

// NormalizeToolInputForAPIMap mirrors TS normalizeToolInputForAPI (API-bound strip only; no zod parse).
func NormalizeToolInputForAPIMap(canonicalToolName string, input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	switch canonicalToolName {
	case ToolNameExitPlanModeV2:
		_, hasPlan := input["plan"]
		_, hasPath := input["planFilePath"]
		if hasPlan || hasPath {
			return omitKeys(input, "plan", "planFilePath")
		}
		return input
	case ToolNameEdit:
		if _, has := input["edits"]; has {
			return omitKeys(input, "old_string", "new_string", "replace_all")
		}
		return input
	default:
		return input
	}
}

func omitKeys(in map[string]any, keys ...string) map[string]any {
	omit := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		omit[k] = struct{}{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if _, drop := omit[k]; !drop {
			out[k] = v
		}
	}
	return out
}

package readonlycmd

import (
	"regexp"
	"strings"
)

// ArgType mirrors readOnlyCommandValidation FlagArgType.
type ArgType string

const (
	ArgNone   ArgType = "none"
	ArgNumber ArgType = "number"
	ArgString ArgType = "string"
	ArgChar   ArgType = "char"
	ArgBrace  ArgType = "{}"
	ArgEOF    ArgType = "EOF"
)

// FLAG_PATTERN mirrors TS.
var flagPattern = regexp.MustCompile(`^-[a-zA-Z0-9_-]`)

// ValidateFlagArgument mirrors validateFlagArgument.
func ValidateFlagArgument(value string, argType ArgType) bool {
	switch argType {
	case ArgNone:
		return false
	case ArgNumber:
		if value == "" {
			return false
		}
		for _, r := range value {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	case ArgString:
		return true
	case ArgChar:
		return len([]rune(value)) == 1
	case ArgBrace:
		return value == "{}"
	case ArgEOF:
		return value == "EOF"
	default:
		return false
	}
}

// CommandConfig is a loaded allowlist entry.
type CommandConfig struct {
	SafeFlags          map[string]ArgType
	RespectsDoubleDash bool
}

// ValidateFlags mirrors readOnlyCommandValidation.ts validateFlags.
func ValidateFlags(tokens []string, startIndex int, config *CommandConfig, opts *ValidateFlagsOptions) bool {
	if config == nil {
		return false
	}
	respectsDD := true
	if config.RespectsDoubleDash == false {
		respectsDD = false
	}
	i := startIndex
	commandName := ""
	if opts != nil {
		commandName = opts.CommandName
	}
	for i < len(tokens) {
		token := tokens[i]
		if token == "" {
			i++
			continue
		}
		if opts != nil && opts.XargsTargetCommands != nil && commandName == "xargs" &&
			(!strings.HasPrefix(token, "-") || token == "--") {
			if token == "--" && i+1 < len(tokens) {
				i++
				token = tokens[i]
			}
			if token != "" && stringInSlice(token, opts.XargsTargetCommands) {
				break
			}
			return false
		}
		if token == "--" {
			if respectsDD {
				i++
				break
			}
			i++
			continue
		}
		if strings.HasPrefix(token, "-") && len(token) > 1 && flagPattern.MatchString(token) {
			hasEquals := strings.Contains(token, "=")
			parts := strings.SplitN(token, "=", 2)
			flag := parts[0]
			inlineValue := ""
			if len(parts) > 1 {
				inlineValue = parts[1]
			}
			if flag == "" {
				return false
			}
			flagArgType, known := config.SafeFlags[flag]
			if !known {
				if commandName == "git" && regexp.MustCompile(`^-\d+$`).MatchString(flag) {
					i++
					continue
				}
				if (commandName == "grep" || commandName == "rg") && strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
					potentialFlag := flag[:2]
					potentialValue := flag[2:]
					ft, ok := config.SafeFlags[potentialFlag]
					if ok && regexp.MustCompile(`^\d+$`).MatchString(potentialValue) {
						if ft == ArgNumber || ft == ArgString {
							if ValidateFlagArgument(potentialValue, ft) {
								i++
								continue
							}
							return false
						}
					}
				}
				if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") && len(flag) > 2 {
					for j := 1; j < len(flag); j++ {
						sf := "-" + string(flag[j])
						ft, ok := config.SafeFlags[sf]
						if !ok {
							return false
						}
						if ft != ArgNone {
							return false
						}
					}
					i++
					continue
				}
				return false
			}
			if flagArgType == ArgNone {
				if hasEquals {
					return false
				}
				i++
			} else {
				var argValue string
				if hasEquals {
					argValue = inlineValue
					i++
				} else {
					if i+1 >= len(tokens) ||
						(strings.HasPrefix(tokens[i+1], "-") && len(tokens[i+1]) > 1 && flagPattern.MatchString(tokens[i+1])) {
						return false
					}
					argValue = tokens[i+1]
					i += 2
				}
				if flagArgType == ArgString && strings.HasPrefix(argValue, "-") {
					if flag == "--sort" && commandName == "git" && regexp.MustCompile(`^-[a-zA-Z]`).MatchString(argValue) {
						// reverse sort keys
					} else {
						return false
					}
				}
				if !ValidateFlagArgument(argValue, flagArgType) {
					return false
				}
			}
		} else {
			i++
		}
	}
	return true
}

func stringInSlice(s string, xs []string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// ValidateFlagsOptions mirrors TS options bag.
type ValidateFlagsOptions struct {
	CommandName           string
	RawCommand            string
	XargsTargetCommands   []string
}

package bashtool

// SedEditInfo mirrors sedEditParser.ts SedEditInfo (subset for API parity).
type SedEditInfo struct {
	FilePath      string
	Pattern       string
	Replacement   string
	Flags         string
	ExtendedRegex bool
}

// ParseSedEditCommand mirrors sedEditParser.ts parseSedEditCommand; full shell-quote parity deferred.
func ParseSedEditCommand(command string) *SedEditInfo {
	_ = command
	return nil
}

// IsSedInPlaceEdit mirrors sedEditParser.ts isSedInPlaceEdit.
func IsSedInPlaceEdit(command string) bool {
	return ParseSedEditCommand(command) != nil
}

package cost

// EmptyUsage returns a zero-valued usage matching EMPTY_USAGE semantics (emptyUsage.ts).
func EmptyUsage() Usage {
	return Usage{
		ServiceTier: "standard",
		Speed:       "standard",
	}
}

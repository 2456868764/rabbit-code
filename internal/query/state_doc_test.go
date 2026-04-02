package query

import "testing"

func TestLoopState_queryTSFieldParity_doc(t *testing.T) {
	// Compile-time / shallow check that extended fields exist for P5.1.1 continuation (item 11).
	var st LoopState
	st.MaxOutputTokensRecoveryCount = 1
	st.HasAttemptedReactiveCompact = true
	st.StopHookActive = true
	_ = st
}

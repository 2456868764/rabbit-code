package cost

import "testing"

func TestFromUsageDelta(t *testing.T) {
	u := FromUsageDelta(1, 2, 3, 4)
	if u.InputTokens != 1 || u.CacheCreationInputTokens != 2 || u.CacheReadInputTokens != 3 || u.OutputTokens != 4 {
		t.Fatalf("%+v", u)
	}
}

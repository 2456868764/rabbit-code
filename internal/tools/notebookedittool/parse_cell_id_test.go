package notebookedittool

import "testing"

func TestParseCellId(t *testing.T) {
	if n, ok := ParseCellId("cell-0"); !ok || n != 0 {
		t.Fatalf("cell-0: got %d %v", n, ok)
	}
	if n, ok := ParseCellId("cell-12"); !ok || n != 12 {
		t.Fatalf("cell-12: got %d %v", n, ok)
	}
	if _, ok := ParseCellId("cell-"); ok {
		t.Fatal("cell- should not match")
	}
	if _, ok := ParseCellId("xcell-1"); ok {
		t.Fatal("xcell-1 should not match")
	}
	if _, ok := ParseCellId(""); ok {
		t.Fatal("empty should not match")
	}
}

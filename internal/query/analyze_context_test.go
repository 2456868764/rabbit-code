package query

import "testing"

func TestReactiveCompactByTranscript_bytesOnly(t *testing.T) {
	if ReactiveCompactByTranscript([]byte("hi"), 1, 0) != true {
		t.Fatal()
	}
	if ReactiveCompactByTranscript([]byte(""), 1, 0) != false {
		t.Fatal()
	}
}

func TestReactiveCompactByTranscript_tokensOnly(t *testing.T) {
	// "abcd" -> 1 token by heuristic; minTokens 2 -> false
	if ReactiveCompactByTranscript([]byte("abcd"), 0, 2) != false {
		t.Fatal()
	}
	if ReactiveCompactByTranscript([]byte("abcde"), 0, 2) != true {
		t.Fatal()
	}
}

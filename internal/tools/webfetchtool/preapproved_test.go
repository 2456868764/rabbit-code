package webfetchtool

import "testing"

func TestIsPreapprovedHost_pathPrefix(t *testing.T) {
	if !IsPreapprovedHost("github.com", "/anthropics/foo") {
		t.Fatal("expected github.com/anthropics prefix")
	}
	if IsPreapprovedHost("github.com", "/anthropics-evil/x") {
		t.Fatal("should not match segment boundary")
	}
	if !IsPreapprovedHost("pkg.go.dev", "/std") {
		t.Fatal("expected hostname-only pkg.go.dev")
	}
}

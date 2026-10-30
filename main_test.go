package main

import "testing"

func TestFoldsAtStartup(t *testing.T) {
	for sub, want := range map[string]bool{"list": true, "add": true, "sync-seed": false, "rebuild": false} {
		if got := foldsAtStartup(sub); got != want {
			t.Errorf("foldsAtStartup(%q) = %v, want %v", sub, got, want)
		}
	}
}

package gui

import "testing"

// A tab can be reattached while its window is still being created. The record
// has to say so, otherwise the detach path shows a window whose record is
// already gone and nothing can close it.
func TestTakeDetachedTabHandlesAWindowThatDoesNotExistYet(t *testing.T) {
	a := &App{detachedTabs: map[string]*detachedAgentTab{}}
	record := &detachedAgentTab{payload: "payload", tabType: "shell"}
	a.detachedTabs["tok"] = record

	got, window := a.takeDetachedTab("tok")
	if got != record {
		t.Fatalf("takeDetachedTab returned %v, want the stored record", got)
	}
	if window != nil {
		t.Fatalf("takeDetachedTab returned a window for a record that has none: %v", window)
	}
	if !record.reattach {
		t.Fatal("the record was not marked, so the detach path would still show the window")
	}
	if _, still := a.detachedTabs["tok"]; still {
		t.Fatal("the record is still registered after being taken")
	}

	if got, window := a.takeDetachedTab("missing"); got != nil || window != nil {
		t.Fatalf("takeDetachedTab on an unknown token = %v, %v; want nils", got, window)
	}
}

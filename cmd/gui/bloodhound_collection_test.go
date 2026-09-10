package gui

import (
	"sync"
	"sync/atomic"
	"testing"

	"siren/internal/bloodhound"
)

// A Wails binding thread and the automation engine can both start a collection,
// so the lazy build has to happen once no matter how many arrive together.
func TestCollectionRunnerIsBuiltOnceUnderConcurrency(t *testing.T) {
	a := &App{}
	var builds atomic.Int64
	build := func() *bloodhound.CollectionRunner {
		builds.Add(1)
		return &bloodhound.CollectionRunner{}
	}

	const goroutines = 16
	results := make([]*bloodhound.CollectionRunner, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = a.collectionRunnerWith(build)
		}()
	}
	wg.Wait()

	if got := builds.Load(); got != 1 {
		t.Fatalf("built the collector %d times, want 1", got)
	}
	for i, runner := range results {
		if runner == nil {
			t.Fatalf("goroutine %d got a nil runner", i)
		}
		if runner != results[0] {
			t.Fatalf("goroutine %d got a different runner", i)
		}
	}
}

// Status and list queries must not build a runner just to answer.
func TestExistingCollectionRunnerDoesNotBuildOne(t *testing.T) {
	a := &App{}
	if a.existingCollectionRunner() != nil {
		t.Fatal("existingCollectionRunner built a runner")
	}
	if _, err := a.BloodHoundCollectionStatus("any"); err == nil {
		t.Fatal("BloodHoundCollectionStatus on a fresh App = nil error, want ErrNotConnected")
	}
}

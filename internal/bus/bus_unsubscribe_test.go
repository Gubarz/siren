package bus

import (
	"runtime"
	"testing"
	"time"
)

// Unsubscribing removed the subscriber from the map but left the channel open,
// so the goroutine delivering events blocked on it forever and the buffered
// events were never released.
func TestUnsubscribeReleasesTheSubscriberGoroutine(t *testing.T) {
	b := New()
	before := runtime.NumGoroutine()

	const subscribers = 20
	unsubscribes := make([]func(), 0, subscribers)
	for range subscribers {
		unsubscribes = append(unsubscribes, b.Subscribe(nil, func(Event) {}))
	}
	for _, unsubscribe := range unsubscribes {
		unsubscribe()
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutines went from %d to %d after unsubscribing %d subscribers",
		before, runtime.NumGoroutine(), subscribers)
}

// Unsubscribing twice must stay safe: the second call closes nothing.
func TestUnsubscribeIsIdempotent(t *testing.T) {
	b := New()
	unsubscribe := b.Subscribe(nil, func(Event) {})

	unsubscribe()
	unsubscribe()

	b.Publish(Event{Type: "anything"})
}

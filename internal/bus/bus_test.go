package bus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"siren/internal/testutil"
)

func TestPublishFansOutToAllSubscribers(t *testing.T) {
	b := New()
	var a, c atomic.Int64
	b.Subscribe(nil, func(Event) { a.Add(1) })
	b.Subscribe(nil, func(Event) { c.Add(1) })
	for i := 0; i < 5; i++ {
		b.Publish(Event{Type: "sliver.session-opened"})
	}
	testutil.WaitFor(t, "both subscribers", func() bool { return a.Load() == 5 && c.Load() == 5 })
}

func TestTypeFilteringAndWildcard(t *testing.T) {
	b := New()
	var got []string
	var mu sync.Mutex
	b.Subscribe([]string{"gui.file-downloaded"}, func(ev Event) {
		mu.Lock()
		got = append(got, ev.Type)
		mu.Unlock()
	})
	b.Publish(Event{Type: "sliver.session-opened"})
	b.Publish(Event{Type: "gui.file-downloaded"})
	testutil.WaitFor(t, "filtered delivery", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == 1 })
	mu.Lock()
	defer mu.Unlock()
	if got[0] != "gui.file-downloaded" {
		t.Fatalf("got %q", got[0])
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := New()
	var n atomic.Int64
	unsub := b.Subscribe(nil, func(Event) { n.Add(1) })
	b.Publish(Event{Type: "x"})
	testutil.WaitFor(t, "first delivery", func() bool { return n.Load() == 1 })
	unsub()
	b.Publish(Event{Type: "x"})
	time.Sleep(50 * time.Millisecond)
	if n.Load() != 1 {
		t.Fatalf("delivered after unsubscribe: %d", n.Load())
	}
}

func TestDropOldestWhenSubscriberLags(t *testing.T) {
	b := New()
	gate := make(chan struct{})
	var received atomic.Int64
	b.Subscribe(nil, func(Event) {
		<-gate // block until released
		received.Add(1)
	})
	total := subscriberCapacity + 50
	for i := 0; i < total; i++ {
		b.Publish(Event{Type: "x", Payload: i})
	}
	close(gate)
	testutil.WaitFor(t, "drain", func() bool { return received.Load() >= subscriberCapacity })
	// White-box: read the dropped counter off the concrete subscription.
	impl := b.(*bus)
	impl.mu.RLock()
	var sub *subscription
	for s := range impl.subs {
		sub = s
	}
	impl.mu.RUnlock()
	if sub == nil {
		t.Fatal("subscription not registered")
	}
	if got := sub.dropped.Load(); got < 1 {
		t.Fatalf("dropped = %d, want at least 1 dropped event", got)
	}
}

func TestSubscriberPanicIsIsolated(t *testing.T) {
	b := New()
	var good atomic.Int64
	b.Subscribe(nil, func(Event) { panic("boom") })
	b.Subscribe(nil, func(Event) { good.Add(1) })
	b.Publish(Event{Type: "x"})
	testutil.WaitFor(t, "good subscriber survives", func() bool { return good.Load() == 1 })
}

func TestPublishFillsZeroTime(t *testing.T) {
	b := New()
	seen := make(chan Event, 1)
	b.Subscribe(nil, func(ev Event) { seen <- ev })
	b.Publish(Event{Type: "x"})
	ev := <-seen
	if ev.Time == 0 {
		t.Fatal("Publish must stamp Time when zero")
	}
}

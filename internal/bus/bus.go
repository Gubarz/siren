package bus

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const subscriberCapacity = 256

type Bus interface {
	Publish(ev Event)
	Subscribe(types []string, h Handler) (unsub func())
}

type subscription struct {
	types   map[string]struct{} // nil = all events
	ch      chan Event
	dropped atomic.Int64
}

func (s *subscription) wants(eventType string) bool {
	if s.types == nil {
		return true
	}
	_, ok := s.types[eventType]
	return ok
}

type bus struct {
	mu   sync.RWMutex
	subs map[*subscription]struct{}
}

func New() Bus {
	return &bus{subs: map[*subscription]struct{}{}}
}

func (b *bus) Publish(ev Event) {
	if ev.Time == 0 {
		ev.Time = time.Now().UnixMilli()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for sub := range b.subs {
		if !sub.wants(ev.Type) {
			continue
		}
		select {
		case sub.ch <- ev:
		default:
			sub.dropOldest(ev)
		}
	}
}

func (s *subscription) dropOldest(ev Event) {
	select {
	case <-s.ch:
	default:
	}
	select {
	case s.ch <- ev:
	default:
	}
	if s.dropped.Add(1) == 1 {
		log.Printf("bus: slow subscriber, dropping oldest events")
	}
}

func (b *bus) Subscribe(types []string, h Handler) (unsub func()) {
	sub := &subscription{ch: make(chan Event, subscriberCapacity)}
	if len(types) > 0 {
		sub.types = make(map[string]struct{}, len(types))
		for _, t := range types {
			sub.types[t] = struct{}{}
		}
	}
	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()
	go sub.run(h)
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, sub)
			b.mu.Unlock()
		})
	}
}

func (s *subscription) run(h Handler) {
	for ev := range s.ch {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("bus: subscriber panic recovered: %v", r)
				}
			}()
			h(ev)
		}()
	}
}

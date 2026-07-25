package journal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"sliver-gui/internal/bus"
)

var ErrDisabled = errors.New("journal disabled")

const (
	recordQueueCapacity = 1024
	flushInterval       = 500 * time.Millisecond
	flushBatchSize      = 200
)

// Service is the journal write path: Record is a non-blocking enqueue; one
// writer goroutine batches inserts so journaling never stalls an RPC.
type Service struct {
	store Store
	bus   bus.Bus // may be nil
	queue chan Entry

	closeOnce sync.Once
	done      chan struct{}
	wg        sync.WaitGroup
}

func NewService(store Store, b bus.Bus) *Service {
	s := &Service{store: store, bus: b, queue: make(chan Entry, recordQueueCapacity), done: make(chan struct{})}
	if store == nil {
		log.Printf("journal: no store — activity journal disabled")
		return s
	}
	s.wg.Add(1)
	go s.writerLoop()
	return s
}

// Record enqueues an entry. It never blocks the caller: a full queue drops
// the entry (logged on first drop), matching the capture-must-not-break-
// operations requirement.
func (s *Service) Record(e Entry) {
	if s.store == nil {
		return
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Time == 0 {
		e.Time = time.Now().UnixMilli()
	}
	select {
	case s.queue <- e:
	default:
		log.Printf("journal: record queue full, dropping %s entry", e.Verb)
	}
}

func (s *Service) Query(ctx context.Context, f Filter) ([]Entry, int, error) {
	if s.store == nil {
		return nil, 0, ErrDisabled
	}
	return s.store.Query(ctx, f)
}

func (s *Service) VerbCounts(ctx context.Context, f Filter) (map[string]int64, error) {
	if s.store == nil {
		return nil, ErrDisabled
	}
	return s.store.VerbCounts(ctx, f)
}

func (s *Service) TimeSeries(ctx context.Context, f TimeSeriesFilter) ([]TimeBucket, error) {
	const maxBuckets = 10000
	if s.store == nil {
		return nil, ErrDisabled
	}
	buckets, err := s.store.TimeSeries(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(buckets) > maxBuckets {
		return nil, fmt.Errorf("time series returned %d buckets (> %d); narrow the range or increase bucket size", len(buckets), maxBuckets)
	}
	return buckets, nil
}

// Close drains the queue, performs a final flush, and closes the store.
func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
	})
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *Service) writerLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	batch := make([]Entry, 0, flushBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := s.store.InsertBatch(context.Background(), batch); err != nil {
			log.Printf("journal: insert batch: %v", err)
		} else {
			s.publish(batch)
		}
		batch = batch[:0]
	}
	for {
		select {
		case e := <-s.queue:
			batch = append(batch, e)
			if len(batch) >= flushBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.done:
			for {
				select {
				case e := <-s.queue:
					batch = append(batch, e)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (s *Service) publish(batch []Entry) {
	if s.bus == nil {
		return
	}
	for _, e := range batch {
		s.bus.Publish(bus.Event{
			Type:         "journal.action-recorded",
			Source:       "journal",
			ConnectionID: e.ConnectionID,
			Payload:      e,
		})
	}
}

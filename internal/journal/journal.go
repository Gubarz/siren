package journal

import "context"

type Entry struct {
	ID            string
	Time          int64
	ConnectionID  string
	ActorKind     string
	RuleID        string
	RuleName      string
	Verb          string
	CommandLine   string
	TargetID      string
	TargetKind    string
	Hostname      string
	Panel         string
	Status        string
	Err           string
	DurationMs    int64
	CorrelationID string
}

type Filter struct {
	ConnectionID string
	TargetID     string
	Verb         string
	ActorKind    string
	Since        int64
	Until        int64
	Limit        int
	Offset       int
}

type Store interface {
	InsertBatch(ctx context.Context, entries []Entry) error
	Query(ctx context.Context, f Filter) ([]Entry, int, error)
	VerbCounts(ctx context.Context, f Filter) (map[string]int64, error)
	Close() error
}

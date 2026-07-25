package journal

import "context"

type Entry struct {
	ID            string `json:"id"`
	Time          int64  `json:"time"`
	ConnectionID  string `json:"connectionID"`
	ActorKind     string `json:"actorKind"`
	RuleID        string `json:"ruleID"`
	RuleName      string `json:"ruleName"`
	Verb          string `json:"verb"`
	CommandLine   string `json:"commandLine"`
	TargetID      string `json:"targetID"`
	TargetKind    string `json:"targetKind"`
	Hostname      string `json:"hostname"`
	Panel         string `json:"panel"`
	Status        string `json:"status"`
	Err           string `json:"err"`
	DurationMs    int64  `json:"durationMs"`
	CorrelationID string `json:"correlationID"`
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

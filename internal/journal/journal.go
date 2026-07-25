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
	Search       string   `json:"search"`
	Verbs        []string `json:"verbs"`
}

type TimeBucket struct {
	Start     int64  `json:"start"`
	Verb      string `json:"verb"`
	ActorKind string `json:"actor_kind"`
	Status    string `json:"status"`
	Count     int64  `json:"count"`
}

type TimeSeriesFilter struct {
	ConnectionID  string `json:"connection_id"`
	TargetID      string `json:"target_id"`
	Verb          string `json:"verb"`
	ActorKind     string `json:"actor_kind"`
	Since         int64  `json:"since"`
	Until         int64  `json:"until"`
	BucketSeconds int64  `json:"bucket_seconds"`
}

type Store interface {
	InsertBatch(ctx context.Context, entries []Entry) error
	Query(ctx context.Context, f Filter) ([]Entry, int, error)
	VerbCounts(ctx context.Context, f Filter) (map[string]int64, error)
	TimeSeries(ctx context.Context, f TimeSeriesFilter) ([]TimeBucket, error)
	Close() error
}

package journal

import "context"

type Overlay struct {
	ActorKind     string
	RuleID        string
	RuleName      string
	CommandLine   string
	Panel         string
	TargetID      string
	TargetKind    string
	Hostname      string
	CorrelationID string
}

type overlayKey struct{}

func WithContext(ctx context.Context, o Overlay) context.Context {
	if existing, ok := OverlayFrom(ctx); ok {
		existing.merge(o)
		o = existing
	}
	return context.WithValue(ctx, overlayKey{}, o)
}

func OverlayFrom(ctx context.Context) (Overlay, bool) {
	o, ok := ctx.Value(overlayKey{}).(Overlay)
	return o, ok
}

func (o *Overlay) merge(next Overlay) {
	if next.ActorKind != "" {
		o.ActorKind = next.ActorKind
	}
	if next.RuleID != "" {
		o.RuleID = next.RuleID
	}
	if next.RuleName != "" {
		o.RuleName = next.RuleName
	}
	if next.CommandLine != "" {
		o.CommandLine = next.CommandLine
	}
	if next.Panel != "" {
		o.Panel = next.Panel
	}
	if next.TargetID != "" {
		o.TargetID = next.TargetID
	}
	if next.TargetKind != "" {
		o.TargetKind = next.TargetKind
	}
	if next.Hostname != "" {
		o.Hostname = next.Hostname
	}
	if next.CorrelationID != "" {
		o.CorrelationID = next.CorrelationID
	}
}

func (e *Entry) ApplyOverlay(o Overlay) {
	if e.ActorKind == "" {
		e.ActorKind = o.ActorKind
	}
	if e.RuleID == "" {
		e.RuleID = o.RuleID
	}
	if e.RuleName == "" {
		e.RuleName = o.RuleName
	}
	if e.CommandLine == "" {
		e.CommandLine = o.CommandLine
	}
	if e.Panel == "" {
		e.Panel = o.Panel
	}
	if e.TargetID == "" {
		e.TargetID = o.TargetID
	}
	if e.TargetKind == "" {
		e.TargetKind = o.TargetKind
	}
	if e.Hostname == "" {
		e.Hostname = o.Hostname
	}
	if e.CorrelationID == "" {
		e.CorrelationID = o.CorrelationID
	}
	if e.ActorKind == "" {
		e.ActorKind = "operator"
	}
}

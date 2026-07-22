package casefile

import (
	"encoding/json"
	"testing"
)

func TestCloneCaseNormalizesNilCollections(t *testing.T) {
	got := cloneCase(&Record{ID: "case-1"})

	if got.AgentIDs == nil || got.LootIDs == nil || got.CredIDs == nil || got.HostIDs == nil || got.CanaryIDs == nil {
		t.Fatalf("cloneCase returned nil collection: %+v", got)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"agentIds", "lootIds", "credIds", "hostIds", "canaryIds"} {
		items, ok := payload[key].([]any)
		if !ok {
			t.Fatalf("%s = %T, want JSON array", key, payload[key])
		}
		if len(items) != 0 {
			t.Fatalf("%s length = %d, want 0", key, len(items))
		}
	}
}

func TestReportFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "blank", in: "", want: "case.md"},
		{name: "plain", in: "Acme Case", want: "Acme_Case.md"},
		{name: "trim underscores", in: " !!! ", want: "case.md"},
		{name: "keeps safe chars", in: "case-01_alpha", want: "case-01_alpha.md"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ReportFilename(tc.in); got != tc.want {
				t.Fatalf("ReportFilename(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

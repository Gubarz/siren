package agents

import (
	"reflect"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	knownagents "siren/internal/localstate/agents"
)

func TestSessionRecordsMapsLiveSessions(t *testing.T) {
	got := sessionRecords(&clientpb.Sessions{Sessions: []*clientpb.Session{{
		ID:            "sess-a",
		Name:          "name-a",
		Hostname:      "host-a",
		Username:      "user-a",
		OS:            "linux",
		Arch:          "amd64",
		RemoteAddress: "10.0.0.1:443",
		Transport:     "mtls",
	}}})
	want := []knownagents.Record{{
		ID:            "sess-a",
		Kind:          "session",
		Name:          "name-a",
		Hostname:      "host-a",
		Username:      "user-a",
		OS:            "linux",
		Arch:          "amd64",
		RemoteAddress: "10.0.0.1:443",
		Transport:     "mtls",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sessionRecords() = %+v, want %+v", got, want)
	}
}

func TestSessionRecordsNilEmptyAndBlank(t *testing.T) {
	for name, in := range map[string]*clientpb.Sessions{
		"nil":       nil,
		"empty":     {},
		"nil entry": {Sessions: []*clientpb.Session{nil}},
	} {
		t.Run(name, func(t *testing.T) {
			got := sessionRecords(in)
			if got == nil {
				t.Fatal("sessionRecords() = nil, want empty slice")
			}
			if len(got) != 0 {
				t.Fatalf("sessionRecords() = %+v, want empty", got)
			}
		})
	}
}

func TestObserveExcludesBeacons(t *testing.T) {
	known := knownagents.New(t.TempDir())
	s := &Service{}
	s.SetKnownAgents(known)
	s.observe([]knownagents.Record{
		{ID: "beacon-b", Kind: "beacon", Hostname: "host-a"},
		{ID: "sess-a", Kind: "session", Hostname: "host-a"},
	})

	list := known.List()
	if len(list) != 1 || list[0].ID != "sess-a" {
		t.Fatalf("List() = %+v, want only sess-a (beacons never persisted)", list)
	}
}

func TestObserveBeaconOnlyIsNoop(t *testing.T) {
	known := knownagents.New(t.TempDir())
	s := &Service{}
	s.SetKnownAgents(known)
	s.observe([]knownagents.Record{{ID: "beacon-b", Kind: "beacon", Hostname: "host-a"}})

	if list := known.List(); len(list) != 0 {
		t.Fatalf("List() = %+v, want empty", list)
	}
}

func TestObserveNilKnownAgentsIsNoop(t *testing.T) {
	s := &Service{}
	s.observe([]knownagents.Record{{ID: "sess-a", Kind: "session"}})
}

func TestObservePersistsRecordsToKnownAgents(t *testing.T) {
	known := knownagents.New(t.TempDir())
	s := &Service{}
	s.SetKnownAgents(known)
	s.observe([]knownagents.Record{{ID: "sess-a", Kind: "session", Hostname: "host-a"}})

	list := known.List()
	if len(list) != 1 || list[0].ID != "sess-a" || list[0].Hostname != "host-a" {
		t.Fatalf("List() = %+v, want observed sess-a", list)
	}
}

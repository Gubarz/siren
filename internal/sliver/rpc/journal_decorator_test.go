package rpc

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"

	"sliver-gui/internal/bus"
	"sliver-gui/internal/journal"
	localjournal "sliver-gui/internal/localstate/journal"
)

// fakeRPC embeds the interface (unimplemented methods panic) and stubs the
// two verbs the tests exercise.
type fakeRPC struct {
	rpcpb.SliverRPCClient
	psErr     error
	psCalls   int
	lastCtx   context.Context
	pollCalls int
}

func (f *fakeRPC) Ps(ctx context.Context, _ *sliverpb.PsReq, _ ...grpc.CallOption) (*sliverpb.Ps, error) {
	f.psCalls++
	f.lastCtx = ctx
	return &sliverpb.Ps{}, f.psErr
}

func (f *fakeRPC) GetSessions(ctx context.Context, _ *commonpb.Empty, _ ...grpc.CallOption) (*clientpb.Sessions, error) {
	f.pollCalls++
	return &clientpb.Sessions{}, nil
}

func newTestStack(t *testing.T) (*journal.Service, bus.Bus) {
	t.Helper()
	store, err := localjournal.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := bus.New()
	svc := journal.NewService(store, b)
	t.Cleanup(func() { _ = svc.Close() })
	return svc, b
}

func waitForEntries(t *testing.T, svc *journal.Service, want int) []journal.Entry {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, _, err := svc.Query(context.Background(), journal.Filter{})
		if err == nil && len(entries) >= want {
			return entries
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d entries", want)
	return nil
}

func TestDecoratorRecordsEntryWithOverlay(t *testing.T) {
	svc, _ := newTestStack(t)
	hook := NewJournalHook(svc)
	hook.SetConnection("h:31337")
	fake := &fakeRPC{}
	client := WrapJournal(fake, hook)

	ctx := journal.WithContext(context.Background(), journal.Overlay{
		ActorKind: "operator", Panel: "console", CommandLine: "ps",
		TargetID: "t1", TargetKind: "session", CorrelationID: "corr-1",
	})
	if _, err := client.Ps(ctx, &sliverpb.PsReq{}); err != nil {
		t.Fatal(err)
	}
	entries := waitForEntries(t, svc, 1)
	e := entries[0]
	if e.Verb != "Ps" || e.Status != "ok" || e.ConnectionID != "h:31337" {
		t.Fatalf("entry: %+v", e)
	}
	if e.ActorKind != "operator" || e.CommandLine != "ps" || e.CorrelationID != "corr-1" {
		t.Fatalf("overlay not applied: %+v", e)
	}
	if e.TargetID != "t1" || e.TargetKind != "session" {
		t.Fatalf("target not applied: %+v", e)
	}
}

func TestDecoratorRecordsErrorStatus(t *testing.T) {
	svc, _ := newTestStack(t)
	fake := &fakeRPC{psErr: errors.New("boom")}
	client := WrapJournal(fake, NewJournalHook(svc))
	_, err := client.Ps(context.Background(), &sliverpb.PsReq{})
	if err == nil {
		t.Fatal("expected error passthrough")
	}
	e := waitForEntries(t, svc, 1)[0]
	if e.Status != "error" || e.Err != "boom" || e.ActorKind != "operator" {
		t.Fatalf("entry: %+v", e)
	}
}

func TestDecoratorDropsPolls(t *testing.T) {
	svc, _ := newTestStack(t)
	fake := &fakeRPC{}
	client := WrapJournal(fake, NewJournalHook(svc))
	if _, err := client.GetSessions(context.Background(), &commonpb.Empty{}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(700 * time.Millisecond) // longer than the flush interval
	entries, total, _ := svc.Query(context.Background(), journal.Filter{})
	if total != 0 || len(entries) != 0 {
		t.Fatalf("poll was journaled: %d", total)
	}
}

func TestNilHookReturnsUnwrapped(t *testing.T) {
	fake := &fakeRPC{}
	client := WrapJournal(fake, nil)
	if _, ok := client.(*journalDecorator); ok {
		t.Fatal("nil hook must not wrap")
	}
}

func TestJournalEventPublished(t *testing.T) {
	svc, b := newTestStack(t)
	seen := make(chan journal.Entry, 1)
	b.Subscribe([]string{"journal.action-recorded"}, func(ev bus.Event) {
		seen <- ev.Payload.(journal.Entry)
	})
	client := WrapJournal(&fakeRPC{}, NewJournalHook(svc))
	if _, err := client.Ps(context.Background(), &sliverpb.PsReq{}); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-seen:
		if e.Verb != "Ps" {
			t.Fatalf("event payload: %+v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no journal.action-recorded bus event")
	}
}

func TestGeneratedDecoratorUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("generator check runs the generator")
	}
	tmp := t.TempDir() + "/journal_decorator.go"
	out, err := exec.Command("go", "run", "./gen", "-out", tmp).CombinedOutput()
	if err != nil {
		t.Fatalf("generator: %v\n%s", err, out)
	}
	want, _ := os.ReadFile("journal_decorator.go")
	got, _ := os.ReadFile(tmp)
	if !bytes.Equal(got, want) {
		t.Fatal("journal_decorator.go is stale — run: go generate ./internal/sliver/rpc")
	}
}

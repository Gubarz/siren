package beacons

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestShouldCancelPendingBeaconTask(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "local cancellation", err: context.Canceled, want: false},
		{name: "wrapped local cancellation", err: errors.Join(errors.New("wait failed"), context.Canceled), want: false},
		{name: "timeout", err: context.DeadlineExceeded, want: true},
		{name: "rpc error", err: errors.New("rpc failed"), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldCancelPendingBeaconTask(tc.err); got != tc.want {
				t.Fatalf("shouldCancelPendingBeaconTask(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func mustMarshal(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// decodeTaskOutput must produce typed output for every task kind the
// frontend can open as a tab — otherwise the tab re-tasks the beacon.
var decodeTaskOutputTests = []struct {
	name      string
	desc      string
	response  func(t *testing.T) []byte
	wantType  string
	wantEmpty func(*TaskOutput) bool
}{
	{
		name: "env",
		desc: "EnvReq",
		response: func(t *testing.T) []byte {
			return mustMarshal(t, &sliverpb.EnvInfo{Variables: []*commonpb.EnvVar{{Key: "FOO", Value: "bar"}}})
		},
		wantType: "env",
		wantEmpty: func(out *TaskOutput) bool {
			return len(out.EnvVars) != 1 || out.EnvVars[0].Key != "FOO" || out.EnvVars[0].Value != "bar"
		},
	},
	{
		name: "processes",
		desc: "PsReq",
		response: func(t *testing.T) []byte {
			return mustMarshal(t, &sliverpb.Ps{Processes: []*commonpb.Process{{Pid: 42, Executable: "cmd.exe"}}})
		},
		wantType:  "processes",
		wantEmpty: func(out *TaskOutput) bool { return len(out.Processes) != 1 || out.Processes[0].Pid != 42 },
	},
	{
		name: "screenshot",
		desc: "ScreenshotReq",
		response: func(t *testing.T) []byte {
			return mustMarshal(t, &sliverpb.Screenshot{Data: []byte("png-bytes")})
		},
		wantType:  "image",
		wantEmpty: func(out *TaskOutput) bool { return out.ImageData == "" },
	},
	{
		name: "files",
		desc: "LsReq",
		response: func(t *testing.T) []byte {
			return mustMarshal(t, &sliverpb.Ls{Path: "/tmp", Files: []*sliverpb.FileInfo{{Name: "a.txt"}}})
		},
		wantType:  "filelist",
		wantEmpty: func(out *TaskOutput) bool { return out.Path != "/tmp" || len(out.Files) != 1 },
	},
	{
		name: "services",
		desc: "ServicesReq",
		response: func(t *testing.T) []byte {
			return mustMarshal(t, &sliverpb.Services{Details: []*sliverpb.ServiceDetails{{Name: "W32Time"}}})
		},
		wantType:  "services",
		wantEmpty: func(out *TaskOutput) bool { return len(out.Services) != 1 || out.Services[0].Name != "W32Time" },
	},
}

func TestDecodeTaskOutput_TypedKinds(t *testing.T) {
	s := &Service{}
	for _, tc := range decodeTaskOutputTests {
		t.Run(tc.name, func(t *testing.T) {
			task := &clientpb.BeaconTask{Description: tc.desc, Response: tc.response(t)}
			out, err := s.decodeTaskOutput(task)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Type != tc.wantType {
				t.Fatalf("expected type %q, got %q", tc.wantType, out.Type)
			}
			if tc.wantEmpty(out) {
				t.Fatalf("typed payload missing or wrong: %+v", out)
			}
		})
	}
}

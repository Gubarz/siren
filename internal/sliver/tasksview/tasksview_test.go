package tasksview

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type decodeCase struct {
	name     string
	method   string
	msg      proto.Message
	kind     string
	wantJSON []string
}

var decodeResponseKindFixtures = []decodeCase{
	{
		name:     "processes",
		method:   "/rpcpb.SliverRPC/Ps",
		msg:      &sliverpb.Ps{Processes: []*commonpb.Process{{Pid: 1, Ppid: 2, Executable: "proc-a"}}},
		kind:     "processes",
		wantJSON: []string{`"Processes"`, `"proc-a"`, `"Pid"`},
	},
	{
		name:     "files",
		method:   "/rpcpb.SliverRPC/Ls",
		msg:      &sliverpb.Ls{Path: "/test/dir", Files: []*sliverpb.FileInfo{{Name: "file-a.txt", Size: 3, Mode: "-rw-r--r--"}}},
		kind:     "files",
		wantJSON: []string{`"Files"`, `"file-a.txt"`, `"Mode"`},
	},
	{
		name:     "image",
		method:   "/rpcpb.SliverRPC/Screenshot",
		msg:      &sliverpb.Screenshot{Data: []byte("png")},
		kind:     "image",
		wantJSON: []string{`"Data"`, `"cG5n"`},
	},
	{
		name:     "env",
		method:   "/rpcpb.SliverRPC/GetEnv",
		msg:      &sliverpb.EnvInfo{Variables: []*commonpb.EnvVar{{Key: "ENV_KEY", Value: "value-a"}}},
		kind:     "env",
		wantJSON: []string{`"Variables"`, `"ENV_KEY"`},
	},
	{
		name:     "services",
		method:   "/rpcpb.SliverRPC/Services",
		msg:      &sliverpb.Services{Details: []*sliverpb.ServiceDetails{{Name: "svc-a"}}},
		kind:     "services",
		wantJSON: []string{`"Details"`, `"svc-a"`},
	},
	{
		name:     "netstat",
		method:   "/rpcpb.SliverRPC/Netstat",
		msg:      &sliverpb.Netstat{},
		kind:     "netstat",
		wantJSON: []string{`{}`},
	},
	{
		name:     "execute stdout text",
		method:   "/rpcpb.SliverRPC/Execute",
		msg:      &sliverpb.Execute{Stdout: []byte("hello"), Response: &commonpb.Response{Err: "boom"}},
		kind:     "text",
		wantJSON: []string{`"Stdout"`, `"aGVsbG8="`, `"Err"`},
	},
	{
		name:     "pwd path text",
		method:   "/rpcpb.SliverRPC/Pwd",
		msg:      &sliverpb.Pwd{Path: "/test/dir"},
		kind:     "text",
		wantJSON: []string{`"Path"`, `/test/dir`},
	},
}

func TestDecodeResponseKinds(t *testing.T) {
	for _, tt := range decodeResponseKindFixtures {
		t.Run(tt.name, func(t *testing.T) {
			assertDecodedResponse(t, tt)
		})
	}
}

func assertDecodedResponse(t *testing.T, tt decodeCase) {
	t.Helper()
	payload, err := proto.Marshal(tt.msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js, b64, kind := Decode(tt.method, "response", payload)
	if kind != tt.kind {
		t.Fatalf("kind = %q, want %q (json %s)", kind, tt.kind, js)
	}
	if js == "" {
		t.Fatal("json empty")
	}
	for _, want := range tt.wantJSON {
		if !strings.Contains(js, want) {
			t.Fatalf("json %s missing %s", js, want)
		}
	}
	if b64 != base64.StdEncoding.EncodeToString(payload) {
		t.Fatalf("base64 = %q", b64)
	}
}

func TestDecodeRequestKindIsJSON(t *testing.T) {
	payload, err := proto.Marshal(&sliverpb.PsReq{FullInfo: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js, b64, kind := Decode("/rpcpb.SliverRPC/Ps", "request", payload)
	if kind != "json" {
		t.Fatalf("kind = %q, want json", kind)
	}
	if js == "" || !strings.Contains(js, `"FullInfo"`) {
		t.Fatalf("json = %s", js)
	}
	if b64 == "" {
		t.Fatal("base64 empty")
	}
}

func TestDecodeUnknownMethodFallsBackToBase64(t *testing.T) {
	js, b64, kind := Decode("/nope/Nope", "response", []byte{1, 2, 3})
	if js != "" {
		t.Fatalf("json = %s", js)
	}
	if kind != "json" {
		t.Fatalf("kind = %q, want json", kind)
	}
	if b64 == "" {
		t.Fatal("base64 empty")
	}
}

func TestDecodeUnmarshalFailureFallsBackToBase64(t *testing.T) {
	js, b64, kind := Decode("/rpcpb.SliverRPC/Ls", "response", []byte{0xff})
	if js != "" {
		t.Fatalf("json = %s", js)
	}
	if kind != "json" {
		t.Fatalf("kind = %q, want json", kind)
	}
	if b64 == "" {
		t.Fatal("base64 empty")
	}
}

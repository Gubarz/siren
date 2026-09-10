package tasksview

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	_ "github.com/bishopfox/sliver/protobuf/rpcpb"
)

func TestDecodeLsResponse(t *testing.T) {
	resp := &sliverpb.Ls{Path: "/tmp", Files: []*sliverpb.FileInfo{{Name: "a.txt", Size: 3}}}
	payload, _ := proto.Marshal(resp)
	js, b64 := Decode("/rpcpb.SliverRPC/Ls", "response", payload)
	if !strings.Contains(js, `"a.txt"`) {
		t.Fatalf("json = %s", js)
	}
	if b64 == "" || b64 == base64.StdEncoding.EncodeToString(nil) {
		t.Fatal("base64 empty")
	}
}

func TestDecodeUnknownMethodFallsBackToBase64(t *testing.T) {
	js, b64 := Decode("/nope/Nope", "response", []byte{1, 2, 3})
	if js != "" {
		t.Fatalf("json = %s", js)
	}
	if b64 == "" {
		t.Fatal("base64 empty")
	}
}

var _ = commonpb.Empty{}

package rpcwrap

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// A request message without the shared envelope is a programming error, but it
// used to be a panic inside the GUI process.
func TestSetRequestReportsAMessageWithoutTheEnvelope(t *testing.T) {
	err := SetRequest(&commonpb.Empty{}, &commonpb.Request{SessionID: "sess"})
	if err == nil {
		t.Fatal("SetRequest accepted a message with no Request field")
	}
	if !strings.Contains(err.Error(), "Request field") {
		t.Fatalf("error = %v, want it to name the missing field", err)
	}
}

func TestSetRequestStampsTheEnvelope(t *testing.T) {
	req := &sliverpb.PsReq{}
	if err := SetRequest(req, &commonpb.Request{SessionID: "sess"}); err != nil {
		t.Fatalf("SetRequest() = %v, want nil", err)
	}
	if got := req.GetRequest().GetSessionID(); got != "sess" {
		t.Fatalf("SessionID = %q, want sess", got)
	}
}

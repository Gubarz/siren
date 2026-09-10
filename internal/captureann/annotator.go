// Package captureann implements revils' capture.Annotator for Siren. It
// resolves the RPC method descriptor from the registered Sliver protos and
// extracts common target fields generically, so no per-method table exists.
package captureann

import (
	"context"
	"encoding/json"

	"github.com/gubarz/revils/capture"

	"siren/internal/execctx"
	"siren/internal/rpcdesc"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Annotator struct {
	operator string
}

func New(operator string) *Annotator {
	return &Annotator{operator: operator}
}

func (a *Annotator) Annotate(ctx context.Context, method, direction string, payload []byte) capture.Annotation {
	runID, stageID := execctx.Run(ctx)
	if runID == "" || stageID == "" {
		current := execctx.Current()
		if runID == "" {
			runID = current.RunID
		}
		if stageID == "" {
			stageID = current.StageID
		}
	}
	ann := capture.Annotation{Operator: a.operator, RunID: runID, StageID: stageID}
	desc := rpcdesc.Message(method, direction)
	if desc == nil || len(payload) == 0 {
		return withPreview(ann, method, direction, len(payload), "", "")
	}
	msg := dynamicpb.NewMessage(desc)
	if err := proto.Unmarshal(payload, msg); err != nil {
		return withPreview(ann, method, direction, len(payload), "", "")
	}
	sessionID := nestedString(msg, "Request", "SessionID")
	beaconID := nestedString(msg, "Request", "BeaconID")
	ann.SessionID = sessionID
	ann.BeaconID = beaconID
	return withPreview(ann, method, direction, len(payload), sessionID, beaconID)
}

// Sliver's protobuf descriptors use the declared CamelCase field names.
func nestedString(msg protoreflect.Message, messageField, scalarField string) string {
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(messageField))
	if fd == nil || fd.Message() == nil {
		return ""
	}
	nested := msg.Get(fd).Message()
	if !nested.IsValid() {
		return ""
	}
	sf := nested.Descriptor().Fields().ByName(protoreflect.Name(scalarField))
	if sf == nil || sf.Kind() != protoreflect.StringKind {
		return ""
	}
	return nested.Get(sf).String()
}

func withPreview(ann capture.Annotation, method, direction string, size int, sessionID, beaconID string) capture.Annotation {
	body, err := json.Marshal(struct {
		Method    string `json:"method"`
		Bytes     int    `json:"bytes"`
		SessionID string `json:"session_id,omitempty"`
		BeaconID  string `json:"beacon_id,omitempty"`
	}{Method: method, Bytes: size, SessionID: sessionID, BeaconID: beaconID})
	if err != nil {
		body = []byte("{}")
	}
	if direction == "response" {
		ann.ResponsePreview = string(body)
	} else {
		ann.RequestPreview = string(body)
	}
	return ann
}

var _ capture.Annotator = (*Annotator)(nil)

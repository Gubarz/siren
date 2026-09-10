// Package tasksview decodes captured RPC payloads into protojson for the
// task views. Descriptors resolve from the registered Sliver protos, so no
// per-method table exists; unknown methods fall back to raw base64 only.
package tasksview

import (
	"encoding/base64"

	_ "github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"siren/internal/rpcdesc"
)

// descriptorKinds maps a response message's full name to the view kind that
// renders it; Sliver declares these fields in CamelCase, so the frontend
// cannot infer the shape from the method name alone.
var descriptorKinds = map[protoreflect.FullName]string{
	"sliverpb.Ps":         "processes",
	"sliverpb.Ls":         "files",
	"sliverpb.Screenshot": "image",
	"sliverpb.EnvInfo":    "env",
	"sliverpb.Services":   "services",
	"sliverpb.Netstat":    "netstat",
	"sliverpb.Pwd":        "text",
}

// Decode returns the protojson rendering of payload, its base64 encoding, and
// the payload kind used by the task views. jsonText is empty when the method
// descriptor is unknown or the payload does not unmarshal; base64Out is always
// populated. Requests and unrecognized responses are kind "json".
func Decode(method, direction string, payload []byte) (jsonText, base64Out, kind string) {
	base64Out = base64.StdEncoding.EncodeToString(payload)
	kind = "json"
	desc := rpcdesc.Message(method, direction)
	if desc == nil {
		return "", base64Out, kind
	}
	msg := dynamicpb.NewMessage(desc)
	if err := proto.Unmarshal(payload, msg); err != nil {
		return "", base64Out, kind
	}
	if direction == "response" {
		kind = classify(desc, msg)
	}
	js, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(msg)
	if err != nil {
		return "", base64Out, "json"
	}
	return string(js), base64Out, kind
}

func classify(desc protoreflect.MessageDescriptor, msg *dynamicpb.Message) string {
	if kind, ok := descriptorKinds[desc.FullName()]; ok {
		return kind
	}
	if hasTextOutput(desc, msg) {
		return "text"
	}
	return "json"
}

// hasTextOutput reports whether a response carries command output either as a
// nested Response.Output byte field or, for Execute, on Stdout/Stderr.
func hasTextOutput(desc protoreflect.MessageDescriptor, msg *dynamicpb.Message) bool {
	if resp := desc.Fields().ByName("Response"); resp != nil && resp.Kind() == protoreflect.MessageKind && msg.Has(resp) {
		if out := resp.Message().Fields().ByName("Output"); out != nil && out.Kind() == protoreflect.BytesKind {
			if len(msg.Get(resp).Message().Get(out).Bytes()) > 0 {
				return true
			}
		}
	}
	if desc.FullName() != "sliverpb.Execute" {
		return false
	}
	for _, name := range []protoreflect.Name{"Stdout", "Stderr"} {
		fd := desc.Fields().ByName(name)
		if fd != nil && fd.Kind() == protoreflect.BytesKind && len(msg.Get(fd).Bytes()) > 0 {
			return true
		}
	}
	return false
}

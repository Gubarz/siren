// Package tasksview decodes captured RPC payloads into protojson for the
// task views. Descriptors resolve from the registered Sliver protos, so no
// per-method table exists; unknown methods fall back to raw base64 only.
package tasksview

import (
	"encoding/base64"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Decode returns the protojson rendering of payload and its base64 encoding.
// jsonText is empty when the method descriptor is unknown or the payload
// does not unmarshal; base64Out is always populated.
func Decode(method, direction string, payload []byte) (jsonText, base64Out string) {
	base64Out = base64.StdEncoding.EncodeToString(payload)
	desc := messageDescriptor(method, direction)
	if desc == nil {
		return "", base64Out
	}
	msg := dynamicpb.NewMessage(desc)
	if err := proto.Unmarshal(payload, msg); err != nil {
		return "", base64Out
	}
	js, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(msg)
	if err != nil {
		return "", base64Out
	}
	return string(js), base64Out
}

func splitMethod(method string) (string, string) {
	for i := len(method) - 1; i >= 0; i-- {
		if method[i] == '/' {
			return method[1:i], method[i+1:]
		}
	}
	return "", ""
}

func messageDescriptor(method, direction string) protoreflect.MessageDescriptor {
	service, rpc := splitMethod(method)
	if service == "" || rpc == "" {
		return nil
	}
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return nil
	}
	sd, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil
	}
	md := sd.Methods().ByName(protoreflect.Name(rpc))
	if md == nil {
		return nil
	}
	if direction == "response" {
		return md.Output()
	}
	return md.Input()
}

// Package rpcdesc resolves Sliver gRPC method descriptors from the global
// protobuf registry. Capture annotations and task views share this lookup.
package rpcdesc

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Message returns the input descriptor for method, or the output descriptor
// when direction is "response". It returns nil for malformed or unregistered
// methods.
func Message(method, direction string) protoreflect.MessageDescriptor {
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

func splitMethod(method string) (string, string) {
	for i := len(method) - 1; i >= 0; i-- {
		if method[i] == '/' {
			return method[1:i], method[i+1:]
		}
	}
	return "", ""
}

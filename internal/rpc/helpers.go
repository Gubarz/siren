package rpc

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// ErrNotConnected is returned when the RPC client is not connected.
var ErrNotConnected = errors.New("not connected")

// ResponseWithError represents a gRPC response that includes a commonpb.Response metadata field.
type ResponseWithError interface {
	GetResponse() *commonpb.Response
}

// CheckResponse inspects the embedded response field for errors.
func CheckResponse(resp ResponseWithError) error {
	if resp == nil || (reflect.ValueOf(resp).Kind() == reflect.Ptr && reflect.ValueOf(resp).IsNil()) {
		return nil
	}
	if r := resp.GetResponse(); r != nil && r.Err != "" {
		return fmt.Errorf("%s", r.Err)
	}
	return nil
}

package registry

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var applyTypedValueTests = []struct {
	name      string
	valueType string
	value     string
	wantErr   bool
	check     func(*sliverpb.RegistryWriteReq) bool
}{
	{"string", "string", "hello", false, func(r *sliverpb.RegistryWriteReq) bool {
		return r.Type == sliverpb.RegistryTypeString && r.StringValue == "hello"
	}},
	{"binary with spaces", "binary", "de ad be ef", false, func(r *sliverpb.RegistryWriteReq) bool {
		return r.Type == sliverpb.RegistryTypeBinary && len(r.ByteValue) == 4 && r.ByteValue[0] == 0xde
	}},
	{"dword hex", "dword", "0xff", false, func(r *sliverpb.RegistryWriteReq) bool {
		return r.Type == sliverpb.RegistryTypeDWORD && r.DWordValue == 255
	}},
	{"qword", "qword", "4294967296", false, func(r *sliverpb.RegistryWriteReq) bool {
		return r.Type == sliverpb.RegistryTypeQWORD && r.QWordValue == 4294967296
	}},
	{"dword overflow", "dword", "4294967296", true, nil},
	{"bad hex", "binary", "zz", true, nil},
	{"unsupported", "blob", "x", true, nil},
}

func TestApplyTypedValue(t *testing.T) {
	for _, tc := range applyTypedValueTests {
		t.Run(tc.name, func(t *testing.T) {
			req := &sliverpb.RegistryWriteReq{}
			err := applyTypedValue(req, tc.valueType, tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q=%q, got nil", tc.valueType, tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.check(req) {
				t.Fatalf("request fields wrong: %+v", req)
			}
		})
	}
}

package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
)

func TestGetOperatorsRequiresConnection(t *testing.T) {
	fake := &fakeServerRPC{}
	svc := &Service{rpc: fake}

	_, err := svc.GetOperators()

	if !errors.Is(err, rpc.ErrNotConnected) {
		t.Fatalf("GetOperators() error = %v, want %v", err, rpc.ErrNotConnected)
	}
	if fake.getOperatorsCalls != 0 {
		t.Fatalf("GetOperators RPC called %d times while disconnected", fake.getOperatorsCalls)
	}
}

func TestGetOperatorsReturnsRPCResponse(t *testing.T) {
	want := &clientpb.Operators{Operators: []*clientpb.Operator{{Name: "alice"}}}
	fake := &fakeServerRPC{connected: true, operators: want}
	svc := &Service{rpc: fake}

	got, err := svc.GetOperators()

	if err != nil {
		t.Fatalf("GetOperators() returned error: %v", err)
	}
	if got != want {
		t.Fatalf("GetOperators() = %p, want %p", got, want)
	}
}

func TestRestartJobsShapesRequestAndPropagatesError(t *testing.T) {
	wantErr := errors.New("restart failed")
	fake := &fakeServerRPC{connected: true, restartJobsErr: wantErr}
	svc := &Service{rpc: fake}

	err := svc.RestartJobs([]uint32{1, 3, 5})

	if !errors.Is(err, wantErr) {
		t.Fatalf("RestartJobs() error = %v, want %v", err, wantErr)
	}
	assertUint32Slice(t, fake.restartJobsReq.JobIDs, []uint32{1, 3, 5})
}

func TestTrafficEncoderMapRequiresConnection(t *testing.T) {
	fake := &fakeServerRPC{}
	svc := &Service{rpc: fake}

	_, err := svc.TrafficEncoderMap()

	if !errors.Is(err, rpc.ErrNotConnected) {
		t.Fatalf("TrafficEncoderMap() error = %v, want %v", err, rpc.ErrNotConnected)
	}
	if fake.trafficEncoderMapCalls != 0 {
		t.Fatalf("TrafficEncoderMap RPC called %d times while disconnected", fake.trafficEncoderMapCalls)
	}
}

func TestAddTrafficEncoderReadsFileAndCallsRPC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alpha.wasm")
	if err := os.WriteFile(path, []byte{0, 97, 115, 109}, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	want := &clientpb.TrafficEncoderTests{}
	fake := &fakeServerRPC{connected: true, trafficEncoderTests: want}
	svc := &Service{rpc: fake}

	got, err := svc.AddTrafficEncoder(path, true)

	if err != nil {
		t.Fatalf("AddTrafficEncoder() returned error: %v", err)
	}
	if got != want {
		t.Fatalf("AddTrafficEncoder() = %p, want %p", got, want)
	}
	req := fake.trafficEncoderAddReq
	if req == nil {
		t.Fatal("TrafficEncoderAdd was not called")
	}
	if req.Wasm.Name != "alpha.wasm" {
		t.Fatalf("encoder name = %q, want alpha.wasm", req.Wasm.Name)
	}
	if string(req.Wasm.Data) != string([]byte{0, 97, 115, 109}) {
		t.Fatalf("encoder data = %v, want wasm bytes", req.Wasm.Data)
	}
	if !req.SkipTests {
		t.Fatal("SkipTests = false, want true")
	}
	if req.TestID == "" {
		t.Fatal("TestID is empty")
	}
}

func TestAddTrafficEncoderRejectsInvalidFilesBeforeRPC(t *testing.T) {
	fake := &fakeServerRPC{connected: true}
	svc := &Service{rpc: fake}

	_, err := svc.AddTrafficEncoder("payload.txt", false)

	if err == nil || err.Error() != "traffic encoder must be a .wasm file" {
		t.Fatalf("AddTrafficEncoder() error = %v, want .wasm validation error", err)
	}
	if fake.trafficEncoderAddReq != nil {
		t.Fatalf("TrafficEncoderAdd request = %#v, want no RPC call", fake.trafficEncoderAddReq)
	}
}

func TestRemoveTrafficEncoderNormalizesNameBeforeRPC(t *testing.T) {
	fake := &fakeServerRPC{connected: true}
	svc := &Service{rpc: fake}

	err := svc.RemoveTrafficEncoder(" /tmp/beta.WASM ")

	if err != nil {
		t.Fatalf("RemoveTrafficEncoder() returned error: %v", err)
	}
	if fake.trafficEncoderRmReq == nil || fake.trafficEncoderRmReq.Wasm.Name != "beta.WASM" {
		t.Fatalf("TrafficEncoderRm request = %#v, want wasm name beta.WASM", fake.trafficEncoderRmReq)
	}
}

func TestInfoMethodsUseExpectedRPCBindings(t *testing.T) {
	fake := &fakeServerRPC{
		connected: true,
		caInfo:    &clientpb.CertificateAuthorityInfo{},
		compiler:  &clientpb.Compiler{},
		canaries:  &clientpb.Canaries{},
		websites:  &clientpb.Websites{},
	}
	svc := &Service{rpc: fake}

	if _, err := svc.CertificateAuthorityInfo(); err != nil {
		t.Fatalf("CertificateAuthorityInfo() returned error: %v", err)
	}
	if _, err := svc.Compiler(); err != nil {
		t.Fatalf("Compiler() returned error: %v", err)
	}
	if _, err := svc.Canaries(); err != nil {
		t.Fatalf("Canaries() returned error: %v", err)
	}
	if _, err := svc.GetWebsites(); err != nil {
		t.Fatalf("GetWebsites() returned error: %v", err)
	}
	if fake.caInfoCalls != 1 || fake.compilerCalls != 1 || fake.canariesCalls != 1 || fake.websitesCalls != 1 {
		t.Fatalf(
			"RPC calls = ca:%d compiler:%d canaries:%d websites:%d, want all 1",
			fake.caInfoCalls,
			fake.compilerCalls,
			fake.canariesCalls,
			fake.websitesCalls,
		)
	}
}

func TestGetCertificatesUsesExpectedRPCBinding(t *testing.T) {
	want := &clientpb.CertificateInfo{}
	fake := &fakeServerRPC{connected: true, certificateInfo: want}
	svc := &Service{rpc: fake}

	got, err := svc.GetCertificates()

	if err != nil {
		t.Fatalf("GetCertificates() returned error: %v", err)
	}
	if got != want {
		t.Fatalf("GetCertificates() = %p, want %p", got, want)
	}
	if fake.certificateInfoCalls != 1 {
		t.Fatalf("GetCertificateInfo called %d times, want 1", fake.certificateInfoCalls)
	}
}

func TestHTTPC2ProfileByNameTrimsNameBeforeRPC(t *testing.T) {
	want := &clientpb.HTTPC2Config{Name: "profile-a"}
	fake := &fakeServerRPC{connected: true, httpC2Profile: want}
	svc := &Service{rpc: fake}

	got, err := svc.HTTPC2ProfileByName(" profile-a ")

	if err != nil {
		t.Fatalf("HTTPC2ProfileByName() returned error: %v", err)
	}
	if got != want {
		t.Fatalf("HTTPC2ProfileByName() = %p, want %p", got, want)
	}
	if fake.httpC2ProfileReq == nil || fake.httpC2ProfileReq.Name != "profile-a" {
		t.Fatalf("GetHTTPC2ProfileByName request = %#v, want trimmed profile name", fake.httpC2ProfileReq)
	}
}

func TestSaveHTTPC2ProfileRejectsBlankNameBeforeRPC(t *testing.T) {
	fake := &fakeServerRPC{connected: true}
	svc := &Service{rpc: fake}

	err := svc.SaveHTTPC2Profile(&clientpb.HTTPC2Config{Name: " "}, true)

	if err == nil || err.Error() != "HTTP C2 profile name is required" {
		t.Fatalf("SaveHTTPC2Profile() error = %v, want name validation error", err)
	}
	if fake.saveHTTPConfigReq != nil {
		t.Fatalf("SaveHTTPC2Profile request = %#v, want no RPC call", fake.saveHTTPConfigReq)
	}
}

func TestSaveHTTPC2ProfileShapesRequestAndPropagatesError(t *testing.T) {
	wantErr := errors.New("save failed")
	config := &clientpb.HTTPC2Config{Name: "profile-a"}
	fake := &fakeServerRPC{connected: true, saveHTTPConfigErr: wantErr}
	svc := &Service{rpc: fake}

	err := svc.SaveHTTPC2Profile(config, true)

	if !errors.Is(err, wantErr) {
		t.Fatalf("SaveHTTPC2Profile() error = %v, want %v", err, wantErr)
	}
	if fake.saveHTTPConfigReq == nil {
		t.Fatal("SaveHTTPC2Profile RPC was not called")
	}
	if fake.saveHTTPConfigReq.C2Config != config {
		t.Fatalf("C2Config = %p, want %p", fake.saveHTTPConfigReq.C2Config, config)
	}
	if !fake.saveHTTPConfigReq.Overwrite {
		t.Fatal("Overwrite = false, want true")
	}
}

type fakeServerRPC struct {
	connected bool

	operators           *clientpb.Operators
	caInfo              *clientpb.CertificateAuthorityInfo
	compiler            *clientpb.Compiler
	canaries            *clientpb.Canaries
	trafficEncoderMap   *clientpb.TrafficEncoderMap
	trafficEncoderTests *clientpb.TrafficEncoderTests
	websites            *clientpb.Websites
	certificateInfo     *clientpb.CertificateInfo
	httpC2Profiles      *clientpb.HTTPC2Configs
	httpC2Profile       *clientpb.HTTPC2Config

	getOperatorsCalls      int
	caInfoCalls            int
	compilerCalls          int
	canariesCalls          int
	trafficEncoderMapCalls int
	websitesCalls          int
	restartJobsReq         *clientpb.RestartJobReq
	trafficEncoderAddReq   *clientpb.TrafficEncoder
	trafficEncoderRmReq    *clientpb.TrafficEncoder
	certificateInfoCalls   int
	httpC2ProfilesCalls    int
	httpC2ProfileReq       *clientpb.C2ProfileReq
	saveHTTPConfigReq      *clientpb.HTTPC2ConfigReq
	restartJobsErr         error
	trafficEncoderAddErr   error
	trafficEncoderRmErr    error
	saveHTTPConfigErr      error
}

func (f *fakeServerRPC) Connected() bool {
	return f.connected
}

func (f *fakeServerRPC) GetOperators(context.Context, *commonpb.Empty) (*clientpb.Operators, error) {
	f.getOperatorsCalls++
	return f.operators, nil
}

func (f *fakeServerRPC) RestartJobs(_ context.Context, req *clientpb.RestartJobReq) (*commonpb.Empty, error) {
	f.restartJobsReq = req
	return &commonpb.Empty{}, f.restartJobsErr
}

func (f *fakeServerRPC) GetCertificateAuthorityInfo(
	context.Context,
	*commonpb.Empty,
) (*clientpb.CertificateAuthorityInfo, error) {
	f.caInfoCalls++
	return f.caInfo, nil
}

func (f *fakeServerRPC) GetCompiler(context.Context, *commonpb.Empty) (*clientpb.Compiler, error) {
	f.compilerCalls++
	return f.compiler, nil
}

func (f *fakeServerRPC) Canaries(context.Context, *commonpb.Empty) (*clientpb.Canaries, error) {
	f.canariesCalls++
	return f.canaries, nil
}

func (f *fakeServerRPC) TrafficEncoderMap(context.Context, *commonpb.Empty) (*clientpb.TrafficEncoderMap, error) {
	f.trafficEncoderMapCalls++
	return f.trafficEncoderMap, nil
}

func (f *fakeServerRPC) TrafficEncoderAdd(
	_ context.Context,
	req *clientpb.TrafficEncoder,
) (*clientpb.TrafficEncoderTests, error) {
	f.trafficEncoderAddReq = req
	return f.trafficEncoderTests, f.trafficEncoderAddErr
}

func (f *fakeServerRPC) TrafficEncoderRm(_ context.Context, req *clientpb.TrafficEncoder) (*commonpb.Empty, error) {
	f.trafficEncoderRmReq = req
	return &commonpb.Empty{}, f.trafficEncoderRmErr
}

func (f *fakeServerRPC) Websites(context.Context, *commonpb.Empty) (*clientpb.Websites, error) {
	f.websitesCalls++
	return f.websites, nil
}

func (f *fakeServerRPC) GetCertificateInfo(
	context.Context,
	*clientpb.CertificatesReq,
) (*clientpb.CertificateInfo, error) {
	f.certificateInfoCalls++
	return f.certificateInfo, nil
}

func (f *fakeServerRPC) GetHTTPC2Profiles(context.Context, *commonpb.Empty) (*clientpb.HTTPC2Configs, error) {
	f.httpC2ProfilesCalls++
	return f.httpC2Profiles, nil
}

func (f *fakeServerRPC) GetHTTPC2ProfileByName(
	_ context.Context,
	req *clientpb.C2ProfileReq,
) (*clientpb.HTTPC2Config, error) {
	f.httpC2ProfileReq = req
	return f.httpC2Profile, nil
}

func (f *fakeServerRPC) SaveHTTPC2Profile(_ context.Context, req *clientpb.HTTPC2ConfigReq) (*commonpb.Empty, error) {
	f.saveHTTPConfigReq = req
	return &commonpb.Empty{}, f.saveHTTPConfigErr
}

func assertUint32Slice(t *testing.T, got, want []uint32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %d, want %d (full slice %v)", i, got[i], want[i], got)
		}
	}
}

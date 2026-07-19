package server

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/rpc"
)

type serverRPC interface {
	Connected() bool
	GetOperators(context.Context, *commonpb.Empty) (*clientpb.Operators, error)
	RestartJobs(context.Context, *clientpb.RestartJobReq) (*commonpb.Empty, error)
	GetCertificateAuthorityInfo(context.Context, *commonpb.Empty) (*clientpb.CertificateAuthorityInfo, error)
	GetCompiler(context.Context, *commonpb.Empty) (*clientpb.Compiler, error)
	Canaries(context.Context, *commonpb.Empty) (*clientpb.Canaries, error)
	TrafficEncoderMap(context.Context, *commonpb.Empty) (*clientpb.TrafficEncoderMap, error)
	TrafficEncoderAdd(context.Context, *clientpb.TrafficEncoder) (*clientpb.TrafficEncoderTests, error)
	TrafficEncoderRm(context.Context, *clientpb.TrafficEncoder) (*commonpb.Empty, error)
	Websites(context.Context, *commonpb.Empty) (*clientpb.Websites, error)
	GetCertificateInfo(context.Context, *clientpb.CertificatesReq) (*clientpb.CertificateInfo, error)
	GetHTTPC2Profiles(context.Context, *commonpb.Empty) (*clientpb.HTTPC2Configs, error)
	GetHTTPC2ProfileByName(context.Context, *clientpb.C2ProfileReq) (*clientpb.HTTPC2Config, error)
	SaveHTTPC2Profile(context.Context, *clientpb.HTTPC2ConfigReq) (*commonpb.Empty, error)
}

type liveServerRPC struct {
	client *rpc.Client
}

func (r liveServerRPC) Connected() bool {
	return r.client != nil && r.client.Connected()
}

func (r liveServerRPC) GetOperators(ctx context.Context, req *commonpb.Empty) (*clientpb.Operators, error) {
	return r.client.RPC.GetOperators(ctx, req)
}

func (r liveServerRPC) RestartJobs(ctx context.Context, req *clientpb.RestartJobReq) (*commonpb.Empty, error) {
	return r.client.RPC.RestartJobs(ctx, req)
}

func (r liveServerRPC) GetCertificateAuthorityInfo(
	ctx context.Context,
	req *commonpb.Empty,
) (*clientpb.CertificateAuthorityInfo, error) {
	return r.client.RPC.GetCertificateAuthorityInfo(ctx, req)
}

func (r liveServerRPC) GetCompiler(ctx context.Context, req *commonpb.Empty) (*clientpb.Compiler, error) {
	return r.client.RPC.GetCompiler(ctx, req)
}

func (r liveServerRPC) Canaries(ctx context.Context, req *commonpb.Empty) (*clientpb.Canaries, error) {
	return r.client.RPC.Canaries(ctx, req)
}

func (r liveServerRPC) TrafficEncoderMap(ctx context.Context, req *commonpb.Empty) (*clientpb.TrafficEncoderMap, error) {
	return r.client.RPC.TrafficEncoderMap(ctx, req)
}

func (r liveServerRPC) TrafficEncoderAdd(
	ctx context.Context,
	req *clientpb.TrafficEncoder,
) (*clientpb.TrafficEncoderTests, error) {
	return r.client.RPC.TrafficEncoderAdd(ctx, req)
}

func (r liveServerRPC) TrafficEncoderRm(ctx context.Context, req *clientpb.TrafficEncoder) (*commonpb.Empty, error) {
	return r.client.RPC.TrafficEncoderRm(ctx, req)
}

func (r liveServerRPC) Websites(ctx context.Context, req *commonpb.Empty) (*clientpb.Websites, error) {
	return r.client.RPC.Websites(ctx, req)
}

func (r liveServerRPC) GetCertificateInfo(
	ctx context.Context,
	req *clientpb.CertificatesReq,
) (*clientpb.CertificateInfo, error) {
	return r.client.RPC.GetCertificateInfo(ctx, req)
}

func (r liveServerRPC) GetHTTPC2Profiles(ctx context.Context, req *commonpb.Empty) (*clientpb.HTTPC2Configs, error) {
	return r.client.RPC.GetHTTPC2Profiles(ctx, req)
}

func (r liveServerRPC) GetHTTPC2ProfileByName(
	ctx context.Context,
	req *clientpb.C2ProfileReq,
) (*clientpb.HTTPC2Config, error) {
	return r.client.RPC.GetHTTPC2ProfileByName(ctx, req)
}

func (r liveServerRPC) SaveHTTPC2Profile(
	ctx context.Context,
	req *clientpb.HTTPC2ConfigReq,
) (*commonpb.Empty, error) {
	return r.client.RPC.SaveHTTPC2Profile(ctx, req)
}

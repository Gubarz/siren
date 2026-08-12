package gui

import (
	"encoding/json"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	uishellcode "siren/internal/sliver/shellcode"
	"siren/internal/sliver/staging"
	"siren/internal/sliver/websites"
)

// ---- Server info ----

type ServerInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Operator string `json:"operator"`
	CA       string `json:"ca"`
}

// GetServerInfo returns the teamserver we're currently connected to, so the UI
// can prefill listener bind hosts / C2 URLs / etc with something useful instead
// of forcing the operator to look it up.
func (a *App) GetServerInfo() ServerInfo {
	if a.RPC == nil || a.RPC.Config == nil {
		return ServerInfo{}
	}
	return ServerInfo{
		Host:     a.RPC.Config.LHost,
		Port:     a.RPC.Config.LPort,
		Operator: a.RPC.Config.Operator,
		CA:       a.RPC.Config.CACertificate,
	}
}

// ---- Listeners / Jobs ----

func (a *App) GetJobs() (*clientpb.Jobs, error) {
	return a.Listeners.GetJobs()
}

func (a *App) KillJob(id uint32) error {
	return a.Listeners.KillJob(id)
}

func (a *App) StartListener(protocol, host string, port uint32, domains string) error {
	return a.Listeners.StartListener(protocol, host, port, domains)
}

func (a *App) StartTCPStagerListener(req staging.TCPListenerRequest) (*clientpb.StagerListener, error) {
	return a.Staging.StartTCPStagerListener(req)
}

// ---- Server / Certificates / Websites ----

func (a *App) GetCertificates() (*clientpb.CertificateInfo, error) {
	return a.Server.GetCertificates()
}

func (a *App) GetWebsites() (*clientpb.Websites, error) {
	return a.Server.GetWebsites()
}

func (a *App) GetAliases() (interface{}, error) {
	return a.Server.GetAliases()
}

// ---- Server misc ----

func (a *App) GetCertificateAuthorityInfo() (*clientpb.CertificateAuthorityInfo, error) {
	return a.Server.CertificateAuthorityInfo()
}

func (a *App) GetCompiler() (*clientpb.Compiler, error) {
	return a.Server.Compiler()
}

func (a *App) GetCanaries() (*clientpb.Canaries, error) {
	return a.Server.Canaries()
}

func (a *App) RestartJobs(jobIDs []uint32) error {
	return a.Server.RestartJobs(jobIDs)
}

func (a *App) LogClient(line string) {
	if a.ClientLog != nil {
		a.ClientLog.Log(line)
	}
}

// ---- Websites ----

func (a *App) GetWebsite(name string) (*clientpb.Website, error) {
	return a.Websites.GetWebsite(name)
}

func (a *App) RemoveWebsite(name string) error {
	return a.Websites.RemoveWebsite(name)
}

func (a *App) AddWebsiteContent(req websites.AddContentRequest) error {
	return a.Websites.AddContent(req)
}

func (a *App) UpdateWebsiteContent(req websites.AddContentRequest) error {
	return a.Websites.UpdateContent(req)
}

func (a *App) RemoveWebsiteContent(name string, paths []string) error {
	return a.Websites.RemoveContent(name, paths)
}

func (a *App) GetTrafficEncoderMap() (*clientpb.TrafficEncoderMap, error) {
	return a.Server.TrafficEncoderMap()
}

func (a *App) AddTrafficEncoder(localPath string, skipTests bool) (*clientpb.TrafficEncoderTests, error) {
	return a.Server.AddTrafficEncoder(localPath, skipTests)
}

func (a *App) RemoveTrafficEncoder(name string) error {
	return a.Server.RemoveTrafficEncoder(name)
}

func (a *App) GetShellcodeEncoderMap() (*clientpb.ShellcodeEncoderMap, error) {
	return uishellcode.EncoderMap(a.RPC)
}

func (a *App) GenerateShellcodeRDI(req uishellcode.RDIRequest) (string, error) {
	return uishellcode.GenerateRDI(a.ctx, a.RPC, req)
}

func (a *App) EncodeShellcode(req uishellcode.EncodeRequest) (string, error) {
	return uishellcode.Encode(a.ctx, a.RPC, req)
}

func (a *App) GetHTTPC2Profiles() (*clientpb.HTTPC2Configs, error) {
	return a.Server.HTTPC2Profiles()
}

func (a *App) GetHTTPC2ProfileByName(name string) (*clientpb.HTTPC2Config, error) {
	return a.Server.HTTPC2ProfileByName(name)
}

// ---- Monitoring Providers ----

func (a *App) MonitorStart() (*commonpb.Response, error) {
	return a.Monitor.MonitorStart()
}

func (a *App) MonitorStop() error {
	return a.Monitor.MonitorStop()
}

func (a *App) GetMonitorProviders() (*clientpb.MonitoringProviders, error) {
	return a.Monitor.ListConfig()
}

func (a *App) AddMonitorProvider(id, providerType, apiKey, apiPassword string) (*commonpb.Response, error) {
	return a.Monitor.AddConfig(&clientpb.MonitoringProvider{
		ID:          id,
		Type:        providerType,
		APIKey:      apiKey,
		APIPassword: apiPassword,
	})
}

func (a *App) RemoveMonitorProvider(id string) (*commonpb.Response, error) {
	return a.Monitor.DelConfig(&clientpb.MonitoringProvider{
		ID: id,
	})
}

func (a *App) SaveHTTPC2ProfileJSON(profileJSON string, overwrite bool) error {
	var config clientpb.HTTPC2Config
	if err := json.Unmarshal([]byte(profileJSON), &config); err != nil {
		return fmt.Errorf("invalid HTTP C2 profile JSON: %w", err)
	}
	return a.Server.SaveHTTPC2Profile(&config, overwrite)
}

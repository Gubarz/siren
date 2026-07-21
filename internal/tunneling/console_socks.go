package tunneling

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/spf13/pflag"
)

type ConsoleCommandResult struct {
	Handled bool   `json:"handled"`
	Output  string `json:"output"`
	Refresh bool   `json:"refresh"`
}

func (s *Service) HandleConsoleSocksCommand(sessionID, line string) ConsoleCommandResult {
	args, err := shellquote.Split(line)
	if err != nil || len(args) == 0 || strings.ToLower(args[0]) != "socks5" {
		return ConsoleCommandResult{}
	}

	result := ConsoleCommandResult{Handled: true}
	if len(args) == 1 {
		result.Output = s.renderSocksList(sessionID)
		return result
	}

	switch strings.ToLower(args[1]) {
	case "start":
		out, err := s.handleConsoleSocksStart(sessionID, args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	case "stop":
		out, err := s.handleConsoleSocksStop(args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	default:
		result.Output = "Usage: socks5 [start|stop]\n"
	}
	return result
}

func (s *Service) handleConsoleSocksStart(sessionID string, args []string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("socks5 start requires an active session")
	}
	flags := pflag.NewFlagSet("socks5 start", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	host := flags.StringP("host", "H", "127.0.0.1", "")
	port := flags.StringP("port", "P", "1081", "")
	user := flags.StringP("user", "u", "", "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	password := ""
	var warnings []string
	if strings.TrimSpace(*user) != "" {
		var err error
		password, err = randomSocksPassword()
		if err != nil {
			return "", err
		}
		warnings = append(warnings,
			"[!] SOCKS proxy authentication credentials are tunneled to the implant",
			"[!] These credentials are recoverable from the implant's memory!",
		)
	}

	lines := append(warnings, fmt.Sprintf("[*] Started SOCKS5 %s %s %s %s", strings.TrimSpace(*host), strings.TrimSpace(*port), strings.TrimSpace(*user), password))
	lines = append(lines, "[!] In-band SOCKS proxies can be a little unstable depending on protocol")
	return strings.Join(lines, "\n") + "\n", nil
}

func (s *Service) handleConsoleSocksStop(args []string) (string, error) {
	flags := pflag.NewFlagSet("socks5 stop", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	id := flags.Uint64P("id", "i", 0, "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if *id == 0 && flags.NArg() > 0 {
		parsed, err := strconv.ParseUint(flags.Arg(0), 10, 64)
		if err != nil {
			return "", fmt.Errorf("must specify a valid socks5 id")
		}
		*id = parsed
	}
	if *id == 0 {
		return "", fmt.Errorf("must specify a valid socks5 id")
	}
	if err := s.StopSocks(*id); err != nil {
		return "", err
	}
	return "[*] Removed socks5\n", nil
}

func (s *Service) renderSocksList(sessionID string) string {
	proxies := s.List()
	socks := make([]ProxyInfo, 0, len(proxies))
	for _, proxy := range proxies {
		if proxy.Kind != "socks" {
			continue
		}
		if sessionID != "" && proxy.SessionID != sessionID {
			continue
		}
		socks = append(socks, proxy)
	}
	if len(socks) == 0 {
		return "[*] No socks5 proxies\n"
	}
	sort.Slice(socks, func(i, j int) bool {
		return socks[i].ID < socks[j].ID
	})

	var b strings.Builder
	fmt.Fprintf(&b, "%-6s  %-12s  %-22s  %-12s  %s\n", "ID", "Session ID", "Bind Address", "Username", "Password")
	for _, proxy := range socks {
		session := proxy.SessionID
		if len(session) > 8 {
			session = session[:8]
		}
		fmt.Fprintf(&b, "%-6d  %-12s  %-22s  %-12s  %s\n", proxy.ID, session, proxy.BindAddr, blankDash(proxy.Username), blankDash(proxy.Password))
	}
	return b.String()
}

func randomSocksPassword() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

func blankDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

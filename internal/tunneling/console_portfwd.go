package tunneling

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var portOnlyPattern = regexp.MustCompile(`^[0-9]+$`)

func (s *Service) handleConsolePortfwdArgs(sessionID string, args []string) ConsoleCommandResult {
	result := ConsoleCommandResult{Handled: true}
	if len(args) == 1 {
		result.Output = s.renderPortfwdList(sessionID)
		return result
	}

	switch strings.ToLower(args[1]) {
	case "add":
		out, err := handleConsolePortfwdAdd(s.StartPortfwd, sessionID, args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	case "rm":
		out, err := s.handleConsolePortfwdRemove(args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	default:
		result.Output = "Usage: portfwd [add|rm]\n"
	}
	return result
}

func handleConsolePortfwdAdd(start func(string, string, string) (uint64, error), sessionID string, args []string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("portfwd add requires an active session")
	}
	flags := pflag.NewFlagSet("portfwd add", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	bindAddr := flags.StringP("bind", "b", "127.0.0.1:8080", "")
	remoteAddr := flags.StringP("remote", "r", "", "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	bind := strings.TrimSpace(*bindAddr)
	remote := strings.TrimSpace(*remoteAddr)
	if portOnlyPattern.MatchString(bind) {
		bind = "127.0.0.1:" + bind
	}
	if bind == "" {
		return "", fmt.Errorf("must specify a bind target host:port")
	}
	if remote == "" {
		return "", fmt.Errorf("must specify a remote target host:port")
	}
	if _, err := start(sessionID, bind, remote); err != nil {
		return "", err
	}
	return fmt.Sprintf("[*] Port forwarding %s -> %s\n", bind, remote), nil
}

func (s *Service) handleConsolePortfwdRemove(args []string) (string, error) {
	flags := pflag.NewFlagSet("portfwd rm", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	id := flags.Uint64P("id", "i", 0, "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if *id == 0 && flags.NArg() > 0 {
		parsed, err := strconv.ParseUint(flags.Arg(0), 10, 64)
		if err != nil {
			return "", fmt.Errorf("must specify a valid portfwd id")
		}
		*id = parsed
	}
	if *id == 0 {
		return "", fmt.Errorf("must specify a valid portfwd id")
	}
	if err := s.StopPortfwd(*id); err != nil {
		return "", err
	}
	return "[*] Removed portfwd\n", nil
}

func (s *Service) renderPortfwdList(sessionID string) string {
	proxies := s.List()
	portfwds := make([]ProxyInfo, 0, len(proxies))
	for _, proxy := range proxies {
		if proxy.Kind == "portfwd" && (sessionID == "" || proxy.SessionID == sessionID) {
			portfwds = append(portfwds, proxy)
		}
	}
	if len(portfwds) == 0 {
		return "[*] No port forwards\n"
	}
	sort.Slice(portfwds, func(i, j int) bool { return portfwds[i].ID < portfwds[j].ID })

	var b strings.Builder
	fmt.Fprintf(&b, "%-6s  %-12s  %-22s  %s\n", "ID", "Session ID", "Bind Address", "Remote Address")
	for _, proxy := range portfwds {
		session := proxy.SessionID
		if len(session) > 8 {
			session = session[:8]
		}
		fmt.Fprintf(&b, "%-6d  %-12s  %-22s  %s\n", proxy.ID, session, proxy.BindAddr, proxy.RemoteAddr)
	}
	return b.String()
}

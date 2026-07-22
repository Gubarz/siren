package tunneling

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

func (s *Service) handleConsoleRportfwdArgs(sessionID string, args []string) ConsoleCommandResult {
	result := ConsoleCommandResult{Handled: true}
	if len(args) == 1 {
		proxies, err := s.ListRportfwds()
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
			return result
		}
		result.Output = formatRportfwdList(proxies, sessionID)
		return result
	}

	switch strings.ToLower(args[1]) {
	case "add":
		out, err := handleConsoleRportfwdAdd(s.startRportfwd, sessionID, args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	case "rm":
		out, err := handleConsoleRportfwdRemove(s.StopRportfwd, sessionID, args[2:])
		result.Output = out
		result.Refresh = err == nil
		if err != nil {
			result.Output = fmt.Sprintf("[!] %s\n", err)
		}
	default:
		result.Output = "Usage: rportfwd [add|rm]\n"
	}
	return result
}

func handleConsoleRportfwdAdd(start func(string, string, string) (uint64, error), sessionID string, args []string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("rportfwd add requires an active session")
	}
	flags := pflag.NewFlagSet("rportfwd add", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	bindAddr := flags.StringP("bind", "b", "", "")
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
		bind = ":" + bind
	}
	if portOnlyPattern.MatchString(remote) {
		remote = "127.0.0.1:" + remote
	}
	if bind == "" {
		return "", fmt.Errorf("must specify an implant bind address")
	}
	if remote == "" {
		return "", fmt.Errorf("must specify an operator forward address")
	}
	if _, err := start(sessionID, bind, remote); err != nil {
		return "", err
	}
	return fmt.Sprintf("[*] Reverse port forwarding %s <- %s\n", remote, bind), nil
}

func handleConsoleRportfwdRemove(stop func(uint64, string) error, sessionID string, args []string) (string, error) {
	flags := pflag.NewFlagSet("rportfwd rm", pflag.ContinueOnError)
	flags.SetOutput(&strings.Builder{})
	id := flags.Uint64P("id", "i", 0, "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if *id == 0 && flags.NArg() > 0 {
		parsed, err := strconv.ParseUint(flags.Arg(0), 10, 64)
		if err != nil {
			return "", fmt.Errorf("must specify a valid rportfwd id")
		}
		*id = parsed
	}
	if *id == 0 {
		return "", fmt.Errorf("must specify a valid rportfwd id")
	}
	if err := stop(*id, sessionID); err != nil {
		return "", err
	}
	return "[*] Removed rportfwd\n", nil
}

func formatRportfwdList(proxies []ProxyInfo, sessionID string) string {
	filtered := make([]ProxyInfo, 0, len(proxies))
	for _, proxy := range proxies {
		if proxy.Kind == "rportfwd" && (sessionID == "" || proxy.SessionID == sessionID) {
			filtered = append(filtered, proxy)
		}
	}
	if len(filtered) == 0 {
		return "[*] No reverse port forwards\n"
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })

	var b strings.Builder
	fmt.Fprintf(&b, "%-6s  %-22s  %s\n", "ID", "Remote Address", "Bind Address")
	for _, proxy := range filtered {
		fmt.Fprintf(&b, "%-6d  %-22s  %s\n", proxy.ID, proxy.RemoteAddr, proxy.BindAddr)
	}
	return b.String()
}

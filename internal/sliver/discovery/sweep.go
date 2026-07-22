package discovery

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/kballard/go-shellquote"
)

const maxSweepHosts = 256

func neighborCommand(osName string) string {
	switch osName {
	case "windows":
		return shellquote.Join("execute", "-o", "--", "arp.exe", "-a")
	case "darwin":
		return shellquote.Join("execute", "-o", "--", "/usr/sbin/arp", "-an")
	default:
		script := `ip neigh 2>/dev/null || arp -an 2>/dev/null`
		return shellquote.Join("execute", "-o", "--", "/bin/sh", "-c", script)
	}
}

func sweepCommand(osName string, hosts []string) string {
	switch osName {
	case "windows":
		quoted := make([]string, 0, len(hosts))
		for _, host := range hosts {
			quoted = append(quoted, "'"+host+"'")
		}
		script := fmt.Sprintf(
			`$p=New-Object System.Net.NetworkInformation.Ping; %s | ForEach-Object { try { $r=$p.Send($_,500); if ($r.Status -eq 'Success') { Write-Output ('DISCOVERY|' + $_ + '||' + $r.Options.Ttl) } } catch {} }`,
			strings.Join(quoted, ","),
		)
		return shellquote.Join(
			"execute", "-o", "--", "powershell.exe",
			"-NoProfile", "-NonInteractive", "-Command", script,
		)
	case "darwin":
		return shellquote.Join("execute", "-o", "--", "/bin/sh", "-c", unixSweepScript(hosts, "-c 1 -W 1000"))
	default:
		return shellquote.Join("execute", "-o", "--", "/bin/sh", "-c", unixSweepScript(hosts, "-c 1 -W 1"))
	}
}

func unixSweepScript(hosts []string, pingArgs string) string {
	return fmt.Sprintf(
		`i=0; for ip in %s; do (out=$(ping %s "$ip" 2>/dev/null) && ttl=$(printf '%%s\n' "$out" | sed -n 's/.*ttl[= ]\([0-9][0-9]*\).*/\1/p' | head -n 1) && printf 'DISCOVERY|%%s||%%s\n' "$ip" "$ttl") & i=$((i+1)); if [ $((i %% 32)) -eq 0 ]; then wait; fi; done; wait`,
		strings.Join(hosts, " "),
		pingArgs,
	)
}

func sweepHosts(value string) ([]string, error) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
	if err != nil || !prefix.Addr().Is4() {
		return nil, fmt.Errorf("enter a valid IPv4 CIDR, for example 192.168.1.0/24")
	}
	prefix = prefix.Masked()
	if prefix.Bits() < 24 {
		return nil, fmt.Errorf("sweeps are limited to /24 networks or smaller")
	}

	var hosts []string
	for address := prefix.Addr(); prefix.Contains(address); address = address.Next() {
		hosts = append(hosts, address.String())
		if len(hosts) > maxSweepHosts {
			return nil, fmt.Errorf("sweep exceeds the %d host limit", maxSweepHosts)
		}
	}
	if len(hosts) > 2 {
		hosts = hosts[1 : len(hosts)-1]
	}
	return hosts, nil
}

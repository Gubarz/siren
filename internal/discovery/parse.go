package discovery

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"
)

var (
	ipPattern          = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	macPattern         = regexp.MustCompile(`(?i)\b[0-9a-f]{2}(?:[:-][0-9a-f]{2}){5}\b`)
	markerPattern      = regexp.MustCompile(`(?m)^DISCOVERY\|([^\s|]+)\|([^\r\n|]*)\|([0-9]*)`)
	arpParenPattern    = regexp.MustCompile(`\(((?:\d{1,3}\.){3}\d{1,3})\)`)
	windowsPingPattern = regexp.MustCompile(`(?im)^\s*Reply from\s+((?:\d{1,3}\.){3}\d{1,3})\s*:.*?\bTTL[= ](\d+)\b`)
	unixPingPattern    = regexp.MustCompile(`(?im)^\s*\d+\s+bytes from\s+(?:[^\s(]+\s+\()?((?:\d{1,3}\.){3}\d{1,3})\)?[:\s].*?\bttl[= ](\d+)\b`)
)

func parseDiscoveryOutput(agentID, method, output string) []NetworkDiscovery {
	now := time.Now().UnixMilli()
	byIP := parseDiscoveryMarkers(agentID, method, output, now)

	for _, line := range strings.Split(output, "\n") {
		updateARPDevice(byIP, agentID, method, line, now)
	}
	return discoveryList(byIP)
}

func parseDiscoveryMarkers(agentID, method, output string, now int64) map[string]NetworkDiscovery {
	byIP := make(map[string]NetworkDiscovery)
	for _, match := range markerPattern.FindAllStringSubmatch(output, -1) {
		if ip, err := netip.ParseAddr(match[1]); err == nil && discoveryHostIP(ip) {
			ttl := parseTTL(match[3])
			byIP[ip.String()] = NetworkDiscovery{
				AgentID: agentID, IP: ip.String(), Hostname: strings.TrimSpace(match[2]),
				OSHint: osHintFromTTL(ttl), TTL: ttl, Method: method, LastSeen: now,
			}
		}
	}
	return byIP
}

func updateARPDevice(byIP map[string]NetworkDiscovery, agentID, method, line string, now int64) {
	ipText := ""
	parenMatch := arpParenPattern.FindStringSubmatch(line)
	mac := macPattern.FindString(line)
	if len(parenMatch) == 2 {
		if mac == "" {
			return
		}
		ipText = parenMatch[1]
	} else if mac != "" {
		ipText = ipPattern.FindString(line)
	}
	ip, err := netip.ParseAddr(ipText)
	if err != nil || !discoveryHostIP(ip) {
		return
	}
	device := byIP[ip.String()]
	device.AgentID = agentID
	device.IP = ip.String()
	device.Method = method
	device.LastSeen = now
	if mac != "" {
		device.MAC = normalizeMAC(mac)
		if !hostMAC(device.MAC) {
			return
		}
		device.Vendor = vendorFromMAC(device.MAC)
	}
	byIP[device.IP] = device
}

func discoveryList(byIP map[string]NetworkDiscovery) []NetworkDiscovery {
	devices := make([]NetworkDiscovery, 0, len(byIP))
	for _, device := range byIP {
		devices = append(devices, device)
	}
	return devices
}

func parsePingDiscoveryOutput(agentID, output string) []NetworkDiscovery {
	now := time.Now().UnixMilli()
	byIP := make(map[string]NetworkDiscovery)
	for _, pattern := range []*regexp.Regexp{windowsPingPattern, unixPingPattern} {
		for _, match := range pattern.FindAllStringSubmatch(output, -1) {
			ip, err := netip.ParseAddr(match[1])
			if err != nil || !discoveryHostIP(ip) {
				continue
			}
			ttl := parseTTL(match[2])
			byIP[ip.String()] = NetworkDiscovery{
				AgentID:  agentID,
				IP:       ip.String(),
				OSHint:   osHintFromTTL(ttl),
				TTL:      ttl,
				Method:   "ping",
				LastSeen: now,
			}
		}
	}

	return discoveryList(byIP)
}

func mergeDiscoveryResults(current, next []NetworkDiscovery) []NetworkDiscovery {
	byKey := make(map[string]NetworkDiscovery, len(current)+len(next))
	for _, device := range append(current, next...) {
		key := device.AgentID + "|" + device.IP
		existing := byKey[key]
		if device.MAC == "" {
			device.MAC = existing.MAC
		}
		if device.Hostname == "" {
			device.Hostname = existing.Hostname
		}
		if device.Vendor == "" {
			device.Vendor = existing.Vendor
		}
		if device.OSHint == "" {
			device.OSHint = existing.OSHint
			device.TTL = existing.TTL
		}
		byKey[key] = device
	}
	merged := make([]NetworkDiscovery, 0, len(byKey))
	for _, device := range byKey {
		merged = append(merged, device)
	}
	return merged
}

func parseTTL(value string) int {
	var ttl int
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &ttl)
	return ttl
}

func discoveryHostIP(ip netip.Addr) bool {
	return ip.Is4() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast() &&
		ip.String() != "255.255.255.255"
}

func osHintFromTTL(ttl int) string {
	switch {
	case ttl <= 0:
		return ""
	case ttl <= 64:
		return "Unix-like"
	case ttl <= 128:
		return "Windows-like"
	default:
		return "Network appliance"
	}
}

func normalizeMAC(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "-", ":"))
}

func hostMAC(value string) bool {
	parts := strings.Split(value, ":")
	if len(parts) != 6 || value == "ff:ff:ff:ff:ff:ff" {
		return false
	}
	var first uint
	if _, err := fmt.Sscanf(parts[0], "%02x", &first); err != nil {
		return false
	}
	return first&1 == 0
}

func vendorFromMAC(value string) string {
	return lookupOUI(value)
}

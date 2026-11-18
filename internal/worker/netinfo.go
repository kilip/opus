package worker

import (
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultNetProbeTarget = "8.8.8.8:80"

// NetInfo holds detected host identity and network interface details.
type NetInfo struct {
	Hostname   string
	IPAddress  string
	MACAddress string
}

// DetectNetInfo detects the local hostname, outbound IP toward masterAddr (falling back to 8.8.8.8:80),
// and the hardware MAC address of the interface owning that IP.
// Errors are non-fatal: unavailable fields are returned as empty strings.
func DetectNetInfo(masterAddr string) NetInfo {
	hostname, _ := os.Hostname()
	ip, mac := detectOutboundInterface(masterAddr)
	return NetInfo{
		Hostname:   hostname,
		IPAddress:  ip,
		MACAddress: mac,
	}
}

func detectOutboundInterface(masterAddr string) (string, string) {
	target := parseDialTarget(masterAddr)
	conn, err := net.DialTimeout("udp", target, 500*time.Millisecond)
	if err != nil && target != defaultNetProbeTarget {
		conn, err = net.DialTimeout("udp", defaultNetProbeTarget, 500*time.Millisecond)
	}
	if err != nil || conn == nil {
		return "", ""
	}
	defer func() { _ = conn.Close() }()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || udpAddr.IP == nil {
		return "", ""
	}
	return udpAddr.IP.String(), findMACForIP(udpAddr.IP)
}

func parseDialTarget(masterAddr string) string {
	addr := strings.TrimSpace(masterAddr)
	if addr == "" {
		return defaultNetProbeTarget
	}
	if strings.Contains(addr, "://") {
		u, err := url.Parse(addr)
		if err != nil || u.Host == "" {
			return defaultNetProbeTarget
		}
		if _, _, err := net.SplitHostPort(u.Host); err == nil {
			return u.Host
		}
		port := "80"
		if u.Scheme == "https" || u.Scheme == "wss" {
			port = "443"
		}
		return net.JoinHostPort(u.Host, port)
	}
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(addr, "80")
}

func findMACForIP(ip net.IP) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var addrIP net.IP
			switch v := a.(type) {
			case *net.IPNet:
				addrIP = v.IP
			case *net.IPAddr:
				addrIP = v.IP
			}
			if addrIP != nil && addrIP.Equal(ip) {
				return iface.HardwareAddr.String()
			}
		}
	}
	return ""
}

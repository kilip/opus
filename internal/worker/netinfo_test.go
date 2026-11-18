package worker

import (
	"net"
	"os"
	"testing"
)

func TestDetectNetInfo(t *testing.T) {
	wantHost, _ := os.Hostname()

	info := DetectNetInfo("http://127.0.0.1:7880")
	if wantHost != "" && info.Hostname != wantHost {
		t.Errorf("got Hostname %q, want %q", info.Hostname, wantHost)
	}
	if info.IPAddress != "" && net.ParseIP(info.IPAddress) == nil {
		t.Errorf("got invalid IPAddress %q", info.IPAddress)
	}
	if info.MACAddress != "" {
		if _, err := net.ParseMAC(info.MACAddress); err != nil {
			t.Errorf("got invalid MACAddress %q: %v", info.MACAddress, err)
		}
	}

	// Empty / unreachable master should never panic or fail, returning valid Hostname if available.
	fallback := DetectNetInfo("")
	if wantHost != "" && fallback.Hostname != wantHost {
		t.Errorf("fallback got Hostname %q, want %q", fallback.Hostname, wantHost)
	}
}

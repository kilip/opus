package service

import (
	"context"
	"net"

	"github.com/kilip/opus/internal/node/domain"
)

const defaultWoLBroadcastAddr = "255.255.255.255:9"

// WakeSender sends a Wake-on-LAN magic packet to the given UDP broadcast address.
type WakeSender func(ctx context.Context, addr string, pkt []byte) error

// BuildMagicPacket constructs a 102-byte Wake-on-LAN frame (6x 0xFF + 16x 6-byte MAC).
func BuildMagicPacket(mac string) ([]byte, error) {
	hw, err := net.ParseMAC(mac)
	if err != nil || len(hw) != 6 {
		return nil, domain.ErrInvalidMACAddress
	}

	pkt := make([]byte, 102)
	for i := range 6 {
		pkt[i] = 0xFF
	}
	for i := 6; i < len(pkt); i += 6 {
		copy(pkt[i:i+6], hw)
	}
	return pkt, nil
}

func sendMagicPacketUDP(_ context.Context, addr string, pkt []byte) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_, err = conn.Write(pkt)
	return err
}

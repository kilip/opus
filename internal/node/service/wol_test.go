package service_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
)

func TestBuildMagicPacket(t *testing.T) {
	pkt, err := service.BuildMagicPacket("AA-BB-CC-DD-EE-FF")
	if err != nil {
		t.Fatalf("BuildMagicPacket failed: %v", err)
	}
	if len(pkt) != 102 {
		t.Fatalf("expected 102 bytes, got %d", len(pkt))
	}
	if !bytes.Equal(pkt[:6], []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}) {
		t.Fatalf("expected 6x 0xFF header, got %x", pkt[:6])
	}
	wantMAC := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	for i := 6; i < 102; i += 6 {
		if !bytes.Equal(pkt[i:i+6], wantMAC) {
			t.Fatalf("repetition at offset %d = %x, want %x", i, pkt[i:i+6], wantMAC)
		}
	}

	for _, bad := range []string{"", "not-a-mac", "00:00:00:00:00:00:00:01"} {
		if _, err := service.BuildMagicPacket(bad); !errors.Is(err, domain.ErrInvalidMACAddress) {
			t.Errorf("BuildMagicPacket(%q) = %v, want ErrInvalidMACAddress", bad, err)
		}
	}
}

func TestNodeService_Wake(t *testing.T) {
	ctx := context.Background()
	client := setupTestClient(t)
	r := repo.New(client)

	var gotAddr string
	var gotPkt []byte
	svc := service.New(r, service.Config{
		WakeSender: func(_ context.Context, addr string, pkt []byte) error {
			gotAddr = addr
			gotPkt = append([]byte(nil), pkt...)
			return nil
		},
	})

	withMAC, err := client.Node.Create().
		SetName("wake-worker").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetMACAddress("11:22:33:44:55:66").
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker with mac: %v", err)
	}
	noMAC, err := client.Node.Create().
		SetName("no-mac-worker").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker without mac: %v", err)
	}
	badMAC, err := client.Node.Create().
		SetName("bad-mac-worker").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetMACAddress("invalid").
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker with bad mac: %v", err)
	}

	if err := svc.Wake(ctx, withMAC.ID.String()); err != nil {
		t.Fatalf("Wake failed: %v", err)
	}
	if gotAddr != "255.255.255.255:9" {
		t.Errorf("expected broadcast addr 255.255.255.255:9, got %q", gotAddr)
	}
	if len(gotPkt) != 102 {
		t.Errorf("expected 102-byte packet, got %d", len(gotPkt))
	}

	if err := svc.Wake(ctx, noMAC.ID.String()); !errors.Is(err, domain.ErrNodeMACNotConfigured) {
		t.Errorf("expected ErrNodeMACNotConfigured for nil MAC, got %v", err)
	}
	if err := svc.Wake(ctx, badMAC.ID.String()); !errors.Is(err, domain.ErrNodeMACNotConfigured) {
		t.Errorf("expected ErrNodeMACNotConfigured for invalid MAC, got %v", err)
	}
	if err := svc.Wake(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

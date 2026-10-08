package flow

import (
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket/layers"
)

// rawIPv4UDP is a minimal IPv4 + UDP packet: 10.0.0.1:12345 -> 10.0.0.2:53.
func rawIPv4UDP() []byte {
	return []byte{
		0x45, 0x00, 0x00, 0x1c, 0x00, 0x00, 0x00, 0x00, 0x40, 0x11, 0x00, 0x00,
		0x0a, 0x00, 0x00, 0x01, 0x0a, 0x00, 0x00, 0x02,
		0x30, 0x39, 0x00, 0x35, 0x00, 0x08, 0x00, 0x00,
	}
}

func TestObserveRawIPv4(t *testing.T) {
	key, l3, flags, ok := Observe(rawIPv4UDP(), layers.LinkTypeRaw)
	if !ok {
		t.Fatal("expected packet to be observed")
	}
	if key.Src != netip.MustParseAddr("10.0.0.1") {
		t.Errorf("src = %v, want 10.0.0.1", key.Src)
	}
	if key.Dst != netip.MustParseAddr("10.0.0.2") {
		t.Errorf("dst = %v, want 10.0.0.2", key.Dst)
	}
	if key.SrcPort != 12345 || key.DstPort != 53 {
		t.Errorf("ports = %d/%d, want 12345/53", key.SrcPort, key.DstPort)
	}
	if key.Proto != uint8(layers.IPProtocolUDP) {
		t.Errorf("proto = %d, want %d", key.Proto, layers.IPProtocolUDP)
	}
	if l3 != 28 {
		t.Errorf("l3 bytes = %d, want 28", l3)
	}
	if flags != 0 {
		t.Errorf("tcp flags = %d, want 0", flags)
	}
	if key.IsIPv6() {
		t.Error("IPv4 flow reported as IPv6")
	}
}

func TestObserveRejectsNonIP(t *testing.T) {
	// 14 zero bytes with an EtherType of 0x0806 (ARP): no IP layer.
	frame := make([]byte, 14)
	frame[12], frame[13] = 0x08, 0x06
	if _, _, _, ok := Observe(frame, layers.LinkTypeEthernet); ok {
		t.Fatal("expected ARP frame to be ignored")
	}
}

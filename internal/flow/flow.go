// Package flow decodes captured frames into unidirectional flow keys and keeps
// the aggregation table that feeds the NetFlow exporter.
package flow

import (
	"net/netip"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// Key identifies a unidirectional flow: the classic 5-tuple plus the IP ToS /
// traffic class. IPv4 and IPv6 addresses share the same key type.
type Key struct {
	Src     netip.Addr
	Dst     netip.Addr
	SrcPort uint16
	DstPort uint16
	Proto   uint8
	TOS     uint8
}

// IsIPv6 reports whether the flow uses IPv6 addressing.
func (k Key) IsIPv6() bool { return k.Src.Is6() || k.Dst.Is6() }

// Observe decodes a captured frame and returns its flow key, its L3 length in
// bytes and its TCP flags. ok is false when the frame carries no IP payload we
// can account for (e.g. ARP, non-IP).
//
// linkType may be layers.LinkTypeEthernet (Linux/macOS capture) or
// layers.LinkTypeRaw (Windows/WinDivert, which hands us a bare IP packet).
func Observe(data []byte, linkType layers.LinkType) (key Key, l3Bytes uint64, tcpFlags uint8, ok bool) {
	pkt := gopacket.NewPacket(data, linkType, gopacket.DecodeOptions{Lazy: true, NoCopy: true})
	net := pkt.NetworkLayer()
	if net == nil {
		return Key{}, 0, 0, false
	}

	switch l := net.(type) {
	case *layers.IPv4:
		src, sOK := netip.AddrFromSlice(l.SrcIP)
		dst, dOK := netip.AddrFromSlice(l.DstIP)
		if !sOK || !dOK {
			return Key{}, 0, 0, false
		}
		key.Src, key.Dst = src.Unmap(), dst.Unmap()
		key.TOS = l.TOS
		key.Proto = uint8(l.Protocol)
		if l.Length > 0 {
			l3Bytes = uint64(l.Length)
		} else {
			l3Bytes = uint64(len(data))
		}
	case *layers.IPv6:
		src, sOK := netip.AddrFromSlice(l.SrcIP)
		dst, dOK := netip.AddrFromSlice(l.DstIP)
		if !sOK || !dOK {
			return Key{}, 0, 0, false
		}
		key.Src, key.Dst = src.Unmap(), dst.Unmap()
		key.TOS = l.TrafficClass
		key.Proto = uint8(l.NextHeader)
		l3Bytes = 40 + uint64(l.Length)
	default:
		return Key{}, 0, 0, false
	}

	// Ports and protocol come from the transport layer when present. This also
	// resolves the protocol across IPv6 extension headers.
	switch t := pkt.TransportLayer().(type) {
	case *layers.TCP:
		key.Proto = uint8(layers.IPProtocolTCP)
		key.SrcPort, key.DstPort = uint16(t.SrcPort), uint16(t.DstPort)
		tcpFlags = tcpFlagsToByte(t)
	case *layers.UDP:
		key.Proto = uint8(layers.IPProtocolUDP)
		key.SrcPort, key.DstPort = uint16(t.SrcPort), uint16(t.DstPort)
	}

	return key, l3Bytes, tcpFlags, true
}

// tcpFlagsToByte packs gopacket's boolean TCP flags into the single byte used
// by the NetFlow TCP_FLAGS field (CWR..FIN). The NS (ECN nonce) bit is dropped.
func tcpFlagsToByte(t *layers.TCP) uint8 {
	var f uint8
	if t.FIN {
		f |= 0x01
	}
	if t.SYN {
		f |= 0x02
	}
	if t.RST {
		f |= 0x04
	}
	if t.PSH {
		f |= 0x08
	}
	if t.ACK {
		f |= 0x10
	}
	if t.URG {
		f |= 0x20
	}
	if t.ECE {
		f |= 0x40
	}
	if t.CWR {
		f |= 0x80
	}
	return f
}

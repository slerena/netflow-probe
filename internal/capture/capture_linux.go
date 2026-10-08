//go:build linux

package capture

import (
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/afpacket"
	"github.com/gopacket/gopacket/layers"
)

type afpacketSource struct {
	t    *afpacket.TPacket
	link layers.LinkType
}

func (s *afpacketSource) Next() ([]byte, gopacket.CaptureInfo, error) {
	return s.t.ReadPacketData()
}

func (s *afpacketSource) LinkType() layers.LinkType { return s.link }

func (s *afpacketSource) Close() error {
	s.t.Close()
	return nil
}

// OpenLive captures on a Linux interface using AF_PACKET (no cgo). It requires
// CAP_NET_RAW or root. The filter argument is currently ignored on Linux; see
// docs/arquitectura.md for the planned kernel-BPF path.
func OpenLive(iface string, _ string) (Source, error) {
	if iface == "" {
		return nil, fmt.Errorf("capture: interface name is required on Linux")
	}
	t, err := afpacket.NewTPacket(afpacket.OptInterface(iface))
	if err != nil {
		return nil, fmt.Errorf("afpacket %s: %w (¿CAP_NET_RAW/root?)", iface, err)
	}
	return &afpacketSource{t: t, link: layers.LinkTypeEthernet}, nil
}

//go:build darwin

package capture

import (
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/bsdbpf"
	"github.com/gopacket/gopacket/layers"
)

type bpfSource struct {
	s    *bsdbpf.BPFSniffer
	link layers.LinkType
}

func (s *bpfSource) Next() ([]byte, gopacket.CaptureInfo, error) {
	return s.s.ReadPacketData()
}

func (s *bpfSource) LinkType() layers.LinkType { return s.link }

func (s *bpfSource) Close() error { return s.s.Close() }

// OpenLive captures on a macOS interface through a /dev/bpf* device (no cgo).
// It requires root. The filter argument is currently ignored on macOS.
func OpenLive(iface string, _ string) (Source, error) {
	if iface == "" {
		return nil, fmt.Errorf("capture: interface name is required on macOS")
	}
	s, err := bsdbpf.NewBPFSniffer(iface, nil)
	if err != nil {
		return nil, fmt.Errorf("bpf %s: %w (¿ejecutando como root?)", iface, err)
	}
	return &bpfSource{s: s, link: layers.LinkTypeEthernet}, nil
}

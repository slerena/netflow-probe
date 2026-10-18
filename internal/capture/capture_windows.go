//go:build windows

package capture

import (
	"fmt"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/imgk/divert-go"
)

type divertSource struct {
	h   *divert.Handle
	buf []byte
}

func (s *divertSource) Next() ([]byte, gopacket.CaptureInfo, error) {
	var addr divert.Address
	n, err := s.h.Recv(s.buf, &addr)
	if err != nil {
		return nil, gopacket.CaptureInfo{}, err
	}
	ci := gopacket.CaptureInfo{
		Timestamp:     time.Now(),
		CaptureLength: int(n),
		Length:        int(n),
	}
	return s.buf[:n], ci, nil
}

// LinkTypeRaw tells the decoder to expect a bare IP packet (IPv4 or IPv6),
// which is exactly what WinDivert hands us at LayerNetwork.
func (s *divertSource) LinkType() layers.LinkType { return layers.LinkTypeRaw }

func (s *divertSource) Close() error { return s.h.Close() }

// OpenLive captures on Windows through WinDivert (no cgo). The iface argument
// is ignored: WinDivert captures the whole host. filter uses WinDivert's filter
// language ("ip or ipv6", "tcp or udp", "not loopback", ...). Requires
// administrator privileges and the WinDivert driver.
func OpenLive(_ string, filter string) (Source, error) {
	if filter == "" {
		filter = "ip or ipv6"
	}
	// FlagSniff: WinDivert entrega una COPIA de los paquetes sin interceptarlos.
	// Sin este flag, WinDivert los saca de la pila y, como solo hacemos Recv
	// (no reinyectamos), el equipo se queda sin red mientras captura.
	h, err := divert.Open(filter, divert.LayerNetwork, divert.PriorityDefault, divert.FlagSniff)
	if err != nil {
		return nil, fmt.Errorf("windivert open (%q): %w (¿proceso elevado y driver WinDivert presente?)", filter, err)
	}
	return &divertSource{h: h, buf: make([]byte, 65535)}, nil
}

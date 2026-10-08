// Package capture abstracts packet sources. Every backend is pure Go
// (CGO_ENABLED=0) and lives in a per-OS file with build tags:
//
//   - Linux:   AF_PACKET via github.com/gopacket/gopacket/afpacket
//   - macOS:   /dev/bpf via github.com/gopacket/gopacket/bsdbpf
//   - Windows: WinDivert via github.com/imgk/divert-go
//   - Others:  unsupported (OpenLive returns an error)
//
// The pcap/pcapng file replay backend (OpenFile) works on every platform and
// needs no privileges, which makes it ideal for tests and dry runs.
package capture

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
)

// Source yields raw captured frames. Next blocks until a frame is available or
// the source fails/ends.
type Source interface {
	Next() (data []byte, ci gopacket.CaptureInfo, err error)
	// LinkType tells the decoder how to interpret the frames (Ethernet on
	// Linux/macOS, raw IP on Windows).
	LinkType() layers.LinkType
	Close() error
}

type packetReader interface {
	ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
	LinkType() layers.LinkType
}

type readerSource struct {
	r    packetReader
	c    io.Closer
	link layers.LinkType
}

func (s *readerSource) Next() ([]byte, gopacket.CaptureInfo, error) {
	return s.r.ReadPacketData()
}

func (s *readerSource) LinkType() layers.LinkType { return s.link }

func (s *readerSource) Close() error {
	if s.c != nil {
		return s.c.Close()
	}
	return nil
}

// OpenFile opens a pcap or pcapng file for replay.
func OpenFile(path string) (Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	br := bufio.NewReaderSize(f, 1<<16)
	magic, err := br.Peek(4)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if isPCAPNG(magic) {
		r, err := pcapgo.NewNgReader(br, pcapgo.DefaultNgReaderOptions)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &readerSource{r: r, c: f, link: r.LinkType()}, nil
	}
	r, err := pcapgo.NewReader(br)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &readerSource{r: r, c: f, link: r.LinkType()}, nil
}

func isPCAPNG(magic []byte) bool {
	return len(magic) >= 4 &&
		magic[0] == 0x0a && magic[1] == 0x0d && magic[2] == 0x0d && magic[3] == 0x0a
}

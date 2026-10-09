//go:build darwin

package capture

import (
	"errors"
	"fmt"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"golang.org/x/sys/unix"
)

// errClosed is returned by Next after Close.
var errClosed = errors.New("captura detenida")

// We implement the macOS capture directly on /dev/bpf* instead of using
// gopacket/bsdbpf, which has proven fragile: it mutates a package-level default
// (breaking repeated opens) and does not retry reads interrupted by signals
// (Go delivers SIGURG), so a capture can die with "interrupted system call" or
// "corrupted BpfHdr struct". This reader retries on EINTR and resynchronises on
// malformed records.

const bpfReadBufLen = 1 << 20

// bpfHdrMinLen is the kernel's struct bpf_hdr size on 64-bit Darwin: a 32-bit
// timeval (8) + caplen (4) + datalen (4) + hdrlen (2) = 18. Go's unix.BpfHdr is
// 20 due to struct padding, so its size must NOT be used as the minimum.
const bpfHdrMinLen = 18

// bpfAlign is BPF_ALIGNMENT, which is sizeof(int32_t) == 4 (not the pointer
// size). gopacket/bsdbpf aligns to 8, which corrupts multi-record buffers.
const bpfAlign = 4

type bpfSource struct {
	fd     int
	buf    []byte
	off    int // start of the current record within buf
	end    int // number of valid bytes in buf
	link   layers.LinkType
	closed atomic.Bool
}

func (s *bpfSource) LinkType() layers.LinkType { return s.link }

func (s *bpfSource) Close() error {
	s.closed.Store(true)
	if s.fd >= 0 {
		err := syscall.Close(s.fd)
		s.fd = -1
		return err
	}
	return nil
}

// Next returns the next captured frame. It blocks until a packet is available.
func (s *bpfSource) Next() ([]byte, gopacket.CaptureInfo, error) {
	for {
		if s.closed.Load() {
			return nil, gopacket.CaptureInfo{}, errClosed
		}
		if s.off+bpfHdrMinLen <= s.end {
			hdr := (*unix.BpfHdr)(unsafe.Pointer(&s.buf[s.off]))
			caplen := int(hdr.Caplen)
			hdrlen := int(hdr.Hdrlen)
			frameStart := s.off + hdrlen
			if hdrlen >= bpfHdrMinLen && hdrlen <= 64 && caplen >= 0 && frameStart+caplen <= s.end {
				frame := make([]byte, caplen)
				copy(frame, s.buf[frameStart:frameStart+caplen])
				ci := gopacket.CaptureInfo{
					Timestamp:     time.Unix(int64(hdr.Tstamp.Sec), int64(hdr.Tstamp.Usec)*1000),
					CaptureLength: caplen,
					Length:        int(hdr.Datalen),
				}
				next := bpfAlignUp(frameStart + caplen)
				if next > s.end {
					next = s.end
				}
				s.off = next
				return frame, ci, nil
			}
			// Malformed/partial header: drop the rest of this buffer.
			s.off = s.end
		}

		n, err := readRetry(s.fd, s.buf)
		if err != nil {
			if s.closed.Load() {
				return nil, gopacket.CaptureInfo{}, errClosed
			}
			return nil, gopacket.CaptureInfo{}, err
		}
		s.off, s.end = 0, n
	}
}

// readRetry performs a blocking read, retrying on EINTR (signals).
func readRetry(fd int, p []byte) (int, error) {
	for {
		n, err := syscall.Read(fd, p)
		if err == syscall.EINTR {
			continue
		}
		return n, err
	}
}

func bpfAlignUp(x int) int {
	return (x + bpfAlign - 1) &^ (bpfAlign - 1)
}

// OpenLive captures on a macOS interface through a /dev/bpf* device (no cgo).
// It requires root (or write access to /dev/bpf*, granted by the app's
// "Conceder acceso a la captura" action). The filter argument is ignored.
func OpenLive(iface string, _ string) (Source, error) {
	if iface == "" {
		return nil, fmt.Errorf("capture: interface name is required on macOS")
	}
	fd, err := openBPFDevice()
	if err != nil {
		return nil, fmt.Errorf("bpf: %w (¿permisos sobre /dev/bpf*?)", err)
	}
	buflen, err := syscall.SetBpfBuflen(fd, bpfReadBufLen)
	if err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bpf buflen: %w", err)
	}
	if err := syscall.SetBpfInterface(fd, iface); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bpf %s: %w", iface, err)
	}
	if err := syscall.SetBpfImmediate(fd, 1); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bpf %s: immediate: %w", iface, err)
	}
	// A read timeout makes blocking reads return periodically (0 bytes when
	// idle) so Close can reliably interrupt the reader; closing the fd alone
	// does not always wake a blocked read on /dev/bpf.
	tv := syscall.Timeval{Sec: 0, Usec: 200000}
	if err := syscall.SetBpfTimeout(fd, &tv); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bpf %s: timeout: %w", iface, err)
	}
	if err := syscall.SetBpfHeadercmpl(fd, 1); err != nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("bpf %s: headercmpl: %w", iface, err)
	}
	// Promiscuous mode is best effort: some interfaces (Wi-Fi) may refuse it and
	// it is not needed to capture this host's own traffic.
	_ = syscall.SetBpfPromisc(fd, 1)

	return &bpfSource{
		fd:   fd,
		buf:  make([]byte, buflen),
		link: layers.LinkTypeEthernet,
	}, nil
}

// openBPFDevice returns the first available /dev/bpfN opened read-write.
func openBPFDevice() (int, error) {
	var lastErr error
	for i := 0; i < 256; i++ {
		fd, err := syscall.Open(fmt.Sprintf("/dev/bpf%d", i), syscall.O_RDWR, 0)
		if err == nil {
			return fd, nil
		}
		if err == syscall.ENOENT {
			break // no more devices exist
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = syscall.ENODEV
	}
	return -1, lastErr
}

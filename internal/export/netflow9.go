// Package export serialises flow records as NetFlow v9 and sends them to an
// external collector (for this project: the Pandora FMS nfcapd collector on
// UDP/9995).
package export

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

const (
	version        = 9
	templateIDIPv4 = 256
	templateIDIPv6 = 257
)

// NetFlow v9 field types we emit (RFC 3954 / IANA IPFIX registry).
const (
	fieldInBytes     = 1
	fieldInPkts      = 2
	fieldProtocol    = 4
	fieldSrcTOS      = 5
	fieldTCPFlags    = 6
	fieldL4SrcPort   = 7
	fieldIPv4SrcAddr = 8
	fieldL4DstPort   = 11
	fieldIPv4DstAddr = 12
	fieldLastSwitch  = 21
	fieldFirstSwitch = 22
	fieldIPv6SrcAddr = 27
	fieldIPv6DstAddr = 28
)

type fieldSpec struct {
	typ    uint16
	length uint16
}

// Order matters: it is the order of the fields in both the template and each
// data record.
var (
	ipv4Fields = []fieldSpec{
		{fieldInBytes, 4},
		{fieldInPkts, 4},
		{fieldProtocol, 1},
		{fieldSrcTOS, 1},
		{fieldTCPFlags, 1},
		{fieldL4SrcPort, 2},
		{fieldIPv4SrcAddr, 4},
		{fieldL4DstPort, 2},
		{fieldIPv4DstAddr, 4},
		{fieldLastSwitch, 4},
		{fieldFirstSwitch, 4},
	}
	ipv6Fields = []fieldSpec{
		{fieldInBytes, 4},
		{fieldInPkts, 4},
		{fieldProtocol, 1},
		{fieldSrcTOS, 1},
		{fieldTCPFlags, 1},
		{fieldL4SrcPort, 2},
		{fieldIPv6SrcAddr, 16},
		{fieldL4DstPort, 2},
		{fieldIPv6DstAddr, 16},
		{fieldLastSwitch, 4},
		{fieldFirstSwitch, 4},
	}
)

// Exporter sends NetFlow v9 datagrams over UDP.
type Exporter struct {
	mu       sync.Mutex
	conn     net.Conn
	addr     string
	sourceID uint32
	seq      uint32
	start    time.Time

	templateRefresh int
	v4TplAge        int
	v6TplAge        int
}

// New dials the collector. The connection is UDP, so it succeeds even if the
// collector is not listening yet.
func New(collector string, sourceID uint32, templateRefresh int) (*Exporter, error) {
	conn, err := net.Dial("udp", collector)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", collector, err)
	}
	return &Exporter{
		conn:            conn,
		addr:            collector,
		sourceID:        sourceID,
		start:           time.Now(),
		templateRefresh: templateRefresh,
		v4TplAge:        templateRefresh, // force templates on first send
		v6TplAge:        templateRefresh,
	}, nil
}

// Close releases the underlying socket.
func (e *Exporter) Close() error {
	if e.conn == nil {
		return nil
	}
	return e.conn.Close()
}

// Send serialises records (refreshing templates as needed) and writes one
// NetFlow v9 datagram. It returns the number of flow records exported.
func (e *Exporter) Send(records []flow.Record, now time.Time) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	pkt := e.encode(records, now)
	if _, err := e.conn.Write(pkt); err != nil {
		return 0, fmt.Errorf("send to %s: %w", e.addr, err)
	}
	e.seq++
	return len(records), nil
}

// encode builds the datagram and advances the exporter's internal state. The
// caller must hold e.mu.
func (e *Exporter) encode(records []flow.Record, now time.Time) []byte {
	var v4, v6 []flow.Record
	for _, r := range records {
		if r.Key.IsIPv6() {
			v6 = append(v6, r)
		} else {
			v4 = append(v4, r)
		}
	}

	needV4Tpl := len(v4) > 0 && e.v4TplAge >= e.templateRefresh
	needV6Tpl := len(v6) > 0 && e.v6TplAge >= e.templateRefresh

	var body bytes.Buffer
	count := 0
	if needV4Tpl {
		writeTemplate(&body, templateIDIPv4, ipv4Fields)
		count++
	}
	if needV6Tpl {
		writeTemplate(&body, templateIDIPv6, ipv6Fields)
		count++
	}
	if len(v4) > 0 {
		writeDataSet(&body, templateIDIPv4, ipv4Fields, v4, e.start)
		count += len(v4)
	}
	if len(v6) > 0 {
		writeDataSet(&body, templateIDIPv6, ipv6Fields, v6, e.start)
		count += len(v6)
	}

	header := make([]byte, 20)
	binary.BigEndian.PutUint16(header[0:], version)
	binary.BigEndian.PutUint16(header[2:], uint16(count))
	binary.BigEndian.PutUint32(header[4:], msSince(e.start, now))
	binary.BigEndian.PutUint32(header[8:], uint32(now.Unix()))
	binary.BigEndian.PutUint32(header[12:], e.seq)
	binary.BigEndian.PutUint32(header[16:], e.sourceID)

	// Advance per-family template ages.
	if len(v4) > 0 {
		if needV4Tpl {
			e.v4TplAge = 0
		} else {
			e.v4TplAge++
		}
	}
	if len(v6) > 0 {
		if needV6Tpl {
			e.v6TplAge = 0
		} else {
			e.v6TplAge++
		}
	}

	return append(header, body.Bytes()...)
}

func writeTemplate(buf *bytes.Buffer, id uint16, fields []fieldSpec) {
	var t bytes.Buffer
	_ = binary.Write(&t, binary.BigEndian, id)
	_ = binary.Write(&t, binary.BigEndian, uint16(len(fields)))
	for _, f := range fields {
		_ = binary.Write(&t, binary.BigEndian, f.typ)
		_ = binary.Write(&t, binary.BigEndian, f.length)
	}
	writeFlowSet(buf, 0, t.Bytes())
}

func writeDataSet(buf *bytes.Buffer, id uint16, fields []fieldSpec, recs []flow.Record, start time.Time) {
	var d bytes.Buffer
	for _, r := range recs {
		writeRecord(&d, fields, r, start)
	}
	writeFlowSet(buf, id, d.Bytes())
}

// writeFlowSet writes a FlowSet header, its content and the 4-byte padding.
func writeFlowSet(buf *bytes.Buffer, id uint16, content []byte) {
	pad := (4 - len(content)%4) % 4
	length := 4 + len(content) + pad
	_ = binary.Write(buf, binary.BigEndian, id)
	_ = binary.Write(buf, binary.BigEndian, uint16(length))
	buf.Write(content)
	for i := 0; i < pad; i++ {
		buf.WriteByte(0)
	}
}

func writeRecord(b *bytes.Buffer, fields []fieldSpec, r flow.Record, start time.Time) {
	for _, f := range fields {
		switch f.typ {
		case fieldInBytes:
			_ = binary.Write(b, binary.BigEndian, uint32(r.Bytes))
		case fieldInPkts:
			_ = binary.Write(b, binary.BigEndian, uint32(r.Packets))
		case fieldProtocol:
			b.WriteByte(r.Key.Proto)
		case fieldSrcTOS:
			b.WriteByte(r.Key.TOS)
		case fieldTCPFlags:
			b.WriteByte(r.TCPFlags)
		case fieldL4SrcPort:
			_ = binary.Write(b, binary.BigEndian, r.Key.SrcPort)
		case fieldIPv4SrcAddr:
			a := r.Key.Src.As4()
			b.Write(a[:])
		case fieldL4DstPort:
			_ = binary.Write(b, binary.BigEndian, r.Key.DstPort)
		case fieldIPv4DstAddr:
			a := r.Key.Dst.As4()
			b.Write(a[:])
		case fieldLastSwitch:
			_ = binary.Write(b, binary.BigEndian, msSince(start, r.Last))
		case fieldFirstSwitch:
			_ = binary.Write(b, binary.BigEndian, msSince(start, r.First))
		case fieldIPv6SrcAddr:
			a := r.Key.Src.As16()
			b.Write(a[:])
		case fieldIPv6DstAddr:
			a := r.Key.Dst.As16()
			b.Write(a[:])
		}
	}
}

// msSince returns the number of milliseconds between start and t, clamped to
// the uint32 range used by NetFlow for relative timestamps.
func msSince(start, t time.Time) uint32 {
	d := t.Sub(start)
	if d < 0 {
		return 0
	}
	return uint32(d.Milliseconds())
}

package export

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

func TestEncodeIPv4TemplateAndData(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	now := start.Add(3 * time.Second)
	e := &Exporter{sourceID: 7, start: start, templateRefresh: 20, v4TplAge: 20, v6TplAge: 20}

	rec := flow.Record{
		Key: flow.Key{
			Src:     netip.MustParseAddr("10.0.0.1"),
			Dst:     netip.MustParseAddr("10.0.0.2"),
			SrcPort: 12345,
			DstPort: 53,
			Proto:   17,
			TOS:     0x10,
		},
		Bytes:   1500,
		Packets: 3,
		First:   start.Add(time.Second),
		Last:    now,
	}

	pkt := e.encode([]flow.Record{rec}, now)

	if got := binary.BigEndian.Uint16(pkt[0:]); got != 9 {
		t.Fatalf("version = %d, want 9", got)
	}
	if got := binary.BigEndian.Uint16(pkt[2:]); got != 2 {
		t.Fatalf("count = %d, want 2 (1 template + 1 record)", got)
	}
	if got := binary.BigEndian.Uint32(pkt[16:]); got != 7 {
		t.Fatalf("sourceID = %d, want 7", got)
	}

	// First FlowSet must be the template (id 0).
	if got := binary.BigEndian.Uint16(pkt[20:]); got != 0 {
		t.Fatalf("first flowset id = %d, want 0 (template)", got)
	}
	tplLen := int(binary.BigEndian.Uint16(pkt[22:]))
	tplID := binary.BigEndian.Uint16(pkt[24:])
	if tplID != templateIDIPv4 {
		t.Fatalf("template id = %d, want %d", tplID, templateIDIPv4)
	}

	// Then the data FlowSet.
	dataOff := 20 + tplLen
	if got := binary.BigEndian.Uint16(pkt[dataOff:]); got != templateIDIPv4 {
		t.Fatalf("data flowset id = %d, want %d", got, templateIDIPv4)
	}
	dataLen := int(binary.BigEndian.Uint16(pkt[dataOff+2:]))
	recSize := 0
	for _, f := range ipv4Fields {
		recSize += int(f.length)
	}
	wantDataLen := 4 + recSize + (4-recSize%4)%4
	if dataLen != wantDataLen {
		t.Fatalf("data flowset length = %d, want %d", dataLen, wantDataLen)
	}

	// Decode the record values in field order.
	rec0 := pkt[dataOff+4:]
	if got := binary.BigEndian.Uint32(rec0[0:]); got != 1500 {
		t.Errorf("in_bytes = %d, want 1500", got)
	}
	if got := binary.BigEndian.Uint32(rec0[4:]); got != 3 {
		t.Errorf("in_pkts = %d, want 3", got)
	}
	if rec0[8] != 17 {
		t.Errorf("protocol = %d, want 17", rec0[8])
	}
	if rec0[9] != 0x10 {
		t.Errorf("tos = %d, want 0x10", rec0[9])
	}
}

func TestEncodeRefreshesTemplateOnlyWhenAged(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	now := start.Add(time.Second)
	e := &Exporter{sourceID: 1, start: start, templateRefresh: 5, v4TplAge: 5, v6TplAge: 5}
	rec := flow.Record{Key: flow.Key{
		Src: netip.MustParseAddr("1.1.1.1"),
		Dst: netip.MustParseAddr("2.2.2.2"),
	}, First: now, Last: now}

	// First packet: template + data => count 2.
	pkt := e.encode([]flow.Record{rec}, now)
	if got := binary.BigEndian.Uint16(pkt[2:]); got != 2 {
		t.Fatalf("first count = %d, want 2", got)
	}
	if got := binary.BigEndian.Uint16(pkt[20:]); got != 0 {
		t.Fatalf("first flowset should be a template, got id %d", got)
	}

	// Second packet (age 1 < refresh): data only => count 1, first flowset is data.
	now = now.Add(time.Second)
	pkt = e.encode([]flow.Record{rec}, now)
	if got := binary.BigEndian.Uint16(pkt[2:]); got != 1 {
		t.Fatalf("second count = %d, want 1", got)
	}
	if got := binary.BigEndian.Uint16(pkt[20:]); got != templateIDIPv4 {
		t.Fatalf("second flowset id = %d, want data (%d)", got, templateIDIPv4)
	}
}

func TestEncodeIPv6UsesIPv6Template(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	now := start.Add(time.Second)
	e := &Exporter{sourceID: 1, start: start, templateRefresh: 20, v4TplAge: 20, v6TplAge: 20}
	rec := flow.Record{Key: flow.Key{
		Src: netip.MustParseAddr("2001:db8::1"),
		Dst: netip.MustParseAddr("2001:db8::2"),
	}, First: now, Last: now}

	pkt := e.encode([]flow.Record{rec}, now)
	if got := binary.BigEndian.Uint16(pkt[24:]); got != templateIDIPv6 {
		t.Fatalf("template id = %d, want %d", got, templateIDIPv6)
	}
}

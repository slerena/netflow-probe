package flow

import (
	"net/netip"
	"testing"
	"time"
)

func mkKey(src string, sport uint16) Key {
	return Key{
		Src:     netip.MustParseAddr(src),
		Dst:     netip.MustParseAddr("192.168.1.1"),
		SrcPort: sport,
		DstPort: 443,
		Proto:   6,
	}
}

func TestTableAggregatesByKey(t *testing.T) {
	base := time.Unix(0, 0)
	now := base
	tbl := NewTable(30*time.Minute, 15*time.Second, 1000)
	tbl.SetClock(func() time.Time { return now })

	k := mkKey("10.0.0.1", 1000)
	tbl.Observe(k, 100, 0, now)
	now = now.Add(time.Second)
	tbl.Observe(k, 200, 0x02, now)

	if got := tbl.Len(); got != 1 {
		t.Fatalf("len = %d, want 1", got)
	}
	now = now.Add(20 * time.Second) // exceed inactive timeout
	recs := tbl.Expire()
	if len(recs) != 1 {
		t.Fatalf("expired %d flows, want 1", len(recs))
	}
	r := recs[0]
	if r.Bytes != 300 || r.Packets != 2 {
		t.Errorf("bytes/pkts = %d/%d, want 300/2", r.Bytes, r.Packets)
	}
	if r.TCPFlags != 0x02 {
		t.Errorf("tcp flags = %#x, want 0x02", r.TCPFlags)
	}
	if tbl.Len() != 0 {
		t.Errorf("table not emptied, len = %d", tbl.Len())
	}
}

func TestTableActiveTimeoutFlushesLiveFlow(t *testing.T) {
	base := time.Unix(0, 0)
	now := base
	tbl := NewTable(10*time.Second, 15*time.Second, 1000)
	tbl.SetClock(func() time.Time { return now })

	k := mkKey("10.0.0.9", 5000)
	tbl.Observe(k, 10, 0, now)

	// Keep the flow busy but let the active timeout pass.
	for i := 0; i < 12; i++ {
		now = now.Add(time.Second)
		tbl.Observe(k, 10, 0, now)
	}
	recs := tbl.Expire()
	if len(recs) != 1 {
		t.Fatalf("expired %d flows, want 1 (active timeout)", len(recs))
	}
	if recs[0].Packets != 13 {
		t.Errorf("packets = %d, want 13", recs[0].Packets)
	}
}

func TestTableOverflowEvictsOldest(t *testing.T) {
	now := time.Unix(0, 0)
	tbl := NewTable(30*time.Minute, 15*time.Second, 2)
	tbl.SetClock(func() time.Time { return now })

	tbl.Observe(mkKey("10.0.0.1", 1), 1, 0, now)
	now = now.Add(time.Second)
	tbl.Observe(mkKey("10.0.0.2", 2), 1, 0, now)
	now = now.Add(time.Second)
	tbl.Observe(mkKey("10.0.0.3", 3), 1, 0, now) // exceeds max, evicts the LRU (port 1)

	if got := tbl.Len(); got != 2 {
		t.Fatalf("len = %d, want 2 (capped)", got)
	}
	recs := tbl.Expire()
	if len(recs) != 1 {
		t.Fatalf("pending evictions = %d, want 1", len(recs))
	}
	if got := recs[0].Key.SrcPort; got != 1 {
		t.Errorf("evicted port = %d, want 1 (oldest)", got)
	}
}

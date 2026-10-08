package flow

import (
	"sort"
	"sync"
	"time"
)

// Record is a completed (or force-expired) flow, ready to be exported.
type Record struct {
	Key      Key
	Bytes    uint64
	Packets  uint64
	TCPFlags uint8
	First    time.Time
	Last     time.Time
}

type entry struct {
	bytes    uint64
	packets  uint64
	tcpFlags uint8
	first    time.Time
	last     time.Time
}

// Table aggregates packets into flows. It is safe for concurrent use: the
// capture goroutine calls Observe while the export goroutine calls Expire.
type Table struct {
	mu      sync.Mutex
	entries map[Key]*entry
	pending []Record

	active   time.Duration
	inactive time.Duration
	max      int

	now func() time.Time
}

// NewTable builds a flow table. active is the maximum lifetime of a flow,
// inactive is the idle timeout and max caps the number of tracked flows.
func NewTable(active, inactive time.Duration, max int) *Table {
	return &Table{
		entries:  make(map[Key]*entry),
		active:   active,
		inactive: inactive,
		max:      max,
		now:      time.Now,
	}
}

// SetClock overrides the time source. Intended for tests.
func (t *Table) SetClock(now func() time.Time) { t.now = now }

// Observe folds one packet into its flow.
func (t *Table) Observe(key Key, bytes uint64, tcpFlags uint8, ts time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e := t.entries[key]
	if e == nil {
		e = &entry{first: ts, last: ts}
		t.entries[key] = e
	}
	e.bytes += bytes
	e.packets++
	e.tcpFlags |= tcpFlags
	if ts.After(e.last) {
		e.last = ts
	}

	if len(t.entries) > t.max {
		t.expireLocked(t.now(), len(t.entries)-t.max)
	}
}

// Expire moves all timed-out flows to the pending buffer and returns everything
// pending (timed-out flows plus any earlier overflow evictions).
func (t *Table) Expire() []Record {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.expireLocked(t.now(), 0)
	out := t.pending
	t.pending = nil
	return out
}

// Flush returns every tracked flow and empties the table. Used on shutdown.
func (t *Table) Flush() []Record {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, e := range t.entries {
		t.pending = append(t.pending, e.record(k))
		delete(t.entries, k)
	}
	out := t.pending
	t.pending = nil
	return out
}

// Len reports how many flows are currently tracked.
func (t *Table) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.entries)
}

// expireLocked appends timed-out flows (and, if force > 0, the force oldest
// remaining flows) to the pending buffer.
func (t *Table) expireLocked(now time.Time, force int) {
	// 1. Time-based expiry (idle or too old).
	for k, e := range t.entries {
		if now.Sub(e.last) >= t.inactive || now.Sub(e.first) >= t.active {
			t.pending = append(t.pending, e.record(k))
			delete(t.entries, k)
		}
	}

	// 2. Overflow eviction: drop the least recently used flows.
	if force <= 0 || len(t.entries) == 0 {
		return
	}
	type candidate struct {
		key Key
		ent *entry
	}
	rest := make([]candidate, 0, len(t.entries))
	for k, e := range t.entries {
		rest = append(rest, candidate{k, e})
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i].ent.last.Before(rest[j].ent.last) })
	if force > len(rest) {
		force = len(rest)
	}
	for _, c := range rest[:force] {
		t.pending = append(t.pending, c.ent.record(c.key))
		delete(t.entries, c.key)
	}
}

func (e *entry) record(k Key) Record {
	return Record{
		Key:      k,
		Bytes:    e.bytes,
		Packets:  e.packets,
		TCPFlags: e.tcpFlags,
		First:    e.first,
		Last:     e.last,
	}
}

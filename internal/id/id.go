// Package id issues unique, time-ordered 64-bit ids without a database round
// trip, so a message can be broadcast before it is written. Layout:
// milliseconds since Epoch in the high bits, a per-millisecond sequence in
// the low 12 bits (4096 ids/ms). Single-node by design.
package id

import (
	"sync"
	"time"
)

const seqBits = 12

var Epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

type Generator struct {
	mu     sync.Mutex
	lastMs int64
	seq    int64
}

func (g *Generator) Next() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	ms := time.Since(Epoch).Milliseconds()
	if ms < g.lastMs {
		ms = g.lastMs // the clock stepped back: stay monotonic
	}
	if ms == g.lastMs {
		g.seq++
		if g.seq == 1<<seqBits {
			ms, g.seq = ms+1, 0 // sequence exhausted: borrow the next millisecond
		}
	} else {
		g.seq = 0
	}
	g.lastMs = ms
	return uint64(ms)<<seqBits | uint64(g.seq)
}

// Floor is the smallest id that could have been issued at t.
func Floor(t time.Time) uint64 {
	ms := t.Sub(Epoch).Milliseconds()
	if ms < 0 {
		return 0
	}
	return uint64(ms) << seqBits
}

// Time is when id was issued, to the millisecond.
func Time(id uint64) time.Time {
	return Epoch.Add(time.Duration(id>>seqBits) * time.Millisecond)
}

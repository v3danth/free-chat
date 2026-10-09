package message

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeInserter struct {
	mu      sync.Mutex
	batches [][]Message
	reject  uint64 // a message id that makes any batch containing it fail
}

func (f *fakeInserter) InsertBatch(_ context.Context, msgs []Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range msgs {
		if m.ID == f.reject {
			return errors.New("foreign key violation")
		}
	}
	f.batches = append(f.batches, append([]Message(nil), msgs...))
	return nil
}

func (f *fakeInserter) stored() []uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []uint64
	for _, b := range f.batches {
		for _, m := range b {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

func TestWriterBatchesAndFlushes(t *testing.T) {
	repo := &fakeInserter{reject: 3}
	w := NewWriter(repo, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	go w.Run(ctx)

	for i := range uint64(250) {
		w.Enqueue(Message{ID: i + 1})
	}
	flushCtx, done := context.WithTimeout(context.Background(), 2*time.Second)
	defer done()
	if err := w.Flush(flushCtx); err != nil {
		t.Fatal(err)
	}

	// One bad row is dropped; the 249 others still land.
	if got := len(repo.stored()); got != 249 {
		t.Fatalf("stored %d messages, want 249", got)
	}
	for _, b := range repo.batches {
		if len(b) > batchSize {
			t.Fatalf("batch of %d exceeds %d", len(b), batchSize)
		}
	}

	w.Enqueue(Message{ID: 1000})
	cancel()
	<-w.Done()
	if ids := repo.stored(); ids[len(ids)-1] != 1000 {
		t.Fatal("shutdown must drain the queue")
	}
}

func TestEnqueueNeverBlocks(t *testing.T) {
	w := NewWriter(&fakeInserter{}, 1) // Run is not started: the queue fills
	if !w.Enqueue(Message{ID: 1}) {
		t.Fatal("first message fits")
	}
	if w.Enqueue(Message{ID: 2}) {
		t.Fatal("a full queue must drop instead of blocking chat")
	}
}

// stallInserter hangs any multi-row insert until its deadline, like a lock
// wait; single rows go through while their own deadline is still ahead.
type stallInserter struct{ fakeInserter }

func (s *stallInserter) InsertBatch(ctx context.Context, msgs []Message) error {
	if len(msgs) > 1 {
		<-ctx.Done()
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.fakeInserter.InsertBatch(ctx, msgs)
}

func TestStalledBatchDoesNotLoseMessages(t *testing.T) {
	repo := &stallInserter{}
	w := NewWriter(repo, 10)
	w.wait = 50 * time.Millisecond
	w.write([]Message{{ID: 1}, {ID: 2}, {ID: 3}})
	if got := repo.stored(); len(got) != 3 {
		t.Fatalf("stored %v, want all 3 after the stalled batch", got)
	}
}

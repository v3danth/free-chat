package message

import (
	"context"
	"log"
	"time"
)

const (
	batchSize   = 100
	batchWindow = 20 * time.Millisecond
	writeWait   = 5 * time.Second // per insert, so one stall cannot pile up
)

type batchInserter interface {
	InsertBatch(ctx context.Context, msgs []Message) error
}

// Writer persists messages off the hot path. Chat delivery never waits on
// MySQL: messages are queued and written in batches of up to 100 every
// 20ms, one round trip and one commit per batch. The trade-off is that a
// crash can lose the last batch window of the log.
type Writer struct {
	repo  batchInserter
	in    chan Message
	flush chan chan struct{}
	done  chan struct{}
	wait  time.Duration // deadline for each insert
}

func NewWriter(repo batchInserter, buffer int) *Writer {
	return &Writer{
		repo:  repo,
		in:    make(chan Message, buffer),
		flush: make(chan chan struct{}),
		done:  make(chan struct{}),
		wait:  writeWait,
	}
}

// Enqueue never blocks. It reports false when the queue is full (the
// database is down or far behind); chat keeps working and the drop is logged.
func (w *Writer) Enqueue(m Message) bool {
	select {
	case w.in <- m:
		return true
	default:
		log.Printf("message writer: queue full, dropped message %d", m.ID)
		return false
	}
}

// Flush returns once everything enqueued before the call is written.
func (w *Writer) Flush(ctx context.Context) error {
	ack := make(chan struct{})
	select {
	case w.flush <- ack:
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-ack:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run writes until ctx is cancelled, then drains the queue and returns.
func (w *Writer) Run(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(batchWindow)
	defer ticker.Stop()

	batch := make([]Message, 0, batchSize)
	write := func() {
		if len(batch) > 0 {
			w.write(batch)
			batch = batch[:0]
		}
	}
	drain := func() {
		for {
			select {
			case m := <-w.in:
				if batch = append(batch, m); len(batch) == batchSize {
					write()
				}
			default:
				write()
				return
			}
		}
	}

	for {
		select {
		case m := <-w.in:
			if batch = append(batch, m); len(batch) == batchSize {
				write()
			}
		case <-ticker.C:
			write()
		case ack := <-w.flush:
			drain()
			close(ack)
		case <-ctx.Done():
			drain()
			return
		}
	}
}

// Done is closed once Run has returned and the queue is drained.
func (w *Writer) Done() <-chan struct{} { return w.done }

func (w *Writer) write(batch []Message) {
	err := w.insert(batch)
	if err == nil {
		return
	}
	log.Printf("message writer: batch of %d failed, retrying one by one: %v", len(batch), err)
	// One bad row (say, its sender was purged) must not lose the others.
	// Each retry gets its own deadline: if the batch timed out, reusing its
	// context would fail every retry at once.
	for _, m := range batch {
		if err := w.insert([]Message{m}); err != nil {
			log.Printf("message writer: dropped message %d: %v", m.ID, err)
		}
	}
}

func (w *Writer) insert(batch []Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), w.wait)
	defer cancel()
	return w.repo.InsertBatch(ctx, batch)
}

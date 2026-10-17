package websocket

import (
	"bytes"
	"strings"
	"testing"
)

func TestBatchQueuedSendsALoneFrameAsItIs(t *testing.T) {
	queue := make(chan []byte, 4)
	first := []byte(`{"type":"chat"}`)
	msg, _, closed := batchQueued(nil, first, queue)
	if &msg[0] != &first[0] || closed {
		t.Fatal("a lone frame must go out untouched, without a copy")
	}
}

func TestBatchQueuedJoinsWaitingFramesOnePerLine(t *testing.T) {
	queue := make(chan []byte, 4)
	queue <- []byte(`{"n":2}`)
	queue <- []byte(`{"n":3}`)
	msg, scratch, closed := batchQueued(nil, []byte(`{"n":1}`), queue)
	if string(msg) != "{\"n\":1}\n{\"n\":2}\n{\"n\":3}" || closed || len(queue) != 0 {
		t.Fatalf("got %q closed=%v", msg, closed)
	}
	// The scratch buffer is reused, and a lone frame afterwards is not
	// mixed up with what it held.
	msg, _, _ = batchQueued(scratch, []byte(`{"n":4}`), queue)
	if string(msg) != `{"n":4}` {
		t.Fatalf("reuse: %q", msg)
	}
}

func TestBatchQueuedReportsAClosedQueue(t *testing.T) {
	queue := make(chan []byte, 2)
	queue <- []byte(`{"n":2}`)
	close(queue)
	msg, _, closed := batchQueued(nil, []byte(`{"n":1}`), queue)
	if !closed || string(msg) != "{\"n\":1}\n{\"n\":2}" {
		t.Fatalf("got %q closed=%v", msg, closed)
	}
}

func TestBatchQueuedStopsAtTheCap(t *testing.T) {
	queue := make(chan []byte, 64)
	frame := []byte(`{"pad":"` + strings.Repeat("x", 4<<10) + `"}`)
	for range cap(queue) {
		queue <- frame
	}
	msg, _, _ := batchQueued(nil, frame, queue)
	if len(msg) > maxBatch+len(frame)+1 || len(queue) == 0 {
		t.Fatalf("batch of %d bytes with %d left; cap is %d", len(msg), len(queue), maxBatch)
	}
	if n := bytes.Count(msg, []byte{'\n'}) + 1; n < 2 {
		t.Fatalf("only %d frames batched", n)
	}
}

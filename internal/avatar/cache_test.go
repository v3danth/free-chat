package avatar

import (
	"fmt"
	"testing"

	"github.com/v3danth/free-chat/internal/httpx"
)

func drawOf(size int) func() (httpx.Payload, error) {
	return func() (httpx.Payload, error) { return httpx.Payload{Body: make([]byte, size)}, nil }
}

func TestRenderCacheStaysInBudget(t *testing.T) {
	c := newRenderCache(10_000)
	for i := range 500 {
		if _, err := c.payload(fmt.Sprint(i), drawOf(100)); err != nil {
			t.Fatal(err)
		}
		if c.used > c.budget {
			t.Fatalf("after %d inserts: used %d over budget %d", i+1, c.used, c.budget)
		}
	}
	if len(c.byKey) != 100 {
		t.Fatalf("want 100 entries of 100 bytes in 10 000, got %d", len(c.byKey))
	}
}

func TestRenderCacheKeepsWhatIsUsed(t *testing.T) {
	c := newRenderCache(10_000)
	draws := 0
	counted := func() (httpx.Payload, error) { draws++; return drawOf(100)() }
	for i := range 2000 {
		// "hot" is asked for on every round; everything else once.
		if _, err := c.payload("hot", counted); err != nil {
			t.Fatal(err)
		}
		if _, err := c.payload(fmt.Sprint(i), drawOf(100)); err != nil {
			t.Fatal(err)
		}
	}
	if draws != 1 {
		t.Fatalf("a face used every round was drawn %d times, want once", draws)
	}
}

func TestRenderCacheSkipsWhatCannotFit(t *testing.T) {
	c := newRenderCache(50)
	if _, err := c.payload("big", drawOf(100)); err != nil {
		t.Fatal(err)
	}
	if len(c.byKey) != 0 || c.used != 0 {
		t.Fatalf("an oversized face must not be kept: %d entries, %d bytes", len(c.byKey), c.used)
	}
}

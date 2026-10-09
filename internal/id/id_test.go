package id

import (
	"testing"
	"time"
)

func TestNextIsUniqueAndOrdered(t *testing.T) {
	var g Generator
	prev := g.Next()
	for range 100_000 { // forces sequence exhaustion within a millisecond
		next := g.Next()
		if next <= prev {
			t.Fatalf("ids must strictly increase: %d then %d", prev, next)
		}
		prev = next
	}
}

func TestFloorAndTime(t *testing.T) {
	var g Generator
	before := time.Now().Add(-time.Millisecond)
	got := g.Next()
	if got < Floor(before) {
		t.Fatalf("id %d is below the floor for a time before it was issued", got)
	}
	if d := time.Since(Time(got)); d < 0 || d > time.Second {
		t.Fatalf("Time(id) is off by %v", d)
	}
	if Floor(Epoch.Add(-time.Hour)) != 0 {
		t.Fatal("times before the epoch floor to 0")
	}
}

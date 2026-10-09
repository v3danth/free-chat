package ratelimit

import (
	"testing"
	"time"
)

func TestStep(t *testing.T) {
	cfg := Config{Rate: 2, Window: time.Minute}
	t0 := time.Unix(1_000_000, 0)

	var w window
	var ok bool
	for i, want := range []bool{true, true, false} {
		if w, ok = step(w, t0, cfg); ok != want {
			t.Fatalf("event %d: allowed = %v, want %v", i, ok, want)
		}
	}
	if r := remaining(w, t0, cfg); r != 0 {
		t.Fatalf("remaining = %d, want 0", r)
	}

	later := t0.Add(cfg.Window)
	if r := remaining(w, later, cfg); r != cfg.Rate {
		t.Fatalf("remaining after window = %d, want %d", r, cfg.Rate)
	}
	if _, ok = step(w, later, cfg); !ok {
		t.Fatal("new window should allow")
	}
}

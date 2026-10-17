package main

import (
	"testing"
	"time"
)

func TestSummarizeUsesNearestRank(t *testing.T) {
	samples := make([]time.Duration, 0, 100)
	for i := 100; i >= 1; i-- { // reversed, to prove it sorts
		samples = append(samples, time.Duration(i)*time.Millisecond)
	}
	s := summarize("x", samples, 2, 10*time.Second)
	want := map[string][2]time.Duration{
		"p50": {s.P50, 50 * time.Millisecond}, "p90": {s.P90, 90 * time.Millisecond},
		"p95": {s.P95, 95 * time.Millisecond}, "p99": {s.P99, 99 * time.Millisecond},
		"max": {s.Max, 100 * time.Millisecond},
	}
	for name, got := range want {
		if got[0] != got[1] {
			t.Errorf("%s = %v, want %v", name, got[0], got[1])
		}
	}
	if s.Count != 100 || s.Errors != 2 || s.PerSecond != 10 {
		t.Errorf("counts: %+v", s)
	}
	if samples[0] != 100*time.Millisecond {
		t.Error("summarize must not reorder the caller's samples")
	}
}

func TestSummarizeHandlesTinyInputs(t *testing.T) {
	if s := summarize("empty", nil, 1, time.Second); s.Count != 0 || s.P99 != 0 {
		t.Fatalf("empty: %+v", s)
	}
	if s := summarize("one", []time.Duration{7}, 0, 0); s.P50 != 7 || s.P99 != 7 || s.PerSecond != 0 {
		t.Fatalf("one sample: %+v", s)
	}
}

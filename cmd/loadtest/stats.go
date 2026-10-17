package main

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// summary is the latency distribution of one target.
type summary struct {
	Name                    string
	Count, Errors           int
	P50, P90, P95, P99, Max time.Duration
	PerSecond               float64
}

// summarize turns raw samples into percentiles by nearest rank, the
// definition load-testing tools use. It sorts a copy; samples is unchanged.
func summarize(name string, samples []time.Duration, errors int, elapsed time.Duration) summary {
	s := summary{Name: name, Count: len(samples), Errors: errors}
	if elapsed > 0 {
		s.PerSecond = float64(len(samples)) / elapsed.Seconds()
	}
	if len(samples) == 0 {
		return s
	}
	sorted := slices.Clone(samples)
	slices.Sort(sorted)
	s.P50, s.P90, s.P95, s.P99 = rank(sorted, 50), rank(sorted, 90), rank(sorted, 95), rank(sorted, 99)
	s.Max = sorted[len(sorted)-1]
	return s
}

// rank is the nearest-rank percentile p of sorted samples.
func rank(sorted []time.Duration, p int) time.Duration {
	i := (p*len(sorted)+99)/100 - 1
	return sorted[max(i, 0)]
}

func table(title string, rows []summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", title)
	fmt.Fprintf(&b, "%-30s %7s %6s %8s %8s %8s %8s %8s %8s\n", "target", "n", "errors", "req/s", "p50", "p90", "p95", "p99", "max")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-30s %7d %6d %8.1f %8s %8s %8s %8s %8s\n", r.Name, r.Count, r.Errors, r.PerSecond,
			ms(r.P50), ms(r.P90), ms(r.P95), ms(r.P99), ms(r.Max))
	}
	return b.String()
}

func ms(d time.Duration) string { return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000) }

package filter

import (
	"slices"
	"testing"
)

func TestApply(t *testing.T) {
	f := New(100, []Rule{
		{Word: "Darn", Action: Mask},
		{Word: "heck", Action: Mask},
		{Word: "scam", Action: Block},
		{Word: "telegram.me", Action: Block},
		{Word: "कमीना", Action: Mask},
	})

	tests := []struct {
		in, want   string
		blocked    bool
		violations []string
	}{
		{"hello", "hello", false, nil},
		{"darn it", "d*** it", false, []string{ViolationProhibited}},
		{"oh, HECK!", "oh, H***!", false, []string{ViolationProhibited}},
		{"darned", "darned", false, nil}, // whole words only
		{"total scam", "total scam", true, []string{ViolationBlocked}},
		{"telegram.me", "telegram.me", true, []string{ViolationBlocked}},
		{"Telegram Me", "Telegram Me", true, []string{ViolationBlocked}},
		// Devanagari: the matras (vowel signs) must stay part of the word.
		{"तू कमीना है", "तू क**** है", false, []string{ViolationProhibited}},
		{"", "", false, nil},
	}
	for _, tt := range tests {
		got := f.Apply(tt.in)
		if got.Content != tt.want || got.Blocked != tt.blocked || !slices.Equal(got.Violations, tt.violations) {
			t.Errorf("Apply(%q) = %+v, want content %q blocked %v violations %v", tt.in, got, tt.want, tt.blocked, tt.violations)
		}
		if got.Filtered != (len(tt.violations) > 0) {
			t.Errorf("Apply(%q).Filtered = %v", tt.in, got.Filtered)
		}
	}
}

func TestTruncatesToMaxRunes(t *testing.T) {
	got := New(10, nil).Apply("héllo wörld!!")
	if got.Content != "héllo wörl" || !slices.Equal(got.Violations, []string{ViolationTooLong}) {
		t.Fatalf("got %+v", got)
	}
}

func TestLiveSwap(t *testing.T) {
	live := NewLive(New(100, nil))
	if live.Load().Apply("darn").Filtered {
		t.Fatal("empty filter should not filter")
	}
	live.Store(New(100, []Rule{{Word: "darn", Action: Mask}}))
	if got := live.Load().Apply("darn").Content; got != "d***" {
		t.Fatalf("after swap: %q", got)
	}
}

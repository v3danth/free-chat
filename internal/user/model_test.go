package user

import (
	"strings"
	"testing"
)

func TestProfileInputParse(t *testing.T) {
	valid := ProfileInput{Name: "cool_guest1", Gender: "female", Age: 25, About: "hi", Location: "Mumbai"}
	with := func(edit func(*ProfileInput)) ProfileInput {
		in := valid
		edit(&in)
		return in
	}

	tests := []struct {
		name string
		in   ProfileInput
		ok   bool
	}{
		{"valid", valid, true},
		{"hindi name with matras", with(func(in *ProfileInput) { in.Name = "राहुल" }), true},
		{"tamil name", with(func(in *ProfileInput) { in.Name = "கார்த்திக்" }), true},
		{"inner space", with(func(in *ProfileInput) { in.Name = "Ravi  K" }), true},
		{"no tags", with(func(in *ProfileInput) { in.Tags = nil }), true},
		{"free-form tags in any script", with(func(in *ProfileInput) { in.Tags = []string{"night owl", "K-pop", "क्रिकेट"} }), true},
		{"location with punctuation", with(func(in *ProfileInput) { in.Location = "St. John's, NL" }), true},
		{"empty location", with(func(in *ProfileInput) { in.Location = "" }), true},
		{"name too short", with(func(in *ProfileInput) { in.Name = "a" }), false},
		{"name too long", with(func(in *ProfileInput) { in.Name = strings.Repeat("a", 33) }), false},
		{"name symbols", with(func(in *ProfileInput) { in.Name = "bad<name>" }), false},
		{"reserved name", with(func(in *ProfileInput) { in.Name = "Real_Admin" }), false},
		{"under 18", with(func(in *ProfileInput) { in.Age = 17 }), false},
		{"over 99", with(func(in *ProfileInput) { in.Age = 120 }), false},
		{"unknown gender", with(func(in *ProfileInput) { in.Gender = "robot" }), false},
		{"four tags", with(func(in *ProfileInput) { in.Tags = []string{"a", "b", "c", "d"} }), false},
		{"tag too long", with(func(in *ProfileInput) { in.Tags = []string{strings.Repeat("a", 21)} }), false},
		{"tag markup", with(func(in *ProfileInput) { in.Tags = []string{"<b>hi</b>"} }), false},
		{"about too long", with(func(in *ProfileInput) { in.About = strings.Repeat("é", 141) }), false},
		{"location too long", with(func(in *ProfileInput) { in.Location = strings.Repeat("a", 41) }), false},
		{"location markup", with(func(in *ProfileInput) { in.Location = "<script>" }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := tt.in.Parse()
			if (err == nil) != tt.ok {
				t.Fatalf("ok = %v, err = %v", tt.ok, err)
			}
			if tt.ok && p.Tags == nil {
				t.Fatal("tags must never be nil, so they encode as []")
			}
		})
	}
}

func TestScreened(t *testing.T) {
	screen := func(s string) (string, bool, bool) {
		switch {
		case strings.Contains(s, "scam"):
			return s, false, true
		case strings.Contains(s, "darn"):
			return strings.ReplaceAll(s, "darn", "d***"), true, false
		}
		return s, false, false
	}
	p := Profile{Name: "ravi", About: "darn good", Location: "Pune"}

	got, err := p.Screened(screen)
	if err != nil || got.About != "d*** good" {
		t.Fatalf("masked about: %+v, %v", got, err)
	}
	if _, err := (Profile{Name: "darnit"}).Screened(screen); err == nil {
		t.Fatal("a name that trips the filter must be rejected, not masked")
	}
	if _, err := (Profile{Name: "ok", Location: "scam city"}).Screened(screen); err == nil {
		t.Fatal("a blocked word in location must be rejected")
	}
}

func TestRoleLadder(t *testing.T) {
	if !RoleAdmin.Outranks(RoleModerator) || RoleModerator.Outranks(RoleModerator) || RoleUser.Outranks(RoleUser) {
		t.Fatal("outrank must be strict")
	}
	if !RoleModerator.AtLeast(RoleModerator) || RoleUser.AtLeast(RoleModerator) {
		t.Fatal("AtLeast is inclusive")
	}
}

func TestParseTags(t *testing.T) {
	got, err := ParseTags([]string{"  night   owl ", "Night Owl", "", "music"})
	if err != nil || strings.Join(got, "|") != "night owl|music" {
		t.Fatalf("got %q, %v; want spaces collapsed, blanks and case-duplicates dropped", got, err)
	}
	// Duplicates do not count towards the limit.
	if _, err := ParseTags([]string{"a", "A", "b", "c"}); err != nil {
		t.Fatalf("three distinct tags must pass: %v", err)
	}
}

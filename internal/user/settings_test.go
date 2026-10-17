package user

import (
	"strings"
	"testing"
	"time"
)

func TestKnockPolicyAdmits(t *testing.T) {
	for _, tc := range []struct {
		policy KnockPolicy
		sender Kind
		want   bool
	}{
		{KnocksAnyone, KindGuest, true},
		{KnocksMembers, KindGuest, false},
		{KnocksMembers, KindMember, true},
		{KnocksNobody, KindMember, false},
	} {
		if got := tc.policy.Admits(tc.sender); got != tc.want {
			t.Errorf("%s admits %s = %v, want %v", tc.policy, tc.sender, got, tc.want)
		}
	}
}

func TestSettingsParsers(t *testing.T) {
	if _, err := ParseAura("ember"); err != nil {
		t.Error(err)
	}
	if a, err := ParseAura(""); err != nil || a != "" {
		t.Errorf("empty clears the aura: %q %v", a, err)
	}
	if _, err := ParseAura("neon"); err == nil {
		t.Error("an aura outside the set must be refused")
	}
	if _, err := ParseKnockPolicy("friends"); err == nil {
		t.Error("an unknown knock policy must be refused")
	}
}

func TestHiddenGenderShowsOnlyOnceKnown(t *testing.T) {
	u := User{ID: 1, Profile: Profile{Name: "asha", Gender: GenderFemale, Age: 25},
		Settings: Settings{HideGender: true, Aura: "rose"}}

	card := ToCard(u, time.Time{})
	if card.Gender != "" {
		t.Fatalf("stranger sees gender %q", card.Gender)
	}
	if card.Aura != "rose" || !strings.HasPrefix(card.Avatar, "/avatar/v2/") {
		t.Fatalf("card = %+v", card)
	}
	if known := ToKnown(u, time.Time{}); known.Gender != GenderFemale {
		t.Fatalf("an open chat sees gender, got %q", known.Gender)
	}
	u.Settings.HideGender = false
	if ToCard(u, time.Time{}).Gender != GenderFemale {
		t.Fatal("gender shows by default")
	}
}

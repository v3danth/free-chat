package user

import (
	"slices"
	"strings"

	"github.com/v3danth/free-chat/internal/apperr"
)

// KnockPolicy is who may start a private chat with someone. Women on
// stranger-chat apps leave when they cannot control who reaches them, so
// the choice is theirs, not the app's.
type KnockPolicy string

const (
	KnocksAnyone  KnockPolicy = "anyone"
	KnocksMembers KnockPolicy = "members"
	KnocksNobody  KnockPolicy = "nobody"
)

// Admits reports whether someone of kind sender may knock under p.
func (p KnockPolicy) Admits(sender Kind) bool {
	switch p {
	case KnocksNobody:
		return false
	case KnocksMembers:
		return sender == KindMember
	}
	return true
}

// Settings are the choices a person makes about how they are seen and
// reached. They are separate from Profile because they are not text others
// read, so the word filter never needs to see them.
type Settings struct {
	// Aura is a glow around the face, one of Auras, or "" for none. Anyone
	// may pick any aura: it says "I chose this", never "I am a woman".
	Aura       string
	KnocksFrom KnockPolicy
	// HideGender keeps gender off the card for anyone without an open chat.
	HideGender bool
}

// Auras are the glows on offer, warm first, after the palette of the film Her.
var Auras = []string{"ember", "rose", "lantern", "peach", "dusk", "sea"}

var (
	errAura      = apperr.New(apperr.Invalid, "aura must be one of "+strings.Join(Auras, ", ")+", or empty")
	errKnockFrom = apperr.New(apperr.Invalid, "knocks_from must be anyone, members or nobody")
)

// ParseAura accepts one of Auras, or "" to clear it.
func ParseAura(s string) (string, error) {
	if s != "" && !slices.Contains(Auras, s) {
		return "", errAura
	}
	return s, nil
}

func ParseKnockPolicy(s string) (KnockPolicy, error) {
	switch p := KnockPolicy(s); p {
	case KnocksAnyone, KnocksMembers, KnocksNobody:
		return p, nil
	}
	return "", errKnockFrom
}

package main

import (
	"github.com/v3danth/free-chat/internal/botclient"
	"log/slog"
	"maps"
	"slices"
	"sync"
)

// persona is a bot's card plus the settings it picks after signing up.
type persona struct {
	botclient.Guest
	Aura string
}

func card(name, gender string, age int, tags []string, about, location, aura string) persona {
	return persona{Guest: botclient.Guest{Name: name, Gender: gender, Age: age, Tags: tags, About: about, Location: location}, Aura: aura}
}

// personas cover every gender the card offers and every aura, plus a few
// with none, so each kind of card and colour shows up on screen.
var personas = []persona{
	card("Mira", "female", 27, []string{"just talk"}, "Night owl, bad at sleeping", "Lisbon", "rose"),
	card("Theo", "male", 31, []string{"dom", "roleplay"}, "Ask me about trains", "Leeds", "ember"),
	card("Juno", "non-binary", 24, []string{"switch"}, "Here for the banter", "", "dusk"),
	card("Kai", "femboy", 22, []string{"sub", "kink curious"}, "", "Osaka", "peach"),
	card("Sam and Alex", "couple", 34, []string{"just talk"}, "Two of us, one card", "Austin", "sea"),
	card("Rowan", "other", 29, []string{"roleplay"}, "Writer, mostly", "Dublin", ""),
	card("Lena", "female", 38, []string{"switch", "just talk"}, "Cooking something", "Berlin", ""),
	card("Ravi", "male", 26, []string{"kink curious"}, "First time here", "Pune", "lantern"),
	card("Noor", "female", 30, []string{"sub"}, "", "Amman", "ember"),
	card("Ezra", "male", 45, []string{"dom"}, "Slow replies, good ones", "Toronto", "sea"),
	card("Ivy", "female", 21, []string{"just talk"}, "Studying, avoiding it", "Melbourne", ""),
	card("Pax", "non-binary", 33, []string{"roleplay", "switch"}, "", "Oslo", "rose"),
}

var roomLines = []string{
	"evening all", "anyone else awake?", "this room is quiet tonight",
	"what is everyone drinking", "hi hi", "who is up for a chat",
	"rain here, again", "just got in", "what music are you on tonight",
}

var knocks = []string{
	"hey, your card made me smile", "hi, how is your night going?",
	"hello from across the room", "you seem fun, want to talk?",
}

var replies = []string{
	"haha yes", "tell me more", "that is fair", "oh nice", "same here honestly",
	"what made you come here tonight?", "ok you got me", "mm, go on",
}

// Odds, in percent, for each choice a bot makes.
const (
	replyToKnockPct  = 60 // otherwise decline (declinePct) or ignore
	declinePct       = 25
	acceptCloserPct  = 70
	askCloserAfter   = 3 // messages each way before a bot asks
	maxRepliesInChat = 12
	reloadPct        = 3 // per move
	leavePct         = 2 // per move
)

// tally counts what happened across all bots, for the summary at the end.
type tally struct {
	mu     sync.Mutex
	counts map[string]int
}

func newTally() *tally { return &tally{counts: map[string]int{}} }

func (t *tally) add(what string) {
	t.mu.Lock()
	t.counts[what]++
	t.mu.Unlock()
}

func (t *tally) log() {
	t.mu.Lock()
	defer t.mu.Unlock()
	args := make([]any, 0, 2*len(t.counts))
	for _, k := range slices.Sorted(maps.Keys(t.counts)) {
		args = append(args, k, t.counts[k])
	}
	slog.Info("simulation summary", args...)
}

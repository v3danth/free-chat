package main

import (
	"maps"
	"slices"
	"time"
)

// This file is the bots' minds, kept pure: each function takes a bot's view
// of the world, one event and a dice roll, and returns the new view and the
// moves to make. The runner in bot.go does all I/O, timing and randomness,
// so every choice here can be tested without a server.

// chat mirrors the web client's CHAT machine from the bot's side.
type chat struct {
	Phase   string // outgoing, incoming, open
	Said    int    // messages this bot sent in the chat
	Heard   int    // messages it received
	Asked   bool   // it asked to get closer
	Invited bool   // the other side asked
}

type world struct {
	Me     uint64
	Online map[uint64]string // id -> name
	Chats  map[uint64]chat
}

// move is one thing a bot does, after Delay. Frame goes to the server;
// Reload and Leave act on the connection itself.
type move struct {
	Delay  time.Duration
	Frame  map[string]any
	Reload bool
	Leave  bool
	Tally  string
}

// frame is an inbound server frame, decoded loosely: bots only read the
// fields they act on.
type frame map[string]any

func (f frame) str(k string) string { s, _ := f[k].(string); return s }
func (f frame) id(k string) uint64  { n, _ := f[k].(float64); return uint64(n) }
func (f frame) flag(k string) bool  { b, _ := f[k].(bool); return b }
func (f frame) obj(k string) frame  { m, _ := f[k].(map[string]any); return m }

// withChat returns a copy of w where peer's chat is c.
func withChat(w world, peer uint64, c chat) world {
	w.Chats = maps.Clone(w.Chats)
	w.Chats[peer] = c
	return w
}

func withoutChat(w world, peer uint64) world {
	w.Chats = maps.Clone(w.Chats)
	delete(w.Chats, peer)
	return w
}

// fromHello is the world a fresh connection starts from: who is here and
// which doors survived the reconnect.
func fromHello(f frame) world {
	w := world{Me: f.obj("you").id("id"), Online: map[uint64]string{}, Chats: map[uint64]chat{}}
	for _, c := range asList(f["online"]) {
		w.Online[c.id("id")] = c.str("name")
	}
	for _, d := range asList(f["doors"]) {
		phase := "incoming"
		switch {
		case d.flag("open"):
			phase = "open"
		case d.flag("knocked_by_me"):
			phase = "outgoing"
		}
		w.Chats[d.id("with")] = chat{Phase: phase, Asked: d.flag("asked_by_me"), Invited: d.flag("asked_by_them")}
	}
	return w
}

func asList(v any) []frame {
	raw, _ := v.([]any)
	out := make([]frame, 0, len(raw))
	for _, r := range raw {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// react is how a bot answers one server frame. roll is 0-99.
func react(w world, f frame, roll int) (world, []move) {
	switch f.str("type") {
	case "hello":
		return fromHello(f), []move{{Frame: map[string]any{"type": "join", "room_id": 1}}}
	case "presence":
		return onPresence(w, f), nil
	case "dm":
		return onDM(w, f, roll)
	case "door":
		peer := f.obj("with").id("id")
		c := w.Chats[peer]
		c.Phase = "open"
		return withChat(w, peer, c), nil
	case "door_closed":
		return withoutChat(w, f.id("with")), []move{{Tally: "door closed: " + f.str("reason")}}
	case "closer":
		return onCloser(w, f, roll)
	case "error":
		return w, []move{{Tally: "refused: " + f.str("code") + ": " + f.str("error")}}
	}
	return w, nil
}

func onPresence(w world, f frame) world {
	who := f.id("user_id")
	w.Online = maps.Clone(w.Online)
	if f.str("event") == "leave" {
		delete(w.Online, who)
		return withoutChat(w, who)
	}
	w.Online[who] = f.obj("user").str("name")
	return w
}

func onDM(w world, f frame, roll int) (world, []move) {
	mine := f.id("from") == w.Me
	peer := f.id("from")
	if mine {
		peer = f.id("to")
	}
	c := w.Chats[peer]
	switch {
	case f.flag("knock") && mine:
		return withChat(w, peer, chat{Phase: "outgoing", Said: 1}), nil
	case f.flag("knock"):
		return answerKnock(withChat(w, peer, chat{Phase: "incoming", Heard: 1}), peer, roll)
	case mine:
		c.Phase, c.Said = "open", c.Said+1
		return withChat(w, peer, c), nil
	}
	c.Phase, c.Heard = "open", c.Heard+1
	w = withChat(w, peer, c)
	if c.Said >= maxRepliesInChat {
		return w, nil
	}
	moves := []move{say(peer, replies[roll%len(replies)], thinkingTime(roll))}
	if !c.Asked && c.Said >= askCloserAfter && c.Heard >= askCloserAfter {
		moves = append(moves, askCloser(peer, thinkingTime(roll)+2*time.Second))
	}
	return w, moves
}

// answerKnock replies, declines or leaves the knock waiting, so the person
// testing sees all three from the knocking side.
func answerKnock(w world, peer uint64, roll int) (world, []move) {
	switch {
	case roll < replyToKnockPct:
		return w, []move{say(peer, replies[roll%len(replies)], thinkingTime(roll))}
	case roll < replyToKnockPct+declinePct:
		return w, []move{{Delay: thinkingTime(roll), Frame: map[string]any{"type": "decline", "user_id": peer}, Tally: "decline"}}
	}
	return w, []move{{Tally: "knock ignored"}}
}

func onCloser(w world, f frame, roll int) (world, []move) {
	peer := f.id("with")
	c, ok := w.Chats[peer]
	if !ok {
		return w, nil
	}
	c.Asked, c.Invited = f.flag("asked_by_me"), f.flag("asked_by_them")
	w = withChat(w, peer, c)
	if c.Invited && !c.Asked && roll < acceptCloserPct {
		return w, []move{askCloser(peer, thinkingTime(roll))}
	}
	return w, nil
}

// initiative is a bot's own move when nothing is asking it to answer:
// talk to the room, knock on the person testing or another bot, reload
// the page, or leave for a while.
func initiative(w world, target string, roll int) []move {
	switch {
	case roll < reloadPct:
		return []move{{Reload: true, Tally: "reload"}}
	case roll < reloadPct+leavePct:
		return []move{{Leave: true, Tally: "leave and return"}}
	}
	if peer, ok := knockable(w, target); ok && roll < 35 {
		return []move{{Frame: map[string]any{"type": "dm", "to": peer, "content": knocks[roll%len(knocks)]}, Tally: "knock"}}
	}
	line := roomLines[roll%len(roomLines)]
	if roll < 45 && slices.Contains(slices.Collect(maps.Values(w.Online)), target) {
		return []move{{Frame: map[string]any{"type": "chat", "room_id": 1, "content": "@" + target + " " + line}, Tally: "mention"}}
	}
	return []move{{Frame: map[string]any{"type": "chat", "room_id": 1, "content": line}, Tally: "room line"}}
}

// knockable picks someone to knock: the person testing first, then anyone
// this bot has no chat with yet.
func knockable(w world, target string) (uint64, bool) {
	var fallback uint64
	for id, name := range w.Online {
		if _, busy := w.Chats[id]; busy || id == w.Me {
			continue
		}
		if target != "" && name == target {
			return id, true
		}
		if fallback == 0 || id < fallback {
			fallback = id
		}
	}
	return fallback, fallback != 0
}

func say(peer uint64, text string, delay time.Duration) move {
	return move{Delay: delay, Frame: map[string]any{"type": "dm", "to": peer, "content": text}, Tally: "dm"}
}

func askCloser(peer uint64, delay time.Duration) move {
	return move{Delay: delay, Frame: map[string]any{"type": "closer", "user_id": peer}, Tally: "closer"}
}

// thinkingTime is how long a reply takes: people type, bots should too.
func thinkingTime(roll int) time.Duration {
	return time.Duration(1500+roll*40) * time.Millisecond
}

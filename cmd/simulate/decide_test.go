package main

import "testing"

func knockFrom(from, to uint64) frame {
	return frame{"type": "dm", "from": float64(from), "to": float64(to), "content": "hi", "knock": true}
}

func TestAnswerKnockCoversEveryOutcome(t *testing.T) {
	w := world{Me: 1, Chats: map[uint64]chat{}}
	for _, tc := range []struct {
		roll int
		want string
	}{{0, "dm"}, {replyToKnockPct, "decline"}, {99, "knock ignored"}} {
		got, moves := react(w, knockFrom(2, 1), tc.roll)
		if got.Chats[2].Phase != "incoming" || len(moves) != 1 || moves[0].Tally != tc.want {
			t.Errorf("roll %d: phase %q moves %+v, want %s", tc.roll, got.Chats[2].Phase, moves, tc.want)
		}
	}
	if len(w.Chats) != 0 {
		t.Fatal("react must not change the world it was given")
	}
}

func TestBotAsksCloserOnceTheChatHasGoneBackAndForth(t *testing.T) {
	w := world{Me: 1, Chats: map[uint64]chat{2: {Phase: "open", Said: askCloserAfter, Heard: askCloserAfter - 1}}}
	_, moves := react(w, frame{"type": "dm", "from": float64(2), "to": float64(1), "content": "hey"}, 10)
	if len(moves) != 2 || moves[1].Tally != "closer" {
		t.Fatalf("want a reply then a closer request, got %+v", moves)
	}
}

func TestHelloRestoresDoors(t *testing.T) {
	w, moves := react(world{}, frame{
		"type": "hello", "you": map[string]any{"id": float64(1)},
		"online": []any{map[string]any{"id": float64(2), "name": "Asha"}},
		"doors":  []any{map[string]any{"with": float64(2), "knocked_by_me": true}},
	}, 0)
	if w.Me != 1 || w.Online[2] != "Asha" || w.Chats[2].Phase != "outgoing" {
		t.Fatalf("world from hello: %+v", w)
	}
	if len(moves) != 1 || moves[0].Frame["type"] != "join" {
		t.Fatalf("hello must join the room: %+v", moves)
	}
}

func TestKnockablePrefersThePersonTesting(t *testing.T) {
	w := world{Me: 1, Online: map[uint64]string{2: "Bot", 3: "Vedant"}, Chats: map[uint64]chat{}}
	if id, ok := knockable(w, "Vedant"); !ok || id != 3 {
		t.Fatalf("got %d %v, want the target", id, ok)
	}
	w.Chats = map[uint64]chat{3: {Phase: "outgoing"}}
	if id, _ := knockable(w, "Vedant"); id != 2 {
		t.Fatalf("already knocking the target: got %d, want another person", id)
	}
}

func TestBotsMentionThePersonTestingInTheRoom(t *testing.T) {
	// Everyone the bot could knock is busy, so its move is a room line.
	w := world{Me: 1, Online: map[uint64]string{3: "Vedant"}, Chats: map[uint64]chat{3: {Phase: "open"}}}
	moves := initiative(w, "Vedant", 40)
	if len(moves) != 1 || moves[0].Tally != "mention" {
		t.Fatalf("want a mention, got %+v", moves)
	}
	if text, _ := moves[0].Frame["content"].(string); text[:7] != "@Vedant" {
		t.Fatalf("mention text: %q", text)
	}
}

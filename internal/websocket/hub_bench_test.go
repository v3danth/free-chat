package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
)

// The hub is the server's hot path: every room line fans out to everyone
// in the room and every arrival sends the newcomer the whole room. These
// benchmarks run the real hub with socketless clients whose queues are
// drained, so they measure the hub alone.

type discardWriter struct{}

func (discardWriter) Enqueue(message.Message) bool { return true }

type nopPresence struct{}

func (nopPresence) Touch(context.Context, uint64) error { return nil }

func benchHub() *Hub {
	return NewHub(Deps{
		Writer: discardWriter{}, IDs: &id.Generator{}, Media: ownImages{}, Users: nopPresence{},
		Blocks:       &memBlocks{pairs: map[[2]uint64]bool{}},
		Limiter:      ratelimit.New(ratelimit.Config{Rate: 1 << 30, Window: time.Minute}),
		KnockLimiter: ratelimit.New(ratelimit.Config{Rate: 1 << 30, Window: time.Minute}),
		Filter:       filter.NewLive(filter.New(100, []filter.Rule{{Word: "darn", Action: filter.Mask}})),
		HistoryLimit: 20, Letters: &memLetters{}, LetterDelay: time.Hour,
	}, []message.Room{{ID: 1, Slug: "main", Name: "Main Room"}}, nil)
}

// benchClient registers a socketless client in room 1 and drains its queue
// until stop is closed, as a fast reader would.
func benchClient(b *testing.B, h *Hub, uid uint64, stop <-chan struct{}) *Client {
	b.Helper()
	u := user.User{ID: uid, Kind: user.KindGuest, Role: user.RoleUser,
		Profile:  user.Profile{Name: fmt.Sprintf("person %d", uid), Gender: user.GenderOther, Age: 30, Tags: []string{"just talk"}},
		Settings: user.Settings{Aura: "ember", KnocksFrom: user.KnocksAnyone}}
	c := newClient(u, map[uint64]struct{}{}, nil)
	go func() {
		for {
			select {
			case _, ok := <-c.send:
				if !ok {
					return
				}
			case <-stop:
				return
			}
		}
	}()
	h.register(c)
	if err := h.join(c, 1); err != nil {
		b.Fatal(err)
	}
	return c
}

func fillRoom(b *testing.B, h *Hub, n int, stop <-chan struct{}) []*Client {
	b.Helper()
	clients := make([]*Client, 0, n)
	for i := range n {
		clients = append(clients, benchClient(b, h, uint64(i+1), stop))
	}
	return clients
}

func BenchmarkRoomMessage(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("members=%d", n), func(b *testing.B) {
			h, stop := benchHub(), make(chan struct{})
			defer close(stop)
			sender := fillRoom(b, h, n, stop)[0]
			cmd := roomSendCmd{room: 1, content: "anyone else awake tonight?"}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := h.sendRoom(sender, cmd); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkArrival is one person joining a room of n: their hello carries
// everyone's card and everyone hears about them.
func BenchmarkArrival(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("online=%d", n), func(b *testing.B) {
			h, stop := benchHub(), make(chan struct{})
			defer close(stop)
			fillRoom(b, h, n, stop)
			newcomer := benchClient(b, h, uint64(n+1), stop)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				h.register(newcomer)
			}
		})
	}
}

// BenchmarkProfileEdit is one card update reaching everyone online.
func BenchmarkProfileEdit(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprintf("online=%d", n), func(b *testing.B) {
			h, stop := benchHub(), make(chan struct{})
			defer close(stop)
			editor := fillRoom(b, h, n, stop)[0]
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				h.publishCard(editor)
			}
		})
	}
}

func TestHelloFrameIsValidJSON(t *testing.T) {
	for _, cards := range [][]json.RawMessage{nil, {json.RawMessage(`{"id":1}`)}, {json.RawMessage(`{"id":1}`), json.RawMessage(`{"id":2}`)}} {
		frame := helloFrame(helloEvent{Type: "hello", Doors: []doorView{}}, cards)
		var got struct {
			Type   string           `json:"type"`
			Online []map[string]any `json:"online"`
		}
		if err := json.Unmarshal(frame, &got); err != nil {
			t.Fatalf("%s: %v", frame, err)
		}
		if got.Type != "hello" || len(got.Online) != len(cards) {
			t.Fatalf("%s: got %+v", frame, got)
		}
	}
}

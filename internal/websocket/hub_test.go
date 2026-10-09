package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/geo"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/user/usertest"
)

// --- in-memory dependencies ---

type memWriter struct {
	mu   sync.Mutex
	msgs []message.Message
}

func (w *memWriter) Enqueue(m message.Message) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgs = append(w.msgs, m)
	return true
}

type ownImages struct{}

// Every image id is owned by the user with the same id.
func (ownImages) Attach(_ context.Context, mediaID, owner uint64) (media.Attachment, error) {
	if mediaID != owner {
		return media.Attachment{}, media.ErrNotOwner
	}
	return media.AttachmentOf(mediaID, "key"), nil
}

type memBlocks struct {
	mu    sync.Mutex
	pairs map[[2]uint64]bool
}

func (b *memBlocks) Between(_ context.Context, u uint64) (map[uint64]struct{}, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	set := map[uint64]struct{}{}
	for p := range b.pairs {
		if p[0] == u {
			set[p[1]] = struct{}{}
		}
		if p[1] == u {
			set[p[0]] = struct{}{}
		}
	}
	return set, nil
}

func (b *memBlocks) Add(_ context.Context, x, y uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pairs[[2]uint64{x, y}] = true
	return nil
}

func (b *memBlocks) Remove(_ context.Context, x, y uint64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.pairs, [2]uint64{x, y})
	return nil
}

type noBans struct{}

func (noBans) IsIPBanned(context.Context, []byte) (bool, error) { return false, nil }

// --- harness ---

type harness struct {
	t      *testing.T
	hub    *Hub
	auth   *auth.Service
	users  *usertest.Memory
	writer *memWriter
	wsURL  string
}

func newHarness(t *testing.T) *harness {
	users := usertest.NewMemory()
	words := filter.NewLive(filter.New(100, []filter.Rule{{Word: "darn", Action: filter.Mask}, {Word: "scam", Action: filter.Block}}))
	authSvc := auth.NewService(users, auth.NewTokens("secret-secret-secret-secret-secret"), noBans{}, geo.None{}, words, "k")
	writer := &memWriter{}
	seed := message.NewRoom(1, 1, 999, "earlier", "before you came", 0)
	hub := NewHub(Deps{
		Writer: writer, IDs: &id.Generator{}, Media: ownImages{}, Users: users,
		Blocks:  &memBlocks{pairs: map[[2]uint64]bool{}},
		Limiter: ratelimit.New(ratelimit.Config{Rate: 50, Window: time.Minute}),
		Filter:  words, HistoryLimit: 20,
	}, []message.Room{{ID: 1, Slug: "main", Name: "Main Room"}},
		map[uint64][]message.Entry{1: {{Message: seed}}})

	mux := http.NewServeMux()
	Routes(mux, hub, authSvc, users, httpx.NewIPResolver(false))
	srv := httptest.NewServer(mux)
	t.Cleanup(func() { hub.Close(); srv.Close() })
	return &harness{t: t, hub: hub, auth: authSvc, users: users, writer: writer,
		wsURL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token="}
}

type peer struct {
	t    *testing.T
	id   uint64
	conn *gws.Conn
}

func (h *harness) join(name string) *peer {
	h.t.Helper()
	s, err := h.auth.CreateGuest(context.Background(), auth.GuestSignup{
		Profile: user.Profile{Name: name, Gender: user.GenderOther, Age: 22, Intent: user.IntentTalk},
	})
	if err != nil {
		h.t.Fatal(err)
	}
	conn, _, err := gws.DefaultDialer.Dial(h.wsURL+s.Token, nil)
	if err != nil {
		h.t.Fatal(err)
	}
	p := &peer{t: h.t, id: s.User.ID, conn: conn}
	p.expect("hello")
	return p
}

func (p *peer) send(frame map[string]any) {
	p.t.Helper()
	if err := p.conn.WriteJSON(frame); err != nil {
		p.t.Fatal(err)
	}
}

// expect skips frames until one of type typ arrives.
func (p *peer) expect(typ string) map[string]any {
	p.t.Helper()
	p.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var f map[string]any
		if err := p.conn.ReadJSON(&f); err != nil {
			p.t.Fatalf("waiting for %q: %v", typ, err)
		}
		if f["type"] == typ {
			return f
		}
	}
}

func (p *peer) expectError(code string) {
	p.t.Helper()
	if e := p.expect("error"); e["code"] != code {
		p.t.Fatalf("want error %s, got %v", code, e)
	}
}

// --- tests ---

func TestRoomPresenceAndHistory(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")
	if ev := asha.expect("presence"); ev["event"] != "join" || ev["user"].(map[string]any)["name"] != "ravi" {
		t.Fatalf("presence = %v", ev)
	}

	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "hi"})
	ravi.expectError("FORBIDDEN") // must join first

	for _, p := range []*peer{asha, ravi} {
		p.send(map[string]any{"type": "join", "room_id": 1})
		hist := p.expect("history")["messages"].([]any)
		if len(hist) != 1 || hist[0].(map[string]any)["content"] != "before you came" {
			t.Fatalf("history served from memory: %v", hist)
		}
	}

	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "darn it"})
	for _, p := range []*peer{asha, ravi} {
		if got := p.expect("chat"); got["content"] != "d*** it" || got["name"] != "ravi" || got["filtered"] != true {
			t.Fatalf("broadcast = %v", got)
		}
	}
	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "total scam"})
	ravi.expectError("INVALID")

	ravi.conn.Close()
	if ev := asha.expect("presence"); ev["event"] != "leave" {
		t.Fatalf("leave = %v", ev)
	}
}

func TestKnockOpensDoorAndReveals(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")

	// A knock cannot carry a photo.
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "media_id": ravi.id})
	ravi.expectError("FORBIDDEN")

	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "hello there"})
	if got := asha.expect("dm"); got["knock"] != true || got["content"] != "hello there" {
		t.Fatalf("knock = %v", got)
	}
	ravi.expect("dm") // echo

	// No second message until she replies.
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "hello??"})
	ravi.expectError("FORBIDDEN")

	asha.send(map[string]any{"type": "dm", "to": ravi.id, "content": "hi!"})
	if d := ravi.expect("door"); d["with"].(map[string]any)["name"] != "asha" {
		t.Fatalf("door for ravi = %v", d)
	}
	asha.expect("door")

	// Door open: photos allowed, both ways.
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "media_id": ravi.id})
	if got := asha.expect("dm"); got["image"] == nil || got["knock"] == true {
		t.Fatalf("photo after door = %v", got)
	}

	h.writer.mu.Lock()
	defer h.writer.mu.Unlock()
	if len(h.writer.msgs) != 3 {
		t.Fatalf("every delivered DM is logged, got %d", len(h.writer.msgs))
	}
}

func TestBlockIsMutualAndSilent(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")

	asha.send(map[string]any{"type": "block", "user_id": ravi.id})
	if ev := ravi.expect("presence"); ev["event"] != "leave" || uint64(ev["user_id"].(float64)) != asha.id {
		t.Fatalf("blocked user should see asha leave: %v", ev)
	}
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "hey"})
	ravi.expectError("NOT_FOUND") // indistinguishable from offline
}

func TestModerationHooks(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")
	for _, p := range []*peer{asha, ravi} {
		p.send(map[string]any{"type": "join", "room_id": 1})
		p.expect("history")
	}

	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "spam spam"})
	msgID := uint64(asha.expect("chat")["id"].(float64))
	h.hub.RemoveMessage(message.Message{ID: msgID, RoomID: 1})
	if ev := asha.expect("removed"); uint64(ev["message_id"].(float64)) != msgID {
		t.Fatalf("removed = %v", ev)
	}

	h.hub.Mute(ravi.id, time.Now().Add(time.Minute))
	ravi.expect("notice")
	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "hello"})
	ravi.expectError("FORBIDDEN")

	h.hub.Kick(ravi.id, "kicked", "bye", time.Time{})
	if n := ravi.expect("notice"); n["code"] != "kicked" {
		t.Fatalf("notice = %v", n)
	}
	if ev := asha.expect("presence"); ev["event"] != "leave" {
		t.Fatalf("kicked user leaves: %v", ev)
	}
}

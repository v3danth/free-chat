package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/v3danth/free-chat/internal/letters"
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

// memLetters keeps letters in memory and refuses a second pending one per
// pair, like the MySQL store.
type memLetters struct {
	mu   sync.Mutex
	sent []letters.Letter
}

func (m *memLetters) Send(_ context.Context, l letters.Letter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, old := range m.sent {
		if old.SenderID == l.SenderID && old.RecipientID == l.RecipientID {
			return letters.ErrPending
		}
	}
	m.sent = append(m.sent, l)
	return nil
}

type noBans struct{}

func (noBans) IsIPBanned(context.Context, []byte) (bool, error) { return false, nil }

// --- harness ---

type harness struct {
	t       *testing.T
	hub     *Hub
	auth    *auth.Service
	users   *usertest.Memory
	writer  *memWriter
	letters *memLetters
	wsURL   string
}

func newHarness(t *testing.T) *harness {
	users := usertest.NewMemory()
	words := filter.NewLive(filter.New(100, []filter.Rule{{Word: "darn", Action: filter.Mask}, {Word: "scam", Action: filter.Block}}))
	writer := &memWriter{}
	box := &memLetters{}
	seed := message.NewRoom(1, 1, 999, "earlier", "before you came", 0)
	hub := NewHub(Deps{
		Writer: writer, IDs: &id.Generator{}, Media: ownImages{}, Users: users,
		Blocks:       &memBlocks{pairs: map[[2]uint64]bool{}},
		Limiter:      ratelimit.New(ratelimit.Config{Rate: 50, Window: time.Minute}),
		KnockLimiter: ratelimit.New(ratelimit.Config{Rate: 3, Window: time.Minute}),
		Filter:       words, HistoryLimit: 20,
		Letters: box, LetterDelay: time.Hour,
	}, []message.Room{{ID: 1, Slug: "main", Name: "Main Room"}},
		map[uint64][]message.Entry{1: {{Message: seed}}})
	authSvc := auth.NewService(users, auth.NewTokens("secret-secret-secret-secret-secret"), noBans{}, hub, geo.None{}, words, "k")

	mux := http.NewServeMux()
	Routes(mux, hub, authSvc, users, httpx.NewIPResolver(false))
	srv := httptest.NewServer(mux)
	t.Cleanup(func() { hub.Close(); srv.Close() })
	return &harness{t: t, hub: hub, auth: authSvc, users: users, writer: writer, letters: box,
		wsURL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token="}
}

type peer struct {
	t       *testing.T
	id      uint64
	conn    *gws.Conn
	token   string
	url     string
	pending [][]byte // frames from a batched message not read yet
}

func (h *harness) join(name string) *peer {
	h.t.Helper()
	s, err := h.auth.CreateGuest(context.Background(), auth.GuestSignup{
		Profile: user.Profile{Name: name, Gender: user.GenderOther, Age: 22},
	})
	if err != nil {
		h.t.Fatal(err)
	}
	return h.connect(s.User.ID, s.Token)
}

func (h *harness) connect(id uint64, token string) *peer {
	h.t.Helper()
	conn, _, err := gws.DefaultDialer.Dial(h.wsURL+token, nil)
	if err != nil {
		h.t.Fatal(err)
	}
	p := &peer{t: h.t, id: id, conn: conn, token: token, url: h.wsURL}
	p.expect("hello")
	return p
}

// joinMember registers a member, optionally with settings, and connects.
func (h *harness) joinMember(name string, settings user.Settings) *peer {
	h.t.Helper()
	ctx := context.Background()
	email := name + "@example.com"
	u, err := h.auth.Register(ctx, auth.Registration{
		Profile: user.Profile{Name: name, Gender: user.GenderFemale, Age: 25}, Email: email, Password: "password1",
	})
	if err != nil {
		h.t.Fatal(err)
	}
	if settings.KnocksFrom == "" {
		settings.KnocksFrom = user.KnocksAnyone
	}
	if err := h.users.UpdateSettings(ctx, u.ID, settings); err != nil {
		h.t.Fatal(err)
	}
	s, err := h.auth.Login(ctx, auth.Credentials{Email: email, Password: "password1"})
	if err != nil {
		h.t.Fatal(err)
	}
	return h.connect(s.User.ID, s.Token)
}

// reconnect opens a second socket with the same session, as a browser does
// after a dropped connection, and returns its hello.
func (p *peer) reconnect() map[string]any {
	p.t.Helper()
	conn, _, err := gws.DefaultDialer.Dial(p.url+p.token, nil)
	if err != nil {
		p.t.Fatal(err)
	}
	p.conn, p.pending = conn, nil
	return p.expect("hello")
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
	if err := p.conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		p.t.Fatal(err)
	}
	for {
		if len(p.pending) == 0 {
			_, msg, err := p.conn.ReadMessage()
			if err != nil {
				p.t.Fatalf("waiting for %q: %v", typ, err)
			}
			p.pending = bytes.Split(msg, []byte{'\n'})
		}
		var f map[string]any
		if err := json.Unmarshal(p.pending[0], &f); err != nil {
			p.t.Fatalf("frame %q: %v", p.pending[0], err)
		}
		p.pending = p.pending[1:]
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

func TestGuestNameIsHeldOnlyWhileOnline(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	if !h.hub.IsNameOnline("ASHA") {
		t.Fatal("a connected guest's name must read as online, ignoring case")
	}
	if _, err := h.auth.CreateGuest(context.Background(), auth.GuestSignup{
		Profile: user.Profile{Name: "asha", Gender: user.GenderOther, Age: 22},
	}); !errors.Is(err, auth.ErrNameInUse) {
		t.Fatalf("joining as someone online: want ErrNameInUse, got %v", err)
	}

	if err := asha.conn.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for h.hub.IsNameOnline("asha") {
		if time.Now().After(deadline) {
			t.Fatal("name still held after the guest left")
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.join("asha")
	if _, err := h.auth.Resume(context.Background(), asha.token, auth.Origin{}); !errors.Is(err, auth.ErrNameClaimed) {
		t.Fatalf("old owner after the name moved on: want ErrNameClaimed, got %v", err)
	}
}

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

	// A masked word asks the sender first; confirming sends it, masked.
	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "darn it"})
	ravi.expectError("RECONSIDER")
	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "darn it", "confirm": true})
	for _, p := range []*peer{asha, ravi} {
		if got := p.expect("chat"); got["content"] != "d*** it" || got["name"] != "ravi" || got["filtered"] != true || got["gender"] != nil {
			t.Fatalf("broadcast = %v", got)
		}
	}
	ravi.send(map[string]any{"type": "chat", "room_id": 1, "content": "total scam"})
	ravi.expectError("INVALID")

	if err := ravi.conn.Close(); err != nil {
		t.Fatal(err)
	}
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

	// Door open: words flow, photos wait for the reveal ladder.
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "media_id": ravi.id})
	ravi.expectError("FORBIDDEN")
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "glad you answered"})
	asha.expect("dm")

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

func TestHelloRestoresDoorsAfterReconnect(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "hello there"})
	asha.expect("dm")
	ravi.expect("dm")

	doors := ravi.reconnect()["doors"].([]any)
	if len(doors) != 1 {
		t.Fatalf("doors after reconnect = %v", doors)
	}
	d := doors[0].(map[string]any)
	if uint64(d["with"].(float64)) != asha.id || d["open"] != false || d["knocked_by_me"] != true {
		t.Fatalf("pending knock not reported: %v", d)
	}
}

// openDoor has from knock and to answer, and drains the frames it causes.
func openDoor(from, to *peer) {
	from.t.Helper()
	from.send(map[string]any{"type": "dm", "to": to.id, "content": "hi"})
	to.expect("dm")
	from.expect("dm")
	to.send(map[string]any{"type": "dm", "to": from.id, "content": "hey"})
	from.expect("door")
	to.expect("door")
}

func TestKnockPolicyIsTheRecipientsChoice(t *testing.T) {
	h := newHarness(t)
	quiet := h.joinMember("quiet", user.Settings{KnocksFrom: user.KnocksNobody})
	picky := h.joinMember("picky", user.Settings{KnocksFrom: user.KnocksMembers})
	guest := h.join("guest")
	member := h.joinMember("member", user.Settings{})

	guest.send(map[string]any{"type": "dm", "to": quiet.id, "content": "hi"})
	guest.expectError("FORBIDDEN")
	guest.send(map[string]any{"type": "dm", "to": picky.id, "content": "hi"})
	if e := guest.expect("error"); e["error"] != errMembersOnly.Msg {
		t.Fatalf("guest knocking a members-only card: %v", e)
	}
	member.send(map[string]any{"type": "dm", "to": picky.id, "content": "hi"})
	if got := picky.expect("dm"); got["knock"] != true {
		t.Fatalf("a member may knock: %v", got)
	}
}

func TestKnocksAreRateLimitedPerSender(t *testing.T) {
	h := newHarness(t)
	sender := h.join("sender")
	others := []*peer{h.join("a1"), h.join("a2"), h.join("a3"), h.join("a4")}
	for _, p := range others[:3] {
		sender.send(map[string]any{"type": "dm", "to": p.id, "content": "hi"})
		p.expect("dm")
	}
	sender.send(map[string]any{"type": "dm", "to": others[3].id, "content": "hi"})
	sender.expectError("RATE_LIMITED")
}

func TestRevealLadderSharesPhotosOnlyWhenBothAsk(t *testing.T) {
	h := newHarness(t)
	ravi := h.join("ravi")
	asha := h.joinMember("asha", user.Settings{HideGender: true})

	if p := ravi.expect("presence"); p["user"].(map[string]any)["gender"] != nil {
		t.Fatalf("a hidden gender reached a stranger: %v", p)
	}
	ravi.send(map[string]any{"type": "closer", "user_id": asha.id})
	ravi.expectError("FORBIDDEN") // no open chat yet

	openDoor(ravi, asha)
	if h.hub.PhotosShared(asha.id, ravi.id) {
		t.Fatal("an open door alone must not share photos")
	}

	ravi.send(map[string]any{"type": "closer", "user_id": asha.id})
	if got := asha.expect("closer"); got["asked_by_them"] != true || got["close"] == true || got["peer"] != nil {
		t.Fatalf("one side asking = %v", got)
	}
	ravi.expect("closer")
	asha.send(map[string]any{"type": "closer", "user_id": ravi.id})
	got := ravi.expect("closer")
	peerCard := got["peer"].(map[string]any)
	if got["close"] != true || peerCard["gender"] != "female" {
		t.Fatalf("both asking = %v", got)
	}
	if !h.hub.PhotosShared(asha.id, ravi.id) {
		t.Fatal("photos are shared once both asked")
	}
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "media_id": ravi.id})
	if dm := asha.expect("dm"); dm["image"] == nil {
		t.Fatalf("photo once close = %v", dm)
	}
}

func TestTomorrowIsPassedOnInAnOpenChat(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")
	openDoor(ravi, asha)
	ravi.send(map[string]any{"type": "tomorrow", "user_id": asha.id})
	if got := asha.expect("tomorrow"); uint64(got["from"].(float64)) != ravi.id {
		t.Fatalf("tomorrow = %v", got)
	}
}

func TestLettersNeedMembersAndArriveLater(t *testing.T) {
	h := newHarness(t)
	asha := h.joinMember("asha", user.Settings{})
	ravi := h.joinMember("ravi", user.Settings{})
	guest := h.join("guest")

	ravi.send(map[string]any{"type": "letter", "to": asha.id, "content": "dear asha"})
	ravi.expectError("FORBIDDEN") // no open chat
	openDoor(guest, asha)
	guest.send(map[string]any{"type": "letter", "to": asha.id, "content": "dear asha"})
	guest.expectError("FORBIDDEN") // guests cannot write letters

	openDoor(ravi, asha)
	ravi.send(map[string]any{"type": "letter", "to": asha.id, "content": "dear asha"})
	if sent := ravi.expect("letter_sent"); int64(sent["deliver_at"].(float64)) < time.Now().Add(50*time.Minute).Unix() {
		t.Fatalf("letter should take LetterDelay: %v", sent)
	}
	ravi.send(map[string]any{"type": "letter", "to": asha.id, "content": "another"})
	ravi.expectError("CONFLICT")

	h.letters.mu.Lock()
	l := h.letters.sent[0]
	h.letters.mu.Unlock()
	if !h.hub.DeliverLetter(l) {
		t.Fatal("an online recipient takes the letter")
	}
	if got := asha.expect("letter"); got["content"] != "dear asha" || got["name"] != "ravi" {
		t.Fatalf("letter = %v", got)
	}
}

func TestReportersAreThanked(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	h.hub.ThankReporters([]uint64{asha.id, 12345})
	if n := asha.expect("notice"); n["code"] != "report_actioned" {
		t.Fatalf("notice = %v", n)
	}
}

func TestKnockSurvivesReloadUntilDeclined(t *testing.T) {
	h := newHarness(t)
	asha, ravi := h.join("asha"), h.join("ravi")
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "hello there"})
	ravi.expect("dm")
	asha.expect("dm")

	d := asha.reconnect()["doors"].([]any)[0].(map[string]any)
	knock, _ := d["knock"].(map[string]any)
	if knock == nil || knock["content"] != "hello there" || uint64(knock["from"].(float64)) != ravi.id {
		t.Fatalf("hello must replay the waiting knock: %v", d)
	}

	ravi.send(map[string]any{"type": "decline", "user_id": asha.id})
	ravi.expectError("NOT_FOUND") // only the person knocked may decline

	asha.send(map[string]any{"type": "decline", "user_id": ravi.id})
	if ev := asha.expect("door_closed"); ev["by_me"] != true || ev["reason"] != "declined" {
		t.Fatalf("decliner: %v", ev)
	}
	if ev := ravi.expect("door_closed"); ev["by_me"] != nil || uint64(ev["with"].(float64)) != asha.id {
		t.Fatalf("knocker: %v", ev)
	}
	if doors := asha.reconnect()["doors"].([]any); len(doors) != 0 {
		t.Fatalf("a declined door must be gone: %v", doors)
	}
	ravi.send(map[string]any{"type": "dm", "to": asha.id, "content": "please?"})
	if e := ravi.expect("error"); e["error"] != errDeclined.Msg {
		t.Fatalf("a declined knocker must not knock again: %v", e)
	}
	// She may still knock on him.
	asha.send(map[string]any{"type": "dm", "to": ravi.id, "content": "actually, hi"})
	if got := ravi.expect("dm"); got["knock"] != true {
		t.Fatalf("decliner knocking: %v", got)
	}
}

func TestOpenDoorCannotBeDeclined(t *testing.T) {
	h := newHarness(t)
	asha, ravi := h.join("asha"), h.join("ravi")
	openDoor(ravi, asha)
	asha.send(map[string]any{"type": "decline", "user_id": ravi.id})
	asha.expectError("NOT_FOUND")
}

// patchMe sends a card edit as p and returns the Self it answers with.
func patchMe(t *testing.T, h *harness, p *peer, body string) map[string]any {
	t.Helper()
	base := "http" + strings.TrimPrefix(strings.TrimSuffix(h.wsURL, "/ws?token="), "ws")
	req, err := http.NewRequest(http.MethodPatch, base+"/me", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var self map[string]any
	if err := json.NewDecoder(res.Body).Decode(&self); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /me %s: %d %v", body, res.StatusCode, self)
	}
	return self
}

func TestEditingSettingsKeepsTheCardAndTheOtherWayRound(t *testing.T) {
	h := newHarness(t)
	asha := h.join("asha")
	ravi := h.join("ravi")
	asha.expect("presence") // ravi arriving

	if self := patchMe(t, h, ravi, `{"aura":"sea"}`); self["aura"] != "sea" {
		t.Fatalf("aura not applied: %v", self)
	}
	if ev := asha.expect("presence"); ev["user"].(map[string]any)["aura"] != "sea" {
		t.Fatalf("others must see the new aura: %v", ev)
	}
	self := patchMe(t, h, ravi, `{"tags":["just talk"],"about":"hi"}`)
	if self["aura"] != "sea" || self["about"] != "hi" {
		t.Fatalf("a card edit must keep settings: %v", self)
	}
	u, err := h.users.GetByID(context.Background(), ravi.id)
	if err != nil || u.Settings.Aura != "sea" || u.Profile.About != "hi" {
		t.Fatalf("stored: %+v, %v", u, err)
	}
}

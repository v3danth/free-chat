package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/user/usertest"
)

// --- fakes ---

type memStore struct {
	words   []Word
	reports []Report
	actions []Action
	ipBans  map[string]time.Time
}

func (m *memStore) ListWords(context.Context) ([]Word, error) { return m.words, nil }
func (m *memStore) AddWord(_ context.Context, w string, a filter.Action, _ uint64) error {
	for _, x := range m.words {
		if x.Word == w {
			return ErrWordExists
		}
	}
	m.words = append(m.words, Word{ID: uint64(len(m.words) + 1), Word: w, Action: a})
	return nil
}
func (m *memStore) RemoveWord(_ context.Context, id uint64) (string, error) {
	for i, w := range m.words {
		if w.ID == id {
			m.words = append(m.words[:i], m.words[i+1:]...)
			return w.Word, nil
		}
	}
	return "", ErrWordNotFound
}
func (m *memStore) BanIP(_ context.Context, h []byte, until time.Time, _ string, _ uint64) error {
	m.ipBans[string(h)] = until
	return nil
}
func (m *memStore) CreateReport(_ context.Context, r Report) error {
	for _, x := range m.reports {
		if *x.ReporterID == *r.ReporterID && x.TargetType == r.TargetType && x.TargetID == r.TargetID {
			return ErrAlreadyReported
		}
	}
	r.Status = StatusOpen
	m.reports = append(m.reports, r)
	return nil
}
func (m *memStore) CountOpen(_ context.Context, t TargetType, id uint64) (int, error) {
	n := 0
	for _, r := range m.reports {
		if r.TargetType == t && r.TargetID == id && r.Status == StatusOpen {
			n++
		}
	}
	return n, nil
}
func (m *memStore) ListReports(context.Context, Status, int) ([]Report, error) { return m.reports, nil }
func (m *memStore) SetReportStatus(context.Context, uint64, Status, uint64) error {
	return nil
}
func (m *memStore) ResolveTarget(_ context.Context, t TargetType, id uint64, s Status, _ uint64) error {
	for i := range m.reports {
		if m.reports[i].TargetType == t && m.reports[i].TargetID == id {
			m.reports[i].Status = s
		}
	}
	return nil
}
func (m *memStore) Log(_ context.Context, a Action) error {
	m.actions = append(m.actions, a)
	return nil
}
func (m *memStore) ListActions(context.Context, int) ([]Action, error) { return m.actions, nil }

type memMessages struct {
	byID   map[uint64]message.Message
	hidden map[uint64]bool
}

func (m *memMessages) Get(_ context.Context, id uint64) (message.Message, error) {
	if msg, ok := m.byID[id]; ok {
		return msg, nil
	}
	return message.Message{}, message.ErrNotFound
}
func (m *memMessages) Before(context.Context, message.Message, int) ([]message.Message, error) {
	return nil, nil
}
func (m *memMessages) BySender(context.Context, uint64, int) ([]message.Message, error) {
	return nil, nil
}
func (m *memMessages) Hide(_ context.Context, id uint64) error {
	m.hidden[id] = true
	return nil
}

type noFlush struct{}

func (noFlush) Flush(context.Context) error { return nil }

type noImages struct{}

func (noImages) Get(context.Context, uint64) (media.Media, error) {
	return media.Media{}, media.ErrNotFound
}
func (noImages) Remove(context.Context, uint64, uint64, bool) (media.Media, error) {
	return media.Media{}, media.ErrNotFound
}

type liveLog struct {
	kicked  []uint64
	removed []uint64
}

func (l *liveLog) Kick(id uint64, _, _ string, _ time.Time) { l.kicked = append(l.kicked, id) }
func (l *liveLog) Mute(uint64, time.Time)                   {}
func (l *liveLog) RemoveMessage(m message.Message)          { l.removed = append(l.removed, m.ID) }
func (l *liveLog) RemoveMedia(uint64)                       {}
func (l *liveLog) UpdateUser(user.User)                     {}

// --- harness ---

type fixture struct {
	svc   *Service
	store *memStore
	msgs  *memMessages
	live  *liveLog
	users *usertest.Memory
	words *filter.Live
}

func newFixture(t *testing.T) *fixture {
	users := usertest.NewMemory()
	f := &fixture{
		store: &memStore{ipBans: map[string]time.Time{}},
		msgs:  &memMessages{byID: map[uint64]message.Message{}, hidden: map[uint64]bool{}},
		live:  &liveLog{},
		users: users,
		words: filter.NewLive(filter.New(100, nil)),
	}
	f.svc = NewService(Deps{Store: f.store, Users: users, Messages: f.msgs, Writer: noFlush{}, Images: noImages{},
		Live: f.live, Filter: f.words, MaxLength: 100, AutoHide: 2})
	return f
}

func (f *fixture) person(t *testing.T, name string, kind user.Kind, role user.Role) auth.Identity {
	t.Helper()
	u, err := f.users.Create(context.Background(), user.User{Kind: kind, Role: role, IPHash: []byte(name),
		Profile: user.Profile{Name: name, Gender: user.GenderOther, Age: 30}})
	if err != nil {
		t.Fatal(err)
	}
	return auth.IdentityOf(u)
}

// --- tests ---

func TestRoleLadder(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	admin := f.person(t, "boss", user.KindMember, user.RoleAdmin)
	mod := f.person(t, "mod1", user.KindMember, user.RoleModerator)
	mod2 := f.person(t, "mod2", user.KindMember, user.RoleModerator)
	guest := f.person(t, "guest", user.KindGuest, user.RoleUser)

	if err := f.svc.Kick(ctx, mod, guest.UserID); err != nil {
		t.Fatalf("mod kicks user: %v", err)
	}
	if err := f.svc.Kick(ctx, mod, mod2.UserID); !errors.Is(err, ErrOutranked) {
		t.Fatalf("mod vs mod: %v", err)
	}
	if err := f.svc.Kick(ctx, mod, admin.UserID); !errors.Is(err, ErrOutranked) {
		t.Fatalf("mod vs admin: %v", err)
	}
	if err := f.svc.Kick(ctx, admin, admin.UserID); !errors.Is(err, ErrSelf) {
		t.Fatalf("self: %v", err)
	}
	if err := f.svc.SetRole(ctx, admin, guest.UserID, user.RoleModerator); !errors.Is(err, ErrMembersOnly) {
		t.Fatalf("guests cannot be staff: %v", err)
	}
}

func TestReportVisibilityAndAutoHide(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	a := f.person(t, "a", user.KindGuest, user.RoleUser)
	b := f.person(t, "b", user.KindGuest, user.RoleUser)
	c := f.person(t, "c", user.KindGuest, user.RoleUser)
	f.msgs.byID[10] = message.NewDirect(10, b.UserID, a.UserID, "a", "nasty dm", 0)
	f.msgs.byID[11] = message.NewRoom(11, 1, a.UserID, "a", "spam", 0)

	in := func(id uint64) ReportInput {
		return ReportInput{TargetType: TargetMessage, TargetID: id, Reason: "harassment"}
	}
	if err := f.svc.Report(ctx, c, in(10)); !errors.Is(err, ErrNotVisible) {
		t.Fatalf("outsider reporting a private message: %v", err)
	}
	if err := f.svc.Report(ctx, a, in(11)); !errors.Is(err, ErrReportSelf) {
		t.Fatalf("self report: %v", err)
	}
	if err := f.svc.Report(ctx, b, in(10)); err != nil {
		t.Fatalf("recipient reports: %v", err)
	}
	if err := f.svc.Report(ctx, b, in(10)); !errors.Is(err, ErrAlreadyReported) {
		t.Fatalf("duplicate report: %v", err)
	}

	// Two reports on the room message reach the auto-hide threshold.
	f.svc.Report(ctx, b, in(11))
	if f.msgs.hidden[11] {
		t.Fatal("hidden too early")
	}
	f.svc.Report(ctx, c, in(11))
	if !f.msgs.hidden[11] || len(f.live.removed) != 1 {
		t.Fatal("message should be auto-hidden and pulled from live screens")
	}
	if got := f.store.reports[0].Evidence; len(got) == 0 {
		t.Fatal("evidence snapshot missing")
	}
}

func TestBanCapsIPBanAndKicks(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	mod := f.person(t, "mod1", user.KindMember, user.RoleModerator)
	troll := f.person(t, "troll", user.KindGuest, user.RoleUser)

	if err := f.svc.Ban(ctx, mod, troll.UserID, 0, true, "spam"); err != nil {
		t.Fatal(err)
	}
	until := f.store.ipBans["troll"]
	if d := time.Until(until); d <= 0 || d > maxIPBan {
		t.Fatalf("permanent account ban must still cap the IP ban, got %v", d)
	}
	u, _ := f.users.GetByID(ctx, troll.UserID)
	if !u.BannedAt(time.Now().AddDate(50, 0, 0)) {
		t.Fatal("d=0 is a permanent ban")
	}
	if len(f.live.kicked) != 1 {
		t.Fatal("a banned user is disconnected immediately")
	}
}

func TestWordsReloadLive(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	mod := f.person(t, "mod1", user.KindMember, user.RoleModerator)

	if err := f.svc.AddWord(ctx, mod, "Telegram.me", filter.Mask); !errors.Is(err, ErrInvalidWord) {
		t.Fatalf("phrases can only block: %v", err)
	}
	if err := f.svc.AddWord(ctx, mod, "Telegram.me", filter.Block); err != nil {
		t.Fatal(err)
	}
	if !f.words.Load().Apply("join telegram.me/x").Blocked {
		t.Fatal("new rule must apply to the live filter immediately")
	}
	if err := f.svc.RemoveWord(ctx, mod, 1); err != nil {
		t.Fatal(err)
	}
	if f.words.Load().Apply("join telegram.me/x").Blocked {
		t.Fatal("removed rule must stop applying")
	}
}

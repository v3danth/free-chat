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
	r.ID = uint64(len(m.reports) + 1)
	m.reports = append(m.reports, r)
	return nil
}
func (m *memStore) GetReport(_ context.Context, id uint64) (Report, error) {
	for _, r := range m.reports {
		if r.ID == id {
			return r, nil
		}
	}
	return Report{}, ErrReportNotFound
}
func (m *memStore) WasActioned(_ context.Context, t TargetType, id uint64) (bool, error) {
	for _, r := range m.reports {
		if r.TargetType == t && r.TargetID == id && r.Status == StatusActioned {
			return true, nil
		}
	}
	return false, nil
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
func (m *memStore) SetReportStatus(_ context.Context, id uint64, s Status, _ uint64) error {
	for i := range m.reports {
		if m.reports[i].ID == id && m.reports[i].Status == StatusOpen {
			m.reports[i].Status = s
			return nil
		}
	}
	return ErrReportNotFound
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
func (m *memMessages) Unhide(_ context.Context, id uint64) error {
	delete(m.hidden, id)
	return nil
}
func (m *memMessages) MediaSeenBy(_ context.Context, mediaID, viewer uint64) (bool, error) {
	for _, msg := range m.byID {
		if msg.MediaID == mediaID && (!msg.IsDirect() || msg.Involves(viewer)) {
			return true, nil
		}
	}
	return false, nil
}

type noFlush struct{}

func (noFlush) Flush(context.Context) error { return nil }

// memImages tracks which images are served, hidden (restorable) or removed.
type memImages struct {
	byID    map[uint64]media.Media
	hidden  map[uint64]bool
	removed map[uint64]bool
}

func (m *memImages) Get(_ context.Context, id uint64) (media.Media, error) {
	if img, ok := m.byID[id]; ok {
		return img, nil
	}
	return media.Media{}, media.ErrNotFound
}
func (m *memImages) Remove(_ context.Context, id, _ uint64, _ bool) (media.Media, error) {
	m.removed[id], m.hidden[id] = true, false
	return m.byID[id], nil
}
func (m *memImages) Hide(_ context.Context, id uint64) (media.Media, error) {
	m.hidden[id] = true
	return m.byID[id], nil
}
func (m *memImages) Restore(_ context.Context, id uint64) (bool, error) {
	was := m.hidden[id]
	m.hidden[id] = false
	return was, nil
}

type liveLog struct {
	kicked  []uint64
	removed []uint64
	doors   map[[2]uint64]bool
}

func (l *liveLog) Kick(id uint64, _, _ string, _ time.Time) { l.kicked = append(l.kicked, id) }
func (l *liveLog) Mute(uint64, time.Time)                   {}
func (l *liveLog) RemoveMessage(m message.Message)          { l.removed = append(l.removed, m.ID) }
func (l *liveLog) RemoveMedia(uint64)                       {}
func (l *liveLog) UpdateUser(user.User)                     {}
func (l *liveLog) DoorOpen(a, b uint64) bool                { return l.doors[[2]uint64{min(a, b), max(a, b)}] }

// --- harness ---

type fixture struct {
	svc    *Service
	store  *memStore
	msgs   *memMessages
	images *memImages
	live   *liveLog
	users  *usertest.Memory
	words  *filter.Live
}

func newFixture(t *testing.T) *fixture {
	users := usertest.NewMemory()
	f := &fixture{
		store:  &memStore{ipBans: map[string]time.Time{}},
		msgs:   &memMessages{byID: map[uint64]message.Message{}, hidden: map[uint64]bool{}},
		images: &memImages{byID: map[uint64]media.Media{}, hidden: map[uint64]bool{}, removed: map[uint64]bool{}},
		live:   &liveLog{doors: map[[2]uint64]bool{}},
		users:  users,
		words:  filter.NewLive(filter.New(100, nil)),
	}
	f.svc = NewService(Deps{Store: f.store, Users: users, Messages: f.msgs, Writer: noFlush{}, Images: f.images,
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

func TestImageReportsNeedVisibilityAndOnlyHide(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	owner := f.person(t, "owner", user.KindMember, user.RoleUser)
	b := f.person(t, "b", user.KindGuest, user.RoleUser)
	c := f.person(t, "c", user.KindGuest, user.RoleUser)
	f.images.byID[5] = media.Media{ID: 5, OwnerID: owner.UserID, Key: "k5"}
	report := func(who auth.Identity) error {
		return f.svc.Report(ctx, who, ReportInput{TargetType: TargetMedia, TargetID: 5, Reason: "spam"})
	}

	// A profile photo nobody was shown cannot be reported by guessing its id.
	if err := report(b); !errors.Is(err, ErrNotVisible) {
		t.Fatalf("unseen image: %v", err)
	}
	// Once a door opens, the other side sees the photo and may report it.
	f.users.UpdateCard(ctx, owner.UserID, nil, "", "", ptr(uint64(5)))
	f.live.doors[[2]uint64{min(owner.UserID, b.UserID), max(owner.UserID, b.UserID)}] = true
	if err := report(b); err != nil {
		t.Fatalf("photo revealed through a door: %v", err)
	}
	// Posted in the room: anyone may report it.
	f.msgs.byID[20] = message.NewRoom(20, 1, owner.UserID, "owner", "", 5)
	if err := report(c); err != nil {
		t.Fatalf("room image: %v", err)
	}

	// Two reports reach the threshold: hidden, not deleted.
	if !f.images.hidden[5] || f.images.removed[5] {
		t.Fatalf("want hidden and restorable, got hidden=%v removed=%v", f.images.hidden[5], f.images.removed[5])
	}
	// Dismissing every report brings it back.
	mod := f.person(t, "mod", user.KindMember, user.RoleModerator)
	for _, r := range f.store.reports {
		if err := f.svc.Dismiss(ctx, mod, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	if f.images.hidden[5] {
		t.Fatal("image should be restored once all reports are dismissed")
	}
}

func TestDismissRestoresAutoHiddenMessageButNotRemovedOnes(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	a := f.person(t, "a", user.KindGuest, user.RoleUser)
	b := f.person(t, "b", user.KindGuest, user.RoleUser)
	c := f.person(t, "c", user.KindGuest, user.RoleUser)
	mod := f.person(t, "mod", user.KindMember, user.RoleModerator)
	f.msgs.byID[11] = message.NewRoom(11, 1, a.UserID, "a", "fine actually", 0)
	f.msgs.byID[12] = message.NewRoom(12, 1, a.UserID, "a", "really bad", 0)
	for _, who := range []auth.Identity{b, c} {
		f.svc.Report(ctx, who, ReportInput{TargetType: TargetMessage, TargetID: 11, Reason: "spam"})
	}
	if !f.msgs.hidden[11] {
		t.Fatal("auto-hide expected")
	}
	f.svc.Dismiss(ctx, mod, 1)
	if !f.msgs.hidden[11] {
		t.Fatal("restored while a report is still open")
	}
	f.svc.Dismiss(ctx, mod, 2)
	if f.msgs.hidden[11] {
		t.Fatal("all reports dismissed: the message should be back")
	}

	// A moderator removes 12; a late report on it, once dismissed, must not undo that.
	if err := f.svc.RemoveMessage(ctx, mod, 12); err != nil {
		t.Fatal(err)
	}
	f.store.reports = append(f.store.reports, Report{ID: 99, TargetType: TargetMessage, TargetID: 12, Status: StatusOpen})
	f.store.reports = append(f.store.reports, Report{ID: 98, TargetType: TargetMessage, TargetID: 12, Status: StatusActioned})
	f.svc.Dismiss(ctx, mod, 99)
	if !f.msgs.hidden[12] {
		t.Fatal("a moderator's removal must stand")
	}
}

func TestContentRemovalFollowsTheRoleLadder(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	admin := f.person(t, "boss", user.KindMember, user.RoleAdmin)
	mod := f.person(t, "mod1", user.KindMember, user.RoleModerator)
	b := f.person(t, "b", user.KindGuest, user.RoleUser)
	c := f.person(t, "c", user.KindGuest, user.RoleUser)
	f.msgs.byID[30] = message.NewRoom(30, 1, admin.UserID, "boss", "admin note", 0)
	f.images.byID[6] = media.Media{ID: 6, OwnerID: admin.UserID, Key: "k6"}

	if err := f.svc.RemoveMessage(ctx, mod, 30); !errors.Is(err, ErrOutranked) {
		t.Fatalf("mod removing the admin's message: %v", err)
	}
	if err := f.svc.RemoveMedia(ctx, mod, 6); !errors.Is(err, ErrOutranked) {
		t.Fatalf("mod removing the admin's image: %v", err)
	}
	// Reports on staff content wait for a human: never auto-hidden.
	for _, who := range []auth.Identity{b, c} {
		f.svc.Report(ctx, who, ReportInput{TargetType: TargetMessage, TargetID: 30, Reason: "spam"})
	}
	if f.msgs.hidden[30] {
		t.Fatal("staff content must not be auto-hidden")
	}
}

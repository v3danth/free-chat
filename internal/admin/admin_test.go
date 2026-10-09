package admin_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/admin"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/moderation"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/websocket"
)

type liveStub struct{}

func (liveStub) Snapshot() websocket.Snapshot { return websocket.Snapshot{Online: 3} }

type creatorStub struct{ got user.Role }

func (c *creatorStub) CreateStaff(_ context.Context, reg auth.Registration, role user.Role) (user.User, error) {
	c.got = role
	return user.User{ID: 9, Kind: user.KindMember, Role: role, Profile: reg.Profile, Email: &reg.Email}, nil
}

type auditStub struct{ logged []moderation.Action }

func (a *auditStub) Log(_ context.Context, act moderation.Action) error {
	a.logged = append(a.logged, act)
	return nil
}

func TestCreateStaffIsAudited(t *testing.T) {
	creator, audit := &creatorStub{}, &auditStub{}
	svc := admin.NewService(nil, liveStub{}, nil, creator, audit)
	reg := auth.Registration{Profile: user.Profile{Name: "mod_alex", Gender: user.GenderOther, Age: 30}, Email: "alex@example.com"}

	m, err := svc.CreateStaff(context.Background(), auth.Identity{UserID: 1}, admin.NewStaff{Registration: reg, Role: user.RoleModerator})
	if err != nil || m.Email != "alex@example.com" || m.Role != user.RoleModerator {
		t.Fatalf("CreateStaff: %+v, %v", m, err)
	}
	if len(audit.logged) != 1 || audit.logged[0].ActorID != 1 || audit.logged[0].TargetUserID != 9 {
		t.Fatalf("audit: %+v", audit.logged)
	}
}

// TestOverviewQueries runs the stats SQL against a real test database.
// It needs TEST_MYSQL_DATABASE (ending in _test) plus MYSQL_HOST, _PORT,
// _USER and _PASSWORD; `make test` passes them from .env.
func TestOverviewQueries(t *testing.T) {
	name := os.Getenv("TEST_MYSQL_DATABASE")
	if !strings.HasSuffix(name, "_test") {
		t.Skip("set TEST_MYSQL_DATABASE to a *_test database to run")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{
		Host: os.Getenv("MYSQL_HOST"), Port: os.Getenv("MYSQL_PORT"), User: os.Getenv("MYSQL_USER"),
		Password: os.Getenv("MYSQL_PASSWORD"), Name: name,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec := func(q string, args ...any) int64 {
		t.Helper()
		res, err := db.ExecContext(ctx, q, args...)
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		n, _ := res.LastInsertId()
		return n
	}
	for _, table := range []string{"reports", "mod_actions", "messages", "blocks", "ip_bans", "media", "users"} {
		exec("DELETE FROM " + table)
	}

	member := exec(`INSERT INTO users (kind, name, gender, age, email, password_hash, country_code) VALUES ('member', 'm1', 'other', 30, 'm1@example.com', 'x', 'IN')`)
	guest := exec(`INSERT INTO users (kind, name, gender, age, country_code) VALUES ('guest', 'g1', 'male', 20, 'IN')`)
	exec(`INSERT INTO users (kind, name, gender, age, banned_until) VALUES ('guest', 'g2', 'female', 22, NOW() + INTERVAL 1 DAY)`)

	ids := &id.Generator{}
	roomMsg := ids.Next()
	exec(`INSERT INTO messages (id, room_id, sender_id, sender_name, body) VALUES (?, 1, ?, 'g1', 'hi')`, roomMsg, guest)
	exec(`INSERT INTO messages (id, recipient_id, sender_id, sender_name, body) VALUES (?, ?, ?, 'g1', 'psst')`, ids.Next(), member, guest)
	exec(`INSERT INTO reports (reporter_id, target_type, target_id, target_user_id, reason, evidence) VALUES (?, 'message', ?, ?, 'spam', '{}')`, member, roomMsg, guest)

	o, err := admin.NewService(db, liveStub{}, nil, nil, nil).Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if o.Now.Online != 3 {
		t.Errorf("live snapshot not passed through: %+v", o.Now)
	}
	if p := o.People; p.Members != 1 || p.Guests24h != 2 || p.Members24h != 1 || p.Guests7d != 2 {
		t.Errorf("people: %+v", p)
	}
	last := o.Messages.PerHour[23]
	if o.Messages.Room24h != 1 || o.Messages.Private24h != 1 || last.Room != 1 || last.Private != 1 {
		t.Errorf("messages: %+v, last hour %+v", o.Messages, last)
	}
	if want := time.Now().UTC().Truncate(time.Hour).Unix(); last.Hour != want {
		t.Errorf("last bucket starts at %d, want %d", last.Hour, want)
	}
	if s := o.Safety; s.OpenReports != 1 || s.Reasons7d["spam"] != 1 || s.Banned != 1 {
		t.Errorf("safety: %+v", s)
	}
	if len(o.Countries) == 0 || o.Countries[0] != (admin.CountryCount{Country: "IN", Signups: 2}) {
		t.Errorf("countries: %+v", o.Countries)
	}
}

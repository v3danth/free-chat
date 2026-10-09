// Package admin is the control panel's backend: live and recent numbers,
// and staff accounts, which only the admin can create.
package admin

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/moderation"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/websocket"
)

// Overview is everything the panel's first screen shows.
type Overview struct {
	Now       websocket.Snapshot `json:"now"`
	People    People             `json:"people"`
	Messages  Messages           `json:"messages"`
	Safety    Safety             `json:"safety"`
	Countries []CountryCount     `json:"signup_countries_7d"`
	At        time.Time          `json:"generated_at"`
}

// People counts sign-ups. Guests are deleted once they have been gone for
// the retention period (7 days by default), so windows up to that are exact.
type People struct {
	Members    int `json:"members_total"`
	Guests24h  int `json:"guests_24h"`
	Members24h int `json:"members_24h"`
	Guests7d   int `json:"guests_7d"`
	Members7d  int `json:"members_7d"`
}

type Messages struct {
	Room24h    int         `json:"room_24h"`
	Private24h int         `json:"private_24h"`
	PerHour    []HourCount `json:"per_hour"` // the last 24 hours, oldest first
}

type HourCount struct {
	Hour    int64 `json:"hour"` // unix seconds at the start of the hour
	Room    int   `json:"room"`
	Private int   `json:"private"`
}

type Safety struct {
	OpenReports int            `json:"open_reports"`
	Reasons7d   map[string]int `json:"report_reasons_7d"`
	Banned      int            `json:"banned_users"`
	Muted       int            `json:"muted_users"`
	IPBans      int            `json:"ip_bans"`
}

type CountryCount struct {
	Country string `json:"country"` // "" when unknown
	Signups int    `json:"signups"`
}

// Live reports who is connected now.
type Live interface {
	Snapshot() websocket.Snapshot
}

type Staff interface {
	ListStaff(ctx context.Context) ([]user.User, error)
}

// Creator makes staff accounts.
type Creator interface {
	CreateStaff(ctx context.Context, reg auth.Registration, role user.Role) (user.User, error)
}

// Audit records who created which staff account.
type Audit interface {
	Log(ctx context.Context, a moderation.Action) error
}

type Service struct {
	db      *sql.DB
	live    Live
	staff   Staff
	creator Creator
	audit   Audit
	now     func() time.Time
}

func NewService(db *sql.DB, live Live, staff Staff, creator Creator, audit Audit) *Service {
	return &Service{db: db, live: live, staff: staff, creator: creator, audit: audit, now: time.Now}
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	now := s.now().UTC()
	o := Overview{Now: s.live.Snapshot(), At: now}
	var err error
	if o.People, err = s.people(ctx); err != nil {
		return o, fmt.Errorf("people: %w", err)
	}
	if o.Messages, err = s.messages(ctx, now); err != nil {
		return o, fmt.Errorf("messages: %w", err)
	}
	if o.Safety, err = s.safety(ctx); err != nil {
		return o, fmt.Errorf("safety: %w", err)
	}
	if o.Countries, err = s.countries(ctx); err != nil {
		return o, fmt.Errorf("countries: %w", err)
	}
	return o, nil
}

func (s *Service) people(ctx context.Context) (People, error) {
	var p People
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(kind = 'member'), 0),
		       COALESCE(SUM(kind = 'guest'  AND created_at >= NOW() - INTERVAL 1 DAY), 0),
		       COALESCE(SUM(kind = 'member' AND created_at >= NOW() - INTERVAL 1 DAY), 0),
		       COALESCE(SUM(kind = 'guest'  AND created_at >= NOW() - INTERVAL 7 DAY), 0),
		       COALESCE(SUM(kind = 'member' AND created_at >= NOW() - INTERVAL 7 DAY), 0)
		FROM users`).Scan(&p.Members, &p.Guests24h, &p.Members24h, &p.Guests7d, &p.Members7d)
	return p, err
}

// messages buckets the last 24 hours by hour. Message ids carry their
// time (milliseconds since id.Epoch, shifted), so no date column is needed.
func (s *Service) messages(ctx context.Context, now time.Time) (Messages, error) {
	first := now.Truncate(time.Hour).Add(-23 * time.Hour)
	m := Messages{PerHour: make([]HourCount, 24)}
	for i := range m.PerHour {
		m.PerHour[i].Hour = first.Add(time.Duration(i) * time.Hour).Unix()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT (id >> 12) DIV 3600000 AS h,
		       SUM(recipient_id IS NULL), SUM(recipient_id IS NOT NULL)
		FROM messages WHERE id >= ? GROUP BY h`, id.Floor(first))
	if err != nil {
		return m, err
	}
	defer rows.Close()
	for rows.Next() {
		var hour int64
		var room, private int
		if err := rows.Scan(&hour, &room, &private); err != nil {
			return m, err
		}
		at := id.Epoch.Add(time.Duration(hour) * time.Hour)
		if i := int(at.Sub(first) / time.Hour); i >= 0 && i < 24 {
			m.PerHour[i].Room, m.PerHour[i].Private = room, private
			m.Room24h += room
			m.Private24h += private
		}
	}
	return m, rows.Err()
}

func (s *Service) safety(ctx context.Context) (Safety, error) {
	sf := Safety{Reasons7d: map[string]int{}}
	err := s.db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM reports WHERE status = 'open'),
		       (SELECT COUNT(*) FROM users WHERE banned_until > NOW()),
		       (SELECT COUNT(*) FROM users WHERE muted_until > NOW()),
		       (SELECT COUNT(*) FROM ip_bans WHERE expires_at > NOW())`).
		Scan(&sf.OpenReports, &sf.Banned, &sf.Muted, &sf.IPBans)
	if err != nil {
		return sf, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT reason, COUNT(*) FROM reports
		WHERE created_at >= NOW() - INTERVAL 7 DAY GROUP BY reason`)
	if err != nil {
		return sf, err
	}
	defer rows.Close()
	for rows.Next() {
		var reason string
		var n int
		if err := rows.Scan(&reason, &n); err != nil {
			return sf, err
		}
		sf.Reasons7d[reason] = n
	}
	return sf, rows.Err()
}

func (s *Service) countries(ctx context.Context) ([]CountryCount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(country_code, ''), COUNT(*) FROM users
		WHERE created_at >= NOW() - INTERVAL 7 DAY
		GROUP BY 1 ORDER BY 2 DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []CountryCount{}
	for rows.Next() {
		var c CountryCount
		if err := rows.Scan(&c.Country, &c.Signups); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// StaffMember is one row of the staff list.
type StaffMember struct {
	user.Card
	Email string `json:"email"`
}

func (s *Service) Staff(ctx context.Context) ([]StaffMember, error) {
	staff, err := s.staff.ListStaff(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]StaffMember, len(staff))
	for i, u := range staff {
		list[i] = StaffMember{Card: user.ToCard(u, time.Time{})}
		if u.Email != nil {
			list[i].Email = *u.Email
		}
	}
	return list, nil
}

// NewStaff is a validated request to create a staff account.
type NewStaff struct {
	Registration auth.Registration
	Role         user.Role
}

func (s *Service) CreateStaff(ctx context.Context, actor auth.Identity, in NewStaff) (StaffMember, error) {
	u, err := s.creator.CreateStaff(ctx, in.Registration, in.Role)
	if err != nil {
		return StaffMember{}, err
	}
	err = s.audit.Log(ctx, moderation.Action{
		ActorID: actor.UserID, Action: moderation.ActSetRole, TargetUserID: u.ID,
		TargetName: u.Profile.Name, Detail: "created staff account as " + string(in.Role),
	})
	if err != nil {
		return StaffMember{}, err
	}
	m := StaffMember{Card: user.ToCard(u, time.Time{})}
	if u.Email != nil {
		m.Email = *u.Email
	}
	return m, nil
}

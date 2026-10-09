package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/user/usertest"
)

const secret = "test-secret-test-secret-test-secret"

type fakeIPBans map[string]bool

func (f fakeIPBans) IsIPBanned(_ context.Context, h []byte) (bool, error) { return f[string(h)], nil }

type fakeGeo map[string]string

func (f fakeGeo) Country(ip string) string { return f[ip] }

func newService() (*auth.Service, *usertest.Memory, fakeIPBans) {
	repo := usertest.NewMemory()
	bans := fakeIPBans{}
	words := filter.NewLive(filter.New(1000, []filter.Rule{{Word: "scam", Action: filter.Block}}))
	return auth.NewService(repo, auth.NewTokens(secret), bans, fakeGeo{"203.0.113.9": "IN"}, words, secret), repo, bans
}

var home = auth.Origin{IP: "203.0.113.9"}

func profile(name string) user.Profile {
	return user.Profile{Name: name, Gender: user.GenderOther, Age: 21}
}

func TestGuestJoinsWithCountryAndResumes(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()

	s, err := svc.CreateGuest(ctx, auth.GuestSignup{Profile: profile("ravi"), Origin: home})
	if err != nil {
		t.Fatalf("CreateGuest: %v", err)
	}
	if s.User.Country != "IN" || s.User.Kind != user.KindGuest {
		t.Fatalf("unexpected user: %+v", s.User)
	}
	u, err := svc.Resume(ctx, s.Token, home)
	if err != nil || u.ID != s.User.ID {
		t.Fatalf("Resume: %+v, %v", u, err)
	}
}

func TestGuestNamesAreNotUnique(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	for range 2 {
		if _, err := svc.CreateGuest(ctx, auth.GuestSignup{Profile: profile("same"), Origin: home}); err != nil {
			t.Fatalf("guests may share a name: %v", err)
		}
	}
}

func TestBannedProfileWordsRejected(t *testing.T) {
	svc, _, _ := newService()
	p := profile("ravi")
	p.About = "total scam"
	if _, err := svc.CreateGuest(context.Background(), auth.GuestSignup{Profile: p, Origin: home}); err == nil {
		t.Fatal("blocked word in about must be rejected")
	}
}

func TestRegisterAndLogin(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	reg := auth.Registration{Profile: profile("alice"), Email: "a@example.com", Password: "hunter22!", Origin: home}

	if _, err := svc.Register(ctx, reg); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := svc.Register(ctx, auth.Registration{Profile: profile("ALICE"), Email: "b@example.com", Password: "hunter22!", Origin: home}); !errors.Is(err, user.ErrNameTaken) {
		t.Fatalf("member names are unique: %v", err)
	}
	if _, err := svc.Login(ctx, auth.Credentials{Email: reg.Email, Password: reg.Password, Origin: home}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	for _, c := range []auth.Credentials{
		{Email: reg.Email, Password: "wrong-pass", Origin: home},
		{Email: "nobody@example.com", Password: reg.Password, Origin: home},
	} {
		if _, err := svc.Login(ctx, c); !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("Login(%s): want ErrInvalidCredentials, got %v", c.Email, err)
		}
	}
}

func TestResumeRejectsEndedSessions(t *testing.T) {
	svc, repo, bans := newService()
	ctx := context.Background()
	s, _ := svc.CreateGuest(ctx, auth.GuestSignup{Profile: profile("ghost"), Origin: home})

	// A ban revokes the token (version bump) and blocks the account.
	until := time.Now().Add(time.Hour)
	repo.SetBan(ctx, s.User.ID, &until)
	if _, err := svc.Resume(ctx, s.Token, home); !errors.Is(err, auth.ErrSessionEnded) {
		t.Fatalf("revoked token: want ErrSessionEnded, got %v", err)
	}

	// A banned network cannot resume or join, even with a fresh profile.
	s2, _ := svc.CreateGuest(ctx, auth.GuestSignup{Profile: profile("other"), Origin: home})
	bans[string(svc.HashIP(home.IP))] = true
	if _, err := svc.Resume(ctx, s2.Token, home); !errors.Is(err, auth.ErrBanned) {
		t.Fatalf("banned IP resume: %v", err)
	}
	if _, err := svc.CreateGuest(ctx, auth.GuestSignup{Profile: profile("new"), Origin: home}); !errors.Is(err, auth.ErrBanned) {
		t.Fatalf("banned IP join: %v", err)
	}

	repo.Delete(s2.User.ID)
	if _, err := svc.Resume(ctx, s2.Token, auth.Origin{IP: "198.51.100.1"}); !errors.Is(err, auth.ErrSessionEnded) {
		t.Fatalf("deleted account: %v", err)
	}
	if _, err := svc.Resume(ctx, "not-a-jwt", home); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("garbage token: %v", err)
	}
}

func TestEnsureAdminIsIdempotent(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	for range 2 {
		if err := svc.EnsureAdmin(ctx, "me@example.com", "long-password", "admin"); err != nil {
			t.Fatalf("EnsureAdmin: %v", err)
		}
	}
	u, err := repo.GetByEmail(ctx, "me@example.com")
	if err != nil || u.Role != user.RoleAdmin || u.Kind != user.KindMember {
		t.Fatalf("admin = %+v, %v", u, err)
	}
}

func TestGuestEndpointContract(t *testing.T) {
	svc, _, _ := newService()
	mux := http.NewServeMux()
	auth.Routes(mux, svc, httpx.NewIPResolver(false))

	post := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/guest", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.9:5555"
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := post(`{"name":"राहुल","gender":"male","age":30,"intent":"night_owl","location":"Delhi"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var got struct {
		Token string         `json:"token"`
		User  map[string]any `json:"user"`
	}
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Token == "" || got.User["name"] != "राहुल" || got.User["country"] != "IN" || got.User["location"] != "Delhi" {
		t.Fatalf("unexpected session view: %s", rec.Body)
	}
	for _, leaked := range []string{"password_hash", "ip_hash", "token_version"} {
		if _, ok := got.User[leaked]; ok {
			t.Fatalf("%s must never be serialized", leaked)
		}
	}

	for _, bad := range []string{`{`, `{"name":"x","gender":"male","age":30}`, `{"name":"bob","gender":"male","age":16}`} {
		if rec := post(bad); rec.Code != http.StatusBadRequest {
			t.Errorf("body %s: status = %d, want 400", bad, rec.Code)
		}
	}
}

func TestCreateStaff(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	reg := auth.Registration{Profile: profile("mod_alex"), Email: "alex@example.com", Password: "a-good-password"}

	if _, err := svc.CreateStaff(ctx, reg, user.RoleUser); err == nil {
		t.Fatal("a plain user role must be refused")
	}
	u, err := svc.CreateStaff(ctx, reg, user.RoleModerator)
	if err != nil || u.Role != user.RoleModerator || u.Kind != user.KindMember {
		t.Fatalf("CreateStaff: %+v, %v", u, err)
	}
	if _, err := svc.Login(ctx, auth.Credentials{Email: "alex@example.com", Password: "a-good-password", Origin: home}); err != nil {
		t.Fatalf("staff cannot log in: %v", err)
	}
	staff, _ := repo.ListStaff(ctx)
	if len(staff) != 1 || staff[0].ID != u.ID {
		t.Fatalf("ListStaff: %+v", staff)
	}
}

func TestLoginAttemptsAreLimited(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, auth.Registration{Profile: profile("member1"), Email: "m@example.com", Password: "right-password", Origin: home}); err != nil {
		t.Fatal(err)
	}
	wrong := auth.Credentials{Email: "m@example.com", Password: "wrong-password", Origin: home}
	for i := range 8 {
		if _, err := svc.Login(ctx, wrong); !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	right := wrong
	right.Password = "right-password"
	if _, err := svc.Login(ctx, right); !errors.Is(err, auth.ErrTooManyLogins) {
		t.Fatalf("ninth attempt on one email must be refused, even with the right password: %v", err)
	}
	// Another email from the same IP still works (the IP limit is higher).
	if _, err := svc.Register(ctx, auth.Registration{Profile: profile("member2"), Email: "n@example.com", Password: "right-password", Origin: home}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(ctx, auth.Credentials{Email: "n@example.com", Password: "right-password", Origin: home}); err != nil {
		t.Fatalf("other email: %v", err)
	}
}

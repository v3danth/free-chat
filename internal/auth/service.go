package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/geo"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
)

var (
	ErrInvalidCredentials = apperr.New(apperr.Unauthorized, "invalid credentials")
	ErrSessionEnded       = apperr.New(apperr.Unauthorized, "session ended, please join again")
	ErrBanned             = apperr.New(apperr.Forbidden, "you are banned")
	ErrTooManySignups     = apperr.New(apperr.RateLimited, "too many new profiles from your network, try again later")
	ErrTooManyLogins      = apperr.New(apperr.RateLimited, "too many sign-in attempts, try again in a few minutes")
)

// dummyHash lets Login spend the same bcrypt time whether or not the email
// exists, so response timing does not reveal registered emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

type IPBans interface {
	IsIPBanned(ctx context.Context, ipHash []byte) (bool, error)
}

// Identity is a resumed, live caller.
type Identity struct {
	UserID uint64
	Kind   user.Kind
	Role   user.Role
	Name   string
}

func IdentityOf(u user.User) Identity {
	return Identity{UserID: u.ID, Kind: u.Kind, Role: u.Role, Name: u.Profile.Name}
}

type Session struct {
	Token string
	User  user.User
}

// Origin is where a request came from, resolved at the HTTP boundary.
type Origin struct {
	IP string
}

type GuestSignup struct {
	Profile user.Profile
	Origin  Origin
}

type Registration struct {
	Profile  user.Profile
	Email    string
	Password string
	Origin   Origin
}

type Credentials struct {
	Email    string
	Password string
	Origin   Origin
}

type Service struct {
	users   user.Repository
	tokens  Tokens
	ipBans  IPBans
	geo     geo.Locator
	words   *filter.Live
	ipKey   []byte
	signups *ratelimit.Limiter
	// Sign-in attempts, per IP and per email: both are checked before the
	// (deliberately slow) password comparison.
	loginsByIP    *ratelimit.Limiter
	loginsByEmail *ratelimit.Limiter
}

func NewService(users user.Repository, tokens Tokens, ipBans IPBans, locator geo.Locator, words *filter.Live, ipKey string) *Service {
	return &Service{
		users:  users,
		tokens: tokens,
		ipBans: ipBans,
		geo:    locator,
		words:  words,
		ipKey:  []byte(ipKey),
		// Generous: Indian mobile carriers put many people behind one IP.
		signups:       ratelimit.New(ratelimit.Config{Rate: 30, Window: 10 * time.Minute}),
		loginsByIP:    ratelimit.New(ratelimit.Config{Rate: 20, Window: 10 * time.Minute}),
		loginsByEmail: ratelimit.New(ratelimit.Config{Rate: 8, Window: 10 * time.Minute}),
	}
}

// HashIP is a keyed hash, so stored values cannot be reversed by
// brute-forcing the IPv4 space.
func (s *Service) HashIP(ip string) []byte {
	m := hmac.New(sha256.New, s.ipKey)
	m.Write([]byte(ip))
	return m.Sum(nil)
}

func (s *Service) CreateGuest(ctx context.Context, g GuestSignup) (Session, error) {
	ipHash, err := s.admit(ctx, g.Origin)
	if err != nil {
		return Session{}, err
	}
	if !s.signups.Allow(bucket(ipHash)) {
		return Session{}, ErrTooManySignups
	}
	profile, err := g.Profile.Screened(s.words.Load().Screen)
	if err != nil {
		return Session{}, err
	}
	u, err := s.users.Create(ctx, user.User{
		Kind:    user.KindGuest,
		Role:    user.RoleUser,
		Profile: profile,
		Country: s.geo.Country(g.Origin.IP),
		IPHash:  ipHash,
	})
	if err != nil {
		return Session{}, err
	}
	return s.session(u)
}

func (s *Service) Register(ctx context.Context, reg Registration) (user.User, error) {
	ipHash, err := s.admit(ctx, reg.Origin)
	if err != nil {
		return user.User{}, err
	}
	if !s.signups.Allow(bucket(ipHash)) {
		return user.User{}, ErrTooManySignups
	}
	profile, err := reg.Profile.Screened(s.words.Load().Screen)
	if err != nil {
		return user.User{}, err
	}
	return s.createMember(ctx, profile, reg.Email, reg.Password, user.RoleUser, s.geo.Country(reg.Origin.IP), ipHash)
}

var errStaffRole = apperr.New(apperr.Invalid, "staff role must be moderator or admin")

// CreateStaff makes a moderator or admin account. Only the admin calls it
// (the route checks); staff cannot sign themselves up.
func (s *Service) CreateStaff(ctx context.Context, reg Registration, role user.Role) (user.User, error) {
	if !role.AtLeast(user.RoleModerator) {
		return user.User{}, errStaffRole
	}
	return s.createMember(ctx, reg.Profile, reg.Email, reg.Password, role, "", nil)
}

func (s *Service) Login(ctx context.Context, c Credentials) (Session, error) {
	ipHash, err := s.admit(ctx, c.Origin)
	if err != nil {
		return Session{}, err
	}
	// Both limits count every attempt, so neither one can be used to probe
	// whether the other is exhausted.
	byIP, byEmail := s.loginsByIP.Allow(bucket(ipHash)), s.loginsByEmail.Allow(bucket(s.HashIP("email:"+c.Email)))
	if !byIP || !byEmail {
		return Session{}, ErrTooManyLogins
	}
	u, err := s.users.GetByEmail(ctx, c.Email)
	if err != nil && !errors.Is(err, user.ErrNotFound) {
		return Session{}, err
	}

	hash := dummyHash
	if err == nil && u.PasswordHash != nil {
		hash = []byte(*u.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(c.Password)) != nil || err != nil {
		return Session{}, ErrInvalidCredentials
	}
	if u.BannedAt(time.Now()) {
		return Session{}, ErrBanned
	}
	return s.session(u)
}

// Resume turns a bearer token into the live user. It is the single check
// used by HTTP and WebSocket: the token must verify, the account must still
// exist, the token must not be revoked, and neither the account nor the
// network may be banned.
func (s *Service) Resume(ctx context.Context, token string, origin Origin) (user.User, error) {
	id, version, err := s.tokens.verify(token)
	if err != nil {
		return user.User{}, err
	}
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, user.ErrNotFound) {
		return user.User{}, ErrSessionEnded
	}
	if err != nil {
		return user.User{}, err
	}
	if u.TokenVersion != version {
		return user.User{}, ErrSessionEnded
	}
	if u.BannedAt(time.Now()) {
		return user.User{}, ErrBanned
	}
	if _, err := s.admit(ctx, origin); err != nil {
		return user.User{}, err
	}
	return u, nil
}

// EnsureAdmin creates the configured admin member, or promotes the member
// that already owns the email. Used once at startup.
func (s *Service) EnsureAdmin(ctx context.Context, email, password, name string) error {
	u, err := s.users.GetByEmail(ctx, email)
	switch {
	case err == nil:
		if u.Role == user.RoleAdmin {
			return nil
		}
		return s.users.SetRole(ctx, u.ID, user.RoleAdmin)
	case !errors.Is(err, user.ErrNotFound):
		return err
	}
	// The bootstrap name may be reserved for everyone else ("admin").
	profile := user.Profile{Name: name, Gender: user.GenderOther, Age: 18}
	_, err = s.createMember(ctx, profile, email, password, user.RoleAdmin, "", nil)
	return err
}

func (s *Service) createMember(ctx context.Context, p user.Profile, email, password string, role user.Role, country string, ipHash []byte) (user.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	return s.users.Create(ctx, user.User{
		Kind:         user.KindMember,
		Role:         role,
		Profile:      p,
		Country:      country,
		Email:        &email,
		PasswordHash: &hashStr,
		IPHash:       ipHash,
	})
}

// admit rejects banned networks and returns the origin's IP hash.
func (s *Service) admit(ctx context.Context, o Origin) ([]byte, error) {
	h := s.HashIP(o.IP)
	banned, err := s.ipBans.IsIPBanned(ctx, h)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrBanned
	}
	return h, nil
}

func (s *Service) session(u user.User) (Session, error) {
	token, err := s.tokens.Issue(u)
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, User: u}, nil
}

// bucket folds an IP hash into a rate-limiter key.
func bucket(h []byte) uint64 {
	var k uint64
	for _, b := range h[:8] {
		k = k<<8 | uint64(b)
	}
	return k
}

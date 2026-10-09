package auth

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/user"
)

func Routes(mux *http.ServeMux, svc *Service, ipOf httpx.IPResolver) {
	origin := func(r *http.Request) Origin { return Origin{IP: ipOf(r)} }

	mux.Handle("POST /auth/guest", httpx.Endpoint(http.StatusCreated,
		func(r *http.Request, in user.ProfileInput) (GuestSignup, error) {
			p, err := in.Parse()
			return GuestSignup{Profile: p, Origin: origin(r)}, err
		},
		svc.CreateGuest, toSessionView))

	mux.Handle("POST /auth/register", httpx.Endpoint(http.StatusCreated,
		func(r *http.Request, in RegisterRequest) (Registration, error) {
			reg, err := in.Parse()
			reg.Origin = origin(r)
			return reg, err
		},
		svc.Register, user.ToSelf))

	mux.Handle("POST /auth/login", httpx.Endpoint(http.StatusOK,
		func(r *http.Request, in loginRequest) (Credentials, error) {
			c, err := in.parse()
			c.Origin = origin(r)
			return c, err
		},
		svc.Login, toSessionView))
}

// AuthedHandler is a handler that only runs for a resumed, live identity.
type AuthedHandler func(http.ResponseWriter, *http.Request, Identity)

func (s *Service) RequireBearer(ipOf httpx.IPResolver, next AuthedHandler) http.HandlerFunc {
	return s.RequireRole(ipOf, user.RoleUser, next)
}

func (s *Service) RequireRole(ipOf httpx.IPResolver, min user.Role, next AuthedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			httpx.Error(w, ErrInvalidToken)
			return
		}
		u, err := s.Resume(r.Context(), token, Origin{IP: ipOf(r)})
		if err != nil {
			httpx.Error(w, err)
			return
		}
		if !u.Role.AtLeast(min) {
			httpx.Error(w, errForbidden)
			return
		}
		next(w, r, IdentityOf(u))
	}
}

var errForbidden = apperr.New(apperr.Forbidden, "you do not have permission to do that")

// --- inbound ---

// RegisterRequest is the body of a member sign-up, also used by the admin
// to create staff accounts.
type RegisterRequest struct {
	user.ProfileInput
	Email    string `json:"email"`
	Password string `json:"password"`
}

var (
	errEmail    = apperr.New(apperr.Invalid, "a valid email is required")
	errPassword = apperr.New(apperr.Invalid, "password must be 8-72 bytes")
)

func (r RegisterRequest) Parse() (Registration, error) {
	profile, err := r.ProfileInput.Parse()
	if err != nil {
		return Registration{}, err
	}
	email, err := parseEmail(r.Email)
	if err != nil {
		return Registration{}, err
	}
	// bcrypt only uses the first 72 bytes and rejects longer input.
	if len(r.Password) < 8 || len(r.Password) > 72 {
		return Registration{}, errPassword
	}
	return Registration{Profile: profile, Email: email, Password: r.Password}, nil
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r loginRequest) parse() (Credentials, error) {
	email := strings.ToLower(strings.TrimSpace(r.Email))
	if email == "" || r.Password == "" {
		return Credentials{}, ErrInvalidCredentials
	}
	return Credentials{Email: email, Password: r.Password}, nil
}

func parseEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email || len(email) > 255 {
		return "", errEmail
	}
	return email, nil
}

// --- outbound ---

type sessionView struct {
	Token string    `json:"token"`
	User  user.Self `json:"user"`
}

func toSessionView(s Session) sessionView {
	return sessionView{Token: s.Token, User: user.ToSelf(s.User)}
}

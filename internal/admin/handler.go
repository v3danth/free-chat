package admin

import (
	"net/http"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/user"
)

// Routes:
//
//	GET  /admin/overview  moderators and admins
//	GET  /admin/staff     admin only
//	POST /admin/staff     admin only; creates a moderator or admin account
func Routes(mux *http.ServeMux, svc *Service, authSvc *auth.Service, ipOf httpx.IPResolver) {
	mod := func(fn auth.AuthedHandler) http.Handler { return authSvc.RequireRole(ipOf, user.RoleModerator, fn) }
	admin := func(fn auth.AuthedHandler) http.Handler { return authSvc.RequireRole(ipOf, user.RoleAdmin, fn) }

	mux.Handle("GET /admin/overview", mod(func(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
		o, err := svc.Overview(r.Context())
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, o)
	}))

	mux.Handle("GET /admin/staff", admin(func(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
		staff, err := svc.Staff(r.Context())
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, staff)
	}))

	mux.Handle("POST /admin/staff", admin(func(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
		in, err := parseNewStaff(w, r)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		m, err := svc.CreateStaff(r.Context(), actor, in)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, m)
	}))
}

var errRole = apperr.New(apperr.Invalid, "role must be moderator or admin")

type newStaffRequest struct {
	auth.RegisterRequest
	Role string `json:"role"`
}

func parseNewStaff(w http.ResponseWriter, r *http.Request) (NewStaff, error) {
	req, err := httpx.DecodeJSON[newStaffRequest](w, r)
	if err != nil {
		return NewStaff{}, err
	}
	role, ok := user.ParseRole(req.Role)
	if !ok {
		return NewStaff{}, errRole
	}
	reg, err := req.RegisterRequest.Parse()
	return NewStaff{Registration: reg, Role: role}, err
}

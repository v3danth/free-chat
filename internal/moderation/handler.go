package moderation

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/user"
)

const (
	listLimit   = 100
	maxDuration = 365 * 24 * time.Hour
)

var (
	errBadID       = apperr.New(apperr.Invalid, "invalid id")
	errBadDuration = apperr.New(apperr.Invalid, "minutes must be between 1 and 525600 (0 = permanent, bans only)")
	errBadRole     = apperr.New(apperr.Invalid, "role must be user, moderator or admin")
	errBadStatus   = apperr.New(apperr.Invalid, "status must be open, actioned or dismissed")
)

func Routes(mux *http.ServeMux, svc *Service, authSvc *auth.Service, ipOf httpx.IPResolver) {
	h := handler{svc: svc}
	anyone := func(fn auth.AuthedHandler) http.Handler { return authSvc.RequireBearer(ipOf, fn) }
	mod := func(fn auth.AuthedHandler) http.Handler { return authSvc.RequireRole(ipOf, user.RoleModerator, fn) }
	admin := func(fn auth.AuthedHandler) http.Handler { return authSvc.RequireRole(ipOf, user.RoleAdmin, fn) }

	mux.Handle("POST /reports", anyone(h.report))

	mux.Handle("GET /admin/reports", mod(h.listReports))
	mux.Handle("POST /admin/reports/{id}/dismiss", mod(h.byID(svc.Dismiss)))
	mux.Handle("POST /admin/messages/{id}/remove", mod(h.byID(svc.RemoveMessage)))
	mux.Handle("POST /admin/media/{id}/remove", mod(h.byID(svc.RemoveMedia)))
	mux.Handle("GET /admin/users/{id}/messages", mod(h.userMessages))
	mux.Handle("POST /admin/users/{id}/kick", mod(h.byID(svc.Kick)))
	mux.Handle("POST /admin/users/{id}/mute", mod(h.mute))
	mux.Handle("POST /admin/users/{id}/ban", mod(h.ban))
	mux.Handle("POST /admin/users/{id}/unban", mod(h.byID(svc.Unban)))
	mux.Handle("PUT /admin/users/{id}/role", admin(h.setRole))
	mux.Handle("GET /admin/words", mod(h.listWords))
	mux.Handle("POST /admin/words", mod(h.addWord))
	mux.Handle("DELETE /admin/words/{id}", mod(h.byID(svc.RemoveWord)))
	mux.Handle("GET /admin/audit", mod(h.audit))
}

type handler struct {
	svc *Service
}

// byID adapts "act on the {id} in the path" to a handler answering 204.
func (h handler) byID(act func(context.Context, auth.Identity, uint64) error) auth.AuthedHandler {
	return func(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
		id, err := pathID(r)
		if err == nil {
			err = act(r.Context(), actor, id)
		}
		respond(w, err)
	}
}

func (h handler) report(w http.ResponseWriter, r *http.Request, id auth.Identity) {
	req, err := httpx.DecodeJSON[struct {
		TargetType string `json:"target_type"`
		TargetID   uint64 `json:"target_id"`
		Reason     string `json:"reason"`
		Note       string `json:"note"`
	}](w, r)
	if err == nil {
		var in ReportInput
		if in, err = ParseReport(req.TargetType, req.Reason, req.TargetID, req.Note); err == nil {
			err = h.svc.Report(r.Context(), id, in)
		}
	}
	respond(w, err)
}

func (h handler) listReports(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
	status := Status(r.URL.Query().Get("status"))
	if status == "" {
		status = StatusOpen
	}
	if status != StatusOpen && status != StatusActioned && status != StatusDismissed {
		httpx.Error(w, errBadStatus)
		return
	}
	list(w, func() ([]Report, error) { return h.svc.Reports(r.Context(), status, listLimit) })
}

func (h handler) userMessages(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
	id, err := pathID(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	list(w, func() ([]EvidenceMessage, error) { return h.svc.UserMessages(r.Context(), id, listLimit) })
}

func (h handler) mute(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
	id, d, _, err := durationRequest(w, r)
	if err == nil && d == 0 {
		err = errBadDuration
	}
	if err == nil {
		err = h.svc.Mute(r.Context(), actor, id, d)
	}
	respond(w, err)
}

func (h handler) ban(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
	id, d, req, err := durationRequest(w, r)
	if err == nil {
		err = h.svc.Ban(r.Context(), actor, id, d, req.IP, req.Reason)
	}
	respond(w, err)
}

func (h handler) setRole(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
	id, err := pathID(r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	req, err := httpx.DecodeJSON[struct {
		Role string `json:"role"`
	}](w, r)
	if err == nil {
		role, ok := user.ParseRole(req.Role)
		if !ok {
			err = errBadRole
		} else {
			err = h.svc.SetRole(r.Context(), actor, id, role)
		}
	}
	respond(w, err)
}

func (h handler) listWords(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
	list(w, func() ([]Word, error) { return h.svc.Words(r.Context()) })
}

func (h handler) addWord(w http.ResponseWriter, r *http.Request, actor auth.Identity) {
	req, err := httpx.DecodeJSON[struct {
		Word   string `json:"word"`
		Action string `json:"action"`
	}](w, r)
	if err == nil {
		err = h.svc.AddWord(r.Context(), actor, req.Word, filter.Action(req.Action))
	}
	respond(w, err)
}

func (h handler) audit(w http.ResponseWriter, r *http.Request, _ auth.Identity) {
	list(w, func() ([]Action, error) { return h.svc.Actions(r.Context(), listLimit) })
}

// --- boundary helpers ---

type sanction struct {
	Minutes int    `json:"minutes"` // 0 = permanent (bans only)
	IP      bool   `json:"ip"`
	Reason  string `json:"reason"`
}

func durationRequest(w http.ResponseWriter, r *http.Request) (uint64, time.Duration, sanction, error) {
	id, err := pathID(r)
	if err != nil {
		return 0, 0, sanction{}, err
	}
	req, err := httpx.DecodeJSON[sanction](w, r)
	if err != nil {
		return 0, 0, sanction{}, err
	}
	d := time.Duration(req.Minutes) * time.Minute
	if req.Minutes < 0 || d > maxDuration || len(req.Reason) > 200 {
		return 0, 0, sanction{}, errBadDuration
	}
	return id, d, req, nil
}

func pathID(r *http.Request) (uint64, error) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errBadID
	}
	return id, nil
}

func respond(w http.ResponseWriter, err error) {
	if err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func list[T any](w http.ResponseWriter, fetch func() ([]T, error)) {
	items, err := fetch()
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if items == nil {
		items = []T{}
	}
	httpx.JSON(w, http.StatusOK, items)
}

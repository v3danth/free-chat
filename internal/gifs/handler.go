package gifs

import (
	"net/http"
	"time"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/ratelimit"
)

var errSearchRate = apperr.New(apperr.RateLimited, "too many searches, slow down a little")

// Routes serves the picker. Imports spend the same upload allowance as
// photos, so a GIF costs exactly what an upload costs.
func Routes(mux *http.ServeMux, lib *Library, images *media.Service, authSvc *auth.Service, ipOf httpx.IPResolver, maxUpload int64) {
	searches := ratelimit.New(ratelimit.Config{Rate: 60, Window: time.Minute})

	mux.Handle("GET /gifs", authSvc.RequireBearer(ipOf, func(w http.ResponseWriter, r *http.Request, id auth.Identity) {
		if !searches.Allow(id.UserID) {
			httpx.Error(w, errSearchRate)
			return
		}
		page, err := lib.Search(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("next"))
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, page)
	}))

	mux.Handle("POST /gifs/import", authSvc.RequireBearer(ipOf, func(w http.ResponseWriter, r *http.Request, id auth.Identity) {
		in, err := httpx.DecodeJSON[struct {
			ID string `json:"id"`
		}](w, r)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		if lib == nil {
			httpx.Error(w, ErrDisabled)
			return
		}
		if err := images.Admit(id.UserID); err != nil {
			httpx.Error(w, err)
			return
		}
		raw, err := lib.Fetch(r.Context(), in.ID, maxUpload)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		m, err := images.Upload(r.Context(), id.UserID, raw)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		httpx.JSON(w, http.StatusCreated, media.ViewOf(m))
	}))
}

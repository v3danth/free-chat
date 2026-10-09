package websocket

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/apperr"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/user"
)

// CheckOrigin is left nil: gorilla then rejects cross-origin browser
// handshakes, while non-browser clients (no Origin header) are allowed.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Profiles interface {
	GetByID(ctx context.Context, id uint64) (user.User, error)
	UpdateCard(ctx context.Context, id uint64, card user.Profile, photoID *uint64) error
}

func Routes(mux *http.ServeMux, hub *Hub, authSvc *auth.Service, profiles Profiles, ipOf httpx.IPResolver) {
	h := handler{hub: hub, auth: authSvc, profiles: profiles}
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) { h.serveWS(w, r, ipOf) })
	mux.Handle("GET /me", authSvc.RequireBearer(ipOf, h.me))
	mux.Handle("PATCH /me", authSvc.RequireBearer(ipOf, h.updateMe))
}

type handler struct {
	hub      *Hub
	auth     *auth.Service
	profiles Profiles
}

// serveWS authenticates before upgrading, so a bad token gets a normal HTTP
// error. Browsers cannot set headers on a WebSocket, hence ?token=.
func (h handler) serveWS(w http.ResponseWriter, r *http.Request, ipOf httpx.IPResolver) {
	u, err := h.auth.Resume(r.Context(), r.URL.Query().Get("token"), auth.Origin{IP: ipOf(r)})
	if err != nil {
		httpx.Error(w, err)
		return
	}
	blocked, err := h.hub.Blocks.Between(r.Context(), u.ID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade has already written the HTTP error.
	}
	c := newClient(u, blocked, conn)
	h.hub.register(c)

	go c.writePump()
	go c.readPump(h.hub)
}

func (h handler) me(w http.ResponseWriter, r *http.Request, id auth.Identity) {
	u, err := h.profiles.GetByID(r.Context(), id.UserID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user.ToSelf(u))
}

// cardUpdate is a partial update: an absent field keeps its value. PhotoID
// stays raw to tell "absent" (keep) from null (clear).
type cardUpdate struct {
	Tags     *[]string       `json:"tags"`
	Color    *string         `json:"color"`
	About    *string         `json:"about"`
	Location *string         `json:"location"`
	PhotoID  json.RawMessage `json:"photo_id"`
}

var (
	errPhotoID    = apperr.New(apperr.Invalid, "photo_id must be an image id or null")
	errGuestPhoto = apperr.New(apperr.Forbidden, "create an account to add a profile photo")
)

// apply merges the update into the current card, parsing each field it sets.
func (req cardUpdate) apply(u user.User) (user.Profile, *uint64, error) {
	p := u.Profile
	var err error
	if req.Tags != nil {
		if p.Tags, err = user.ParseTags(*req.Tags); err != nil {
			return p, nil, err
		}
	}
	if req.Color != nil {
		if p.Color, err = user.ParseColor(*req.Color); err != nil {
			return p, nil, err
		}
	}
	if req.About != nil {
		if p.About, err = user.ParseAbout(*req.About); err != nil {
			return p, nil, err
		}
	}
	if req.Location != nil {
		if p.Location, err = user.ParseLocation(*req.Location); err != nil {
			return p, nil, err
		}
	}

	photo := u.PhotoID
	if u.PhotoKey == nil {
		photo = nil // a photo a moderator removed is not kept
	}
	if len(req.PhotoID) > 0 {
		photo = nil
		if string(req.PhotoID) != "null" {
			var id uint64
			if json.Unmarshal(req.PhotoID, &id) != nil || id == 0 {
				return p, nil, errPhotoID
			}
			if u.Kind != user.KindMember {
				return p, nil, errGuestPhoto // guests are their face; photos are for members
			}
			photo = &id
		}
	}
	return p, photo, nil
}

// updateMe edits the card fields others see. Name, age and gender are fixed
// for the visit: changing them mid-chat would make impersonation trivial.
func (h handler) updateMe(w http.ResponseWriter, r *http.Request, id auth.Identity) {
	req, err := httpx.DecodeJSON[cardUpdate](w, r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	current, err := h.profiles.GetByID(r.Context(), id.UserID)
	if err != nil {
		httpx.Error(w, err)
		return
	}

	p, photo, err := req.apply(current)
	if err == nil {
		p, err = p.Screened(h.hub.Filter.Load().Screen)
	}
	if err != nil {
		httpx.Error(w, err)
		return
	}

	if err := h.profiles.UpdateCard(r.Context(), id.UserID, p, photo); err != nil {
		httpx.Error(w, err)
		return
	}
	fresh, err := h.profiles.GetByID(r.Context(), id.UserID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	h.hub.UpdateUser(fresh)
	httpx.JSON(w, http.StatusOK, user.ToSelf(fresh))
}

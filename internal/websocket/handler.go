package websocket

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/v3danth/free-chat/internal/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Restrict in production
	},
}

type Handler struct {
	hub     *Hub
	jwt     *auth.JWTManager
	authSvc *auth.Service
}

func NewHandler(
	hub *Hub,
	jwt *auth.JWTManager,
	authSvc *auth.Service,
) *Handler {
	return &Handler{
		hub:     hub,
		jwt:     jwt,
		authSvc: authSvc,
	}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	username := r.URL.Query().Get("username")

	var claims *auth.Claims
	var err error

	// Direct guest login: if no token but username provided, create guest directly
	if token == "" && username != "" {
		username = strings.TrimSpace(username)
		if username == "" {
			http.Error(w, "username is required", http.StatusBadRequest)
			return
		}

		if len(username) > 32 {
			http.Error(w, "username too long", http.StatusBadRequest)
			return
		}

		token, _, err = h.authSvc.CreateGuestUserDirect(r.Context(), username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
	}

	if token == "" {
		http.Error(w, "token or username is required", http.StatusUnauthorized)
		return
	}

	claims, err = h.jwt.Validate(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:      h.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   claims.UserID,
		username: claims.Username,
		userType: claims.UserType,
		rooms:    make(map[uint64]bool),
	}

	// Send the token back to guest users so they can reconnect
	if r.URL.Query().Get("username") != "" {
		tokenResp := struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}{
			Type:  "auth",
			Token: token,
		}
		if data, err := json.Marshal(tokenResp); err == nil {
			client.send <- data
		}
	}

	h.hub.register <- client

	go client.Write()
	go client.Read()
}

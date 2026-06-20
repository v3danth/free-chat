package auth

import (
	"encoding/json"
	"net/http"

	"github.com/v3danth/free-chat/internal/user"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Guest Signup Handler
type guestRequest struct {
	Username string      `json:"username"`
	Gender   user.Gender `json:"gender"`
	Age      uint8       `json:"age"`
	About    string      `json:"about"`
}

func (h *Handler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	var req guestRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	u, err := h.service.CreateGuestUser(
		r.Context(),
		req.Username,
		req.Gender,
		req.Age,
		req.About,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(u)
}

// Registered user Handler
type registerRequest struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Gender   user.Gender `json:"gender"`
	Age      uint8       `json:"age"`
	About    string      `json:"about"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	u, err := h.service.RegisterUser(
		r.Context(),
		req.Username,
		req.Email,
		req.Password,
		req.Gender,
		req.Age,
		req.About,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(u)
}

// Registered user login Handler
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

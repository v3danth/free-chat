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
	return &Handler{service: service}
}

type guestRequest struct {
	Username string      `json:"username"`
	Gender   user.Gender `json:"gender"`
	Age      uint8       `json:"age"`
	About    string      `json:"about"`
}

func (h *Handler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req guestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	token, u, err := h.service.CreateGuestUser(
		r.Context(),
		req.Username,
		req.Gender,
		req.Age,
		req.About,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(guestResponse{
		Token: token,
		User:  u,
	})
}

type guestResponse struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

type registerRequest struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Gender   user.Gender `json:"gender"`
	Age      uint8       `json:"age"`
	About    string      `json:"about"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "username, email, and password are required", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, u, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		Token: token,
		User:  u,
	})
}

type loginResponse struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

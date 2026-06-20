package main

import (
	"log"
	"net/http"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/config"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/user"
)

func main() {
	// 1. Load config
	cfg := config.Load()

	// 2. Connect DB
	db, err := database.New(cfg)
	if err != nil {
		log.Fatal("db connection failed:", err)
	}
	defer db.Close()

	// 3. Build repository
	userRepo := user.NewRepository(db)

	// 4. Build service
	authService := auth.NewService(userRepo)

	// 5. Build handler
	authHandler := auth.NewHandler(authService)

	// 6. Register routes
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/guest", authHandler.CreateGuest)
	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)

	// 7. Start server
	addr := ":" + cfg.ServerPort

	log.Println("server running on", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

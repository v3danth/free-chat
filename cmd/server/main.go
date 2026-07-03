package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/config"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/guest"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/websocket"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// 3. Build repositories
	userRepo := user.NewRepository(db)
	msgRepo := message.NewRepository(db)
	mediaRepo := media.NewRepository(db)

	// 4. Build storage
	storage, err := media.NewLocalStorage(media.StorageConfig{
		BasePath: cfg.MediaStoragePath,
		BaseURL:  "/media",
	})
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}

	// 5. Build services
	authService := auth.NewService(userRepo, cfg.JWTSecret)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)

	mediaService := media.NewService(mediaRepo, storage, media.UploadConfig{
		MaxImageSize:  cfg.MediaMaxImageSize,
		MaxGIFSize:    cfg.MediaMaxGIFSize,
		MaxVoiceSize:  cfg.MediaMaxVoiceSize,
		FlagThreshold: cfg.MediaFlagThreshold,
	})

	// 6. Parse banned words
	bannedWords := parseBannedWords(cfg.FilterBannedWords)

	// 7. Build WebSocket hub
	hub := websocket.NewHub(
		userRepo,
		msgRepo,
		mediaRepo,
		storage,
		cfg.MessageHistoryLimit,
		ratelimit.Config{
			Rate:   cfg.MessageRateLimit,
			Window: cfg.MessageRateWindow,
		},
		filter.Config{
			MaxMessageLength: cfg.MessageMaxLength,
			BannedWords:      bannedWords,
		},
	)
	go hub.Run()

	// 8. Start guest cleanup goroutine
	guestCleaner := guest.NewCleaner(userRepo, guest.CleanerConfig{
		SweepInterval:   cfg.GuestSweepInterval,
		MaxInactiveTime: cfg.GuestMaxInactiveTime,
	})
	go guestCleaner.Start()
	defer guestCleaner.Stop()

	// 9. Build handlers
	authHandler := auth.NewHandler(authService)
	wsHandler := websocket.NewHandler(hub, jwtManager, authService)
	mediaHandler := media.NewHandler(mediaService, jwtManager, storage)

	// 10. Register routes
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("./web")))
	// WebSocket
	mux.HandleFunc("/ws", wsHandler.ServeWS)

	// Auth
	mux.HandleFunc("/auth/guest", authHandler.CreateGuest)
	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)

	// Media upload
	mux.HandleFunc("/media/upload/image", mediaHandler.UploadImage)
	mux.HandleFunc("/media/upload/gif", mediaHandler.UploadGIF)
	mux.HandleFunc("/media/upload/voice", mediaHandler.UploadVoice)
	mux.HandleFunc("/media/flag", mediaHandler.FlagMedia)

	// Media serving (static files)
	mux.Handle("/media/files/", http.StripPrefix("/media/files/", http.FileServer(http.Dir(cfg.MediaStoragePath))))

	// 11. Start server with graceful shutdown
	addr := ":" + cfg.ServerPort
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Shutdown channel
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	go func() {
		log.Printf("server running on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdownCh
	log.Printf("received signal %v, shutting down...", sig)

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Println("server stopped")
}

func parseBannedWords(commaSeparated string) []string {
	if commaSeparated == "" {
		return nil
	}

	words := strings.Split(commaSeparated, ",")
	result := make([]string, 0, len(words))

	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" {
			result = append(result, w)
		}
	}

	return result
}

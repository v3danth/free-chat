package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/v3danth/free-chat/internal/admin"
	"github.com/v3danth/free-chat/internal/auth"
	"github.com/v3danth/free-chat/internal/avatar"
	"github.com/v3danth/free-chat/internal/block"
	"github.com/v3danth/free-chat/internal/config"
	"github.com/v3danth/free-chat/internal/database"
	"github.com/v3danth/free-chat/internal/filter"
	"github.com/v3danth/free-chat/internal/geo"
	"github.com/v3danth/free-chat/internal/httpx"
	"github.com/v3danth/free-chat/internal/id"
	"github.com/v3danth/free-chat/internal/janitor"
	"github.com/v3danth/free-chat/internal/media"
	"github.com/v3danth/free-chat/internal/message"
	"github.com/v3danth/free-chat/internal/moderation"
	"github.com/v3danth/free-chat/internal/ratelimit"
	"github.com/v3danth/free-chat/internal/user"
	"github.com/v3danth/free-chat/internal/websocket"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run is the composition root: the only place that knows every package and
// the only place with process-level side effects.
// schema lists columns added after the first migration, with the file that
// adds each, so an out-of-date database is caught at startup.
var schema = []database.Requirement{
	{Table: "users", Column: "tags", Migration: "migrations/002_tags.sql"},
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Open(ctx, database.Config{
		Host: cfg.MySQLHost, Port: cfg.MySQLPort, User: cfg.MySQLUser,
		Password: cfg.MySQLPassword, Name: cfg.MySQLDatabase,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.RequireColumns(ctx, db, schema...); err != nil {
		return err
	}

	locator, closeGeo, err := openGeo(cfg.GeoIPPath)
	if err != nil {
		return err
	}
	defer closeGeo()

	// Repositories.
	users := user.NewRepository(db)
	messages := message.NewRepository(db)
	modStore := moderation.NewRepository(db)
	blocks := block.NewRepository(db)

	storage, err := media.NewStorage(cfg.MediaStoragePath)
	if err != nil {
		return err
	}
	mediaSvc := media.NewService(media.NewRepository(db), storage)

	words := filter.NewLive(filter.New(cfg.MessageMaxLength, nil))
	authSvc := auth.NewService(users, auth.NewTokens(cfg.JWTSecret), modStore, locator, words, cfg.JWTSecret)

	// The message log is written in the background; it gets its own context
	// so it can drain after the HTTP server and sockets are gone.
	writer := message.NewWriter(messages, 10_000)
	writerCtx, stopWriter := context.WithCancel(context.Background())
	go writer.Run(writerCtx)
	defer func() {
		stopWriter()
		<-writer.Done()
	}()

	hub, err := newHub(ctx, cfg, messages, writer, mediaSvc, users, blocks, words)
	if err != nil {
		return err
	}

	modSvc := moderation.NewService(moderation.Deps{
		Store: modStore, Users: users, Messages: messages, Writer: writer, Images: mediaSvc,
		Live: hub, Filter: words, MaxLength: cfg.MessageMaxLength, AutoHide: cfg.ReportAutoHideThreshold,
	})
	if err := modSvc.ReloadWords(ctx); err != nil {
		return fmt.Errorf("load banned words: %w", err)
	}

	if cfg.AdminEmail != "" {
		if err := authSvc.EnsureAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword, cfg.AdminName); err != nil {
			return fmt.Errorf("bootstrap admin: %w", err)
		}
	}

	ipOf := httpx.NewIPResolver(cfg.BehindProxy)
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("GET /faces", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./web/faces.html") })
	avatar.Routes(mux, cfg.AvatarStyle)
	auth.Routes(mux, authSvc, ipOf)
	media.Routes(mux, mediaSvc, authSvc, ipOf, cfg.MediaMaxUploadSize)
	websocket.Routes(mux, hub, authSvc, users, ipOf)
	moderation.Routes(mux, modSvc, authSvc, ipOf)
	admin.Routes(mux, admin.NewService(db, hub, users, authSvc, modStore), authSvc, ipOf)

	go janitor.Run(ctx, janitor.NewMySQLStore(db), mediaSvc, hub, janitor.Config{
		Retention: cfg.Retention, Interval: cfg.JanitorInterval,
	})

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second, // large uploads on slow links
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	server.RegisterOnShutdown(hub.Close) // hijacked WebSockets are not closed by Shutdown

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("server running on %s", server.Addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Println("shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Println("server stopped")
	return nil
}

// newHub loads the live rooms and their recent messages once, so joining a
// room is served from memory.
func newHub(ctx context.Context, cfg config.Config, messages *message.MySQLRepository, writer *message.Writer,
	mediaSvc *media.Service, users *user.MySQLRepository, blocks *block.MySQLRepository, words *filter.Live,
) (*websocket.Hub, error) {
	rooms, err := messages.LiveRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("load rooms: %w", err)
	}
	recent := make(map[uint64][]message.Entry, len(rooms))
	for _, r := range rooms {
		if recent[r.ID], err = messages.Recent(ctx, r.ID, cfg.MessageHistoryLimit); err != nil {
			return nil, fmt.Errorf("load history for room %d: %w", r.ID, err)
		}
	}
	return websocket.NewHub(websocket.Deps{
		Writer: writer, IDs: &id.Generator{}, Media: mediaSvc, Users: users, Blocks: blocks,
		Limiter:      ratelimit.New(ratelimit.Config{Rate: cfg.MessageRateLimit, Window: cfg.MessageRateWindow}),
		Filter:       words,
		HistoryLimit: cfg.MessageHistoryLimit,
	}, rooms, recent), nil
}

func openGeo(path string) (geo.Locator, func(), error) {
	if path == "" {
		log.Println("geo: GEOIP_DB_PATH not set, country flags disabled")
		return geo.None{}, func() {}, nil
	}
	db, err := geo.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open geoip database: %w", err)
	}
	return db, func() { db.Close() }, nil
}

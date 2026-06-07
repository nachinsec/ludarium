package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nachinsec/ludarium/internal/ai"
	"github.com/nachinsec/ludarium/internal/api"
	"github.com/nachinsec/ludarium/internal/auth"
	"github.com/nachinsec/ludarium/internal/config"
	"github.com/nachinsec/ludarium/internal/db"
	"github.com/nachinsec/ludarium/internal/igdb"
	"github.com/nachinsec/ludarium/internal/psn"
	"github.com/nachinsec/ludarium/internal/secret"
	"github.com/nachinsec/ludarium/internal/syncer"
	"github.com/nachinsec/ludarium/internal/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	sessions := auth.NewSessionManager(store, cfg)
	steam := auth.NewSteam(cfg)
	igdbClient := igdb.NewClient(cfg.IGDBClientID, cfg.IGDBSecret)
	aiClient := ai.NewClient(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
	cipher, err := secret.New(cfg.SessionSecret)
	if err != nil {
		slog.Error("cipher", "err", err)
		os.Exit(1)
	}
	psnClient := psn.NewClient()
	sync := syncer.New(store, steam, igdbClient, psnClient, cipher)

	router := api.NewRouter(api.Deps{
		Config:   cfg,
		Store:    store,
		Sessions: sessions,
		Steam:    steam,
		Syncer:   sync,
		AI:       aiClient,
		IGDB:     igdbClient,
		Cipher:   cipher,
		Frontend: web.Handler(),
	})

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.Addr(), "base_url", cfg.BaseURL)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

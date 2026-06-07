package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/nachinsec/ludarium/internal/ai"
	"github.com/nachinsec/ludarium/internal/auth"
	"github.com/nachinsec/ludarium/internal/config"
	"github.com/nachinsec/ludarium/internal/db"
	"github.com/nachinsec/ludarium/internal/igdb"
	"github.com/nachinsec/ludarium/internal/secret"
	"github.com/nachinsec/ludarium/internal/syncer"
)

// Deps are the dependencies the HTTP layer needs.
type Deps struct {
	Config   *config.Config
	Store    *db.Store
	Sessions *auth.SessionManager
	Steam    *auth.Steam
	Syncer   *syncer.Syncer
	AI       *ai.Client
	IGDB     *igdb.Client
	Cipher   *secret.Cipher
	Frontend http.Handler
}

// NewRouter builds the application's HTTP handler.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(d.Sessions.Middleware)

	h := &handlers{d: d}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.health)

		r.Route("/auth", func(r chi.Router) {
			// Public
			r.Post("/register", h.register)
			r.Post("/login", h.login)
			r.Post("/logout", h.logout)
			r.Get("/me", h.me)
			r.Get("/steam/login", h.steamLogin)
			r.Get("/steam/callback", h.steamCallback)

			// Authenticated
			r.Group(func(r chi.Router) {
				r.Use(d.Sessions.RequireUser)
				r.Patch("/profile", h.updateProfile)
				r.Delete("/connections/steam", h.disconnectSteam)
				r.Post("/connections/psn", h.connectPSN)
				r.Delete("/connections/psn", h.disconnectPSN)
				r.Get("/ai", h.getAISettings)
				r.Patch("/ai", h.setAISettings)
			})
		})

		// Library + sync (all require an authenticated user).
		r.Group(func(r chi.Router) {
			r.Use(d.Sessions.RequireUser)
			r.Get("/library", h.getLibrary)
			r.Patch("/library/{id}", h.updateLibraryEntry)
			r.Get("/stats", h.getStats)
			r.Post("/recommend", h.recommend)
			r.Post("/sync/steam", h.syncSteam)
			r.Post("/sync/psn", h.syncPSN)
			r.Post("/enrich", h.enrich)
			r.Get("/igdb/search", h.igdbSearch)
			r.Get("/igdb/popular", h.igdbPopular)
			r.Get("/igdb/upcoming", h.igdbUpcoming)
			r.Get("/oracle/candidates", h.oracleCandidates)
			r.Post("/library/add", h.addGame)
		})
	})

	// Everything else is served by the embedded frontend (SPA fallback).
	r.NotFound(d.Frontend.ServeHTTP)

	return r
}

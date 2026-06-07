package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/nachinsec/ludarium/internal/ai"
	"github.com/nachinsec/ludarium/internal/auth"
	"github.com/nachinsec/ludarium/internal/db"
	"github.com/nachinsec/ludarium/internal/igdb"
)

type handlers struct {
	d Deps
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// me returns the authenticated user and their linked connections, or null.
func (h *handlers) me(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	conns, _ := h.d.Store.ListConnections(r.Context(), user.ID)
	if conns == nil {
		conns = []db.Connection{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "connections": conns})
}

// --- email/password ------------------------------------------------------

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handlers) register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decode(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if len(req.Username) < 3 || len(req.Username) > 20 {
		writeError(w, http.StatusBadRequest, "username must be 3-20 characters")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}

	user, err := h.d.Store.CreateUserWithPassword(r.Context(), req.Username, req.Email, hash)
	switch {
	case errors.Is(err, db.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "username already taken")
		return
	case errors.Is(err, db.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already registered")
		return
	case err != nil:
		slog.Error("register", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	if err := h.d.Sessions.Issue(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

type loginReq struct {
	Login    string `json:"login"` // username or email
	Password string `json:"password"`
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decode(w, r, &req) {
		return
	}
	req.Login = strings.TrimSpace(strings.ToLower(req.Login))

	user, hash, err := h.d.Store.CredentialsByLogin(r.Context(), req.Login)
	if err != nil || hash == "" || !auth.CheckPassword(hash, req.Password) {
		// Same message whether the user exists or not (avoid user enumeration).
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := h.d.Sessions.Issue(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) {
	h.d.Sessions.Destroy(w, r)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- profile -------------------------------------------------------------

type profileReq struct {
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
}

func (h *handlers) updateProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var req profileReq
	if !decode(w, r, &req) {
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" || len(req.DisplayName) > 40 {
		writeError(w, http.StatusBadRequest, "display name must be 1-40 characters")
		return
	}
	if err := h.d.Store.UpdateProfile(r.Context(), user.ID, req.DisplayName, req.AvatarURL); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	updated, _ := h.d.Store.GetUserByID(r.Context(), user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"user": updated})
}

// --- steam ---------------------------------------------------------------

func (h *handlers) steamLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.d.Steam.AuthURL(), http.StatusFound)
}

// steamCallback links, logs in, or creates an account depending on state.
func (h *handlers) steamCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	steamID, err := h.d.Steam.Verify(r.Context(), r.Form)
	if err != nil {
		slog.Warn("steam verify failed", "err", err)
		http.Redirect(w, r, "/?auth=failed", http.StatusFound)
		return
	}

	// Case 1: already logged in → link Steam to this account.
	if current := auth.UserFromContext(r.Context()); current != nil {
		if err := h.d.Store.LinkSteam(r.Context(), current.ID, steamID); err != nil {
			slog.Warn("link steam", "err", err)
			http.Redirect(w, r, "/settings?steam=linkfailed", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/settings?steam=linked", http.StatusFound)
		return
	}

	// Case 2: a user already linked this Steam account → log them in.
	if user, err := h.d.Store.UserBySteamID(r.Context(), steamID); err == nil {
		_ = h.d.Sessions.Issue(r.Context(), w, user.ID)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Case 3: brand-new Steam login → create an account (profile data if key set).
	name, avatar := "", ""
	if p, err := h.d.Steam.FetchProfile(r.Context(), steamID); err == nil {
		name, avatar = p.Name, p.Avatar
	}
	user, err := h.d.Store.CreateUserFromSteam(r.Context(), generateUsername(), steamID, name, avatar)
	if err != nil {
		slog.Error("create steam user", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = h.d.Sessions.Issue(r.Context(), w, user.ID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *handlers) disconnectSteam(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	// Don't let a steam-only account orphan itself out of any login method.
	if !user.HasPassword {
		writeError(w, http.StatusBadRequest, "set a password before disconnecting Steam")
		return
	}
	if err := h.d.Store.DeleteConnection(r.Context(), user.ID, "steam"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not disconnect")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type connectPSNReq struct {
	NPSSO string `json:"npsso"`
}

func (h *handlers) connectPSN(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var req connectPSNReq
	if !decode(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.NPSSO) == "" {
		writeError(w, http.StatusBadRequest, "paste your NPSSO token")
		return
	}
	if err := h.d.Syncer.ConnectPSN(r.Context(), user.ID, strings.TrimSpace(req.NPSSO)); err != nil {
		slog.Warn("connect psn", "err", err)
		writeError(w, http.StatusBadGateway, "could not connect PlayStation — is the NPSSO valid and fresh?")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) disconnectPSN(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if err := h.d.Store.DeleteConnection(r.Context(), user.ID, "psn"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not disconnect")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) syncPSN(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	sum, err := h.d.Syncer.SyncPSN(r.Context(), user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "connect your PlayStation account first")
		return
	}
	if err != nil {
		slog.Warn("psn sync", "err", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// --- library + sync ------------------------------------------------------

func (h *handlers) getLibrary(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	items, err := h.d.Store.ListLibrary(r.Context(), user.ID)
	if err != nil {
		slog.Error("list library", "err", err)
		writeError(w, http.StatusInternalServerError, "could not load library")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": items})
}

func (h *handlers) getLibraryGame(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.d.Store.GetLibraryItem(r.Context(), user.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "game not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load game")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"game": item})
}

// aiClientFor returns the user's own AI client if configured, else the server default.
func (h *handlers) aiClientFor(ctx context.Context, userID int64) *ai.Client {
	s, err := h.d.Store.GetUserAISettings(ctx, userID)
	if err == nil && s.BaseURL != "" && s.APIKeyEnc != "" {
		if key, derr := h.d.Cipher.Decrypt(s.APIKeyEnc); derr == nil {
			model := s.Model
			if model == "" {
				model = h.d.Config.AIModel
			}
			return ai.NewClient(s.BaseURL, key, model)
		}
	}
	return h.d.AI
}

func (h *handlers) getAISettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	s, err := h.d.Store.GetUserAISettings(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load AI settings")
		return
	}
	// The key itself is never returned — only whether one is set.
	writeJSON(w, http.StatusOK, map[string]any{
		"baseUrl": s.BaseURL, "model": s.Model, "hasKey": s.APIKeyEnc != "",
	})
}

type aiSettingsReq struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	APIKey  string `json:"apiKey"`
}

func (h *handlers) setAISettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var req aiSettingsReq
	if !decode(w, r, &req) {
		return
	}
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")

	// Empty base URL clears the override (back to the server default).
	if req.BaseURL == "" {
		if err := h.d.Store.SetUserAISettings(r.Context(), user.ID, "", "", ""); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	// Keep the existing key when the field is left blank.
	cur, _ := h.d.Store.GetUserAISettings(r.Context(), user.ID)
	enc := cur.APIKeyEnc
	if req.APIKey != "" {
		e, err := h.d.Cipher.Encrypt(req.APIKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not encrypt key")
			return
		}
		enc = e
	}
	if err := h.d.Store.SetUserAISettings(r.Context(), user.ID, req.BaseURL, req.Model, enc); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) recommend(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	client := h.aiClientFor(r.Context(), user.ID)
	if !client.Configured() {
		writeError(w, http.StatusBadRequest, "no AI configured — set yours in Settings, or ask the admin")
		return
	}

	// Optional: games the user picked in the Oracle quiz (their current mood).
	var req struct {
		Preferred []string `json:"preferred"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	items, err := h.d.Store.ListLibrary(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load library")
		return
	}

	games := make([]ai.GameInfo, 0, len(items))
	owned := make([]string, 0, len(items))
	for _, it := range items {
		rating := 0
		if it.Rating != nil {
			rating = *it.Rating
		}
		games = append(games, ai.GameInfo{Title: it.Title, Status: it.Status, Rating: rating, Hours: it.Hours})
		owned = append(owned, it.Title)
	}

	taste := ai.Taste(games)
	if len(req.Preferred) > 0 {
		taste = taste[:0]
		for _, t := range req.Preferred {
			taste = append(taste, ai.GameInfo{Title: t})
		}
	}

	recs, err := ai.RecommendNew(r.Context(), client, taste, owned)
	if err != nil {
		slog.Warn("recommend", "err", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"recommendations": h.groundRecommendations(r, user.ID, recs)})
}

type recCard struct {
	Title       string   `json:"title"`
	Reason      string   `json:"reason"`
	CoverURL    string   `json:"coverUrl"`
	IGDBID      int      `json:"igdbId"`
	ReleaseYear *int     `json:"releaseYear"`
	Genres      []string `json:"genres"`
}

// groundRecommendations resolves each LLM title against IGDB to attach a real
// cover/metadata and drops anything the user already owns.
func (h *handlers) groundRecommendations(r *http.Request, userID int64, recs []ai.Recommendation) []recCard {
	out := make([]recCard, 0, len(recs))
	if !h.d.IGDB.Configured() {
		for _, rec := range recs {
			out = append(out, recCard{Title: rec.Title, Reason: rec.Reason})
		}
		return out
	}
	ownedIDs, _ := h.d.Store.OwnedIGDBIDs(r.Context(), userID)
	for _, rec := range recs {
		card := recCard{Title: rec.Title, Reason: rec.Reason}
		if matches, err := h.d.IGDB.Search(r.Context(), rec.Title); err == nil && len(matches) > 0 {
			g := matches[0]
			if ownedIDs[g.IGDBID] {
				continue
			}
			card.Title, card.CoverURL, card.IGDBID = g.Name, g.CoverURL, g.IGDBID
			card.ReleaseYear, card.Genres = g.ReleaseYear, g.Genres
		}
		out = append(out, card)
		time.Sleep(260 * time.Millisecond)
	}
	return out
}

func (h *handlers) igdbSearch(w http.ResponseWriter, r *http.Request) {
	if !h.d.IGDB.Configured() {
		writeError(w, http.StatusBadRequest, "IGDB is not configured on this server")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{"games": []any{}})
		return
	}
	games, err := h.d.IGDB.Search(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusBadGateway, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": games})
}

func (h *handlers) igdbPopular(w http.ResponseWriter, r *http.Request) {
	h.igdbList(w, r, h.d.IGDB.Popular)
}

func (h *handlers) oracleCandidates(w http.ResponseWriter, r *http.Request) {
	h.igdbList(w, r, h.d.IGDB.Candidates)
}

func (h *handlers) igdbUpcoming(w http.ResponseWriter, r *http.Request) {
	h.igdbList(w, r, h.d.IGDB.Upcoming)
}

func (h *handlers) igdbList(w http.ResponseWriter, r *http.Request, fn func(context.Context) ([]igdb.Game, error)) {
	if !h.d.IGDB.Configured() {
		writeError(w, http.StatusBadRequest, "IGDB is not configured on this server")
		return
	}
	games, err := fn(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "IGDB request failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": games})
}

type addGameReq struct {
	IGDBID   int    `json:"igdbId"`
	Status   string `json:"status"`
	Platform string `json:"platform"`
}

func (h *handlers) addGame(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.d.IGDB.Configured() {
		writeError(w, http.StatusBadRequest, "IGDB is not configured on this server")
		return
	}
	var req addGameReq
	if !decode(w, r, &req) {
		return
	}
	status := req.Status
	if status == "" {
		status = "backlog"
	}
	if !validStatus[status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	g, err := h.d.IGDB.GameByID(r.Context(), req.IGDBID)
	if err != nil || g == nil {
		writeError(w, http.StatusBadRequest, "game not found on IGDB")
		return
	}
	genres, _ := json.Marshal(g.Genres)
	details := db.MarshalGameDetails(g.Summary, g.Screenshots, g.Score)
	gameID, err := h.d.Store.UpsertGameByIGDBID(r.Context(), g.IGDBID, g.Name, g.CoverURL, g.ReleaseYear, g.Developer, string(genres), details)
	if err != nil {
		slog.Error("add game upsert", "err", err)
		writeError(w, http.StatusInternalServerError, "could not add game")
		return
	}
	if err := h.d.Store.AddLibraryEntry(r.Context(), user.ID, gameID, status, req.Platform); err != nil {
		writeError(w, http.StatusInternalServerError, "could not add to library")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) enrich(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !h.d.IGDB.Configured() {
		writeError(w, http.StatusBadRequest, "IGDB is not configured on this server (set IGDB_CLIENT_ID/SECRET)")
		return
	}
	sum, err := h.d.Syncer.EnrichSteam(r.Context(), user.ID)
	if err != nil {
		slog.Error("enrich", "err", err)
		writeError(w, http.StatusBadGateway, "enrichment failed")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (h *handlers) getStats(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	stats, err := h.d.Store.Stats(r.Context(), user.ID)
	if err != nil {
		slog.Error("stats", "err", err)
		writeError(w, http.StatusInternalServerError, "could not load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

var validStatus = map[string]bool{
	"backlog": true, "playing": true, "ongoing": true, "completed": true, "dropped": true, "wishlist": true,
}

type updateEntryReq struct {
	Status string `json:"status"`
	Rating *int   `json:"rating"`
	Notes  string `json:"notes"`
}

func (h *handlers) updateLibraryEntry(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateEntryReq
	if !decode(w, r, &req) {
		return
	}
	if !validStatus[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		writeError(w, http.StatusBadRequest, "rating must be 1-5")
		return
	}

	err = h.d.Store.UpdateLibraryEntry(r.Context(), user.ID, id, req.Status, req.Rating, req.Notes)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "game not found")
		return
	}
	if err != nil {
		slog.Error("update entry", "err", err)
		writeError(w, http.StatusInternalServerError, "could not update game")
		return
	}

	item, _ := h.d.Store.GetLibraryItem(r.Context(), user.ID, id)
	writeJSON(w, http.StatusOK, map[string]any{"game": item})
}

func (h *handlers) syncSteam(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())

	steamID, err := h.d.Store.SteamIDForUser(r.Context(), user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "connect your Steam account first")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Steam connection")
		return
	}

	summary, err := h.d.Syncer.SyncSteam(r.Context(), user.ID, steamID)
	if err != nil {
		slog.Warn("steam sync", "err", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// --- helpers -------------------------------------------------------------

func generateUsername() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return "player_" + hex.EncodeToString(b)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

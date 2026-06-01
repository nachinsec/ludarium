package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrUsernameTaken = errors.New("username already taken")
	ErrEmailTaken    = errors.New("email already registered")
)

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) firstUserRole(ctx context.Context) (string, error) {
	n, err := s.CountUsers(ctx)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "instance_admin", nil
	}
	return "user", nil
}

// CreateUserWithPassword registers a new email/password account.
func (s *Store) CreateUserWithPassword(ctx context.Context, username, email, passwordHash string) (*User, error) {
	role, err := s.firstUserRole(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (username, display_name, email, password_hash, role)
		VALUES (?, ?, ?, ?, ?)`,
		username, username, nullable(email), passwordHash, role)
	if err != nil {
		return nil, mapUserConflict(err)
	}
	id, _ := res.LastInsertId()
	return s.GetUserByID(ctx, id)
}

// CreateUserFromSteam creates a passwordless account plus its Steam connection.
func (s *Store) CreateUserFromSteam(ctx context.Context, username, steamID, displayName, avatar string) (*User, error) {
	role, err := s.firstUserRole(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (username, display_name, avatar_url, role)
		VALUES (?, ?, ?, ?)`,
		username, fallback(displayName, username), avatar, role)
	if err != nil {
		return nil, mapUserConflict(err)
	}
	id, _ := res.LastInsertId()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO connections (user_id, provider, external_id) VALUES (?, 'steam', ?)`,
		id, steamID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, id)
}

// CredentialsByLogin loads a user and their password hash by email or username.
func (s *Store) CredentialsByLogin(ctx context.Context, login string) (*User, string, error) {
	var u User
	var email sql.NullString
	var hash string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, avatar_url, role, created_at, password_hash
		FROM users WHERE username = ? OR email = ?`, login, login).
		Scan(&u.ID, &u.Username, &u.DisplayName, &email, &u.AvatarURL, &u.Role, &u.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	u.Email = email.String
	u.HasPassword = hash != ""
	return &u, hash, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, display_name, email, avatar_url, role, created_at,
		       (password_hash != '') AS has_pw
		FROM users WHERE id = ?`, id))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	var email sql.NullString
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &email, &u.AvatarURL, &u.Role, &u.CreatedAt, &u.HasPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Email = email.String
	return &u, nil
}

func (s *Store) UpdateProfile(ctx context.Context, userID int64, displayName, avatar string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET display_name = ?, avatar_url = ?, updated_at = datetime('now')
		WHERE id = ?`, displayName, avatar, userID)
	return err
}

// --- Connections ---------------------------------------------------------

func (s *Store) UserBySteamID(ctx context.Context, steamID string) (*User, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id FROM connections WHERE provider = 'steam' AND external_id = ?`,
		steamID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, id)
}

func (s *Store) LinkSteam(ctx context.Context, userID int64, steamID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO connections (user_id, provider, external_id) VALUES (?, 'steam', ?)`,
		userID, steamID)
	return err
}

func (s *Store) ListConnections(ctx context.Context, userID int64) ([]Connection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, provider, external_id, created_at FROM connections WHERE user_id = ?`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.Provider, &c.ExternalID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) DeleteConnection(ctx context.Context, userID int64, provider string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM connections WHERE user_id = ? AND provider = ?`, userID, provider)
	return err
}

// SteamIDForUser returns the Steam ID linked to a user, or ErrNotFound.
func (s *Store) SteamIDForUser(ctx context.Context, userID int64) (string, error) {
	var steamID string
	err := s.db.QueryRowContext(ctx, `
		SELECT external_id FROM connections
		WHERE user_id = ? AND provider = 'steam'`, userID).Scan(&steamID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return steamID, err
}

// --- Games & library -----------------------------------------------------

// UpsertGameBySteamAppID inserts or refreshes the shared metadata cache.
func (s *Store) UpsertGameBySteamAppID(ctx context.Context, appID int, title, coverURL string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO games (steam_appid, title, cover_url) VALUES (?, ?, ?)
		ON CONFLICT(steam_appid) DO UPDATE SET
			title      = excluded.title,
			cover_url  = excluded.cover_url,
			updated_at = datetime('now')
		RETURNING id`, appID, title, coverURL).Scan(&id)
	return id, err
}

// UpsertLibraryEntry refreshes playtime only; status/rating/notes are left intact.
func (s *Store) UpsertLibraryEntry(ctx context.Context, userID, gameID int64, hours float64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO library_entries (user_id, game_id, hours, platform)
		VALUES (?, ?, ?, 'PC / Steam')
		ON CONFLICT(user_id, game_id) DO UPDATE SET
			hours      = excluded.hours,
			updated_at = datetime('now')`,
		userID, gameID, hours)
	return err
}

func (s *Store) CountLibrary(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM library_entries WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// SteamGameRef identifies a game pending IGDB enrichment.
type SteamGameRef struct {
	ID         int64
	SteamAppID int
}

// ListUnenrichedSteamGames returns the user's Steam games not yet enriched.
func (s *Store) ListUnenrichedSteamGames(ctx context.Context, userID int64) ([]SteamGameRef, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT g.id, g.steam_appid
		FROM games g JOIN library_entries le ON le.game_id = g.id
		WHERE le.user_id = ? AND g.steam_appid IS NOT NULL AND g.enriched_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SteamGameRef
	for rows.Next() {
		var r SteamGameRef
		if err := rows.Scan(&r.ID, &r.SteamAppID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateGameMetadata fills IGDB metadata and marks the game enriched. If the
// igdb_id clashes (two Steam entries → same IGDB game), it retries without it.
func (s *Store) UpdateGameMetadata(ctx context.Context, gameID int64, igdbID int, cover string, year *int, developer, genresJSON string) error {
	const withID = `UPDATE games SET igdb_id = ?,
		cover_url = CASE WHEN ? <> '' THEN ? ELSE cover_url END,
		release_year = ?, developer = ?, genres = ?,
		enriched_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`
	_, err := s.db.ExecContext(ctx, withID, igdbID, cover, cover, ratingArg(year), developer, genresJSON, gameID)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: games.igdb_id") {
		_, err = s.db.ExecContext(ctx, `UPDATE games SET
			cover_url = CASE WHEN ? <> '' THEN ? ELSE cover_url END,
			release_year = ?, developer = ?, genres = ?,
			enriched_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`,
			cover, cover, ratingArg(year), developer, genresJSON, gameID)
	}
	return err
}

// AISettings holds a user's own AI provider config. APIKeyEnc is encrypted at
// rest and never leaves the server.
type AISettings struct {
	BaseURL   string
	Model     string
	APIKeyEnc string
}

func (s *Store) GetUserAISettings(ctx context.Context, userID int64) (AISettings, error) {
	var a AISettings
	err := s.db.QueryRowContext(ctx,
		`SELECT ai_base_url, ai_model, ai_api_key FROM users WHERE id = ?`, userID).
		Scan(&a.BaseURL, &a.Model, &a.APIKeyEnc)
	return a, err
}

func (s *Store) SetUserAISettings(ctx context.Context, userID int64, baseURL, model, apiKeyEnc string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET ai_base_url = ?, ai_model = ?, ai_api_key = ?, updated_at = datetime('now')
		WHERE id = ?`, baseURL, model, apiKeyEnc, userID)
	return err
}

// MarkGameEnriched records that we tried (e.g. no IGDB match) so we skip it next time.
func (s *Store) MarkGameEnriched(ctx context.Context, gameID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE games SET enriched_at = datetime('now') WHERE id = ?`, gameID)
	return err
}

// OwnedIGDBIDs returns the set of IGDB ids already in the user's library.
func (s *Store) OwnedIGDBIDs(ctx context.Context, userID int64) (map[int]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.igdb_id FROM games g JOIN library_entries le ON le.game_id = g.id
		WHERE le.user_id = ? AND g.igdb_id IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int]bool)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// UpsertGameByIGDBID inserts or refreshes a game from IGDB metadata (no Steam appid).
func (s *Store) UpsertGameByIGDBID(ctx context.Context, igdbID int, title, cover string, year *int, developer, genresJSON string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO games (igdb_id, title, cover_url, release_year, developer, genres, enriched_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(igdb_id) DO UPDATE SET
			title = excluded.title, cover_url = excluded.cover_url,
			release_year = excluded.release_year, developer = excluded.developer,
			genres = excluded.genres, updated_at = datetime('now')
		RETURNING id`, igdbID, title, cover, ratingArg(year), developer, genresJSON).Scan(&id)
	return id, err
}

// AddLibraryEntry adds a game to a user's library, leaving an existing entry untouched.
func (s *Store) AddLibraryEntry(ctx context.Context, userID, gameID int64, status string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO library_entries (user_id, game_id, status) VALUES (?, ?, ?)
		ON CONFLICT(user_id, game_id) DO NOTHING`, userID, gameID, status)
	return err
}

const libraryCols = `le.id, g.title, g.cover_url, le.status, le.rating, le.hours,
	le.platform, g.developer, g.release_year, le.notes`

func scanLibraryItem(sc interface{ Scan(...any) error }) (LibraryItem, error) {
	var it LibraryItem
	var rating, releaseYear sql.NullInt64
	err := sc.Scan(&it.ID, &it.Title, &it.CoverURL, &it.Status, &rating,
		&it.Hours, &it.Platform, &it.Developer, &releaseYear, &it.Notes)
	if err != nil {
		return it, err
	}
	if rating.Valid {
		v := int(rating.Int64)
		it.Rating = &v
	}
	if releaseYear.Valid {
		v := int(releaseYear.Int64)
		it.ReleaseYear = &v
	}
	return it, nil
}

// ListLibrary joins game metadata with the user's entries, most-played first.
func (s *Store) ListLibrary(ctx context.Context, userID int64) ([]LibraryItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+libraryCols+`
		FROM library_entries le
		JOIN games g ON g.id = le.game_id
		WHERE le.user_id = ?
		ORDER BY le.hours DESC, g.title ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]LibraryItem, 0)
	for rows.Next() {
		it, err := scanLibraryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetLibraryItem returns a single entry owned by the user.
func (s *Store) GetLibraryItem(ctx context.Context, userID, entryID int64) (*LibraryItem, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+libraryCols+`
		FROM library_entries le
		JOIN games g ON g.id = le.game_id
		WHERE le.id = ? AND le.user_id = ?`, entryID, userID)
	it, err := scanLibraryItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// UpdateLibraryEntry updates the user's own fields and auto-stamps start/finish
// dates on the first transition into playing/completed.
func (s *Store) UpdateLibraryEntry(ctx context.Context, userID, entryID int64, status string, rating *int, notes string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE library_entries SET
			status      = ?,
			rating      = ?,
			notes       = ?,
			started_at  = CASE WHEN ? = 'playing'   AND started_at  IS NULL THEN datetime('now') ELSE started_at  END,
			finished_at = CASE WHEN ? = 'completed' AND finished_at IS NULL THEN datetime('now') ELSE finished_at END,
			updated_at  = datetime('now')
		WHERE id = ? AND user_id = ?`,
		status, ratingArg(rating), notes, status, status, entryID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func ratingArg(r *int) any {
	if r == nil {
		return nil
	}
	return *r
}

// Stats aggregates a user's library for the dashboard.
func (s *Store) Stats(ctx context.Context, userID int64) (*Stats, error) {
	st := &Stats{ByStatus: make(map[string]StatusStat, len(AllStatuses))}
	for _, k := range AllStatuses {
		st.ByStatus[k] = StatusStat{}
	}

	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(hours), 0) FROM library_entries WHERE user_id = ?`,
		userID).Scan(&st.TotalGames, &st.TotalHours)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*), COALESCE(SUM(hours), 0)
		FROM library_entries WHERE user_id = ? GROUP BY status`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var ss StatusStat
		if err := rows.Scan(&status, &ss.Count, &ss.Hours); err != nil {
			return nil, err
		}
		st.ByStatus[status] = ss
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(rating), 0), COUNT(rating)
		FROM library_entries WHERE user_id = ? AND rating IS NOT NULL`,
		userID).Scan(&st.AvgRating, &st.RatingCount)
	if err != nil {
		return nil, err
	}

	top, err := s.db.QueryContext(ctx, `
		SELECT g.title, le.hours
		FROM library_entries le JOIN games g ON g.id = le.game_id
		WHERE le.user_id = ? ORDER BY le.hours DESC LIMIT 5`, userID)
	if err != nil {
		return nil, err
	}
	defer top.Close()
	st.TopGames = make([]TopGame, 0, 5)
	for top.Next() {
		var g TopGame
		if err := top.Scan(&g.Title, &g.Hours); err != nil {
			return nil, err
		}
		st.TopGames = append(st.TopGames, g)
	}
	if err := top.Err(); err != nil {
		return nil, err
	}

	// Genres are stored as a JSON array per game; json_each expands them.
	genres, err := s.db.QueryContext(ctx, `
		SELECT j.value, COUNT(*), COALESCE(SUM(le.hours), 0)
		FROM library_entries le
		JOIN games g ON g.id = le.game_id, json_each(g.genres) j
		WHERE le.user_id = ?
		GROUP BY j.value
		ORDER BY SUM(le.hours) DESC, COUNT(*) DESC
		LIMIT 10`, userID)
	if err != nil {
		return nil, err
	}
	defer genres.Close()
	st.Genres = make([]GenreStat, 0)
	for genres.Next() {
		var g GenreStat
		if err := genres.Scan(&g.Name, &g.Count, &g.Hours); err != nil {
			return nil, err
		}
		st.Genres = append(st.Genres, g)
	}
	return st, genres.Err()
}

// --- Sessions ------------------------------------------------------------

func (s *Store) CreateSession(ctx context.Context, id string, userID int64, ttl time.Duration) error {
	expires := time.Now().Add(ttl).UTC().Format("2006-01-02 15:04:05")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expires)
	return err
}

// UserBySession resolves an unexpired session token to its user.
func (s *Store) UserBySession(ctx context.Context, sessionID string) (*User, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id FROM sessions WHERE id = ? AND expires_at > datetime('now')`,
		sessionID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, id)
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

// --- helpers -------------------------------------------------------------

func mapUserConflict(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed: users.username") {
		return ErrUsernameTaken
	}
	if strings.Contains(msg, "UNIQUE constraint failed: users.email") {
		return ErrEmailTaken
	}
	return err
}

// nullable turns an empty string into a SQL NULL so UNIQUE(email) ignores it.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func fallback(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

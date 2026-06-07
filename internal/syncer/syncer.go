// Package syncer imports a user's games from external providers into their library.
package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nachinsec/ludarium/internal/auth"
	"github.com/nachinsec/ludarium/internal/db"
	"github.com/nachinsec/ludarium/internal/igdb"
	"github.com/nachinsec/ludarium/internal/psn"
	"github.com/nachinsec/ludarium/internal/secret"
)

type Syncer struct {
	store  *db.Store
	steam  *auth.Steam
	igdb   *igdb.Client
	psn    *psn.Client
	cipher *secret.Cipher
}

func New(store *db.Store, steam *auth.Steam, igdbClient *igdb.Client, psnClient *psn.Client, cipher *secret.Cipher) *Syncer {
	return &Syncer{store: store, steam: steam, igdb: igdbClient, psn: psnClient, cipher: cipher}
}

type Summary struct {
	Total int `json:"total"`
	Added int `json:"added"`
}

type EnrichSummary struct {
	Checked int `json:"checked"`
	Matched int `json:"matched"`
}

// EnrichSteam backfills IGDB metadata for the user's not-yet-enriched Steam
// games. Rate-limited to stay under IGDB's 4 req/s ceiling.
func (s *Syncer) EnrichSteam(ctx context.Context, userID int64) (EnrichSummary, error) {
	refs, err := s.store.ListUnenrichedSteamGames(ctx, userID)
	if err != nil {
		return EnrichSummary{}, err
	}

	var sum EnrichSummary
	for _, ref := range refs {
		game, err := s.igdb.GameBySteamAppID(ctx, ref.SteamAppID)
		if err != nil {
			continue // transient (e.g. rate limit) — leave it for a later run
		}
		sum.Checked++
		if game == nil {
			_ = s.store.MarkGameEnriched(ctx, ref.ID)
			continue
		}
		genres, _ := json.Marshal(game.Genres)
		details := db.MarshalGameDetails(game.Summary, game.Screenshots, game.Score)
		if err := s.store.UpdateGameMetadata(ctx, ref.ID, game.IGDBID, game.CoverURL,
			game.ReleaseYear, game.Developer, string(genres), details); err == nil {
			sum.Matched++
		}
		time.Sleep(260 * time.Millisecond)
	}
	return sum, nil
}

// Portrait capsule (3:4) — fits the cards; missing ones 404 and fall back in the UI.
func steamCoverURL(appID int) string {
	return fmt.Sprintf(
		"https://cdn.cloudflare.steamstatic.com/steam/apps/%d/library_600x900.jpg", appID)
}

// SyncSteam refreshes playtime without touching the user's status/rating/notes.
func (s *Syncer) SyncSteam(ctx context.Context, userID int64, steamID string) (Summary, error) {
	games, err := s.steam.OwnedGames(ctx, steamID)
	if err != nil {
		return Summary{}, err
	}
	if len(games) == 0 {
		return Summary{}, fmt.Errorf(
			"no games returned — make sure your Steam profile and game details are public")
	}

	before, err := s.store.CountLibrary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	for _, g := range games {
		gameID, err := s.store.UpsertGameBySteamAppID(ctx, g.AppID, g.Name, steamCoverURL(g.AppID))
		if err != nil {
			return Summary{}, fmt.Errorf("upsert game %q: %w", g.Name, err)
		}
		hours := float64(g.PlaytimeMinutes) / 60.0
		if err := s.store.UpsertLibraryEntry(ctx, userID, gameID, hours, "PC / Steam"); err != nil {
			return Summary{}, fmt.Errorf("upsert entry %q: %w", g.Name, err)
		}
	}

	after, err := s.store.CountLibrary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Total: len(games), Added: after - before}, nil
}

// ConnectPSN exchanges a fresh NPSSO for a refresh token and stores it encrypted.
func (s *Syncer) ConnectPSN(ctx context.Context, userID int64, npsso string) error {
	tok, err := s.psn.ExchangeNPSSO(ctx, npsso)
	if err != nil {
		return err
	}
	if tok.AccountID == "" || tok.RefreshToken == "" {
		return fmt.Errorf("PlayStation did not return a usable session")
	}
	enc, err := s.cipher.Encrypt(tok.RefreshToken)
	if err != nil {
		return err
	}
	return s.store.LinkPSN(ctx, userID, tok.AccountID, enc)
}

// SyncPSN imports the user's PlayStation play history (with playtime + platform).
func (s *Syncer) SyncPSN(ctx context.Context, userID int64) (Summary, error) {
	_, encSecret, err := s.store.PSNConnection(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	refreshToken, err := s.cipher.Decrypt(encSecret)
	if err != nil {
		return Summary{}, err
	}

	tok, err := s.psn.Refresh(ctx, refreshToken)
	if err != nil {
		return Summary{}, fmt.Errorf("PlayStation session expired — reconnect it in Settings")
	}
	// PSN may rotate the refresh token; persist the new one.
	if tok.RefreshToken != "" && tok.RefreshToken != refreshToken {
		if enc, e := s.cipher.Encrypt(tok.RefreshToken); e == nil {
			_ = s.store.SetPSNSecret(ctx, userID, enc)
		}
	}

	games, err := s.psn.Library(ctx, tok.AccessToken)
	if err != nil {
		return Summary{}, err
	}

	before, err := s.store.CountLibrary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	for _, g := range games {
		gameID, err := s.resolvePSNGame(ctx, g)
		if err != nil {
			continue
		}
		if err := s.store.UpsertLibraryEntry(ctx, userID, gameID, float64(g.PlaytimeMinutes)/60.0, g.Platform); err != nil {
			return Summary{}, err
		}
	}
	after, err := s.store.CountLibrary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Total: len(games), Added: after - before}, nil
}

// resolvePSNGame matches a PSN title to IGDB by name (for cover/metadata and to
// dedupe with Steam), falling back to a bare game when there's no match.
func (s *Syncer) resolvePSNGame(ctx context.Context, g psn.Game) (int64, error) {
	if s.igdb.Configured() {
		if matches, err := s.igdb.Search(ctx, g.Name); err == nil && len(matches) > 0 {
			m := matches[0]
			genres, _ := json.Marshal(m.Genres)
			details := db.MarshalGameDetails(m.Summary, m.Screenshots, m.Score)
			id, err := s.store.UpsertGameByIGDBID(ctx, m.IGDBID, m.Name, m.CoverURL, m.ReleaseYear, m.Developer, string(genres), details)
			time.Sleep(260 * time.Millisecond) // respect IGDB's rate limit
			if err == nil {
				return id, nil
			}
		}
	}
	return s.store.GetOrCreateBareGame(ctx, g.Name, g.ImageURL)
}

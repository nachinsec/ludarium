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
)

type Syncer struct {
	store *db.Store
	steam *auth.Steam
	igdb  *igdb.Client
}

func New(store *db.Store, steam *auth.Steam, igdbClient *igdb.Client) *Syncer {
	return &Syncer{store: store, steam: steam, igdb: igdbClient}
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
		if err := s.store.UpdateGameMetadata(ctx, ref.ID, game.IGDBID, game.CoverURL,
			game.ReleaseYear, game.Developer, string(genres)); err == nil {
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
		if err := s.store.UpsertLibraryEntry(ctx, userID, gameID, hours); err != nil {
			return Summary{}, fmt.Errorf("upsert entry %q: %w", g.Name, err)
		}
	}

	after, err := s.store.CountLibrary(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Total: len(games), Added: after - before}, nil
}

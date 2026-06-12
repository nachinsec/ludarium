package db

import "encoding/json"

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatarUrl"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
	HasPassword bool   `json:"hasPassword"` // false for steam-only accounts
	Visibility  string `json:"visibility"`  // 'private' | 'public'
}

// UserCard is a member's public-facing summary for the directory.
type UserCard struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	Visibility  string `json:"visibility"`
	GameCount   int    `json:"gameCount"`
	IsFollowing bool   `json:"isFollowing"`
}

// PublicProfile is a member's profile header, viewed by someone else.
type PublicProfile struct {
	UserCard
	Followers int  `json:"followers"`
	Following int  `json:"following"`
	IsSelf    bool `json:"isSelf"`
	CanView   bool `json:"canView"` // false when private and not self
}

// FeedItem is one activity entry: a followed member added/has a game.
type FeedItem struct {
	UserID      int64  `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	Title       string `json:"title"`
	CoverURL    string `json:"coverUrl"`
	Status      string `json:"status"`
	At          string `json:"at"`
}

type Connection struct {
	ID         int64  `json:"id"`
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	CreatedAt  string `json:"createdAt"`
}

// BingoBoard is a user's conference bingo card; Data is opaque JSON owned by
// the frontend (squares + marks + freeCenter).
type BingoBoard struct {
	ID         int64           `json:"id"`
	Title      string          `json:"title"`
	Visibility string          `json:"visibility"`
	Data       json.RawMessage `json:"data"`
	UpdatedAt  string          `json:"updatedAt"`
}

// LibraryItem is a game's metadata joined with the user's personal entry.
type LibraryItem struct {
	ID          int64    `json:"id"`
	IGDBID      int      `json:"igdbId"` // 0 = no IGDB match (manual/unmatched)
	Title       string   `json:"title"`
	CoverURL    string   `json:"coverUrl"`
	Status      string   `json:"status"`
	Rating      *int     `json:"rating"`
	Hours       float64  `json:"hours"`
	Platform    string   `json:"platform"`
	Developer   string   `json:"developer"`
	ReleaseYear *int     `json:"releaseYear"`
	Notes       string   `json:"notes"`
	Genres      []string `json:"genres"`
	StartedAt   *string  `json:"startedAt"`
	FinishedAt  *string  `json:"finishedAt"`
	Summary     string   `json:"summary"`
	Screenshots []string `json:"screenshots"`
	Score       int      `json:"score"` // IGDB community rating 0..100, 0 = none
}

// GameDetails is the richer IGDB metadata stored as JSON in games.details.
type GameDetails struct {
	Summary     string   `json:"summary"`
	Screenshots []string `json:"screenshots"`
	Score       int      `json:"score"`
}

// MarshalGameDetails serializes details for storage.
func MarshalGameDetails(summary string, screenshots []string, score int) string {
	b, _ := json.Marshal(GameDetails{Summary: summary, Screenshots: screenshots, Score: score})
	return string(b)
}

type StatusStat struct {
	Count int     `json:"count"`
	Hours float64 `json:"hours"`
}

type TopGame struct {
	Title string  `json:"title"`
	Hours float64 `json:"hours"`
}

type GenreStat struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Hours float64 `json:"hours"`
}

type Stats struct {
	TotalGames  int                   `json:"totalGames"`
	TotalHours  float64               `json:"totalHours"`
	ByStatus    map[string]StatusStat `json:"byStatus"`
	AvgRating   float64               `json:"avgRating"`
	RatingCount int                   `json:"ratingCount"`
	TopGames    []TopGame             `json:"topGames"`
	Genres      []GenreStat           `json:"genres"`
}

// AllStatuses is the canonical status list/order used across the app.
var AllStatuses = []string{"playing", "ongoing", "completed", "backlog", "dropped", "wishlist"}

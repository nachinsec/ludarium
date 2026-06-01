package db

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatarUrl"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
	HasPassword bool   `json:"hasPassword"` // false for steam-only accounts
}

type Connection struct {
	ID         int64  `json:"id"`
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	CreatedAt  string `json:"createdAt"`
}

// LibraryItem is a game's metadata joined with the user's personal entry.
type LibraryItem struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	CoverURL    string  `json:"coverUrl"`
	Status      string  `json:"status"`
	Rating      *int    `json:"rating"`
	Hours       float64 `json:"hours"`
	Platform    string  `json:"platform"`
	Developer   string  `json:"developer"`
	ReleaseYear *int    `json:"releaseYear"`
	Notes       string  `json:"notes"`
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

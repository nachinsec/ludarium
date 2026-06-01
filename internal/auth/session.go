package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/nachinsec/ludarium/internal/config"
	"github.com/nachinsec/ludarium/internal/db"
)

const (
	cookieName    = "gl_session"
	sessionTTL    = 30 * 24 * time.Hour
	contextUserID ctxKey = iota
)

type ctxKey int

type SessionManager struct {
	store *db.Store
	cfg   *config.Config
}

func NewSessionManager(store *db.Store, cfg *config.Config) *SessionManager {
	return &SessionManager{store: store, cfg: cfg}
}

// Issue creates a session for the user and writes the session cookie.
func (m *SessionManager) Issue(ctx context.Context, w http.ResponseWriter, userID int64) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	if err := m.store.CreateSession(ctx, token, userID, sessionTTL); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.HasPrefix(m.cfg.BaseURL, "https"),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
	return nil
}

// Destroy clears the current session and expires the cookie.
func (m *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = m.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// Middleware loads the authenticated user (if any) into the request context.
func (m *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err == nil && c.Value != "" {
			if user, err := m.store.UserBySession(r.Context(), c.Value); err == nil {
				ctx := context.WithValue(r.Context(), contextUserID, user)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireUser is middleware that rejects unauthenticated requests.
func (m *SessionManager) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext returns the authenticated user, or nil.
func UserFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(contextUserID).(*db.User)
	return u
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

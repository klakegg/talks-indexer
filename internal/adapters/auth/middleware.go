package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/javaBin/talks-indexer/internal/adapters/session"
)

// ContextKey for storing user info in request context
type ContextKey string

const (
	// SessionKey is the context key for the authenticated session
	SessionKey ContextKey = "session"

	sessionCookieName = "session"
	stateCookieName   = "oauth_state"
	returnURLCookie   = "return_url"
)

// GetSession retrieves the session from the context, returns nil if not present
func GetSession(ctx context.Context) *session.Session {
	if sess, ok := ctx.Value(SessionKey).(*session.Session); ok {
		return sess
	}
	return nil
}

// Middleware protects routes with OIDC authentication
type Middleware struct {
	store         session.Store
	authenticator *Authenticator
	secureCookies bool
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(store session.Store, auth *Authenticator, secureCookies bool) *Middleware {
	return &Middleware{
		store:         store,
		authenticator: auth,
		secureCookies: secureCookies,
	}
}

// RequireAuth wraps a handler requiring authentication
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			m.redirectToLogin(w, r)
			return
		}

		sess, err := m.store.Get(r.Context(), cookie.Value)
		if err != nil || sess == nil {
			m.redirectToLogin(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), SessionKey, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// redirectToLogin generates state, stores it, and redirects to OIDC provider
func (m *Middleware) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     returnURLCookie,
		Value:    r.URL.Path,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, m.authenticator.AuthURL(state), http.StatusFound)
}

// generateState generates a cryptographically secure random state parameter
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

package auth

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/javaBin/talks-indexer/internal/adapters/session"
)

// Handler handles auth-related HTTP requests
type Handler struct {
	store         session.Store
	authenticator *Authenticator
	sessionTTL    time.Duration
	secureCookies bool
}

// NewHandler creates a new auth handler
func NewHandler(store session.Store, auth *Authenticator, secureCookies bool) *Handler {
	return &Handler{
		store:         store,
		authenticator: auth,
		sessionTTL:    24 * time.Hour,
		secureCookies: secureCookies,
	}
}

// HandleCallback handles the OIDC callback
func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil {
		slog.ErrorContext(ctx, "missing state cookie")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		slog.ErrorContext(ctx, "state mismatch", "expected", stateCookie.Value, "got", state)
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	h.clearCookie(w, stateCookieName)

	code := r.URL.Query().Get("code")
	if code == "" {
		slog.ErrorContext(ctx, "missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	email, err := h.authenticator.Exchange(ctx, code)
	if err != nil {
		slog.ErrorContext(ctx, "OIDC exchange failed", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	sess, err := h.store.Create(ctx, email, h.sessionTTL)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create session", "error", err)
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		MaxAge:   int(h.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	slog.InfoContext(ctx, "user authenticated", "email", email)

	returnURL := "/admin"
	if cookie, err := r.Cookie(returnURLCookie); err == nil && isValidReturnURL(cookie.Value) {
		returnURL = cookie.Value
	}
	h.clearCookie(w, returnURLCookie)

	http.Redirect(w, r, returnURL, http.StatusFound)
}

// HandleLogout handles user logout
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.store.Delete(ctx, cookie.Value); err != nil {
			slog.ErrorContext(ctx, "failed to delete session", "error", err)
		}
	}

	h.clearCookie(w, sessionCookieName)

	slog.InfoContext(ctx, "user logged out")
	http.Redirect(w, r, "/", http.StatusFound)
}

// clearCookie clears a cookie by setting MaxAge to -1
func (h *Handler) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// isValidReturnURL validates the return URL to prevent open redirects
func isValidReturnURL(url string) bool {
	return strings.HasPrefix(url, "/admin")
}

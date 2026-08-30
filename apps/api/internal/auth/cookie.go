package auth

import (
	"net/http"
	"strings"
	"time"
)

const SessionCookieName = "careeros_session"

// CookieConfig controls session cookie attributes.
type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Path     string
	MaxAge   time.Duration
}

// DefaultCookieConfig returns cookie settings from environment flags.
func DefaultCookieConfig(secure bool, sameSite string) CookieConfig {
	site := http.SameSiteLaxMode
	switch strings.ToLower(sameSite) {
	case "none":
		site = http.SameSiteNoneMode
	case "strict":
		site = http.SameSiteStrictMode
	}
	return CookieConfig{
		Secure:   secure,
		SameSite: site,
		Path:     "/",
		MaxAge:   0, // set per token expiry
	}
}

// SetSessionCookie writes the JWT session cookie.
func SetSessionCookie(w http.ResponseWriter, token string, maxAge time.Duration, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     cfg.Path,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     cfg.Path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

// SessionTokenFromRequest reads the session cookie if present.
func SessionTokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}

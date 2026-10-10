package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/careeros/api/internal/auth"
	"github.com/careeros/api/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type fakeRevocations struct {
	revoked map[string]bool
	err     error
}

func (f fakeRevocations) IsTokenRevoked(_ context.Context, tokenID string) (bool, error) {
	return f.revoked[tokenID], f.err
}

const testSecret = "test-secret-at-least-16-chars"

func newTokens() *auth.TokenManager {
	return auth.NewTokenManager(&config.Config{JWTSecret: testSecret, JWTExpiryHours: 1})
}

func runAuthenticate(t *testing.T, revocations RevocationChecker, token string) (int, auth.Session) {
	t.Helper()
	var session auth.Session
	handler := Authenticate(newTokens(), revocations)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ = auth.SessionFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, session
}

func TestAuthenticate_RevocationAndSession(t *testing.T) {
	token, err := newTokens().Generate(&auth.User{ID: uuid.New(), Email: "student@gram.edu"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := newTokens().Validate(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	code, session := runAuthenticate(t, fakeRevocations{}, token)
	if code != http.StatusNoContent {
		t.Fatalf("valid token: status = %d", code)
	}
	if session.TokenID != claims.ID || session.ExpiresAt.IsZero() {
		t.Fatalf("session not propagated: %+v", session)
	}

	if code, _ := runAuthenticate(t, fakeRevocations{revoked: map[string]bool{claims.ID: true}}, token); code != http.StatusUnauthorized {
		t.Fatalf("revoked token: status = %d, want 401", code)
	}

	if code, _ := runAuthenticate(t, fakeRevocations{err: errors.New("db down")}, token); code != http.StatusInternalServerError {
		t.Fatalf("revocation lookup failure: status = %d, want 500", code)
	}
}

func TestAuthenticate_RejectsTokenWithoutID(t *testing.T) {
	claims := auth.Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if code, _ := runAuthenticate(t, fakeRevocations{}, token); code != http.StatusUnauthorized {
		t.Fatalf("token without jti: status = %d, want 401", code)
	}
}

func TestRequireAdmin(t *testing.T) {
	adminID := uuid.New()
	guard := RequireAdmin(
		map[uuid.UUID]struct{}{adminID: {}},
		map[string]struct{}{"admin@gram.edu": {}},
	)
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name  string
		id    uuid.UUID
		email string
		want  int
	}{
		{"admin by user ID", adminID, "someone@gram.edu", http.StatusNoContent},
		{"admin by email", uuid.New(), "Admin@Gram.edu", http.StatusNoContent},
		{"regular user", uuid.New(), "student@gram.edu", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := auth.WithUserEmail(auth.WithUserID(context.Background(), tt.id), tt.email)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

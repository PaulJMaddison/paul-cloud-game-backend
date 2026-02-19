package testutil

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
)

func TestTimeout(t *testing.T) time.Duration {
	t.Helper()
	v := os.Getenv("TEST_TIMEOUT_SECONDS")
	if v == "" {
		return 10 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		t.Logf("invalid TEST_TIMEOUT_SECONDS=%q, using default 10", v)
		return 10 * time.Second
	}
	return time.Duration(n) * time.Second
}

func Context(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), TestTimeout(t))
}

func MustJWT(t *testing.T, userID, username string) string {
	t.Helper()
	secret := os.Getenv("LOGIN_JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret"
	}
	auth := login.NewAuthenticator(secret, 24*time.Hour)
	tok, err := auth.GenerateToken(userID, username)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}
	return tok
}

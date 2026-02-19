package login

import (
	"testing"
	"time"
)

func TestHashVerifyAndToken(t *testing.T) {
	auth := NewAuthenticator("test-secret", time.Hour)

	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if err := auth.VerifyPassword(hash, "password123"); err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if err := auth.VerifyPassword(hash, "wrong"); err == nil {
		t.Fatal("expected invalid password")
	}

	token, err := auth.GenerateToken("u1", "alice")
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	userID, username, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if userID != "u1" || username != "alice" {
		t.Fatalf("unexpected claims: %s %s", userID, username)
	}
}

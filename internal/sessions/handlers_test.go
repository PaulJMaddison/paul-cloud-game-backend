package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeAuth struct{}

func (fakeAuth) ParseToken(string) (string, string, error) {
	return "user-1", "alice", nil
}

type fakeCreateRepo struct{}

func (fakeCreateRepo) CreateSession(_ context.Context, ownerUserID, status string, _ []string) (Session, error) {
	return Session{ID: "sess-1", OwnerUserID: ownerUserID, Status: status, CreatedAt: time.Now().UTC()}, nil
}

func (fakeCreateRepo) IsMember(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func TestCreateSessionHappyPath(t *testing.T) {
	svc := NewService(fakeCreateRepo{}, fakeAuth{}, nil)
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
	req.Header.Set("Authorization", "Bearer token")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.Code)
	}

	var body map[string]Session
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["session"].ID != "sess-1" {
		t.Fatalf("unexpected session id: %s", body["session"].ID)
	}
}

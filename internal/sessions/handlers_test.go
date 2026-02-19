package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func (fakeCreateRepo) ListUsers(_ context.Context) ([]User, error) {
	return []User{{ID: "user-1", Username: "alice", CreatedAt: time.Now().UTC()}}, nil
}

func (fakeCreateRepo) ListSessions(_ context.Context) ([]Session, error) {
	return []Session{{ID: "sess-1", OwnerUserID: "user-1", Status: "created", CreatedAt: time.Now().UTC()}}, nil
}

func TestCreateSessionHappyPath(t *testing.T) {
	svc := NewService(fakeCreateRepo{}, fakeAuth{}, nil, nil)
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

func TestAdminUsersRequiresToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "dev-admin")
	svc := NewService(fakeCreateRepo{}, fakeAuth{}, nil, nil)
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", res.Code)
	}
}

func TestAdminUsersHappyPath(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "dev-admin")
	svc := NewService(fakeCreateRepo{}, fakeAuth{}, nil, nil)
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	req.Header.Set("X-Admin-Token", "dev-admin")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "alice") {
		t.Fatalf("expected alice in response body: %s", res.Body.String())
	}
}

func TestMain(m *testing.M) {
	_ = os.Unsetenv("ADMIN_TOKEN")
	os.Exit(m.Run())
}

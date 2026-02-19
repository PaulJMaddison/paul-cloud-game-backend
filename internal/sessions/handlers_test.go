package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

type fakeAuth struct{}

func (fakeAuth) ParseToken(string) (string, string, error) { return "user-1", "alice", nil }

type fakeCreateRepo struct {
	createCalls int
	members     []string
}

func (f *fakeCreateRepo) CreateSession(_ context.Context, ownerUserID, status string, members []string) (Session, error) {
	f.createCalls++
	f.members = append([]string(nil), members...)
	return Session{ID: "sess-1", OwnerUserID: ownerUserID, Status: status, CreatedAt: time.Now().UTC()}, nil
}
func (f *fakeCreateRepo) IsMember(_ context.Context, _, _ string) (bool, error) { return true, nil }
func (f *fakeCreateRepo) ListUsers(_ context.Context) ([]User, error) {
	return []User{{ID: "user-1", Username: "alice", CreatedAt: time.Now().UTC()}}, nil
}
func (f *fakeCreateRepo) ListSessions(_ context.Context) ([]Session, error) {
	return []Session{{ID: "sess-1", OwnerUserID: "user-1", Status: "created", CreatedAt: time.Now().UTC()}}, nil
}

func TestCreateSessionHappyPath(t *testing.T) {
	t.Parallel()
	repo := &fakeCreateRepo{}
	svc := NewService(repo, fakeAuth{}, nil, nil)
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
}

func TestAdminUsersRequiresToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "dev-admin")
	svc := NewService(&fakeCreateRepo{}, fakeAuth{}, nil, nil)
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", res.Code)
	}
	var er apierror.Response
	_ = json.Unmarshal(res.Body.Bytes(), &er)
	if er.Code != "unauthorized" {
		t.Fatalf("unexpected code %s", er.Code)
	}
}

func TestAdminUsersHappyPath(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "dev-admin")
	svc := NewService(&fakeCreateRepo{}, fakeAuth{}, nil, nil)
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

func TestHandleMatchedEventCreatesSession(t *testing.T) {
	t.Parallel()
	repo := &fakeCreateRepo{}
	svc := NewService(repo, fakeAuth{}, nil, nil)
	raw, err := contracts.MarshalV1("evt-1", contracts.EventMatchmakingMatched, time.Now().UTC(), "corr-1", nil, contracts.MatchmakingMatchedV1{MatchID: "m1", UserIDs: []string{"u1", "u2"}})
	if err != nil {
		t.Fatal(err)
	}
	svc.HandleMatchedEvent(&nats.Msg{Data: raw})
	if repo.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", repo.createCalls)
	}
}

//go:build integration

package login

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/itest"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

func TestLoginFlowWithRealPostgres(t *testing.T) {
	h := itest.Start(t)
	sqlRaw, _ := os.ReadFile("../../deploy/sql/migrations/001_init.sql")
	itest.RunSQL(t, h.PostgresURL, string(sqlRaw))

	db, err := sql.Open("pgx", h.PostgresURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	nc := itest.NATS(t, h.NATSURL)
	svc := NewService(NewPostgresRepository(db), NewAuthenticator("local-dev-secret", 24*time.Hour), nc)
	hdl := NewHandler(svc)
	mux := http.NewServeMux()
	hdl.Register(mux)

	body := []byte(`{"username":"alice","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", res.Code, res.Body.String())
	}
	var loginResp LoginResponse
	if err := json.Unmarshal(res.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+loginResp.Token)
	resMe := httptest.NewRecorder()
	mux.ServeHTTP(resMe, reqMe)
	if resMe.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", resMe.Code)
	}

	reqBad := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	reqBad.Header.Set("Authorization", "Bearer bad")
	resBad := httptest.NewRecorder()
	mux.ServeHTTP(resBad, reqBad)
	if resBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resBad.Code)
	}
	var er apierror.Response
	_ = json.Unmarshal(resBad.Body.Bytes(), &er)
	if er.Code != "unauthorized" {
		t.Fatalf("expected unauthorized code got %s", er.Code)
	}
}

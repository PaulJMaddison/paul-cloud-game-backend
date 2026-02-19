package login

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

type fakeService struct {
	loginResp LoginResponse
	loginErr  error
	meResp    UserProfile
	meErr     error
	parseErr  error
}

func (f fakeService) Login(context.Context, LoginRequest, string) (LoginResponse, error) {
	return f.loginResp, f.loginErr
}
func (f fakeService) Me(context.Context, string) (UserProfile, error) { return f.meResp, f.meErr }
func (f fakeService) ParseToken(string) (string, string, error) {
	if f.parseErr != nil {
		return "", "", f.parseErr
	}
	return "u1", "alice", nil
}

func TestLoginHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		svc  fakeService
		body string
		code int
		err  string
	}{
		{name: "success", svc: fakeService{loginResp: LoginResponse{Token: "jwt", User: UserProfile{ID: "u1", Username: "alice", CreatedAt: time.Now().UTC()}}}, body: `{"username":"alice","password":"password123"}`, code: http.StatusOK},
		{name: "bad json", svc: fakeService{}, body: `{`, code: http.StatusBadRequest, err: "invalid_json"},
		{name: "validation", svc: fakeService{}, body: `{"username":"ab","password":"short"}`, code: http.StatusBadRequest, err: "validation_failed"},
		{name: "auth failure", svc: fakeService{loginErr: ErrInvalidCredentials}, body: `{"username":"alice","password":"password123"}`, code: http.StatusUnauthorized, err: "invalid_credentials"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := NewHandler(tc.svc)
			mux := http.NewServeMux()
			h.Register(mux)
			req := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewBufferString(tc.body))
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			if res.Code != tc.code {
				t.Fatalf("expected %d got %d", tc.code, res.Code)
			}
			if tc.err != "" {
				var e apierror.Response
				if err := json.Unmarshal(res.Body.Bytes(), &e); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if e.Code != tc.err {
					t.Fatalf("expected code %s got %s", tc.err, e.Code)
				}
			}
		})
	}
}

func TestHandleMeUnauthorized(t *testing.T) {
	t.Parallel()
	h := NewHandler(fakeService{parseErr: errors.New("bad")})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", res.Code)
	}
	var e apierror.Response
	_ = json.Unmarshal(res.Body.Bytes(), &e)
	if e.Code != "unauthorized" {
		t.Fatalf("unexpected error code: %s", e.Code)
	}
}

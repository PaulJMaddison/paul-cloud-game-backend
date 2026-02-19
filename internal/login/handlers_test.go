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

func TestHandleLoginSuccess(t *testing.T) {
	h := NewHandler(fakeService{loginResp: LoginResponse{Token: "jwt", User: UserProfile{ID: "u1", Username: "alice", CreatedAt: time.Now().UTC()}}})
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(LoginRequest{Username: "alice", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.Code)
	}
}

func TestHandleLoginValidation(t *testing.T) {
	h := NewHandler(fakeService{})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/login", bytes.NewBufferString(`{"username":"ab","password":"short"}`))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", res.Code)
	}
}

func TestHandleMeUnauthorized(t *testing.T) {
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
}

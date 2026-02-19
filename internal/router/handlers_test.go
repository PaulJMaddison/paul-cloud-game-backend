package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRouter struct {
	instanceID string
	err        error
}

func (f fakeRouter) Route(context.Context, string, json.RawMessage) (string, error) {
	return f.instanceID, f.err
}

func TestHandleRouteAccepted(t *testing.T) {
	h := NewHandler(fakeRouter{instanceID: "gw-1"})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewBufferString(`{"user_id":"u1","message":{"type":"ping"}}`))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202 got %d", res.Code)
	}
}

func TestHandleRouteOffline(t *testing.T) {
	h := NewHandler(fakeRouter{err: ErrOffline})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewBufferString(`{"user_id":"u1","message":{"type":"ping"}}`))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", res.Code)
	}
}

func TestHandleRouteBadRequest(t *testing.T) {
	h := NewHandler(fakeRouter{err: errors.New("should not be called")})
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewBufferString(`{"message":{"type":"ping"}}`))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", res.Code)
	}
}

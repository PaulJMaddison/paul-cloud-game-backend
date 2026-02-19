package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/apierror"
)

type fakeRouter struct {
	instanceID string
	err        error
}

func (f fakeRouter) Route(context.Context, string, json.RawMessage) (string, error) {
	return f.instanceID, f.err
}

func TestHandleRoute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		r       fakeRouter
		code    int
		errCode string
	}{
		{"accepted", `{"user_id":"u1","message":{"type":"ping"}}`, fakeRouter{instanceID: "gw-1"}, http.StatusAccepted, ""},
		{"offline", `{"user_id":"u1","message":{"type":"ping"}}`, fakeRouter{err: ErrOffline}, http.StatusNotFound, "offline"},
		{"badrequest", `{"message":{"type":"ping"}}`, fakeRouter{}, http.StatusBadRequest, "validation_failed"},
		{"internal", `{"user_id":"u1","message":{"type":"ping"}}`, fakeRouter{err: errors.New("boom")}, http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := NewHandler(tc.r)
			mux := http.NewServeMux()
			h.Register(mux)
			req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewBufferString(tc.body))
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			if res.Code != tc.code {
				t.Fatalf("expected %d got %d", tc.code, res.Code)
			}
			if tc.errCode != "" {
				var e apierror.Response
				_ = json.Unmarshal(res.Body.Bytes(), &e)
				if e.Code != tc.errCode {
					t.Fatalf("expected error code %s got %s", tc.errCode, e.Code)
				}
			}
		})
	}
}

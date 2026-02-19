//go:build integration

package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/itest"
)

func TestRouteWithRealRedisAndNATS(t *testing.T) {
	h := itest.Start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	redis := itest.Redis(t, h.RedisAddr)
	nc := itest.NATS(t, h.NATSURL)

	if err := redis.Set(ctx, "pcgb:user_gateway:u1", "gw-1", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	sub, err := nc.SubscribeSync(SendToUserSubject)
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(NewRedisLookup(redis), nc, false)
	hdl := NewHandler(svc)
	mux := http.NewServeMux()
	hdl.Register(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/route", bytes.NewBufferString(`{"user_id":"u1","message":{"type":"ping"}}`))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusAccepted {
		t.Fatalf("expected 202 got %d", res.Code)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("expected published nats message: %v", err)
	}
	var env Envelope
	if err := json.Unmarshal(msg.Data, &env); err != nil {
		t.Fatal(err)
	}
	if env.UserID != "u1" || env.GatewayInstanceID != "gw-1" {
		t.Fatalf("unexpected payload %+v", env)
	}
}

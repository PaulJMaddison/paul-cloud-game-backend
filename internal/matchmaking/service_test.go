package matchmaking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
)

func TestBuildMatch(t *testing.T) {
	_, ok := BuildMatch([]string{"only-one"}, "m-1")
	if ok {
		t.Fatalf("expected no match for single user")
	}

	result, ok := BuildMatch([]string{"u-1", "u-2", "u-3"}, "m-2")
	if !ok {
		t.Fatalf("expected match for two users")
	}
	if result.UserA != "u-1" || result.UserB != "u-2" || result.MatchID != "m-2" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestEnqueueAndProcessOnce_WithFakeRedisQueue(t *testing.T) {
	ctx := context.Background()
	queue := &fakeRedisQueue{}
	publisher := &fakePublisher{}
	svc := NewService(queue, publisher)
	svc.newID = fixedIDs("evt-1", "evt-2", "match-1", "corr-1", "evt-3", "evt-4")
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

	if err := svc.Enqueue(ctx, "u-1", "corr-enq-1"); err != nil {
		t.Fatalf("enqueue u-1: %v", err)
	}
	if err := svc.Enqueue(ctx, "u-2", "corr-enq-2"); err != nil {
		t.Fatalf("enqueue u-2: %v", err)
	}
	if err := svc.ProcessOnce(ctx); err != nil {
		t.Fatalf("process once: %v", err)
	}

	if len(publisher.events) != 5 {
		t.Fatalf("expected 5 published messages, got %d", len(publisher.events))
	}
	if publisher.events[2].subject != contracts.SubjectMatchmakingMatch {
		t.Fatalf("unexpected subject at #3: %s", publisher.events[2].subject)
	}
	matchedEnv, err := contracts.UnmarshalEnvelope(publisher.events[2].data)
	if err != nil {
		t.Fatalf("unmarshal matched envelope: %v", err)
	}
	matchedPayload, err := contracts.DecodeV1Payload(matchedEnv)
	if err != nil {
		t.Fatalf("decode matched payload: %v", err)
	}
	matched := matchedPayload.(contracts.MatchmakingMatchedV1)
	if matched.MatchID != "match-1" || matched.UserIDs[0] != "u-1" || matched.UserIDs[1] != "u-2" {
		t.Fatalf("unexpected matched payload: %+v", matched)
	}

	userMsgEnv, err := contracts.UnmarshalEnvelope(publisher.events[3].data)
	if err != nil {
		t.Fatalf("unmarshal user msg envelope: %v", err)
	}
	userPayload, err := contracts.DecodeV1Payload(userMsgEnv)
	if err != nil {
		t.Fatalf("decode user msg payload: %v", err)
	}
	gatewayMsg := userPayload.(contracts.GatewaySendToUserV1)
	var body map[string]string
	if err := json.Unmarshal(gatewayMsg.Message, &body); err != nil {
		t.Fatalf("unmarshal gateway message: %v", err)
	}
	if body["type"] != "match_found" || body["other_user_id"] != "u-2" || body["match_id"] != "match-1" {
		t.Fatalf("unexpected gateway body: %+v", body)
	}
}

func TestHTTPEnqueueRequiresJWT(t *testing.T) {
	queue := &fakeRedisQueue{}
	publisher := &fakePublisher{}
	svc := NewService(queue, publisher)
	auth := login.NewAuthenticator("test-secret", time.Hour)
	h := NewHandler(svc, auth)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/v1/matchmaking/enqueue", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rr.Code)
	}

	token, err := auth.GenerateToken("u-99", "player99")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/v1/matchmaking/enqueue", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("expected accepted, got %d", rr2.Code)
	}
	if len(queue.users) != 1 || queue.users[0] != "u-99" {
		t.Fatalf("user not queued correctly: %+v", queue.users)
	}
}

type fakeRedisQueue struct {
	users []string
}

func (f *fakeRedisQueue) Enqueue(_ context.Context, userID string) error {
	f.users = append(f.users, userID)
	return nil
}

func (f *fakeRedisQueue) DequeuePair(_ context.Context) ([]string, error) {
	if len(f.users) < 2 {
		return nil, nil
	}
	pair := []string{f.users[0], f.users[1]}
	f.users = f.users[2:]
	return pair, nil
}

type fakePublisher struct {
	events []published
}

type published struct {
	subject string
	data    []byte
}

func (f *fakePublisher) Publish(subject string, data []byte) error {
	f.events = append(f.events, published{subject: subject, data: data})
	return nil
}

func fixedIDs(ids ...string) func() (string, error) {
	i := 0
	return func() (string, error) {
		if i >= len(ids) {
			return "overflow-id", nil
		}
		id := ids[i]
		i++
		return id, nil
	}
}
